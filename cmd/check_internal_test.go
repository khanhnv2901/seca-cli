package cmd

import (
	"testing"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/check"
	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
)

func TestCanonicalTargetNormalizes(t *testing.T) {
	target := "example.com/path#section"
	got := canonicalTarget(target)
	want := "http://example.com/path"
	if got != want {
		t.Fatalf("canonicalTarget() = %s, want %s", got, want)
	}
}

func TestTargetSetAdd(t *testing.T) {
	set := newTargetSet()
	if !set.Add("https://example.com") {
		t.Fatal("expected first add to succeed")
	}
	if set.Add("https://example.com/") {
		t.Fatal("expected canonical duplicate to be rejected")
	}
	if set.Add("https://example.com/#frag") {
		t.Fatal("expected fragment duplicate to be rejected")
	}
	if !set.Add("https://example.com/login") {
		t.Fatal("expected unique path to be added")
	}
}

func TestResultAdapterToDomain_Success(t *testing.T) {
	adapter := &resultAdapter{}
	tlsExpiry := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	checkResult := checker.CheckResult{
		Status:       "ok",
		HTTPStatus:   200,
		ResponseTime: 123.45,
		TLSExpiry:    tlsExpiry.Format(time.RFC3339),
	}

	domainResult, err := adapter.toDomain("https://example.com", checkResult)
	if err != nil {
		t.Fatalf("toDomain() error = %v", err)
	}

	if domainResult.Status() != check.CheckStatusOK {
		t.Fatalf("expected OK status, got %s", domainResult.Status())
	}
	if domainResult.HTTPStatus() != 200 {
		t.Fatalf("expected HTTP 200, got %d", domainResult.HTTPStatus())
	}
	if domainResult.ResponseTime() != 123.45 {
		t.Fatalf("expected response time to propagate, got %f", domainResult.ResponseTime())
	}
	if !domainResult.TLSExpiry().Equal(tlsExpiry) {
		t.Fatalf("expected TLS expiry to be parsed, got %v want %v", domainResult.TLSExpiry(), tlsExpiry)
	}
}

func TestResultAdapterToDomain_ErrorPropagation(t *testing.T) {
	adapter := &resultAdapter{}
	checkResult := checker.CheckResult{
		Status: "fail",
		Error:  "connection timeout",
	}

	domainResult, err := adapter.toDomain("https://example.com", checkResult)
	if err != nil {
		t.Fatalf("toDomain() error = %v", err)
	}

	if domainResult.Status() != check.CheckStatusError {
		t.Fatalf("expected error status, got %s", domainResult.Status())
	}
	if domainResult.Error() != "connection timeout" {
		t.Fatalf("expected error message to propagate, got %s", domainResult.Error())
	}
}
