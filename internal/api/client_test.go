package api

import (
	"errors"
	"net/http"
	"testing"
)

func TestDoRequestNon2xx(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte("forbidden"))
	})
	_, c := testServer(t, mux)

	err := c.get("/test", nil)
	if err == nil {
		t.Fatal("expected error for 403")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("got status %d, want 403", apiErr.StatusCode)
	}
}

func TestDoRequest429RateLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte("too many requests"))
	})
	_, c := testServer(t, mux)

	err := c.get("/test", nil)
	if !IsRateLimited(err) {
		t.Errorf("expected IsRateLimited to be true, got false for: %v", err)
	}
}

func TestGetContentSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo/items", func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); accept != "text/plain" {
			t.Errorf("expected Accept: text/plain, got %s", accept)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("file content here"))
	})
	_, c := testServer(t, mux)

	content, err := c.getContent("/git/repositories/repo/items?path=/file.go&$format=text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "file content here" {
		t.Errorf("got %q, want %q", content, "file content here")
	}
}

func TestGetContent404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo/items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	})
	_, c := testServer(t, mux)

	_, err := c.getContent("/git/repositories/repo/items?path=/missing.go&$format=text")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got: %v", err)
	}
}

func TestErrorHints(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"401", &APIError{StatusCode: 401}},
		{"404", &APIError{StatusCode: 404}},
		{"429", &APIError{StatusCode: 429}},
		{"500", &APIError{StatusCode: 500}},
		{"generic", errors.New("something")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := ErrorHint(tt.err)
			if hint == "" {
				t.Error("hint should not be empty")
			}
		})
	}
}
