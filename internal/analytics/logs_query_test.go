package analytics

import "testing"

func TestCanUseFastLogsPath(t *testing.T) {
	if !canUseFastLogsPath("timestamp", logsQueryOptions{}, false) {
		t.Fatalf("expected timestamp sort to use fast path")
	}
	if canUseFastLogsPath("ip", logsQueryOptions{}, false) {
		t.Fatalf("expected ip sort to require join")
	}
	if canUseFastLogsPath("timestamp", logsQueryOptions{ipFilter: "127.0.0.1"}, false) {
		t.Fatalf("expected ip filter to disable fast path")
	}
	if canUseFastLogsPath("timestamp", logsQueryOptions{includeNewVisitor: true}, false) {
		t.Fatalf("expected new visitor query to disable fast path")
	}
	if canUseFastLogsPath("timestamp", logsQueryOptions{}, true) {
		t.Fatalf("expected distinct ip query to disable fast path")
	}
}

func TestNeedsLogsJoinForFilters(t *testing.T) {
	if needsLogsJoinForFilters(logsQueryOptions{}) {
		t.Fatalf("expected empty filters to skip join")
	}
	if !needsLogsJoinForFilters(logsQueryOptions{excludeInternal: true}) {
		t.Fatalf("expected excludeInternal to require join")
	}
	if !needsLogsJoinForFilters(logsQueryOptions{urlFilter: "/api"}) {
		t.Fatalf("expected urlFilter to require join")
	}
}

func TestBuildLogsOrderClause(t *testing.T) {
	got := buildLogsOrderClause("l.timestamp", "desc", "l.id")
	want := "l.timestamp desc, l.id desc"
	if got != want {
		t.Fatalf("unexpected order clause: got %q want %q", got, want)
	}
}

func TestShouldUseExactLogsCount(t *testing.T) {
	if shouldUseExactLogsCount(logsQueryOptions{}) {
		t.Fatalf("expected broad query to skip exact count")
	}
	if shouldUseExactLogsCount(logsQueryOptions{pageviewOnly: true}) {
		t.Fatalf("expected pageview-only query to skip exact count")
	}
	if !shouldUseExactLogsCount(logsQueryOptions{timeStart: 1710000000}) {
		t.Fatalf("expected time-bounded query to keep exact count")
	}
	if !shouldUseExactLogsCount(logsQueryOptions{filter: "/api"}) {
		t.Fatalf("expected keyword search to keep exact count")
	}
	if !shouldUseExactLogsCount(logsQueryOptions{statusCode: 500}) {
		t.Fatalf("expected exact status code filter to keep exact count")
	}
}
