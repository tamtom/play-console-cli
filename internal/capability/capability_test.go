package capability

import "testing"

func TestList_FiltersByStatusAndProvider(t *testing.T) {
	items, err := List(Filter{
		Status:   StatusOfficial,
		Provider: "android-publisher-api",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected official Android Publisher capabilities")
	}
	for _, item := range items {
		if item.Status != StatusOfficial || item.Provider != "android-publisher-api" {
			t.Fatalf("filter returned unexpected item: %+v", item)
		}
	}
}

func TestList_RejectsUnknownStatus(t *testing.T) {
	if _, err := List(Filter{Status: "experimental"}); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestList_ReturnsIndependentSlice(t *testing.T) {
	first, err := List(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	first[0].ID = "changed"

	second, err := List(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if second[0].ID == "changed" {
		t.Fatal("List leaked mutable catalog state")
	}
}

func TestCatalogIncludesNewOfficialAPIFamiliesAndSigningBoundary(t *testing.T) {
	items, err := List(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Capability{}
	for _, item := range items {
		byID[item.ID] = item
	}
	wantOfficial := []string{"app.enterprise_kms_signing", "app.third_party_store", "app.reporting_metric_sets", "app.checks_repo_scans", "app.play_integrity", "app.developer_id_status"}
	for _, id := range wantOfficial {
		if byID[id].Status != StatusOfficial {
			t.Errorf("%s status = %q", id, byID[id].Status)
		}
	}
	if byID["app.standard_signing_enrollment"].Status != StatusManual {
		t.Errorf("standard signing must remain manual")
	}
	if experiment := byID["app.store_listing_experiments"]; experiment.Status != StatusManual || experiment.Command != "gplay experiments" {
		t.Errorf("store listing experiment boundary = %#v", experiment)
	}
}
