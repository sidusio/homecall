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

	err = util.WithTransaction(s.db, func(db util.DB) error {
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
			return fmt.Errorf("failed to create tenant: %w", err)
		}

		memberId, err := util.RandomString(16)
		if err != nil {
			return fmt.Errorf("failed to generate member id: %w", err)
		}

		// Insert user tenant
		stmt = UserTenant.INSERT(
			UserTenant.MemberID,
			UserTenant.UserID,
			UserTenant.TenantID,
			UserTenant.Role,
		).VALUES(
			String(memberId),
			SELECT(User.ID).FROM(User).WHERE(User.IdpUserID.EQ(String(authDetails.Subject))).LIMIT(1),
			SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(tenantID))).LIMIT(1),
			enum.TenantRole.Admin,
		)
		_, err = stmt.ExecContext(ctx, s.db)
		if err != nil {
			return fmt.Errorf("failed to create user tenant: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
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
	).WHERE(User.IdpUserID.EQ(String(authDetails.Subject)))

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

func (s *Service) CreateTenantInvite(ctx context.Context, req *connect.Request[homecallv1alpha.CreateTenantInviteRequest]) (*connect.Response[homecallv1alpha.CreateTenantInviteResponse], error) {
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

	inviteID, err := util.RandomString(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite ID: %w", err)
	}

	// Insert tenant invite
	stmt := TenantInvite.INSERT(
		TenantInvite.InviteID,
		TenantInvite.TenantID,
		TenantInvite.Email,
		TenantInvite.Role,
	).VALUES(
		inviteID,
		SELECT(Tenant.ID).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId()))).LIMIT(1),
		normalizeEmail(
			req.Msg.GetEmail(),
		),
		role,
	)
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant invite: %w", err)

	}

	tenant := model.Tenant{}
	err = SELECT(Tenant.Name).FROM(Tenant).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId()))).LIMIT(1).QueryContext(ctx, s.db, &tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.CreateTenantInviteResponse]{
		Msg: &homecallv1alpha.CreateTenantInviteResponse{
			TenantInvite: &homecallv1alpha.TenantInvite{
				Id:         inviteID,
				TenantId:   req.Msg.GetTenantId(),
				Email:      normalizeEmail(req.Msg.GetEmail()),
				Role:       req.Msg.GetRole(),
				TenantName: tenant.Name,
			},
		},
	}, nil
}

func (s *Service) ListTenantInvites(ctx context.Context, req *connect.Request[homecallv1alpha.ListTenantInvitesRequest]) (*connect.Response[homecallv1alpha.ListTenantInvitesResponse], error) {
	var stmt SelectStatement
	if req.Msg.GetTenantId() != "" {
		err := s.CanAccessTenant(ctx, req.Msg.GetTenantId(), true)
		if err != nil {
			return nil, fmt.Errorf("failed access tenant invites: %w", err)
		}

		stmt = SELECT(
			TenantInvite.InviteID,
			TenantInvite.Email,
			TenantInvite.Role,
			Tenant.TenantID,
			Tenant.Name,
		).FROM(
			TenantInvite.
				LEFT_JOIN(Tenant, TenantInvite.TenantID.EQ(Tenant.ID)),
		).WHERE(Tenant.TenantID.EQ(String(req.Msg.GetTenantId())))
	} else {
		authDetails := auth.GetAuth(ctx)
		if authDetails == nil {
			return nil, fmt.Errorf("no auth details")
		}

		stmt = SELECT(
			TenantInvite.InviteID,
			TenantInvite.Email,
			TenantInvite.Role,
			Tenant.TenantID,
			Tenant.Name,
		).FROM(
			TenantInvite.
				LEFT_JOIN(Tenant, TenantInvite.TenantID.EQ(Tenant.ID)),
		).WHERE(TenantInvite.Email.EQ(String(normalizeEmail(authDetails.VerifiedEmail))))
	}

	var dbInvites []struct {
		model.TenantInvite
		model.Tenant
	}
	err := stmt.QueryContext(ctx, s.db, &dbInvites)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant invites: %w", err)
	}

	invites := make([]*homecallv1alpha.TenantInvite, len(dbInvites))
	for i, dbInvite := range dbInvites {
		role := homecallv1alpha.Role_ROLE_MEMBER
		if dbInvite.TenantInvite.Role == "admin" {
			role = homecallv1alpha.Role_ROLE_ADMIN
		}

		invites[i] = &homecallv1alpha.TenantInvite{
			Id:         dbInvite.TenantInvite.InviteID,
			TenantId:   dbInvite.Tenant.TenantID,
			Email:      normalizeEmail(dbInvite.TenantInvite.Email),
			Role:       role,
			TenantName: dbInvite.Tenant.Name,
		}
	}

	return &connect.Response[homecallv1alpha.ListTenantInvitesResponse]{
		Msg: &homecallv1alpha.ListTenantInvitesResponse{
			TenantInvites: invites,
		},
	}, nil
}

