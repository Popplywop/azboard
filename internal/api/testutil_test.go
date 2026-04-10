package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/popplywop/azboard/internal/config"
)

// testServer creates an httptest.Server with the given handler and returns
// a *Client whose baseURL and orgURL point at the test server.
// The server is automatically closed when the test finishes.
func testServer(t *testing.T, handler http.Handler) (*httptest.Server, *Client) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	c := &Client{
		baseURL:    ts.URL,
		orgURL:     ts.URL,
		project:    "TestProject",
		httpClient: ts.Client(),
		authMethod: config.AuthPAT,
		token:      "test-pat-token",
		tokenExp:   time.Now().Add(24 * time.Hour),
	}
	return ts, c
}

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
