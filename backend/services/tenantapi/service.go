package tenantapi

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"errors"
	"fmt"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"log/slog"
	homecallv1alpha "sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/gen/jetdb/public/enum"
	"sidus.io/home-call/gen/jetdb/public/model"
	. "sidus.io/home-call/gen/jetdb/public/table"
	"sidus.io/home-call/services/auth"
	"sidus.io/home-call/util"
	"strings"
	"time"
)

var _ homecallv1alphaconnect.TenantServiceHandler = (*Service)(nil)

var ErrNoAccess = errors.New("no access")

func NewService(
	db *sql.DB,
	logger *slog.Logger,
	defaultDeviceLimit int,
) *Service {
	return &Service{
		db:                 db,
		logger:             logger,
		defaultDeviceLimit: defaultDeviceLimit,
	}
}

type Service struct {
	db                 *sql.DB
	logger             *slog.Logger
	defaultDeviceLimit int
}

func (s *Service) CreateTenant(ctx context.Context, req *connect.Request[homecallv1alpha.CreateTenantRequest]) (*connect.Response[homecallv1alpha.CreateTenantResponse], error) {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return nil, fmt.Errorf("no auth details")
	}

	tenantID, err := generateTenantID(req.Msg.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to generate tenant ID: %w", err)
	}

	createdAt := time.Now()

	// TODO: Transactions

	// Insert tenant
	stmt := Tenant.INSERT(Tenant.TenantID, Tenant.Name, Tenant.MaxDevices, Tenant.CreatedAt).
		MODEL(model.Tenant{
			TenantID:   tenantID,
			Name:       req.Msg.GetName(),
			MaxDevices: int32(s.defaultDeviceLimit),
			CreatedAt:  createdAt,
		})
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Ensure the user exists
	stmt = User.INSERT(User.Email).VALUES(String(authDetails.Subject)).ON_CONFLICT(User.Email).DO_NOTHING()
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Insert user tenant
	stmt = UserTenant.INSERT(
		UserTenant.UserID,
		UserTenant.TenantID,
		UserTenant.Role,
	).VALUES(
		SELECT(User.ID).FROM(User).WHERE(User.Email.EQ(String(authDetails.Subject))).LIMIT(1),
		SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(tenantID))).LIMIT(1),
		enum.TenantRole.Admin,
	)
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create user tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.CreateTenantResponse]{
		Msg: &homecallv1alpha.CreateTenantResponse{
			Tenant: &homecallv1alpha.Tenant{
				Id:         tenantID,
				Name:       req.Msg.GetName(),
				MaxDevices: int64(s.defaultDeviceLimit),
			},
		},
	}, nil

}

func (s *Service) ListTenants(ctx context.Context, req *connect.Request[homecallv1alpha.ListTenantsRequest]) (*connect.Response[homecallv1alpha.ListTenantsResponse], error) {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return nil, fmt.Errorf("no auth details")
	}

	stmt := SELECT(
		Tenant.TenantID,
		Tenant.Name,
		Tenant.MaxDevices,
	).FROM(
		Tenant.
			LEFT_JOIN(UserTenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)),
	).WHERE(User.Email.EQ(String(authDetails.Subject)))

	var dbTenants []model.Tenant
	err := stmt.QueryContext(ctx, s.db, &dbTenants)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	tenants := make([]*homecallv1alpha.Tenant, len(dbTenants))
	for i, dbTenant := range dbTenants {
		tenants[i] = &homecallv1alpha.Tenant{
			Id:         dbTenant.TenantID,
			Name:       dbTenant.Name,
			MaxDevices: int64(dbTenant.MaxDevices),
		}

	}

	return &connect.Response[homecallv1alpha.ListTenantsResponse]{
		Msg: &homecallv1alpha.ListTenantsResponse{
			Tenants: tenants,
		},
	}, nil
}

