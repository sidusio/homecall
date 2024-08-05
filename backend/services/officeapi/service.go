package officeapi

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"errors"
	fm "firebase.google.com/go/v4/messaging"
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
	"sidus.io/home-call/notifications"
	"sidus.io/home-call/services/tenantapi"
	"sidus.io/home-call/util"
	"time"
)

var _ homecallv1alphaconnect.OfficeServiceHandler = (*Service)(nil)

func NewService(
	db *sql.DB,
	broker *messaging.Broker,
	jitsiApp *jitsi.App,
	logger *slog.Logger,
	tenantService *tenantapi.Service,
	notificationService notifications.Service,
) *Service {
	return &Service{
		db:                  db,
		broker:              broker,
		jitsiApp:            jitsiApp,
		logger:              logger,
		tenantService:       tenantService,
		notificationService: notificationService,
	}
}

type Service struct {
	db                  *sql.DB
	broker              *messaging.Broker
	jitsiApp            *jitsi.App
	logger              *slog.Logger
	tenantService       *tenantapi.Service
	notificationService notifications.Service
}

func (s *Service) CreateDevice(ctx context.Context, req *connect.Request[homecallv1alpha.CreateDeviceRequest]) (*connect.Response[homecallv1alpha.CreateDeviceResponse], error) {
	err := s.tenantService.CanAccessTenant(ctx, req.Msg.GetTenantId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

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

	err = util.WithTransaction(s.db, func(tx util.DB) error {
		// Insert device
		insertDeviceStmt := Device.INSERT(Device.DeviceID, Device.Name, Device.TenantID).VALUES(
			deviceId,
			req.Msg.GetName(),
			SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId()))).LIMIT(1),
		)
		_, err = insertDeviceStmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to insert device: %w", err)
		}

		// Insert enrollment
		insertEnrollmentStmt := Enrollment.INSERT(Enrollment.ID, Enrollment.Key, Enrollment.DeviceSettings).QUERY(
			SELECT(Device.ID, String(enrollmentKey), Json(string(deviceSettings))).FROM(Device).WHERE(Device.DeviceID.EQ(String(deviceId))))
		_, err = insertEnrollmentStmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to insert enrollment: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[homecallv1alpha.CreateDeviceResponse]{
		Msg: &homecallv1alpha.CreateDeviceResponse{
			Device: &homecallv1alpha.Device{
				Id:            deviceId,
				Name:          req.Msg.GetName(),
				EnrollmentKey: enrollmentKey,
				Online:        false,
				TenantId:      req.Msg.GetTenantId(),
			},
		},
	}, nil
}

func (s *Service) UpdateDevice(ctx context.Context, req *connect.Request[homecallv1alpha.UpdateDeviceRequest]) (*connect.Response[homecallv1alpha.UpdateDeviceResponse], error) {
	err := s.tenantService.CanAccessDevice(ctx, req.Msg.GetDeviceId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access device: %w", err)
	}

	device, err := s.getDevice(ctx, req.Msg.GetDeviceId())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	newName := req.Msg.GetName()

	updateStmt := Device.UPDATE().SET(Device.Name.SET(String(newName))).WHERE(Device.DeviceID.EQ(String(device.GetId())))
	_, err = updateStmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	device.Name = newName

	return &connect.Response[homecallv1alpha.UpdateDeviceResponse]{
		Msg: &homecallv1alpha.UpdateDeviceResponse{
			Device: device,
		},
	}, nil
}

func (s *Service) RemoveDevice(ctx context.Context, req *connect.Request[homecallv1alpha.RemoveDeviceRequest]) (*connect.Response[homecallv1alpha.RemoveDeviceResponse], error) {
	err := s.tenantService.CanAccessDevice(ctx, req.Msg.GetDeviceId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access device: %w", err)
	}

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
		Enrollment.DeviceSettings,
		Enrollment.Key,
		Tenant.TenantID,
		DeviceNotificationToken.UpdatedAt.IS_NOT_NULL().
			AND(DeviceNotificationToken.UpdatedAt.GT(CAST(NOW()).AS_TIMESTAMP().SUB(INTERVALd(time.Hour)))).
			AS("Online"),
	).FROM(
		Device.
			LEFT_JOIN(Enrollment, Device.ID.EQ(Enrollment.ID)).
			LEFT_JOIN(Tenant, Device.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(DeviceNotificationToken, Device.ID.EQ(DeviceNotificationToken.DeviceID)),
	).WHERE(Device.DeviceID.EQ(String(deviceID)))
	var device struct {
		model.Device
		model.Enrollment
		model.Tenant
		Online bool
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
		Online:        device.Online,
		TenantId:      device.Tenant.TenantID,
	}, nil
}

