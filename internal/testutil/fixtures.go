package testutil

// ---------------------------------------------------------------------------
// JSON fixture data for common Google Play Android Publisher API responses.
// These are plain Go maps; tests marshal them as needed.
// ---------------------------------------------------------------------------

// EditFixture returns a sample Edits.insert / Edits.get response.
func EditFixture() map[string]interface{} {
	return map[string]interface{}{
		"id":                "edit-123",
		"expiryTimeSeconds": "1700000000",
	}
}

// TrackFixture returns a sample Tracks.get response with a single release.
func TrackFixture() map[string]interface{} {
	return map[string]interface{}{
		"track": "production",
		"releases": []map[string]interface{}{
			{
				"name":         "1.0.0",
				"versionCodes": []string{"100"},
				"status":       "completed",
				"userFraction": 1.0,
			},
		},
	}
}

// TracksListFixture returns a sample Tracks.list response.
func TracksListFixture() map[string]interface{} {
	return map[string]interface{}{
		"kind": "androidpublisher#tracksListResponse",
		"tracks": []map[string]interface{}{
			TrackFixture(),
			{
				"track": "beta",
				"releases": []map[string]interface{}{
					{
						"name":         "1.1.0-beta",
						"versionCodes": []string{"110"},
						"status":       "completed",
					},
				},
			},
		},
	}
}

// ReviewFixture returns a sample Reviews.get response.
func ReviewFixture() map[string]interface{} {
	return map[string]interface{}{
		"reviewId":   "review-abc",
		"authorName": "Test User",
		"comments": []map[string]interface{}{
			{
				"userComment": map[string]interface{}{
					"text":           "Great app!",
					"starRating":     5,
					"lastModified":   map[string]interface{}{"seconds": "1700000000"},
					"appVersionCode": 100,
					"appVersionName": "1.0.0",
					"device":         "Pixel 6",
				},
			},
		},
	}
}

// ReviewsListFixture returns a sample Reviews.list response.
func ReviewsListFixture() map[string]interface{} {
	return map[string]interface{}{
		"reviews": []map[string]interface{}{
			ReviewFixture(),
		},
		"tokenPagination": map[string]interface{}{
			"nextPageToken": "",
		},
	}
}

// ListingFixture returns a sample Listings.get response.
func ListingFixture() map[string]interface{} {
	return map[string]interface{}{
		"language":         "en-US",
		"title":            "My App",
		"fullDescription":  "A great application.",
		"shortDescription": "Great app",
		"video":            "",
	}
}

// ListingsListFixture returns a sample Listings.list response.
func ListingsListFixture() map[string]interface{} {
	return map[string]interface{}{
		"kind": "androidpublisher#listingsListResponse",
		"listings": []map[string]interface{}{
			ListingFixture(),
		},
	}
}

// ---------------------------------------------------------------------------
// Error fixtures
// ---------------------------------------------------------------------------

// ErrorFixture404 returns a Google-style 404 error response body.
func ErrorFixture404() map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"code":    404,
			"message": "No application was found for the given package name.",
			"status":  "NOT_FOUND",
		},
	}
}

// ErrorFixture403 returns a Google-style 403 error response body.
func ErrorFixture403() map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"code":    403,
			"message": "The caller does not have permission.",
			"status":  "PERMISSION_DENIED",
		},
	}
}

// ErrorFixture409 returns a Google-style 409 error response body.
func ErrorFixture409() map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"code":    409,
			"message": "Another edit is already open for this application.",
			"status":  "ALREADY_EXISTS",
		},
	}
}