func (s *Service) RemoveTenant(ctx context.Context, req *connect.Request[homecallv1alpha.RemoveTenantRequest]) (*connect.Response[homecallv1alpha.RemoveTenantResponse], error) {
	err := s.CanAccessTenant(ctx, req.Msg.GetId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	// Remove all members but preserve the tenant
	stmt := UserTenant.
		DELETE().
		WHERE(UserTenant.TenantID.EQ(
			IntExp(SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetId()))).LIMIT(1))),
		)
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to remove tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.RemoveTenantResponse]{
		Msg: &homecallv1alpha.RemoveTenantResponse{},
	}, nil
}

func (s *Service) CreateTenantMember(ctx context.Context, req *connect.Request[homecallv1alpha.CreateTenantMemberRequest]) (*connect.Response[homecallv1alpha.CreateTenantMemberResponse], error) {
	err := s.CanAccessTenant(ctx, req.Msg.GetTenantId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	//nolint:ineffassign,staticcheck
	role := enum.TenantRole.User
	switch req.Msg.GetRole() {
	case homecallv1alpha.Role_ROLE_ADMIN:
		role = enum.TenantRole.Admin
	case homecallv1alpha.Role_ROLE_MEMBER:
		role = enum.TenantRole.User
	default:
		return nil, fmt.Errorf("invalid role")
	}

	// Ensure the user exists
	stmt := User.INSERT(User.Email).VALUES(String(req.Msg.GetEmail())).ON_CONFLICT(User.Email).DO_NOTHING()
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Insert user tenant
	stmt = UserTenant.INSERT(
		UserTenant.UserID,
		UserTenant.TenantID,
		UserTenant.Role,
	).VALUES(
		SELECT(User.ID).FROM(User).WHERE(User.Email.EQ(String(req.Msg.GetEmail()))).LIMIT(1),
		SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId()))).LIMIT(1),
		role,
	)
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create user tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.CreateTenantMemberResponse]{
		Msg: &homecallv1alpha.CreateTenantMemberResponse{
			TenantMember: &homecallv1alpha.TenantMember{
				Email:    req.Msg.GetEmail(),
				Role:     req.Msg.GetRole(),
				TenantId: req.Msg.GetTenantId(),
			},
		},
	}, nil
}

func (s *Service) RemoveTenantMember(ctx context.Context, req *connect.Request[homecallv1alpha.RemoveTenantMemberRequest]) (*connect.Response[homecallv1alpha.RemoveTenantMemberResponse], error) {
	err := s.CanAccessTenant(ctx, req.Msg.GetTenantId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	// Make sure the user is not removing themselves
	authDetails := auth.GetAuth(ctx)
	if authDetails != nil && authDetails.Subject == req.Msg.GetEmail() {
		return nil, fmt.Errorf("cannot remove yourself")
	}

	// Remove user tenant
	stmt := UserTenant.
		DELETE().
		WHERE(
			UserTenant.TenantID.EQ(
				IntExp(SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId()))).LIMIT(1)),
			).AND(UserTenant.UserID.EQ(
				IntExp(SELECT(User.ID).FROM(User).WHERE(User.Email.EQ(String(req.Msg.GetEmail()))).LIMIT(1)),
			)),
		)
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to remove user tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.RemoveTenantMemberResponse]{
		Msg: &homecallv1alpha.RemoveTenantMemberResponse{},
	}, nil
}

func (s *Service) UpdateTenantMember(ctx context.Context, req *connect.Request[homecallv1alpha.UpdateTenantMemberRequest]) (*connect.Response[homecallv1alpha.UpdateTenantMemberResponse], error) {
	err := s.CanAccessTenant(ctx, req.Msg.GetTenantId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	//nolint:ineffassign,staticcheck
	role := enum.TenantRole.User
	switch req.Msg.GetRole() {
	case homecallv1alpha.Role_ROLE_ADMIN:
		role = enum.TenantRole.Admin
	case homecallv1alpha.Role_ROLE_MEMBER:
		role = enum.TenantRole.User
	default:
		return nil, fmt.Errorf("invalid role")
	}

	// Update user tenant
	stmt := UserTenant.UPDATE(UserTenant.Role).SET(role).
		WHERE(
			UserTenant.TenantID.EQ(
				IntExp(SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId()))).LIMIT(1)),
			).AND(UserTenant.UserID.EQ(
				IntExp(SELECT(User.ID).FROM(User).WHERE(User.Email.EQ(String(req.Msg.GetEmail()))).LIMIT(1)),
			)),
		)
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to update user tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.UpdateTenantMemberResponse]{
		Msg: &homecallv1alpha.UpdateTenantMemberResponse{},
	}, nil
}