func (s *Service) WaitForEnrollment(ctx context.Context, req *connect.Request[homecallv1alpha.WaitForEnrollmentRequest], stream *connect.ServerStream[homecallv1alpha.WaitForEnrollmentResponse]) error {
	err := s.tenantService.CanAccessDevice(ctx, req.Msg.GetDeviceId(), true)
	if err != nil {
		return fmt.Errorf("failed access device: %w", err)
	}

	device, err := s.getDevice(ctx, req.Msg.GetDeviceId())
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	if device.EnrollmentKey == "" {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("device is already enrolled"))
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
	tenantId := req.Msg.GetTenantId()

	err := s.tenantService.CanAccessTenant(ctx, tenantId, false)
	if err != nil {
		return nil, fmt.Errorf("failed access device: %w", err)
	}

	devicesStmt := SELECT(
		Device.DeviceID,
		Device.Name, DeviceNotificationToken.UpdatedAt.IS_NOT_NULL().
			AND(DeviceNotificationToken.UpdatedAt.GT(CAST(NOW()).AS_TIMESTAMP().SUB(INTERVALd(time.Hour)))).
			AS("Online"),
		Enrollment.Key,
	).FROM(Device.
		LEFT_JOIN(Enrollment, Device.ID.EQ(Enrollment.ID)).
		LEFT_JOIN(Tenant, Device.TenantID.EQ(Tenant.ID)).
		LEFT_JOIN(DeviceNotificationToken, Device.ID.EQ(DeviceNotificationToken.DeviceID)),
	).WHERE(Tenant.TenantID.EQ(String(tenantId)))

	var devices []struct {
		model.Device
		model.Enrollment
		Online bool
	}
	err = devicesStmt.QueryContext(ctx, s.db, &devices)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	var deviceResponses []*homecallv1alpha.Device
	for _, device := range devices {
		deviceResponses = append(deviceResponses, &homecallv1alpha.Device{
			Id:            device.Device.DeviceID,
			Name:          device.Device.Name,
			EnrollmentKey: device.Enrollment.Key,
			Online:        device.Online,
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
	err := s.tenantService.CanAccessDevice(ctx, req.Msg.GetDeviceId(), false)
	if err != nil {
		return nil, fmt.Errorf("failed access device: %w", err)
	}

	device, err := s.getDevice(ctx, req.Msg.GetDeviceId())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	if device.EnrollmentKey != "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("device is not enrolled"))
	}

	if !device.Online {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("device is offline"))
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

	callId := uuid.New().String()

	err = util.WithTransaction(s.db, func(db util.DB) error {
		// Store call in device outbox
		insertCallStmt := DeviceCallOutbox.
			INSERT(
				DeviceCallOutbox.CallID,
				DeviceCallOutbox.DeviceID,
				DeviceCallOutbox.JitsiRoomID,
				DeviceCallOutbox.JitsiJwt,
			).
			VALUES(
				String(callId),
				SELECT(Device.ID).FROM(Device).WHERE(Device.DeviceID.EQ(String(device.GetId()))),
				String(jitsiCall.RoomName()),
				String(deviceToken),
			)
		_, err = insertCallStmt.ExecContext(ctx, s.db)
		if err != nil {
			return fmt.Errorf("failed to insert call: %w", err)
		}

		tokenRow := model.DeviceNotificationToken{}
		err = SELECT(DeviceNotificationToken.NotificationToken).
			FROM(Device.LEFT_JOIN(DeviceNotificationToken, Device.ID.EQ(DeviceNotificationToken.DeviceID))).
			WHERE(Device.DeviceID.EQ(String(device.GetId()))).LIMIT(1).QueryContext(ctx, s.db, &tokenRow)
		if err != nil {
			return fmt.Errorf("failed to get notification token: %w", err)
		}

		err = s.notificationService.SendNotification(ctx, &fm.Message{
			Token: tokenRow.NotificationToken,
			Data: map[string]string{
				"callId": callId,
				"type":   "call",
			},
			Notification: &fm.Notification{
				Title: "Inkommande samtal",
				Body:  "Du har ett inkommande samtal, klicka här för att svara",
			},
			Android: &fm.AndroidConfig{
				// Required for background/quit data-only messages on Android
				Priority: "high",
			},
			APNS: &fm.APNSConfig{
				Payload: &fm.APNSPayload{
					Aps: &fm.Aps{
						// Required for background/quit data-only messages on iOS
						ContentAvailable: true,
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[homecallv1alpha.StartCallResponse]{
		Msg: &homecallv1alpha.StartCallResponse{
			CallId:      callId,
			JitsiJwt:    officeToken,
			JitsiRoomId: jitsiCall.RoomName(),
		},
	}, nil
}
