package cmd

import "testing"

type stubAppContext struct {
	Operator string
}

func TestValidateCheckParams_Success(t *testing.T) {
	params := checkParams{
		ID:         "eng123",
		ROEConfirm: true,
	}
	appCtx := &AppContext{Operator: "tester"}
	if err := validateCheckParams(params, appCtx, false, 0); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}
}

func TestValidateCheckParams_MissingID(t *testing.T) {
	params := checkParams{ROEConfirm: true}
	appCtx := &AppContext{Operator: "tester"}
	if err := validateCheckParams(params, appCtx, false, 0); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestValidateCheckParams_InvalidID(t *testing.T) {
	params := checkParams{ID: "bad/id", ROEConfirm: true}
	appCtx := &AppContext{Operator: "tester"}
	if err := validateCheckParams(params, appCtx, false, 0); err == nil {
		t.Fatal("expected error for invalid id")
	}
}

func TestValidateCheckParams_ROENotConfirmed(t *testing.T) {
	params := checkParams{ID: "eng123"}
	appCtx := &AppContext{Operator: "tester"}
	if err := validateCheckParams(params, appCtx, false, 0); err == nil {
		t.Fatal("expected error when --roe-confirm missing")
	}
}

func TestValidateCheckParams_OperatorRequired(t *testing.T) {
	params := checkParams{ID: "eng123", ROEConfirm: true}
	appCtx := &AppContext{Operator: ""}
	if err := validateCheckParams(params, appCtx, false, 0); err == nil {
		t.Fatal("expected error when operator missing")
	}
}

func TestValidateCheckParams_ComplianceModeRetention(t *testing.T) {
	params := checkParams{ID: "eng123", ROEConfirm: true, ComplianceMode: true}
	appCtx := &AppContext{Operator: "tester"}
	err := validateCheckParams(params, appCtx, true, 0)
	if err == nil || err.Error() != "in compliance mode, --audit-append-raw requires --retention-days=<N>" {
		t.Fatalf("expected retention error, got %v", err)
	}
}

func TestValidateCheckParams_ComplianceModeNoOperator(t *testing.T) {
	params := checkParams{ID: "eng123", ROEConfirm: true, ComplianceMode: true}
	appCtx := &AppContext{Operator: ""}
	err := validateCheckParams(params, appCtx, false, 0)
	if err == nil {
		t.Fatal("expected operator error in compliance mode")
	}
}

func TestValidateCheckParams_ComplianceModePasses(t *testing.T) {
	params := checkParams{ID: "eng123", ROEConfirm: true, ComplianceMode: true}
	appCtx := &AppContext{Operator: "tester"}
	if err := validateCheckParams(params, appCtx, true, 10); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}
}
