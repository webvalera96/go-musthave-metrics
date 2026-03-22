package grpcserver

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTrustedSubnetUnaryInterceptor_NoSubnet_Passes(t *testing.T) {
	t.Helper()
	ic := TrustedSubnetUnaryInterceptor(nil)
	called := false
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}
	out, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{}, h)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" || !called {
		t.Fatalf("expected handler to run, got %v, called=%v", out, called)
	}
}

func TestTrustedSubnetUnaryInterceptor_DeniedWithoutIP(t *testing.T) {
	t.Helper()
	_, n, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	ic := TrustedSubnetUnaryInterceptor(n)
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler must not run")
		return nil, nil
	}
	_, err = ic(context.Background(), nil, &grpc.UnaryServerInfo{}, h)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}

func TestTrustedSubnetUnaryInterceptor_AllowsTrustedIP(t *testing.T) {
	t.Helper()
	_, n, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	ic := TrustedSubnetUnaryInterceptor(n)
	md := metadata.Pairs("x-real-ip", "10.1.2.3")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	called := false
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}
	out, err := ic(ctx, nil, &grpc.UnaryServerInfo{}, h)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" || !called {
		t.Fatalf("expected ok, got %v called=%v", out, called)
	}
}
