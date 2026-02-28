package config

import "testing"

func TestRequireEnv(t *testing.T) {
	t.Setenv("TEST_REQUIRED_VAR", "value")

	if err := RequireEnv("TEST_REQUIRED_VAR"); err != nil {
		t.Errorf("expected no error for set var, got: %v", err)
	}

	err := RequireEnv("TEST_REQUIRED_VAR", "TEST_MISSING_VAR")
	if err == nil {
		t.Fatal("expected error for missing var")
	}
	if got := err.Error(); got != "missing required env vars: TEST_MISSING_VAR" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestRequireEnv_MultipleMissing(t *testing.T) {
	err := RequireEnv("MISSING_A", "MISSING_B")
	if err == nil {
		t.Fatal("expected error for multiple missing vars")
	}
	if got := err.Error(); got != "missing required env vars: MISSING_A, MISSING_B" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestRequireEnv_NoArgs(t *testing.T) {
	if err := RequireEnv(); err != nil {
		t.Errorf("expected no error with no args, got: %v", err)
	}
}
