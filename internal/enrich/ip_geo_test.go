package enrich

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestGetIPLocationBatchFallsBackToUnknownWhenRemoteUnavailable(t *testing.T) {
	ResetIPGeoCache()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked", http.StatusForbidden)
	}))
	defer server.Close()

	originalClientFactory := newIPAPIHTTPClient
	newIPAPIHTTPClient = func() *http.Client {
		return server.Client()
	}
	defer func() {
		newIPAPIHTTPClient = originalClientFactory
	}()

	originalURL := resolveIPAPIURLFunc
	resolveIPAPIURLFunc = func() string {
		return server.URL
	}
	defer func() {
		resolveIPAPIURLFunc = originalURL
	}()

	results, failures, err := GetIPLocationBatch([]string{"203.0.113.10", "198.51.100.20"})
	if err == nil {
		t.Fatal("expected remote lookup error")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, ip := range []string{"203.0.113.10", "198.51.100.20"} {
		entry, ok := results[ip]
		if !ok {
			t.Fatalf("missing result for %s", ip)
		}
		if entry.Domestic != "未知" || entry.Global != "未知" || entry.Source != "unknown" {
			t.Fatalf("unexpected fallback result for %s: %+v", ip, entry)
		}
		if failures[ip] != "http_status" {
			t.Fatalf("expected http_status failure for %s, got %q", ip, failures[ip])
		}
	}
}

func TestQueryIPLocationRemoteBatchStopsAfterFirstServiceFailure(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Error(w, "blocked", http.StatusTooManyRequests)
	}))
	defer server.Close()

	originalClientFactory := newIPAPIHTTPClient
	newIPAPIHTTPClient = func() *http.Client {
		return server.Client()
	}
	defer func() {
		newIPAPIHTTPClient = originalClientFactory
	}()

	originalURL := resolveIPAPIURLFunc
	resolveIPAPIURLFunc = func() string {
		return server.URL
	}
	defer func() {
		resolveIPAPIURLFunc = originalURL
	}()

	ips := make([]string, 0, 250)
	for i := 0; i < 250; i++ {
		ips = append(ips, "203.0.113.10")
	}

	_, failures, err := queryIPLocationRemoteBatch(ips)
	if err == nil {
		t.Fatal("expected remote batch error")
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected 1 remote request, got %d", hits)
	}
	if len(failures) != 1 {
		t.Fatalf("expected failures to be deduplicated by IP, got %d", len(failures))
	}
	if failures["203.0.113.10"] != "http_status" {
		t.Fatalf("unexpected failure reason: %q", failures["203.0.113.10"])
	}
}
