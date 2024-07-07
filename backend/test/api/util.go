package api

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"os"
	"sidus.io/home-call/app"
	homecallv1alpha "sidus.io/home-call/gen/connect/homecall/v1alpha"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/postgresdb"
	"sidus.io/home-call/services/auth"
	"sidus.io/home-call/util"
	"strconv"
	"testing"
	"time"
)

func WithApp(t *testing.T, f func(t *testing.T, ctx context.Context, apiAddress string)) {
	t.Helper()
	postgresdb.WithPostgres(t, func(t *testing.T, connectionDetails postgresdb.DirectConfig) {
		ctx, cancel := context.WithCancel(context.Background())

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		port, err := getNextAvailablePort()
		require.NoError(t, err)

		var cfg app.Config
		err = envconfig.Process("TEST_HOMECALL", &cfg)
		if err != nil && err.Error() != "required key DB_PASSWORD missing value" {
			require.NoError(t, err)
		}
		cfg.DBHost = connectionDetails.Hostname
		cfg.DBPort = connectionDetails.Port
		cfg.DBUser = connectionDetails.UserName
		cfg.DBPassword = connectionDetails.Password
		cfg.DBName = connectionDetails.Database
		cfg.Port = port
		cfg.AuthDisabled = true
		cfg.JitsiKeyRaw = dummyPemKey

		eg, ctx := errgroup.WithContext(ctx)
		eg.Go(func() error {
			return app.Run(ctx, logger, cfg)
		})

		eg.Go(func() error {
			defer cancel()

			err = waitForAddress(ctx, fmt.Sprintf("localhost:%s", port))
			if err != nil {
				return fmt.Errorf("failed to wait for address: %w", err)
			}

			f(t, ctx, fmt.Sprintf("http://localhost:%s", port))
			return nil
		})

		err = eg.Wait()
		require.NoError(t, err)
	})
}

func createTestTenant(t *testing.T, name string, adminUser string, tenantClient homecallv1alphaconnect.TenantServiceClient) *homecallv1alpha.Tenant {
	ctx := context.Background()

	createRsp, err := tenantClient.CreateTenant(ctx, auth.WithDummyToken(adminUser, &connect.Request[homecallv1alpha.CreateTenantRequest]{
		Msg: &homecallv1alpha.CreateTenantRequest{
			Name: name,
		},
	}))
	require.NoError(t, err)

	return createRsp.Msg.GetTenant()
}

func randomUser(t *testing.T) string {
	user, err := util.RandomString(10)
	require.NoError(t, err)
	return user + "@example.com"
}

func getNextAvailablePort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", fmt.Errorf("failed to get next available port: %w", err)
	}
	defer l.Close()

	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func waitForAddress(ctx context.Context, address string) error {
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, _ := net.Dial("tcp", address)
			if conn != nil {
				defer conn.Close()
				return
			}
			time.Sleep(time.Second / 10)
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done")
	case <-time.After(time.Second * 10):
		return fmt.Errorf("timeout")
	case <-done:
		return nil
	}
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
