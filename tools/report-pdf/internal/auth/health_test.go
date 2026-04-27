package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchAPIVersion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","version":"v0.27.1","services":{"database":"healthy"}}`))
	}))
	defer server.Close()

	v, err := FetchAPIVersion(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v0.27.1" {
		t.Errorf("version = %q, want %q", v, "v0.27.1")
	}
}

func TestFetchAPIVersion_AcceptsServiceUnavailable(t *testing.T) {
	// /health returns 503 when the DB is down — but the API process
	// itself is running and the version field is populated. We still
	// want the version for the colophon in that case.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unhealthy","version":"v0.27.1","services":{"database":"unhealthy"}}`))
	}))
	defer server.Close()

	v, err := FetchAPIVersion(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v0.27.1" {
		t.Errorf("version = %q, want %q", v, "v0.27.1")
	}
}

func TestFetchAPIVersion_RejectsClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := FetchAPIVersion(server.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestFetchAPIVersion_RejectsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	_, err := FetchAPIVersion(server.URL)
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestFetchAPIVersion_ConnectionRefused(t *testing.T) {
	_, err := FetchAPIVersion("http://localhost:1")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}
