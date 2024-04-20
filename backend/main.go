package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/gen/jetdb/public/model"
	"sidus.io/home-call/migrations"
	"sidus.io/home-call/postgresdb"
	"strings"
	"time"

	"connectrpc.com/connect"

	. "github.com/go-jet/jet/v2/postgres"
	. "sidus.io/home-call/gen/jetdb/public/table"
)

const (
	callsTopic = "homecall.calls"
	appName    = "homecall"
)

type Config struct {
	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     string `envconfig:"DB_PORT" default:"5432"`
	DBUser     string `envconfig:"DB_USER" default:"homecall"`
	DBPassword string `envconfig:"DB_PASSWORD" required:"true"`
	DBName     string `envconfig:"DB_NAME" default:"homecall"`

	Port string `envconfig:"PORT" default:"8080"`

	JitsiAppId   string `envconfig:"JITSI_APP_ID" required:"true"`
	JitsiKeyId   string `envconfig:"JITSI_KEY_ID" required:"true"`
	JitsiKeyFile string `envconfig:"JITSI_KEY_FILE" required:"true"`
}

type Call struct {
	DeviceID    string
	ID          string
	JitsiJwt    string
	JitsiRoomId string
}

func NewOfficeService(
	db *sql.DB,
	pubSub message.Publisher,
	jitsiAppId string,
	jitsiKeyId string,
	jitsiPrivateKey *rsa.PrivateKey,
) *OfficeService {
	return &OfficeService{
		db:              db,
		pubSub:          pubSub,
		jitsiAppId:      jitsiAppId,
		jitsiKeyId:      jitsiKeyId,
		jitsiPrivateKey: jitsiPrivateKey,
	}
}

type OfficeService struct {
	db              *sql.DB
	pubSub          message.Publisher
	jitsiAppId      string
	jitsiKeyId      string
	jitsiPrivateKey *rsa.PrivateKey
}

func randomString() (string, error) {
	validBytes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := strings.Builder{}
	b.Grow(32)
	for i := 0; i < 32; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(validBytes))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		b.WriteByte(validBytes[idx.Int64()])
	}
	return b.String(), nil
}

// EnrollDevice starts the enrollment process for a new device.
func (s *OfficeService) EnrollDevice(ctx context.Context, req *connect.Request[homecallv1alpha.EnrollDeviceRequest]) (*connect.Response[homecallv1alpha.EnrollDeviceResponse], error) {
	enrollmentKey, err := randomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate enrollment key: %w", err)
	}

	deviceSettings, err := protojson.Marshal(req.Msg.GetSettings())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device settings: %w", err)
	}

	insertEnrollmentStmt := Enrollment.INSERT(Enrollment.Key, Enrollment.DeviceName, Enrollment.DeviceSettings).MODEL(
		&model.Enrollment{
			Key:            enrollmentKey,
			DeviceName:     req.Msg.GetName(),
			DeviceSettings: string(deviceSettings),
		},
	)

	_, err = insertEnrollmentStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to insert enrollment: %w", err)
	}
	return &connect.Response[homecallv1alpha.EnrollDeviceResponse]{
		Msg: &homecallv1alpha.EnrollDeviceResponse{
			EnrollmentKey: enrollmentKey,
		},
	}, nil
}

// ListDevices returns a list of all devices.
func (s *OfficeService) ListDevices(ctx context.Context, req *connect.Request[homecallv1alpha.ListDevicesRequest]) (*connect.Response[homecallv1alpha.ListDevicesResponse], error) {
	devicesStmt := SELECT(
		Device.DeviceID,
		Device.Name,
	).FROM(
		Device,
	)

	var devices []model.Device
	err := devicesStmt.QueryContext(ctx, s.db, &devices)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	var deviceResponses []*homecallv1alpha.Device
	for _, device := range devices {
		deviceResponses = append(deviceResponses, &homecallv1alpha.Device{
			Id:   device.DeviceID,
			Name: device.Name,
		})

	}
	return &connect.Response[homecallv1alpha.ListDevicesResponse]{
		Msg: &homecallv1alpha.ListDevicesResponse{
			Devices: deviceResponses,
		},
	}, nil
}

type JitsiClaims struct {
	Room    string            `json:"room"`
	Context JitsiClaimContext `json:"context"`
	jwt.RegisteredClaims
	Audience string `json:"aud"`
}

type JitsiClaimContext struct {
	User     JitsiClaimUser     `json:"user"`
	Features JitsiClaimFeatures `json:"features"`
}

type JitsiClaimUser struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Avatar             string `json:"avatar"`
	Email              string `json:"email"`
	Moderator          bool   `json:"moderator"`
	HiddenFromRecorder bool   `json:"hidden-from-recorder"`
}

type JitsiClaimFeatures struct {
	Livestreaming bool `json:"livestreaming"`
	OutboundCall  bool `json:"outbound-call"`
	Transcription bool `json:"transcription"`
	Recording     bool `json:"recording"`
}

