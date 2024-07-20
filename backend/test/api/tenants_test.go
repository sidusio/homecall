package api

import (
	"bytes"
	"connectrpc.com/connect"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	homecallv1alpha "sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/services/auth"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		app, err := NewTestApp()
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

	/*wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		waitResp, err := c.deviceClient.WaitForCall(ctx, &connect.Request[homecallv1alpha.WaitForCallRequest]{
			Msg: &homecallv1alpha.WaitForCallRequest{
				DeviceId: device.Msg.GetDevice().GetId(),
			},
		})
		require.NoError(t, err)
		defer waitResp.Close()

		for waitResp.Receive() {
			msg := waitResp.Msg()
			require.NotEmpty(t, msg.GetCallId())
			// TODO check jwt
			// TODO check that call details align with device
			return
		}

		require.NoError(t, waitResp.Err())
		require.Fail(t, "no call received")
	}() */

	/*resp, err := c.officeClient.StartCall(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.StartCallRequest]{
		Msg: &homecallv1alpha.StartCallRequest{
			DeviceId: device.Msg.GetDevice().GetId(),
		},
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)*/

	//wg.Wait()
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
