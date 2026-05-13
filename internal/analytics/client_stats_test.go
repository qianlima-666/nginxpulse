package analytics

import "testing"

func TestClientStatsOrderByExpr(t *testing.T) {
	if got := clientStatsOrderByExpr("url"); got != "pv DESC, uv DESC" {
		t.Fatalf("clientStatsOrderByExpr(url) = %q, want %q", got, "pv DESC, uv DESC")
	}

	for _, statsType := range []string{"referer", "user_browser", "user_device", "location"} {
		if got := clientStatsOrderByExpr(statsType); got != "uv DESC, pv DESC" {
			t.Fatalf("clientStatsOrderByExpr(%q) = %q, want %q", statsType, got, "uv DESC, pv DESC")
		}
	}
}
