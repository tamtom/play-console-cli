package listings

import "testing"

func TestValidateVideoURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ", true},
		{"https://youtube.com/watch?v=abc123", true},
		{"https://vimeo.com/123", false},
		{"not-a-url", false},
	}
	for _, tt := range tests {
		err := ValidateVideoURL(tt.url)
		if tt.valid && err != nil {
			t.Errorf("ValidateVideoURL(%q) = %v, want nil", tt.url, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateVideoURL(%q) = nil, want error", tt.url)
		}
	}
}
