package main

import (
	"context"
	"crypto/rand"
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
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/gen/jetdb/public/model"
	"sidus.io/home-call/postgresdb"
	"strings"
	"time"

	"connectrpc.com/connect"

	. "github.com/go-jet/jet/v2/postgres"
	. "sidus.io/home-call/gen/jetdb/public/table"
)

const (
	callsTopic = "homecall.calls"
)

type Call struct {
	DeviceID    string
	ID          string
	JitsiJwt    string
	JitsiRoomId string
}

func NewOfficeService(db *sql.DB, pubSub message.Publisher) *OfficeService {
	return &OfficeService{
		db:     db,
		pubSub: pubSub,
	}
}

type OfficeService struct {
	db     *sql.DB
	pubSub message.Publisher
}

func randomString() (string, error) {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}
	return string(b), nil
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
		Device.ID,
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

// StartCall starts a call with the specified device.
func (s *OfficeService) StartCall(context.Context, *connect.Request[homecallv1alpha.StartCallRequest]) (*connect.Response[homecallv1alpha.StartCallResponse], error) {
	return nil, errors.New("not implemented")
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err := run(ctx, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context, logger *slog.Logger) error {
	eg, ctx := errgroup.WithContext(ctx)
	db, err := postgresdb.NewDirectConnection(ctx, postgresdb.DirectConfig{
		Hostname: "localhost",
		Port:     "5432",
		UserName: "postgres",
		Password: "password",
		Database: "homecall",
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
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
	callBroadcaster.AddSubscription("homecall.calls")
	eg.Go(func() error {
		err := callBroadcaster.Run(ctx)
		if err != nil {
			return fmt.Errorf("failed to run call-broadcaster: %w", err)
		}
		return nil
	})

	deviceService := NewDeviceService(db, callBroadcaster)
	officeService := NewOfficeService(db, pubSub)

	mux := http.NewServeMux()
	mux.Handle(homecallv1alphaconnect.NewDeviceServiceHandler(deviceService))
	mux.Handle(homecallv1alphaconnect.NewOfficeServiceHandler(officeService))

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck
		w.Write([]byte("pong"))

	})
	server := &http.Server{
		Handler: h2c.NewHandler(mux, &http2.Server{}),
		Addr:    ":8080",
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
