package corti

import (
	"strings"
	"testing"
)

func TestGetTokenRejectsPlaceholderCredentials(t *testing.T) {
	tm := &TokenManager{
		clientID:     "your_client_id_here",
		clientSecret: "your_client_secret_here",
		authURL:      "http://example.invalid/token",
	}

	_, err := tm.GetToken()
	if err == nil {
		t.Fatal("expected placeholder credentials to be rejected")
	}
	if !strings.Contains(err.Error(), "example placeholder values") {
		t.Fatalf("expected placeholder credential error, got %q", err.Error())
	}
}

func TestIsPlaceholderCredential(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "client id placeholder", value: "your_client_id_here", want: true},
		{name: "client secret placeholder", value: "your_client_secret_here", want: true},
		{name: "trims whitespace", value: " your_client_id_here ", want: true},
		{name: "real looking credential", value: "xstek-prod-client", want: false},
		{name: "empty is not placeholder", value: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlaceholderCredential(tt.value); got != tt.want {
				t.Fatalf("isPlaceholderCredential(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}
