package offers

import (
	"context"
	"strings"
	"testing"
)

func TestUpdateCommand_EmptyJSON_NoUpdateMask_ReturnsError(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--product-id", "test",
		"--base-plan-id", "monthly",
		"--offer-id", "trial",
		"--json", `{}`,
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty JSON without --update-mask")
	}
	if !strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("error should mention no updatable fields, got: %s", err.Error())
	}
}

func TestUpdateCommand_OnlyImmutableFields_ReturnsError(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--product-id", "test",
		"--base-plan-id", "monthly",
		"--offer-id", "trial",
		"--json", `{"packageName":"com.example","productId":"test","basePlanId":"monthly","offerId":"trial"}`,
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for only immutable fields")
	}
	if !strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("error should mention no updatable fields, got: %s", err.Error())
	}
}

func TestUpdateCommand_WithExplicitUpdateMask_SkipsDeriving(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--product-id", "test",
		"--base-plan-id", "monthly",
		"--offer-id", "trial",
		"--json", `{}`,
		"--update-mask", "phases",
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil && strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("explicit --update-mask should skip derive; got: %s", err.Error())
	}
}
