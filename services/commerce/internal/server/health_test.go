package server

import (
	"context"
	"testing"

	v1 "kratos-payment-lab/api/commerce/v1"
)

func TestHealthService_Check(t *testing.T) {
	s := NewHealthService()
	resp, err := s.Check(context.Background(), &v1.CheckRequest{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if resp.Service != "commerce" {
		t.Errorf("Check() Service = %q, want %q", resp.Service, "commerce")
	}
	if resp.Status != "SERVING" {
		t.Errorf("Check() Status = %q, want %q", resp.Status, "SERVING")
	}
}
