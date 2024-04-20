package officeapi

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"fmt"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/jetdb/public/model"
	"sidus.io/home-call/gen/jetdb/public/table"
	"sidus.io/home-call/jitsi"
	"sidus.io/home-call/messaging"
	"sidus.io/home-call/util"
)

func NewOfficeService(
	db *sql.DB,
	broker *messaging.Broker,
	jitsiApp *jitsi.App,
) *OfficeService {
	return &OfficeService{
		db:       db,
		broker:   broker,
		jitsiApp: jitsiApp,
	}
}

type OfficeService struct {
	db       *sql.DB
	broker   *messaging.Broker
	jitsiApp *jitsi.App
}

// EnrollDevice starts the enrollment process for a new device.
func (s *OfficeService) EnrollDevice(ctx context.Context, req *connect.Request[homecallv1alpha.EnrollDeviceRequest]) (*connect.Response[homecallv1alpha.EnrollDeviceResponse], error) {
	enrollmentKey, err := util.RandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate enrollment key: %w", err)
	}

	deviceSettings, err := protojson.Marshal(req.Msg.GetSettings())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device settings: %w", err)
	}

	insertEnrollmentStmt := table.Enrollment.INSERT(table.Enrollment.Key, table.Enrollment.DeviceName, table.Enrollment.DeviceSettings).MODEL(
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
	devicesStmt := postgres.SELECT(
		table.Device.DeviceID,
		table.Device.Name,
	).FROM(
		table.Device,
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
func (s *OfficeService) StartCall(ctx context.Context, req *connect.Request[homecallv1alpha.StartCallRequest]) (*connect.Response[homecallv1alpha.StartCallResponse], error) {
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
		DeviceID:    req.Msg.GetDeviceId(),
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
