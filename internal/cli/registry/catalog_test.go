package registry

import (
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestCatalog_CommandsForBuildsOnlySelectedCommand(t *testing.T) {
	selectedBuilds := 0
	unselectedBuilds := 0
	catalog := &Catalog{specs: []CommandSpec{
		commandSpec("selected", "Selected command.", func() *ffcli.Command {
			selectedBuilds++
			return &ffcli.Command{Name: "selected"}
		}),
		commandSpec("unselected", "Unselected command.", func() *ffcli.Command {
			unselectedBuilds++
			return &ffcli.Command{Name: "unselected"}
		}),
	}}

	commands := catalog.CommandsFor("selected")
	if len(commands) != 2 {
		t.Fatalf("command count = %d, want 2", len(commands))
	}
	if selectedBuilds != 1 {
		t.Fatalf("selected factory calls = %d, want 1", selectedBuilds)
	}
	if unselectedBuilds != 0 {
		t.Fatalf("unselected factory calls = %d, want 0", unselectedBuilds)
	}
	if commands[1].Name != "unselected" || commands[1].ShortHelp != "Unselected command." {
		t.Fatalf("unselected metadata was not preserved: %#v", commands[1])
	}
}

func TestCatalog_MetadataCarriesAgentDiscoveryFields(t *testing.T) {
	catalog := NewCatalog("test", nil)
	specs := catalog.Specs()
	if len(specs) == 0 {
		t.Fatal("expected command specs")
	}
	for _, spec := range specs {
		if spec.Path == "" || spec.Summary == "" || spec.Stability == "" || spec.Provider == "" {
			t.Fatalf("incomplete command spec: %#v", spec)
		}
	}
}
