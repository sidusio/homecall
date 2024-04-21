package deviceapi

import (
	"connectrpc.com/connect"
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"fmt"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"log/slog"
	"sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/gen/jetdb/public/model"
	. "sidus.io/home-call/gen/jetdb/public/table"
	"sidus.io/home-call/messaging"
	"strings"
	"time"
)

var _ homecallv1alphaconnect.DeviceServiceHandler = (*Service)(nil)

func NewService(db *sql.DB, broker *messaging.Broker, logger *slog.Logger) *Service {
	return &Service{
		db:     db,
		broker: broker,
		logger: logger,
	}
}

type Service struct {
	db     *sql.DB
	broker *messaging.Broker
	logger *slog.Logger
}

func (s *Service) Enroll(ctx context.Context, req *connect.Request[homecallv1alpha.EnrollRequest]) (*connect.Response[homecallv1alpha.EnrollResponse], error) {
	enrollmentStmt := SELECT(
		Enrollment.ID,
		Enrollment.Key,
		Enrollment.DeviceSettings,
		Device.DeviceID,
		Device.Name,
	).FROM(
		Enrollment.
			LEFT_JOIN(Device, Enrollment.ID.EQ(Device.ID)),
	).WHERE(
		Enrollment.Key.EQ(String(req.Msg.GetEnrollmentKey())),
	).LIMIT(1)

	var enrollment struct {
		model.Enrollment
		model.Device
	}
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

	deviceUpdateStmt := Device.UPDATE().SET(
		Device.PublicKey.SET(String(req.Msg.GetPublicKey())),
	).WHERE(Device.ID.EQ(Enrollment.ID))

	_, err = deviceUpdateStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to insert device: %w", err)
	}

	enrollmentDeleteStmt := Enrollment.DELETE().WHERE(Enrollment.ID.EQ(Int32(enrollment.Enrollment.ID)))
	_, err = enrollmentDeleteStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to delete enrollment: %w", err)
	}

	err = s.broker.PublishEnrollment(enrollment.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to publish enrollment: %w", err)
	}

	return &connect.Response[homecallv1alpha.EnrollResponse]{
		Msg: &homecallv1alpha.EnrollResponse{
			DeviceId: enrollment.DeviceID,
			Settings: &deviceSettings,
			Name:     enrollment.Name,
		},
	}, nil
}

func (s *Service) WaitForCall(ctx context.Context, req *connect.Request[homecallv1alpha.WaitForCallRequest], stream *connect.ServerStream[homecallv1alpha.WaitForCallResponse]) error {
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

	if device.PublicKey == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("device not enrolled"))
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(*device.PublicKey))
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	err = verifyDeviceToken(bearerToken, publicKey, deviceId)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
	}

	setLastSeen := func() {
		now := time.Now()
		timestamp := Timestamp(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
		deviceUpdateStmt := Device.UPDATE().SET(
			Device.LastSeen.SET(timestamp),
		).WHERE(Device.DeviceID.EQ(String(deviceId)))

		_, err = deviceUpdateStmt.ExecContext(ctx, s.db)
		if err != nil {
			s.logger.Error("failed to update last seen", "error", err)
		}
	}

	go func() {
		setLastSeen()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				setLastSeen()
			}
		}
	}()

	err = s.broker.SubscribeToCalls(ctx, deviceId, func(call messaging.Call) error {
		err = stream.Send(&homecallv1alpha.WaitForCallResponse{
			CallId:      call.ID,
			JitsiJwt:    call.JitsiJwt,
			JitsiRoomId: call.JitsiRoomId,
		})
		if err != nil {
			return fmt.Errorf("failed to send call to client: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to calls: %w", err)
	}
	return nil
}

func verifyDeviceToken(token string, publicKey *rsa.PublicKey, deviceId string) error {
	_, err := jwt.Parse(
		token,
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
		return fmt.Errorf("invalid token: %w", err)
	}
	return nil
}
