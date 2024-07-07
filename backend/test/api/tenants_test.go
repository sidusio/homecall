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
	"net/http"
	homecallv1alpha "sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/services/auth"
	"testing"
)

type clients struct {
	tenantClient homecallv1alphaconnect.TenantServiceClient
	officeClient homecallv1alphaconnect.OfficeServiceClient
	deviceClient homecallv1alphaconnect.DeviceServiceClient
}

type tenantTest = func(t *testing.T, ctx context.Context, tenant *homecallv1alpha.Tenant, adminUser string, c clients)

func TestApi(t *testing.T) {
	t.Parallel()
	WithApp(t, func(t *testing.T, ctx context.Context, apiAddress string) {
		tenantClient := homecallv1alphaconnect.NewTenantServiceClient(http.DefaultClient, apiAddress)
		officeClient := homecallv1alphaconnect.NewOfficeServiceClient(http.DefaultClient, apiAddress)
		deviceClient := homecallv1alphaconnect.NewDeviceServiceClient(http.DefaultClient, apiAddress)

		testCases := []struct {
			name string
			test tenantTest
		}{
			{
				name: "delete tenants",
				test: testTenantDelete,
			},
			{
				name: "tenant members add",
				test: testTenantMembersAdd,
			},
			{
				name: "device call",
				test: testDeviceCall,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				adminUser := randomUser(t)
				tenant := createTestTenant(t, testCase.name, adminUser, tenantClient)

				testCase.test(t, ctx, tenant, adminUser, clients{
					tenantClient: tenantClient,
					officeClient: officeClient,
					deviceClient: deviceClient,
				})

			})
		}
	})

}

func testDeviceCall(t *testing.T, ctx context.Context, tenant *homecallv1alpha.Tenant, adminUser string, c clients) {
	device, err := c.officeClient.CreateDevice(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateDeviceRequest]{
		Msg: &homecallv1alpha.CreateDeviceRequest{
			Name:     fmt.Sprintf("test-%s", randomUser(t)),
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

	_, err = c.deviceClient.Enroll(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.EnrollRequest]{
		Msg: &homecallv1alpha.EnrollRequest{
			EnrollmentKey: device.Msg.GetDevice().GetEnrollmentKey(),
			PublicKey:     publicKey.String(),
		},
	}))
	require.NoError(t, err)

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

	resp, err := c.officeClient.StartCall(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.StartCallRequest]{
		Msg: &homecallv1alpha.StartCallRequest{
			DeviceId: device.Msg.GetDevice().GetId(),
		},
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)

	//wg.Wait()
}

func testTenantMembersAdd(t *testing.T, ctx context.Context, tenant *homecallv1alpha.Tenant, adminUser string, c clients) {
	memberUser := randomUser(t)
	nonMemberUser := randomUser(t)
	newAdminUser := randomUser(t)
	newMemberUser := randomUser(t)

	// Check if member is added
	isMember := func(t *testing.T, email string, role homecallv1alpha.Role) bool {
		resp, err := c.tenantClient.ListTenantMembers(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.ListTenantMembersRequest]{
			Msg: &homecallv1alpha.ListTenantMembersRequest{
				TenantId: tenant.Id,
			},
		}))
		require.NoError(t, err)
		for _, m := range resp.Msg.TenantMembers {
			if m.Email == email && m.Role == role {
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
			resp, err := c.tenantClient.CreateTenantMember(ctx, auth.WithDummyToken(step.asUser, &connect.Request[homecallv1alpha.CreateTenantMemberRequest]{
				Msg: &homecallv1alpha.CreateTenantMemberRequest{
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
				require.True(t, isMember(t, step.addUser, step.withRole))

				assert.Equal(t, resp.Msg.GetTenantMember().GetRole(), step.withRole)
				assert.Equal(t, resp.Msg.GetTenantMember().GetEmail(), step.addUser)
				assert.Equal(t, resp.Msg.GetTenantMember().GetTenantId(), tenant.Id)
			}
		})
	}
}

func testTenantDelete(t *testing.T, ctx context.Context, tenant *homecallv1alpha.Tenant, adminUser string, c clients) {
	memberUser := randomUser(t)
	nonMemberUser := randomUser(t)

	// Check if tenant is deleted
	isDeleted := func(t *testing.T) bool {
		resp, err := c.tenantClient.ListTenants(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.ListTenantsRequest]{
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
	_, err := c.tenantClient.CreateTenantMember(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateTenantMemberRequest]{
		Msg: &homecallv1alpha.CreateTenantMemberRequest{
			TenantId: tenant.Id,
			Email:    memberUser,
			Role:     homecallv1alpha.Role_ROLE_MEMBER,
		},
	}))
	require.NoError(t, err)

	// Attempt to delete tenant as non-member
	_, err = c.tenantClient.RemoveTenant(ctx, auth.WithDummyToken(nonMemberUser, &connect.Request[homecallv1alpha.RemoveTenantRequest]{
		Msg: &homecallv1alpha.RemoveTenantRequest{
			Id: tenant.Id,
		},
	}))
	require.Error(t, err)
	require.False(t, isDeleted(t))

	// Attempt to delete tenant as member
	_, err = c.tenantClient.RemoveTenant(ctx, auth.WithDummyToken(memberUser, &connect.Request[homecallv1alpha.RemoveTenantRequest]{
		Msg: &homecallv1alpha.RemoveTenantRequest{
			Id: tenant.Id,
		},
	}))
	require.Error(t, err)
	require.False(t, isDeleted(t))

	// Attempt to delete tenant as admin
	_, err = c.tenantClient.RemoveTenant(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.RemoveTenantRequest]{
		Msg: &homecallv1alpha.RemoveTenantRequest{
			Id: tenant.Id,
		},
	}))
	require.NoError(t, err)
	require.True(t, isDeleted(t))
}
