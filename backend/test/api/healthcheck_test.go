package api

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	resp, err := http.DefaultClient.Get(fmt.Sprintf("%s/healthz", globalTestApp.ApiAddress()))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
