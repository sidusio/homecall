package api

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"net"
	homecallv1alpha "sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/services/auth"
	"sidus.io/home-call/util"
	"strconv"
	"strings"
	"testing"
)

func createTestTenant(name string, adminUser string, tenantClient homecallv1alphaconnect.TenantServiceClient) (*homecallv1alpha.Tenant, error) {
	ctx := context.Background()

	createRsp, err := tenantClient.CreateTenant(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateTenantRequest]{
		Msg: &homecallv1alpha.CreateTenantRequest{
			Name: name,
		},
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create test tenant: %w", err)
	}

	return createRsp.Msg.GetTenant(), nil
}

func randomUser() string {
	user, err := util.RandomString(10)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(user) + "@example.com"
}

func getNextAvailablePort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", fmt.Errorf("failed to get next available port: %w", err)
	}
	defer l.Close()

	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}

var dummyPemKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC8yUGgnYiyPxKw
n0M6RrstX7sfXXvEaK3SQypjgxxt5XAXy8yJJpjoJ0LconGkROqhQc3lt23dkscb
CN7vHye3FD0rLrGfygnhNNtv2lK9MHNuWI5uq7WAU5PzobTs2KimdkHXbEE4IIGw
8f1rs+7esykFHAGxeK3f9shp+DiZDhQi9iINAQJagmLC0y2JSeLAvjLWUZKLtufD
7qQKRa9GnjtklrSml0B0BL6iRvX3Ove22WWk03fEFVQPHwt8jAmSdZSIhrQ0Jgzz
YfGhE2bLPJFjyDa3aL3XaAB5Kfb5vom0Meawg2F2qFpPdG04275OvzsaaMs17Eg6
8oLrXRWXAgMBAAECggEAQiqBe2MrSVnM2aWAIPkwWjdOtLAFlHGh1mte/HCz8ppz
Hov5vGoQNnGoP/8ZOFtFJs6S9PvEoF90tDd4NzPirgqEY9GiRKBBtTJa5ImO7SsB
kf+ssAIzg24HkWCwMkC/X1RcQD37X8oY2mT+DpUKV/hQHK/TshlbS39Jf8aVQ6Lu
mE0ZWWOslqTZ2cbbtKwjdPg/mZzlkP1xqANRUMmIPhYMllYf3T8yZGlKcCdMUxBR
rdjn4LEKoHlH03svHyrEtY4ofEkn9av/30D1+EG7Oa1jfSctddRY2KLAShoAVv92
NLwo85bBF/SdDNiJima0pivxpB5n2anGfnrLXOrLhQKBgQDrmGlhgOVEmMDI5xT1
etjHRdNHRm5A7gXPo9Plt6vWK4FNIParCK8CjkO+KfgdmzGQXKwHnwXstSlaiSoW
uVIWjB38FEt9P24BoD+2jTXTTesuFzCivByCqgHCN747JWTef5kBhc4q2BjkAAsC
/7WyiuXj+E6A3jgRhD2pcVDNzQKBgQDNIv9Z44xCDAjB9/RvYBZ78kOOqaUD22ny
d1l4euGoBtHkxQvtNBrg5o3ohP6CvOo3eePVj0CNfN7PEXeLHmgWGFwhlG2iBM63
+8vNftRZHKYM98fedwuU1dBKbYIQVR/RODciAufwUvzrUwK9rHiR9GyjOfmoVcH5
753DkvSs8wKBgQCULuARgOYzuDSB6L7JDESvSh7y1LziQBQNnwjXkygU5HZGkfY0
a5jQbbT0NiemT4fkOjXF8WLjmKrzFBUSB+w23Fi7xfQZSj0h7q5EXxs81eSXr+Ra
ZyEzmkTS6QbQ4ttIC0+sooGjdxpoxhInB7k8HJsuQW73JU50zg2OtwRQ/QKBgFkl
HFKzz//juuqQFmlQGHVEkpcsoclLUH9N3lO9EtMyI4SHHOe3/PY/OuwQ34lxD1eM
YLYtyp+x5CGYNZr/W7w+Wcs99WazMCJECg0DUMRo7sAz7Wd/1EiZoiq17A+s7ma9
RzhAiwqlBcQ+DrLegIbs8Uj9qMC+g81Zk/WppyqBAoGBALSJCfA3WZaK9zE1hX4Z
TpjAWojTcYtuk0ruH5lPcyKImQYphyyT2W32BnfaHP6wSscZAws4sc5yDcbcHle3
aOlWmWafVx8Cj1l/a3l+N4oOZ4RPP+7XDovmVQJD8VjydvmWGgNEbLM0ImMQ1oGr
jQy6/JpIaFelxUNSamglf5+2
-----END PRIVATE KEY-----
`
