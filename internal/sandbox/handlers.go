package sandbox

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// endpointHandlers maps official discovery method IDs to sandbox behavior.
// Add entries here as the black-box suite grows; an unknown ID fails loudly
// with 501 instead of faking success.
func endpointHandlers() map[string]func(s *Server, w http.ResponseWriter, r *http.Request, params map[string]string) {
	return map[string]func(s *Server, w http.ResponseWriter, r *http.Request, params map[string]string){
		"androidpublisher.edits.insert":          editsInsert,
		"androidpublisher.edits.get":             editsGet,
		"androidpublisher.edits.delete":          editsDelete,
		"androidpublisher.edits.validate":        editsValidate,
		"androidpublisher.edits.commit":          editsCommit,
		"androidpublisher.edits.tracks.list":     tracksList,
		"androidpublisher.edits.tracks.get":      tracksGet,
		"androidpublisher.edits.tracks.update":   tracksUpdate,
		"androidpublisher.edits.listings.list":   listingsList,
		"androidpublisher.edits.listings.get":    listingsGet,
		"androidpublisher.edits.listings.update": listingsUpdate,
		"androidpublisher.edits.listings.delete": listingsDelete,
		"androidpublisher.edits.bundles.list":    bundlesList,
		"androidpublisher.edits.bundles.upload":  bundlesUpload,
		"androidpublisher.reviews.list":          reviewsList,
		"androidpublisher.inappproducts.list":    inappproductsList,
	}
}

func requirePackage(w http.ResponseWriter, params map[string]string) bool {
	if params["packageName"] != Package {
		writeError(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("Package not found: %s.", params["packageName"]))
		return false
	}
	return true
}

func (s *Server) edit(w http.ResponseWriter, params map[string]string) (*editState, bool) {
	e, ok := s.edits[params["editId"]]
	if !ok || e.pkg != params["packageName"] {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT",
			fmt.Sprintf("Invalid edit ID: %s.", params["editId"]))
		return nil, false
	}
	return e, true
}

// newEditState seeds a fresh edit with the app baseline: two tracks with one
// release each and one en-US listing.
func newEditState() *editState {
	return &editState{
		pkg: Package,
		tracks: map[string]map[string]any{
			"internal": {
				"track": "internal",
				"releases": []any{map[string]any{
					"name": "9 (1.9)", "status": "completed",
					"versionCodes": []any{"9"},
				}},
			},
			"production": {
				"track": "production",
				"releases": []any{map[string]any{
					"name": "8 (1.8)", "status": "completed",
					"versionCodes": []any{"8"},
				}},
			},
		},
		listings: map[string]map[string]any{
			"en-US": {
				"language":         "en-US",
				"title":            "Sandbox App",
				"shortDescription": "A seeded sandbox listing.",
				"fullDescription":  "The sandbox server seeds this listing for tests.",
			},
		},
	}
}

func editsInsert(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	s.nextEdit++
	id := fmt.Sprintf("sandbox-edit-%d", s.nextEdit)
	s.edits[id] = newEditState()
	writeJSON(w, map[string]any{"id": id, "expiryTimeSeconds": "9999999999"})
}

func editsGet(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	if _, ok := s.edit(w, params); !ok {
		return
	}
	writeJSON(w, map[string]any{"id": params["editId"], "expiryTimeSeconds": "9999999999"})
}

func editsDelete(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	if _, ok := s.edit(w, params); !ok {
		return
	}
	delete(s.edits, params["editId"])
	w.WriteHeader(http.StatusNoContent)
}

func editsValidate(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	if _, ok := s.edit(w, params); !ok {
		return
	}
	writeJSON(w, map[string]any{"id": params["editId"], "expiryTimeSeconds": "9999999999"})
}

func editsCommit(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	if _, ok := s.edit(w, params); !ok {
		return
	}
	// A committed edit is gone: later calls on it must fail like the real API.
	delete(s.edits, params["editId"])
	writeJSON(w, map[string]any{"id": params["editId"], "expiryTimeSeconds": "9999999999"})
}

func tracksList(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	tracks := []any{}
	for _, tr := range e.tracks {
		tracks = append(tracks, tr)
	}
	writeJSON(w, map[string]any{"kind": "androidpublisher#tracksListResponse", "tracks": tracks})
}

func tracksGet(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	tr, ok := e.tracks[params["track"]]
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("Track not found: %s.", params["track"]))
		return
	}
	writeJSON(w, tr)
}

func tracksUpdate(s *Server, w http.ResponseWriter, r *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	if _, ok := e.tracks[params["track"]]; !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("Track not found: %s.", params["track"]))
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid track body.")
		return
	}
	body["track"] = params["track"]
	e.tracks[params["track"]] = body
	writeJSON(w, body)
}

func listingsList(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	listings := []any{}
	for _, l := range e.listings {
		listings = append(listings, l)
	}
	writeJSON(w, map[string]any{"kind": "androidpublisher#listingsListResponse", "listings": listings})
}

func listingsGet(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	l, ok := e.listings[params["language"]]
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("Listing not found: %s.", params["language"]))
		return
	}
	writeJSON(w, l)
}

func listingsUpdate(s *Server, w http.ResponseWriter, r *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid listing body.")
		return
	}
	body["language"] = params["language"]
	e.listings[params["language"]] = body
	writeJSON(w, body)
}

func listingsDelete(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	delete(e.listings, params["language"])
	w.WriteHeader(http.StatusNoContent)
}

func bundlesList(s *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	writeJSON(w, map[string]any{"kind": "androidpublisher#bundlesListResponse", "bundles": e.bundles})
}

func bundlesUpload(s *Server, w http.ResponseWriter, r *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	e, ok := s.edit(w, params)
	if !ok {
		return
	}
	// Consume the upload body; content is not inspected. The version code
	// grows monotonically from the seeded internal release.
	_ = r.Body.Close()
	bundle := map[string]any{
		"versionCode": 10 + len(e.bundles),
		"sha256":      fmt.Sprintf("sandbox-sha-%d", len(e.bundles)+1),
	}
	e.bundles = append(e.bundles, bundle)
	writeJSON(w, bundle)
}

func reviewsList(_ *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	writeJSON(w, map[string]any{"reviews": []any{
		map[string]any{
			"reviewId":   "sandbox-review-1",
			"authorName": "Sandbox Tester",
			"comments": []any{map[string]any{
				"userComment": map[string]any{"text": "Works great.", "starRating": 5},
			}},
		},
	}})
}

func inappproductsList(_ *Server, w http.ResponseWriter, _ *http.Request, params map[string]string) {
	if !requirePackage(w, params) {
		return
	}
	writeJSON(w, map[string]any{
		"kind":         "androidpublisher#inappproductsListResponse",
		"inappproduct": []any{},
	})
}
