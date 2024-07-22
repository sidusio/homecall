package api

import (
	"bytes"
	"connectrpc.com/connect"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"firebase.google.com/go/v4/messaging"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	homecallv1alpha "sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/notifications/directorynotifications"
	"sidus.io/home-call/services/auth"
	"sidus.io/home-call/util"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		notificationsDir, err := os.MkdirTemp("", "crudb-mock-notifications")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(notificationsDir)

		app, err := NewTestApp(WithNotificationDir(notificationsDir))
		if err != nil {
			panic(err)
		}
		err = app.Start(ctx)
		if err != nil {
			panic(err)
		}
		defer func() {
			err := app.Stop()
			if err != nil {
				panic(err)
			}
		}()

		globalTestApp = app
		return m.Run()
	}())
}

var (
	globalTestApp *TestApp
)

func TestDeviceCall(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	adminUser := randomUser()
	tenant, err := createTestTenant(t.Name(), adminUser, globalTestApp.TenantClient())
	require.NoError(t, err)

	device, err := globalTestApp.OfficeClient().CreateDevice(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateDeviceRequest]{
		Msg: &homecallv1alpha.CreateDeviceRequest{
			Name:     fmt.Sprintf("test-%s", randomUser()),
			TenantId: tenant.Id,
			DefaultSettings: &homecallv1alpha.DeviceSettings{
				AutoAnswer:             true,
				AutoAnswerDelaySeconds: 10,
			},
		},
	}))
	require.NoError(t, err)

	// Generate dummy rsa key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	// RSA PEM encoded
	var publicKey bytes.Buffer
	err = pem.Encode(&publicKey, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey),
	})
	require.NoError(t, err)

	deviceToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Subject:   device.Msg.GetDevice().GetId(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "homecall-device",
		Audience:  jwt.ClaimStrings{"homecall"},
	}).SignedString(key)
	require.NoError(t, err)

	// attempt call before enrollment
	_, err = globalTestApp.OfficeClient().StartCall(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.StartCallRequest]{
		Msg: &homecallv1alpha.StartCallRequest{
			DeviceId: device.Msg.GetDevice().GetId(),
		},
	}))
	cErr := &connect.Error{}
	require.ErrorAs(t, err, &cErr)
	require.Equal(t, connect.CodeFailedPrecondition, cErr.Code())

	// Enroll device
	_, err = globalTestApp.DeviceClient().Enroll(ctx, &connect.Request[homecallv1alpha.EnrollRequest]{
		Msg: &homecallv1alpha.EnrollRequest{
			EnrollmentKey: device.Msg.GetDevice().GetEnrollmentKey(),
			PublicKey:     publicKey.String(),
		},
	})
	require.NoError(t, err)

	// attempt call before token registration
	_, err = globalTestApp.OfficeClient().StartCall(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.StartCallRequest]{
		Msg: &homecallv1alpha.StartCallRequest{
			DeviceId: device.Msg.GetDevice().GetId(),
		},
	}))
	cErr = &connect.Error{}
	require.ErrorAs(t, err, &cErr)
	require.Equal(t, connect.CodeFailedPrecondition, cErr.Code())

	// Register device token
	deviceNotificationToken, err := util.RandomString(10)
	require.NoError(t, err)

	_, err = globalTestApp.DeviceClient().UpdateNotificationToken(ctx, auth.WithToken(deviceToken, &connect.Request[homecallv1alpha.UpdateNotificationTokenRequest]{
		Msg: &homecallv1alpha.UpdateNotificationTokenRequest{
			NotificationToken: deviceNotificationToken,
		},
	}))
	require.NoError(t, err)

	// attempt call after enrollment
	call, err := globalTestApp.OfficeClient().StartCall(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.StartCallRequest]{
		Msg: &homecallv1alpha.StartCallRequest{
			DeviceId: device.Msg.GetDevice().GetId(),
		},
	}))
	require.NoError(t, err)

	message := &messaging.Message{}
	err = filepath.Walk(path.Join(globalTestApp.NotificationsDir(), directorynotifications.DevicesDirectory, deviceNotificationToken), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			err = message.UnmarshalJSON(content)
			require.NoError(t, err)
		}
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, "call", message.Data["type"])
	assert.Equal(t, call.Msg.GetCallId(), message.Data["callId"])

	// Get call details
	callDetails, err := globalTestApp.DeviceClient().GetCallDetails(ctx, auth.WithToken(deviceToken, &connect.Request[homecallv1alpha.GetCallDetailsRequest]{
		Msg: &homecallv1alpha.GetCallDetailsRequest{
			CallId: message.Data["callId"],
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, call.Msg.GetCallId(), callDetails.Msg.GetCallId())
	assert.Equal(t, call.Msg.GetJitsiRoomId(), callDetails.Msg.GetJitsiRoomId())
	assert.NotEqual(t, call.Msg.GetJitsiJwt(), callDetails.Msg.GetJitsiJwt())
	assert.NotEmpty(t, call.Msg.GetJitsiJwt())
	assert.NotEmpty(t, callDetails.Msg.GetJitsiJwt())
}

func TestTenantMemberAdmin(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	adminUser := randomUser()
	tenant, err := createTestTenant(t.Name(), adminUser, globalTestApp.TenantClient())
	require.NoError(t, err)

	memberUser := randomUser()
	nonMemberUser := randomUser()

	invite, err := globalTestApp.TenantClient().CreateTenantInvite(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateTenantInviteRequest]{
		Msg: &homecallv1alpha.CreateTenantInviteRequest{
			TenantId: tenant.Id,
			Email:    memberUser,
			Role:     homecallv1alpha.Role_ROLE_MEMBER,
		},
	}))
	require.NoError(t, err)

	_, err = globalTestApp.TenantClient().AcceptTenantInvite(ctx, auth.WithDummyToken(memberUser, &connect.Request[homecallv1alpha.AcceptTenantInviteRequest]{
		Msg: &homecallv1alpha.AcceptTenantInviteRequest{
			Id: invite.Msg.GetTenantInvite().GetId(),
		},
	}))
	require.NoError(t, err)

	tenantMembers, err := globalTestApp.TenantClient().ListTenantMembers(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.ListTenantMembersRequest]{
		Msg: &homecallv1alpha.ListTenantMembersRequest{
			TenantId: tenant.Id,
		},
	}))
	require.NoError(t, err)

	var adminTenantMember, memberTenantMember *homecallv1alpha.TenantMember
	for _, m := range tenantMembers.Msg.TenantMembers {
		if m.VerifiedEmail == adminUser {
			adminTenantMember = m
		} else if m.VerifiedEmail == memberUser {
			memberTenantMember = m
		}
	}
	require.NotNil(t, adminTenantMember)
	require.NotNil(t, memberTenantMember)

	// non-member should not be able to update or remove tenant members
	_, err = globalTestApp.TenantClient().UpdateTenantMember(ctx, auth.WithDummyToken(nonMemberUser, &connect.Request[homecallv1alpha.UpdateTenantMemberRequest]{
		Msg: &homecallv1alpha.UpdateTenantMemberRequest{
			Id: adminTenantMember.GetId(),
		},
	}))
	require.Error(t, err)
	_, err = globalTestApp.TenantClient().RemoveTenantMember(ctx, auth.WithDummyToken(nonMemberUser, &connect.Request[homecallv1alpha.RemoveTenantMemberRequest]{
		Msg: &homecallv1alpha.RemoveTenantMemberRequest{
			Id: memberTenantMember.GetId(),
		},
	}))
	require.Error(t, err)

	// member should not be able to remove or update members
	_, err = globalTestApp.TenantClient().UpdateTenantMember(ctx, auth.WithDummyToken(memberUser, &connect.Request[homecallv1alpha.UpdateTenantMemberRequest]{
		Msg: &homecallv1alpha.UpdateTenantMemberRequest{
			Id: memberTenantMember.GetId(),
		},
	}))
	require.Error(t, err)
	_, err = globalTestApp.TenantClient().RemoveTenantMember(ctx, auth.WithDummyToken(memberUser, &connect.Request[homecallv1alpha.RemoveTenantMemberRequest]{
		Msg: &homecallv1alpha.RemoveTenantMemberRequest{
			Id: adminTenantMember.GetId(),
		},
	}))
	require.Error(t, err)

	// admin should be able to update and remove members
	_, err = globalTestApp.TenantClient().UpdateTenantMember(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.UpdateTenantMemberRequest]{
		Msg: &homecallv1alpha.UpdateTenantMemberRequest{
			Id:   memberTenantMember.GetId(),
			Role: homecallv1alpha.Role_ROLE_ADMIN,
		},
	}))
	require.NoError(t, err)
	_, err = globalTestApp.TenantClient().RemoveTenantMember(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.RemoveTenantMemberRequest]{
		Msg: &homecallv1alpha.RemoveTenantMemberRequest{
			Id: memberTenantMember.GetId(),
		},
	}))
	require.NoError(t, err)

	// admin should not be able to update or remove self
	_, err = globalTestApp.TenantClient().UpdateTenantMember(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.UpdateTenantMemberRequest]{
		Msg: &homecallv1alpha.UpdateTenantMemberRequest{
			Id: adminTenantMember.GetId(),
		},
	}))
	require.Error(t, err)
	_, err = globalTestApp.TenantClient().RemoveTenantMember(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.RemoveTenantMemberRequest]{
		Msg: &homecallv1alpha.RemoveTenantMemberRequest{
			Id: adminTenantMember.GetId(),
		},
	}))
	require.Error(t, err)
}

