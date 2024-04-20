package deviceapi

import (
	"connectrpc.com/connect"
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/jetdb/public/model"
	"sidus.io/home-call/gen/jetdb/public/table"
	"sidus.io/home-call/messaging"
	"strings"
	"time"
)

func NewService(db *sql.DB, broker *messaging.Broker) *Service {
	return &Service{
		db:     db,
		broker: broker,
	}
}

type Service struct {
	db     *sql.DB
	broker *messaging.Broker
}

func (s *Service) Enroll(ctx context.Context, req *connect.Request[homecallv1alpha.EnrollRequest]) (*connect.Response[homecallv1alpha.EnrollResponse], error) {
	enrollmentStmt := postgres.SELECT(
		table.Enrollment.AllColumns.Except(table.Enrollment.Key),
	).FROM(
		table.Enrollment.
			LEFT_JOIN(table.Device, table.Enrollment.ID.EQ(table.Device.EnrollmentID)),
	).WHERE(
		table.Enrollment.Key.EQ(postgres.String(req.Msg.GetEnrollmentKey())).
			AND(table.Device.ID.IS_NULL()),
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
	deviceInsertStmt := table.Device.INSERT(table.Device.AllColumns.Except(table.Device.ID)).MODEL(
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

	deviceStmt := postgres.SELECT(
		table.Device.PublicKey,
	).FROM(
		table.Device,
	).WHERE(
		table.Device.DeviceID.EQ(postgres.String(deviceId)),
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

	err = verifyDeviceToken(bearerToken, publicKey, deviceId)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
	}

	err = s.broker.SubscribeToCalls(ctx, func(call messaging.Call) error {
		if call.DeviceID != deviceId {
			return nil
		}

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