func (s *Service) RemoveTenantInvite(ctx context.Context, req *connect.Request[homecallv1alpha.RemoveTenantInviteRequest]) (*connect.Response[homecallv1alpha.RemoveTenantInviteResponse], error) {
	err := s.CanAccessTenantInvite(ctx, req.Msg.GetId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant invite: %w", err)
	}

	stmt := TenantInvite.
		DELETE().
		WHERE(TenantInvite.InviteID.EQ(String(req.Msg.GetId())))
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to remove tenant invite: %w", err)
	}

	return &connect.Response[homecallv1alpha.RemoveTenantInviteResponse]{
		Msg: &homecallv1alpha.RemoveTenantInviteResponse{},
	}, nil
}

func (s *Service) AcceptTenantInvite(ctx context.Context, req *connect.Request[homecallv1alpha.AcceptTenantInviteRequest]) (*connect.Response[homecallv1alpha.AcceptTenantInviteResponse], error) {
	err := s.CanAcceptTenantInvite(ctx, req.Msg.GetId())
	if err != nil {
		return nil, fmt.Errorf("failed access tenant invite: %w", err)
	}

	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return nil, fmt.Errorf("no auth details")
	}

	memberId, err := util.RandomString(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate member id: %w", err)

	}

	err = util.WithTransaction(s.db, func(db util.DB) error {
		// Insert user tenant
		stmt := UserTenant.INSERT(
			UserTenant.MemberID,
			UserTenant.UserID,
			UserTenant.TenantID,
			UserTenant.Role,
		).VALUES(
			String(memberId),
			SELECT(User.ID).FROM(User).WHERE(User.IdpUserID.EQ(String(authDetails.Subject))).LIMIT(1),
			SELECT(TenantInvite.TenantID).FROM(TenantInvite).WHERE(TenantInvite.InviteID.EQ(String(req.Msg.GetId()))).LIMIT(1),
			SELECT(TenantInvite.Role).FROM(TenantInvite).WHERE(TenantInvite.InviteID.EQ(String(req.Msg.GetId()))).LIMIT(1),
		)
		_, err = stmt.ExecContext(ctx, s.db)
		if err != nil {
			return fmt.Errorf("failed to create user tenant: %w", err)
		}

		// Remove the invite
		delStmt := TenantInvite.DELETE().WHERE(TenantInvite.InviteID.EQ(String(req.Msg.GetId())))
		_, err = delStmt.ExecContext(ctx, s.db)
		if err != nil {
			return fmt.Errorf("failed to remove tenant invite: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[homecallv1alpha.AcceptTenantInviteResponse]{
		Msg: &homecallv1alpha.AcceptTenantInviteResponse{},
	}, nil
}

func (s *Service) RemoveTenantMember(ctx context.Context, req *connect.Request[homecallv1alpha.RemoveTenantMemberRequest]) (*connect.Response[homecallv1alpha.RemoveTenantMemberResponse], error) {
	err := s.CanAccessTenantMember(ctx, req.Msg.GetId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	// Make sure the user is not removing themselves
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return nil, fmt.Errorf("no auth details")
	}

	var result struct{ Count int }
	err = SELECT(COUNT(UserTenant.MemberID).AS("count")).
		FROM(User.LEFT_JOIN(UserTenant, User.ID.EQ(UserTenant.UserID))).
		WHERE(
			User.IdpUserID.EQ(String(authDetails.Subject)).
				AND(UserTenant.MemberID.EQ(String(req.Msg.GetId()))),
		).GROUP_BY(UserTenant.MemberID).
		LIMIT(1).QueryContext(ctx, s.db, &result)
	if err != nil && !errors.Is(err, qrm.ErrNoRows) {
		return nil, fmt.Errorf("failed to query user tenant: %w", err)
	}
	if !errors.Is(err, qrm.ErrNoRows) || (err != nil && result.Count > 0) {
		return nil, fmt.Errorf("cannot remove yourself")
	}

	// Remove user tenant
	stmt := UserTenant.
		DELETE().
		WHERE(UserTenant.MemberID.EQ(String(req.Msg.GetId())))
	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to remove user tenant: %w", err)
	}

	return &connect.Response[homecallv1alpha.RemoveTenantMemberResponse]{
		Msg: &homecallv1alpha.RemoveTenantMemberResponse{},
	}, nil
}

func (s *Service) UpdateTenantMember(ctx context.Context, req *connect.Request[homecallv1alpha.UpdateTenantMemberRequest]) (*connect.Response[homecallv1alpha.UpdateTenantMemberResponse], error) {
	err := s.CanAccessTenantMember(ctx, req.Msg.GetId(), true)
	if err != nil {
		return nil, fmt.Errorf("failed access tenant: %w", err)
	}

	// Make sure the user is not updating themselves
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return nil, fmt.Errorf("no auth details")
	}

	var result struct{ Count int }
	err = SELECT(COUNT(UserTenant.MemberID).AS("count")).
		FROM(User.LEFT_JOIN(UserTenant, User.ID.EQ(UserTenant.UserID))).
		WHERE(
			User.IdpUserID.EQ(String(authDetails.Subject)).
				AND(UserTenant.MemberID.EQ(String(req.Msg.GetId()))),
		).GROUP_BY(UserTenant.MemberID).
		LIMIT(1).QueryContext(ctx, s.db, &result)
	if err != nil && !errors.Is(err, qrm.ErrNoRows) {
		return nil, fmt.Errorf("failed to query user tenant: %w", err)
	}
	if !errors.Is(err, qrm.ErrNoRows) || (err != nil && result.Count > 0) {
		return nil, fmt.Errorf("cannot update yourself")
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
		WHERE(UserTenant.MemberID.EQ(String(req.Msg.GetId())))
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
		User.IdpUserID,
		User.DisplayName,
		UserTenant.MemberID,
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
			Id:            dbMember.UserTenant.MemberID,
			TenantId:      req.Msg.GetTenantId(),
			Subject:       dbMember.IdpUserID,
			VerifiedEmail: normalizeEmail(dbMember.Email),
			DisplayName:   dbMember.DisplayName,
			Role:          role,
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
		AND(User.IdpUserID.EQ(String(authDetails.Subject)))
	if adminRequired {
		conditions = conditions.AND(UserTenant.Role.EQ(enum.TenantRole.Admin))
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(User.ID).AS("count")).FROM(
		Tenant.
			LEFT_JOIN(UserTenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)),
	).WHERE(conditions).GROUP_BY(User.ID).LIMIT(1)
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

func (s *Service) CanAccessTenantMember(ctx context.Context, memberID string, adminRequired bool) error {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return ErrNoAccess
	}

	targetUserTenant := UserTenant.AS("target_user_tenant")
	conditions := targetUserTenant.MemberID.EQ(String(memberID)).
		AND(User.IdpUserID.EQ(String(authDetails.Subject)))
	if adminRequired {
		conditions = conditions.AND(UserTenant.Role.EQ(enum.TenantRole.Admin))
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(User.ID).AS("count")).FROM(
		targetUserTenant.
			LEFT_JOIN(Tenant, targetUserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(UserTenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)),
	).WHERE(conditions).GROUP_BY(User.ID).LIMIT(1)
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

func (s *Service) CanAccessTenantInvite(ctx context.Context, inviteID string, adminRequired bool) error {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return ErrNoAccess
	}

	conditions := TenantInvite.InviteID.EQ(String(inviteID)).
		AND(User.IdpUserID.EQ(String(authDetails.Subject)))
	if adminRequired {
		conditions = conditions.AND(UserTenant.Role.EQ(enum.TenantRole.Admin))
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(User.ID).AS("count")).FROM(
		UserTenant.
			LEFT_JOIN(Tenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)).
			LEFT_JOIN(TenantInvite, TenantInvite.TenantID.EQ(Tenant.ID)),
	).WHERE(conditions).GROUP_BY(User.ID).LIMIT(1)
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

func (s *Service) CanAcceptTenantInvite(ctx context.Context, inviteID string) error {
	authDetails := auth.GetAuth(ctx)
	if authDetails == nil {
		return ErrNoAccess
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(TenantInvite.ID).AS("count")).FROM(
		TenantInvite,
	).WHERE(
		TenantInvite.Email.EQ(String(normalizeEmail(authDetails.VerifiedEmail))).
			AND(TenantInvite.InviteID.EQ(String(inviteID))),
	).GROUP_BY(TenantInvite.ID).LIMIT(1)
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
		AND(User.IdpUserID.EQ(String(authDetails.Subject)))
	if adminRequired {
		conditions = conditions.AND(UserTenant.Role.EQ(enum.TenantRole.Admin))
	}

	// Check if the user has access to the tenant
	stmt := SELECT(COUNT(User.ID).AS("count")).FROM(
		Device.
			LEFT_JOIN(Tenant, Device.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(UserTenant, UserTenant.TenantID.EQ(Tenant.ID)).
			LEFT_JOIN(User, User.ID.EQ(UserTenant.UserID)),
	).WHERE(conditions).GROUP_BY(User.ID).LIMIT(1)
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

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