func TestTenantMembersAdd(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	adminUser := randomUser()
	tenant, err := createTestTenant(t.Name(), adminUser, globalTestApp.TenantClient())
	require.NoError(t, err)

	memberUser := randomUser()
	nonMemberUser := randomUser()
	newAdminUser := randomUser()
	newMemberUser := randomUser()

	// Check if member is added
	isMember := func(t *testing.T, email string, role homecallv1alpha.Role) bool {
		resp, err := globalTestApp.TenantClient().ListTenantMembers(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.ListTenantMembersRequest]{
			Msg: &homecallv1alpha.ListTenantMembersRequest{
				TenantId: tenant.Id,
			},
		}))
		require.NoError(t, err)
		for _, m := range resp.Msg.TenantMembers {
			if m.VerifiedEmail == email && m.Role == role {
				return true
			}
		}
		return false
	}

	steps := []struct {
		description string
		asUser      string
		addUser     string
		withRole    homecallv1alpha.Role
		expectError bool
	}{
		{
			description: "setup member",
			asUser:      adminUser,
			addUser:     memberUser,
			withRole:    homecallv1alpha.Role_ROLE_MEMBER,
			expectError: false,
		},
		{
			description: "attempt to add member as non-member",
			asUser:      nonMemberUser,
			addUser:     newMemberUser,
			withRole:    homecallv1alpha.Role_ROLE_MEMBER,
			expectError: true,
		},
		{
			description: "attempt to add admin as non-member",
			asUser:      nonMemberUser,
			addUser:     newAdminUser,
			withRole:    homecallv1alpha.Role_ROLE_ADMIN,
			expectError: true,
		},
		{
			description: "attempt to add self as member as non-member",
			asUser:      nonMemberUser,
			addUser:     nonMemberUser,
			withRole:    homecallv1alpha.Role_ROLE_MEMBER,
			expectError: true,
		},
		{
			description: "attempt to add self as admin as non-member",
			asUser:      nonMemberUser,
			addUser:     nonMemberUser,
			withRole:    homecallv1alpha.Role_ROLE_ADMIN,
			expectError: true,
		},
		{
			description: "attempt to add member as member",
			asUser:      memberUser,
			addUser:     newMemberUser,
			withRole:    homecallv1alpha.Role_ROLE_MEMBER,
			expectError: true,
		},
		{
			description: "attempt to add admin as member",
			asUser:      memberUser,
			addUser:     newAdminUser,
			withRole:    homecallv1alpha.Role_ROLE_ADMIN,
			expectError: true,
		},
		{
			description: "attempt to add self as admin as member",
			asUser:      memberUser,
			addUser:     memberUser,
			withRole:    homecallv1alpha.Role_ROLE_ADMIN,
			expectError: true,
		},
		{
			description: "attempt member as admin",
			asUser:      adminUser,
			addUser:     newMemberUser,
			withRole:    homecallv1alpha.Role_ROLE_MEMBER,
			expectError: false,
		},
		{
			description: "attempt admin as admin",
			asUser:      adminUser,
			addUser:     newAdminUser,
			withRole:    homecallv1alpha.Role_ROLE_ADMIN,
			expectError: false,
		},
	}

	for i, step := range steps {
		t.Run(fmt.Sprintf("step %d: %s", i, step.description), func(t *testing.T) {
			invite, err := globalTestApp.TenantClient().CreateTenantInvite(ctx, auth.WithDummyToken(step.asUser, &connect.Request[homecallv1alpha.CreateTenantInviteRequest]{
				Msg: &homecallv1alpha.CreateTenantInviteRequest{
					TenantId: tenant.Id,
					Email:    step.addUser,
					Role:     step.withRole,
				},
			}))
			if step.expectError {
				require.Error(t, err)
				require.False(t, isMember(t, step.addUser, step.withRole))
			} else {
				require.NoError(t, err)
				assert.Equal(t, invite.Msg.GetTenantInvite().GetRole(), step.withRole)
				assert.Equal(t, invite.Msg.GetTenantInvite().GetEmail(), step.addUser)
				assert.Equal(t, invite.Msg.GetTenantInvite().GetTenantId(), tenant.Id)

				_, err := globalTestApp.TenantClient().AcceptTenantInvite(ctx, auth.WithDummyToken(step.addUser, &connect.Request[homecallv1alpha.AcceptTenantInviteRequest]{
					Msg: &homecallv1alpha.AcceptTenantInviteRequest{
						Id: invite.Msg.GetTenantInvite().GetId(),
					},
				}))
				require.NoError(t, err)
				require.True(t, isMember(t, step.addUser, step.withRole))
			}
		})
	}
}

