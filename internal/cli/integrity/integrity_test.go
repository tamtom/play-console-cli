package integrity

import (
	"context"
	"strings"
	"testing"
)

func TestDecodeRequiresTokenBeforeAuthentication(t *testing.T) {
	cmd := DecodeCommand(false)
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--token") {
		t.Fatalf("expected token error, got %v", err)
	}
}

func TestDeviceRecallRequiresSecurityAcknowledgement(t *testing.T) {
	cmd := DeviceRecallWriteCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--json", `{"integrityToken":"x","newValues":{"bitFirst":true}}`, "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--security-fraud-abuse-use") {
		t.Fatalf("expected use restriction error, got %v", err)
	}
}
