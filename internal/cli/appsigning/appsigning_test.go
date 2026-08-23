package appsigning

import (
	"context"
	"strings"
	"testing"
)

func TestEnroll_RequiresEnterpriseKMSAcknowledgement(t *testing.T) {
	cmd := EnrollCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "app.example", "--json", `{}`, "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--enterprise-self-hosted-kms") {
		t.Fatalf("expected enterprise scope error, got %v", err)
	}
}

func TestRotate_RejectsUnknownReasonBeforeAuthentication(t *testing.T) {
	cmd := RotateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "app.example", "--confirm-package", "app.example", "--json", `{"keyRotationReason":"BECAUSE"}`,
		"--enterprise-self-hosted-kms", "--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "keyRotationReason") {
		t.Fatalf("expected reason validation error, got %v", err)
	}
}

func TestEnroll_RequiresExactPackageConfirmation(t *testing.T) {
	cmd := EnrollCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "app.example", "--confirm-package", "other.example", "--json", `{}`, "--enterprise-self-hosted-kms", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "must exactly match") {
		t.Fatalf("expected package mismatch, got %v", err)
	}
}
