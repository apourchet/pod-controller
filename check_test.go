package controller

import (
	"errors"
	"net/http"
	"os/exec"
	"testing"

	"github.com/apourchet/fakenet"
	"github.com/stretchr/testify/require"
)

func TestShellCheck(t *testing.T) {
	t.Run("healthy_run", func(t *testing.T) {
		cmd := exec.Command("sleep", "0.1")
		check := NewShellCheck(cmd)
		success, err := check.Run()
		require.True(t, success)
		require.NoError(t, err)
	})
	t.Run("unhealthy_run", func(t *testing.T) {
		cmd := exec.Command("unknowncommand", "unknownarg")
		check := NewShellCheck(cmd)
		success, err := check.Run()
		require.False(t, success)
		require.Error(t, err)
	})
}

func TestHTTPCheck(t *testing.T) {
	t.Run("200_OK", func(t *testing.T) {
		net := fakenet.New()
		net.CatchAll(http.StatusBadRequest, "malformed request")
		net.InterceptURL("http://bogus/", http.StatusOK, "OK")

		check := NewHTTPCheck("bogus", "/")
		check.Client = net
		check.AddHeader("X-HEADER-NAME", "value")

		success, err := check.Run()
		require.True(t, success)
		require.NoError(t, err)
	})
	t.Run("400_unhealthy", func(t *testing.T) {
		net := fakenet.New()
		net.CatchAll(http.StatusBadRequest, "Bad Request")

		check := NewHTTPCheck("bogus", "/")
		check.Client = net

		success, err := check.Run()
		require.False(t, success)
		require.Equal(t, ErrBadStatusCode, err)
	})
	t.Run("bad_url", func(t *testing.T) {
		net := fakenet.New()
		net.CatchAll(http.StatusOK, "OK")

		check := NewHTTPCheck("bogus", "/")
		check.Client = net
		check.Scheme = "bad_scheme!"

		success, err := check.Run()
		require.False(t, success)
		require.Error(t, err)
	})
	t.Run("error_response", func(t *testing.T) {
		net := fakenet.New()
		catchall := fakenet.CatchAllInterceptor(nil, errors.New("Fell through to the catch all"))
		net.Intercept(catchall)

		check := NewHTTPCheck("bogus", "/")
		check.Client = net

		success, err := check.Run()
		require.False(t, success)
		require.Error(t, err)
	})
}