// StartCall starts a call with the specified device.
func (s *OfficeService) StartCall(ctx context.Context, req *connect.Request[homecallv1alpha.StartCallRequest]) (*connect.Response[homecallv1alpha.StartCallResponse], error) {
	// Create jitsi room
	roomName, err := randomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate room name: %w", err)
	}

	// Create jitsi jwt for office
	officeToken := jwt.NewWithClaims(jwt.SigningMethodRS256, JitsiClaims{
		Room: roomName,
		Context: JitsiClaimContext{
			User: JitsiClaimUser{
				ID:                 "office",
				Name:               "office",
				Avatar:             "",
				Email:              "",
				Moderator:          false,
				HiddenFromRecorder: true,
			},
			Features: JitsiClaimFeatures{
				Livestreaming: false,
				OutboundCall:  false,
				Transcription: false,
				Recording:     false,
			},
		},
		RegisteredClaims: jwt.RegisteredClaims{
			// Jitsi requires audience to be set as a string
			//Audience:  []string{"jitsi"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chat",
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   s.jitsiAppId,
		},
		Audience: "jitsi",
	})
	officeToken.Header["kid"] = s.jitsiKeyId

	signedOfficeToken, err := officeToken.SignedString(s.jitsiPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign office token: %w", err)
	}

	// Create jitsi jwt for device
	deviceToken := jwt.NewWithClaims(jwt.SigningMethodRS256, JitsiClaims{
		Room: roomName,
		Context: JitsiClaimContext{
			User: JitsiClaimUser{
				ID:                 req.Msg.GetDeviceId(),
				Name:               "Johan",
				Avatar:             "",
				Email:              "",
				Moderator:          false,
				HiddenFromRecorder: true,
			},
			Features: JitsiClaimFeatures{
				Livestreaming: false,
				OutboundCall:  false,
				Transcription: false,
				Recording:     false,
			},
		},
		RegisteredClaims: jwt.RegisteredClaims{
			// Jitsi requires audience to be set as a string
			//Audience:  []string{"jitsi"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chat",
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   s.jitsiAppId,
		},
		Audience: "jitsi",
	})
	deviceToken.Header["kid"] = s.jitsiKeyId

	signedDeviceToken, err := deviceToken.SignedString(s.jitsiPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign office token: %w", err)
	}

	// Broadcast call
	call := Call{
		DeviceID:    req.Msg.GetDeviceId(),
		ID:          uuid.New().String(),
		JitsiJwt:    signedDeviceToken,
		JitsiRoomId: fmt.Sprintf("%s/%s", s.jitsiAppId, roomName),
	}

	callJson, err := json.Marshal(call)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal call: %w", err)
	}

	err = s.pubSub.Publish(callsTopic, message.NewMessage(uuid.New().String(), callJson))
	if err != nil {
		return nil, fmt.Errorf("failed to publish call: %w", err)
	}

	return &connect.Response[homecallv1alpha.StartCallResponse]{
		Msg: &homecallv1alpha.StartCallResponse{
			CallId:      call.ID,
			JitsiJwt:    signedOfficeToken,
			JitsiRoomId: fmt.Sprintf("%s/%s", s.jitsiAppId, roomName),
		},
	}, nil
}

func NewDeviceService(db *sql.DB, callsBroadcaster message.Subscriber) *DeviceService {
	return &DeviceService{
		db:               db,
		callsBroadcaster: callsBroadcaster,
	}
}

type DeviceService struct {
	db               *sql.DB
	callsBroadcaster message.Subscriber
}

func (s *DeviceService) Enroll(ctx context.Context, req *connect.Request[homecallv1alpha.EnrollRequest]) (*connect.Response[homecallv1alpha.EnrollResponse], error) {
	enrollmentStmt := SELECT(
		Enrollment.AllColumns.Except(Enrollment.Key),
	).FROM(
		Enrollment.
			LEFT_JOIN(Device, Enrollment.ID.EQ(Device.EnrollmentID)),
	).WHERE(
		Enrollment.Key.EQ(String(req.Msg.GetEnrollmentKey())).
			AND(Device.ID.IS_NULL()),
	).LIMIT(1)

	var enrollment model.Enrollment
	err := enrollmentStmt.QueryContext(ctx, s.db, &enrollment)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid enrollment key"))
		}
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	var deviceSettings homecallv1alpha.DeviceSettings
	err = protojson.Unmarshal([]byte(enrollment.DeviceSettings), &deviceSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal device settings: %w", err)
	}

	deviceId := uuid.New().String()
	deviceInsertStmt := Device.INSERT(Device.AllColumns.Except(Device.ID)).MODEL(
		&model.Device{
			EnrollmentID: enrollment.ID,
			DeviceID:     deviceId,
			Name:         enrollment.DeviceName,
			PublicKey:    req.Msg.GetPublicKey(),
		})

	_, err = deviceInsertStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to insert device: %w", err)
	}

	return &connect.Response[homecallv1alpha.EnrollResponse]{
		Msg: &homecallv1alpha.EnrollResponse{
			DeviceId: deviceId,
			Settings: &deviceSettings,
		},
	}, nil
}

