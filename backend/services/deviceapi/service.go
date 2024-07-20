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
	jose "gopkg.in/go-jose/go-jose.v2/jwt"
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
		Device.ID,
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
	).WHERE(Device.ID.EQ(Int32(enrollment.Device.ID)))

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

func (s *Service) UpdateNotificationToken(ctx context.Context, req *connect.Request[homecallv1alpha.UpdateNotificationTokenRequest]) (*connect.Response[homecallv1alpha.UpdateNotificationTokenResponse], error) {
	deviceId, err := s.verifyDeviceToken(ctx, req)
	if err != nil {
		cErr := &connect.Error{}
		if errors.As(err, &cErr) {
			return nil, err
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	deviceIdExpression := SELECT(Device.ID).FROM(Device).WHERE(Device.DeviceID.EQ(String(deviceId))).LIMIT(1)

	updateStmt := DeviceNotificationToken.
		INSERT(DeviceNotificationToken.DeviceID, DeviceNotificationToken.NotificationToken, DeviceNotificationToken.UpdatedAt).
		VALUES(deviceIdExpression, req.Msg.GetNotificationToken(), CAST(NOW()).AS_TIMESTAMP()).
		ON_CONFLICT(DeviceNotificationToken.DeviceID).
		DO_UPDATE(SET(
			DeviceNotificationToken.NotificationToken.SET(String(req.Msg.GetNotificationToken())),
			DeviceNotificationToken.UpdatedAt.SET(CAST(NOW()).AS_TIMESTAMP()),
		))

	_, err = updateStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to update fcm token: %w", err)
	}

	return &connect.Response[homecallv1alpha.UpdateNotificationTokenResponse]{
		Msg: &homecallv1alpha.UpdateNotificationTokenResponse{},
	}, nil
}

func (s *Service) GetCallDetails(ctx context.Context, req *connect.Request[homecallv1alpha.GetCallDetailsRequest]) (*connect.Response[homecallv1alpha.GetCallDetailsResponse], error) {
	deviceId, err := s.verifyDeviceToken(ctx, req)
	if err != nil {
		cErr := &connect.Error{}
		if errors.As(err, &cErr) {
			return nil, err
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	callStmt := SELECT(DeviceCallOutbox.JitsiJwt, DeviceCallOutbox.JitsiRoomID).
		FROM(DeviceCallOutbox.LEFT_JOIN(Device, DeviceCallOutbox.DeviceID.EQ(Device.ID))).
		WHERE(
			Device.DeviceID.EQ(String(deviceId)).
				AND(DeviceCallOutbox.CallID.EQ(String(req.Msg.GetCallId()))).
				AND(DeviceCallOutbox.CreatedAt.GT(CAST(NOW()).AS_TIMESTAMP().SUB(INTERVALd(time.Hour)))),
		).LIMIT(1)

	var call struct {
		model.DeviceCallOutbox
	}
	err = callStmt.QueryContext(ctx, s.db, &call)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("call not found"))
		}
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	return &connect.Response[homecallv1alpha.GetCallDetailsResponse]{
		Msg: &homecallv1alpha.GetCallDetailsResponse{
			JitsiJwt:    call.JitsiJwt,
			JitsiRoomId: call.JitsiRoomID,
			CallId:      req.Msg.GetCallId(),
		},
	}, nil
}

func (s *Service) verifyDeviceToken(ctx context.Context, req connect.AnyRequest) (string, error) {
	// Verify bearer token
	bearerToken := strings.TrimSpace(
		strings.TrimPrefix(
			req.Header().Get("Authorization"),
			"Bearer "),
	)
	if bearerToken == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("missing bearer token"))
	}

	parsedToken, err := jose.ParseSigned(bearerToken)
	if err != nil {
		return "", fmt.Errorf("could not parse the token: %w", err)
	}

	unsafeClaims := jose.Claims{}
	err = parsedToken.UnsafeClaimsWithoutVerification(&unsafeClaims)
	if err != nil {
		return "", fmt.Errorf("could not parse the claims: %w", err)
	}

	deviceId := unsafeClaims.Subject
	if deviceId == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("missing device id"))
	}

	deviceStmt := SELECT(
		Device.PublicKey,
	).FROM(
		Device,
	).WHERE(
		Device.DeviceID.EQ(String(deviceId)),
	).LIMIT(1)

	var device model.Device
	err = deviceStmt.QueryContext(ctx, s.db, &device)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return "", connect.NewError(connect.CodeUnauthenticated, errors.New("invalid device id"))
		}
		return "", fmt.Errorf("failed to query database: %w", err)

	}

	if device.PublicKey == nil {
		return "", connect.NewError(connect.CodeFailedPrecondition, errors.New("device not enrolled"))
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(*device.PublicKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	err = verifyDeviceToken(bearerToken, publicKey, deviceId)
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
	}
	return deviceId, nil
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