func (s *Service) ListTenantMembers(ctx context.Context, req *connect.Request[homecallv1alpha.ListTenantMembersRequest]) (*connect.Response[homecallv1alpha.ListTenantMembersResponse], error) {
	err := s.CanAccessTenant(ctx, req.Msg.GetTenantId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	stmt := SELECT(
		User.Email,
		UserTenant.Role,
	).FROM(
		UserTenant.
			LEFT_JOIN(User, UserTenant.UserID.EQ(User.ID)).
			LEFT_JOIN(Tenant, Tenant.ID.EQ(UserTenant.TenantID)),
	).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId())))

	var dbMembers []struct {
		model.UserTenant
		model.User
	}
	err = stmt.QueryContext(ctx, s.db, &dbMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant members: %w", err)
	}

	members := make([]*homecallv1alpha.TenantMember, len(dbMembers))
	for i, dbMember := range dbMembers {
		role := homecallv1alpha.Role_ROLE_MEMBER
		if dbMember.Role == "admin" {
			role = homecallv1alpha.Role_ROLE_ADMIN
		}

		members[i] = &homecallv1alpha.TenantMember{
			Email:    dbMember.Email,
			Role:     role,
			TenantId: req.Msg.GetTenantId(),
		}
	}

	return &connect.Response[homecallv1alpha.ListTenantMembersResponse]{
		Msg: &homecallv1alpha.ListTenantMembersResponse{
			TenantMembers: members,
		},
	}, nil
}

func (s *Service) CanAccessTenant(ctx context.Context, tenantID string, adminRequired bool) error {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return ErrNoAccess
	}

	conditions := Tenant.TenantID.EQ(String(tenantID)).
		AND(User.Email.EQ(String(authDetails.Subject)))
	if adminRequired {
		conditions = conditions.AND(UserTenant.Role.EQ(enum.TenantRole.Admin))
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(User.ID).AS("count")).FROM(
		Tenant.
			LEFT_JOIN(UserTenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)),
	).WHERE(conditions).GROUP_BY(User.ID).LIMIT(1)
	query, args := stmt.Sql()
	s.logger.Debug("Querying tenant access", "query", query, "args", args) // TODO
	var result struct{ Count int }
	err := stmt.QueryContext(ctx, s.db, &result)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return ErrNoAccess
		}
		return fmt.Errorf("failed to query tenant access: %w", err)
	}

	if result.Count == 0 {
		return ErrNoAccess
	}

	return nil
}

func (s *Service) CanAccessDevice(ctx context.Context, deviceID string, adminRequired bool) error {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return ErrNoAccess
	}

	conditions := Device.DeviceID.EQ(String(deviceID)).
		AND(User.Email.EQ(String(authDetails.Subject)))
	if adminRequired {
		conditions = conditions.AND(UserTenant.Role.EQ(String("admin")))
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(Int(1)).AS("count")).FROM(
		Device.
			LEFT_JOIN(Tenant, Device.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(UserTenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)),
	).WHERE(conditions).GROUP_BY(User.ID).LIMIT(1)
	var result struct{ Count int }
	err := stmt.QueryContext(ctx, s.db, &result)
	if err != nil {
		return fmt.Errorf("failed to query tenant access: %w", err)
	}

	if result.Count == 0 {
		return ErrNoAccess
	}

	return nil
}

func generateTenantID(name string) (string, error) {
	allowed := "abcdefghijklmnopqrstuvwxyz0123456789-"
	tenantID := ""
	for _, r := range strings.ToLower(name) {
		if strings.Contains(allowed, string(r)) {
			tenantID += string(r)
		}
		if r == ' ' {
			tenantID += "_"
		}
	}

	random, err := util.RandomString(6)

	if err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}

	return tenantID + "-" + random, nil
}