func (s *DeviceService) WaitForCall(ctx context.Context, req *connect.Request[homecallv1alpha.WaitForCallRequest], stream *connect.ServerStream[homecallv1alpha.WaitForCallResponse]) error {
	// Verify bearer token
	bearerToken := strings.TrimSpace(
		strings.TrimPrefix(
			req.Header().Get("Authorization"),
			"Bearer "),
	)
	if bearerToken == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing bearer token"))
	}

	deviceId := req.Msg.GetDeviceId()
	if deviceId == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("missing device id"))
	}

	deviceStmt := SELECT(
		Device.PublicKey,
	).FROM(
		Device,
	).WHERE(
		Device.DeviceID.EQ(String(deviceId)),
	).LIMIT(1)

	var device model.Device
	err := deviceStmt.QueryContext(ctx, s.db, &device)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid device id"))
		}
		return fmt.Errorf("failed to query database: %w", err)

	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(device.PublicKey))
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Get subject from token
	_, err = jwt.Parse(
		bearerToken,
		func(token *jwt.Token) (interface{}, error) {
			return publicKey, nil
		},
		jwt.WithAudience("homecall"),
		jwt.WithIssuer("homecall-device"),
		jwt.WithIssuedAt(),
		jwt.WithSubject(deviceId),
		jwt.WithLeeway(time.Second*30),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{
			jwt.SigningMethodRS256.Alg(),
			jwt.SigningMethodRS384.Alg(),
			jwt.SigningMethodRS512.Alg(),
		}),
	)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
	}

	callChan, err := s.callsBroadcaster.Subscribe(ctx, callsTopic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to calls: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-callChan:
			if !ok {
				return nil
			}

			var call Call
			err := json.Unmarshal(msg.Payload, &call)
			if err != nil {
				msg.Nack()
				return fmt.Errorf("failed to unmarshal call: %w", err)
			}

			if call.DeviceID != deviceId {
				msg.Ack()
				continue
			}

			err = stream.Send(&homecallv1alpha.WaitForCallResponse{
				CallId:      call.ID,
				JitsiJwt:    call.JitsiJwt,
				JitsiRoomId: call.JitsiRoomId,
			})
			if err != nil {
				msg.Nack()
				return fmt.Errorf("failed to send call: %w", err)
			}
		}
	}
}

func main() {
	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt)
	go func(ctx context.Context, cleanup context.CancelFunc) {
		<-ctx.Done()
		cleanup()
	}(ctx, cleanup)

	var cfg Config
	err := envconfig.Process(appName, &cfg)
	if err != nil {
		slog.Error("failed to process env vars", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err = run(ctx, logger, cfg)
	if err != nil {
		logger.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context, logger *slog.Logger, cfg Config) error {
	eg, ctx := errgroup.WithContext(ctx)
	db, err := postgresdb.NewDirectConnection(ctx, postgresdb.DirectConfig{
		Hostname: cfg.DBHost,
		Port:     cfg.DBPort,
		UserName: cfg.DBUser,
		Password: cfg.DBPassword,
		Database: cfg.DBName,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	migrator, err := postgresdb.NewMigrator(ctx, db, logger, postgresdb.MigrationConfig{
		ApplyMigrations: true,
		MigrationsFS:    migrations.Migrations,
		MigrationsPath:  ".",
	})
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	err = migrator.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	pubSub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            0,
		Persistent:                     false,
		BlockPublishUntilSubscriberAck: false,
	}, watermill.NewSlogLogger(logger))

	callBroadcaster, err := gochannel.NewFanOut(pubSub, watermill.NewSlogLogger(logger))
	if err != nil {
		return fmt.Errorf("failed to create call broadcaster: %w", err)
	}
	callBroadcaster.AddSubscription(callsTopic)
	eg.Go(func() error {
		err := callBroadcaster.Run(ctx)
		if err != nil {
			return fmt.Errorf("failed to run call-broadcaster: %w", err)
		}
		return nil
	})

	jitsiKeyData, err := os.ReadFile(cfg.JitsiKeyFile)
	if err != nil {
		return fmt.Errorf("failed to read jitsi key file: %w", err)
	}
	jitsiKey, err := jwt.ParseRSAPrivateKeyFromPEM(jitsiKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse jitsi key: %w", err)
	}

	deviceService := NewDeviceService(db, callBroadcaster)
	officeService := NewOfficeService(
		db,
		pubSub,
		cfg.JitsiAppId,
		cfg.JitsiKeyId,
		jitsiKey,
	)

	mux := http.NewServeMux()
	mux.Handle(homecallv1alphaconnect.NewDeviceServiceHandler(deviceService))
	mux.Handle(homecallv1alphaconnect.NewOfficeServiceHandler(officeService))

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck
		w.Write([]byte("pong"))

	})
	server := &http.Server{
		Handler: h2c.NewHandler(mux, &http2.Server{}),
		Addr:    fmt.Sprintf(":%s", cfg.Port),
	}
	eg.Go(func() error {
		err := listenAndServe(ctx, server, 15*time.Second)
		if err != nil {
			return fmt.Errorf("failed to listen and serve: %w", err)
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("application crashed: %w", err)
	}
	return nil
}

func listenAndServe(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	eg.Go(func() error {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to listen and serve: %w", err)
		}
		return nil
	})

	return eg.Wait()
}