func TestTenantDelete(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	adminUser := randomUser()
	tenant, err := createTestTenant(t.Name(), adminUser, globalTestApp.TenantClient())
	require.NoError(t, err)

	memberUser := randomUser()
	nonMemberUser := randomUser()

	// Check if tenant is deleted
	isDeleted := func(t *testing.T) bool {
		resp, err := globalTestApp.TenantClient().ListTenants(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.ListTenantsRequest]{
			Msg: &homecallv1alpha.ListTenantsRequest{},
		}))
		require.NoError(t, err)
		for _, t := range resp.Msg.Tenants {
			if t.Id == tenant.Id {
				return false
			}
		}
		return true
	}

	// Setup
	// Create a member
	invite, err := globalTestApp.TenantClient().CreateTenantInvite(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateTenantInviteRequest]{
		Msg: &homecallv1alpha.CreateTenantInviteRequest{
			TenantId: tenant.Id,
			Email:    memberUser,
			Role:     homecallv1alpha.Role_ROLE_MEMBER,
		},
	}))
	require.NoError(t, err)
	_, err = globalTestApp.TenantClient().AcceptTenantInvite(ctx, auth.WithDummyToken(memberUser, &connect.Request[homecallv1alpha.AcceptTenantInviteRequest]{
		Msg: &homecallv1alpha.AcceptTenantInviteRequest{
			Id: invite.Msg.GetTenantInvite().GetId(),
		},
	}))
	require.NoError(t, err)

	// Attempt to delete tenant as non-member
	_, err = globalTestApp.TenantClient().RemoveTenant(ctx, auth.WithDummyToken(nonMemberUser, &connect.Request[homecallv1alpha.RemoveTenantRequest]{
		Msg: &homecallv1alpha.RemoveTenantRequest{
			Id: tenant.Id,
		},
	}))
	require.Error(t, err)
	require.False(t, isDeleted(t))

	// Attempt to delete tenant as member
	_, err = globalTestApp.TenantClient().RemoveTenant(ctx, auth.WithDummyToken(memberUser, &connect.Request[homecallv1alpha.RemoveTenantRequest]{
		Msg: &homecallv1alpha.RemoveTenantRequest{
			Id: tenant.Id,
		},
	}))
	require.Error(t, err)
	require.False(t, isDeleted(t))

	// Attempt to delete tenant as admin
	_, err = globalTestApp.TenantClient().RemoveTenant(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.RemoveTenantRequest]{
		Msg: &homecallv1alpha.RemoveTenantRequest{
			Id: tenant.Id,
		},
	}))
	require.NoError(t, err)
	require.True(t, isDeleted(t))
}
