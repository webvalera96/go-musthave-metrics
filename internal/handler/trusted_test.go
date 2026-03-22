package handler

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrustedSubnetMiddleware_AllowsWhenUnset(t *testing.T) {
	called := false
	h := TrustedSubnetMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.True(t, called)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestTrustedSubnetMiddleware_ForbiddenNoHeader(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	h := TrustedSubnetMiddleware(ipNet, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not run")
	}))
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTrustedSubnetMiddleware_AllowsInSubnet(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	h := TrustedSubnetMiddleware(ipNet, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set(XRealIPHeader, "10.1.2.3")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestTrustedSubnetMiddleware_SkipsGetPing(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	called := false
	h := TrustedSubnetMiddleware(ipNet, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.True(t, called)
	require.Equal(t, http.StatusOK, rr.Code)
}
