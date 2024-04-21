package officeapi

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"errors"
	"fmt"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"log/slog"
	"sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/gen/jetdb/public/model"
	. "sidus.io/home-call/gen/jetdb/public/table"
	"sidus.io/home-call/jitsi"
	"sidus.io/home-call/messaging"
	"sidus.io/home-call/util"
	"time"
)

var _ homecallv1alphaconnect.OfficeServiceHandler = (*Service)(nil)

func NewService(
	db *sql.DB,
	broker *messaging.Broker,
	jitsiApp *jitsi.App,
	logger *slog.Logger,
) *Service {
	return &Service{
		db:       db,
		broker:   broker,
		jitsiApp: jitsiApp,
		logger:   logger,
	}
}

type Service struct {
	db       *sql.DB
	broker   *messaging.Broker
	jitsiApp *jitsi.App
	logger   *slog.Logger
}

func (s *Service) CreateDevice(ctx context.Context, req *connect.Request[homecallv1alpha.CreateDeviceRequest]) (*connect.Response[homecallv1alpha.CreateDeviceResponse], error) {
	// Prepare data
	deviceId := uuid.New().String()

	deviceSettings, err := protojson.Marshal(req.Msg.GetDefaultSettings())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device settings: %w", err)
	}

	enrollmentKey, err := util.RandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate enrollment key: %w", err)
	}

	// Setup transaction
	var txOk bool
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if !txOk {
			err := tx.Rollback()
			s.logger.Error("failed to rollback transaction", "error", err)
		}
	}()

	// Insert device
	insertDeviceStmt := Device.INSERT(Device.DeviceID, Device.Name).VALUES(deviceId, req.Msg.GetName())
	_, err = insertDeviceStmt.ExecContext(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to insert device: %w", err)
	}

	// Insert enrollment
	insertEnrollmentStmt := Enrollment.INSERT(Enrollment.ID, Enrollment.Key, Enrollment.DeviceSettings).QUERY(
		SELECT(Device.ID, String(enrollmentKey), Json(string(deviceSettings))).FROM(Device).WHERE(Device.DeviceID.EQ(String(deviceId))))
	_, err = insertEnrollmentStmt.ExecContext(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to insert enrollment: %w", err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	txOk = true

	return &connect.Response[homecallv1alpha.CreateDeviceResponse]{
		Msg: &homecallv1alpha.CreateDeviceResponse{
			Device: &homecallv1alpha.Device{
				Id:            deviceId,
				Name:          req.Msg.GetName(),
				EnrollmentKey: enrollmentKey,
				Online:        false,
			},
		},
	}, nil
}

func (s *Service) RemoveDevice(ctx context.Context, req *connect.Request[homecallv1alpha.RemoveDeviceRequest]) (*connect.Response[homecallv1alpha.RemoveDeviceResponse], error) {
	device, err := s.getDevice(ctx, req.Msg.GetDeviceId())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	deleteStmt := Device.DELETE().WHERE(Device.DeviceID.EQ(String(device.GetId())))
	_, err = deleteStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to delete device: %w", err)
	}

	return &connect.Response[homecallv1alpha.RemoveDeviceResponse]{
		Msg: &homecallv1alpha.RemoveDeviceResponse{
			Device: device,
		},
	}, nil
}

func (s *Service) getDevice(ctx context.Context, deviceID string) (*homecallv1alpha.Device, error) {
	deviceStmt := SELECT(
		Device.DeviceID,
		Device.Name,
		Device.LastSeen,
	).FROM(Device.LEFT_JOIN(Enrollment, Device.ID.EQ(Enrollment.ID))).
		WHERE(Device.DeviceID.EQ(String(deviceID)))
	var device struct {
		model.Device
		model.Enrollment
	}
	err := deviceStmt.QueryContext(ctx, s.db, &device)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("device not found"))
		}
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	return &homecallv1alpha.Device{
		Id:            device.Device.DeviceID,
		Name:          device.Device.Name,
		EnrollmentKey: device.Enrollment.Key,
		Online:        device.Device.LastSeen != nil && device.Device.LastSeen.After(time.Now().Add(-2*time.Minute)),
	}, nil
}

func (s *Service) WaitForEnrollment(ctx context.Context, req *connect.Request[homecallv1alpha.WaitForEnrollmentRequest], stream *connect.ServerStream[homecallv1alpha.WaitForEnrollmentResponse]) error {
	device, err := s.getDevice(ctx, req.Msg.GetDeviceId())
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	err = s.broker.SubscribeToEnrollment(ctx, device.GetId(), func() error {
		device, err := s.getDevice(ctx, device.GetId())
		if err != nil {
			return fmt.Errorf("failed to get device: %w", err)
		}

		err = stream.Send(&homecallv1alpha.WaitForEnrollmentResponse{
			Device: device,
		})
		if err != nil {
			return fmt.Errorf("failed to send enrollment to client: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to enrollments: %w", err)
	}
	return nil
}

// ListDevices returns a list of all devices.
func (s *Service) ListDevices(ctx context.Context, req *connect.Request[homecallv1alpha.ListDevicesRequest]) (*connect.Response[homecallv1alpha.ListDevicesResponse], error) {
	devicesStmt := SELECT(
		Device.DeviceID,
		Device.Name,
		Device.LastSeen,
		Enrollment.Key,
	).FROM(Device.LEFT_JOIN(Enrollment, Device.ID.EQ(Enrollment.ID)))

	var devices []struct {
		model.Device
		model.Enrollment
	}
	err := devicesStmt.QueryContext(ctx, s.db, &devices)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	var deviceResponses []*homecallv1alpha.Device
	for _, device := range devices {
		deviceResponses = append(deviceResponses, &homecallv1alpha.Device{
			Id:            device.Device.DeviceID,
			Name:          device.Device.Name,
			EnrollmentKey: device.Enrollment.Key,
			Online:        device.Device.LastSeen != nil && device.Device.LastSeen.After(time.Now().Add(-2*time.Minute)),
		})

	}
	return &connect.Response[homecallv1alpha.ListDevicesResponse]{
		Msg: &homecallv1alpha.ListDevicesResponse{
			Devices: deviceResponses,
		},
	}, nil
}

// StartCall starts a call with the specified device.
func (s *Service) StartCall(ctx context.Context, req *connect.Request[homecallv1alpha.StartCallRequest]) (*connect.Response[homecallv1alpha.StartCallResponse], error) {
	device, err := s.getDevice(ctx, req.Msg.GetDeviceId())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create jitsi room
	jitsiCall, err := s.jitsiApp.NewCall()
	if err != nil {
		return nil, fmt.Errorf("failed to create call: %w", err)
	}

	// Create jitsi jwt for office
	officeToken, err := jitsiCall.OfficeJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to create office token: %w", err)
	}

	// Create jitsi jwt for device
	deviceToken, err := jitsiCall.DeviceJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to create device token: %w", err)
	}

	// Broadcast call
	call := messaging.Call{
		DeviceID:    device.GetId(),
		ID:          uuid.New().String(),
		JitsiJwt:    deviceToken,
		JitsiRoomId: jitsiCall.RoomName(),
	}

	err = s.broker.PublishCall(call)
	if err != nil {
		return nil, fmt.Errorf("failed to publish call: %w", err)
	}

	return &connect.Response[homecallv1alpha.StartCallResponse]{
		Msg: &homecallv1alpha.StartCallResponse{
			CallId:      call.ID,
			JitsiJwt:    officeToken,
			JitsiRoomId: jitsiCall.RoomName(),
		},
	}, nil
}
