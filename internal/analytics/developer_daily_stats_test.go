package analytics

import (
	"strings"
	"testing"
)

func TestDeveloperIssueQueriesJoinURLDimension(t *testing.T) {
	topIssuesQuery := buildDeveloperTopIssuesQuery("site1", false)
	if !strings.Contains(topIssuesQuery, `JOIN "site1_dim_url" u ON u.id = l.url_id`) {
		t.Fatalf("expected top issues query to join dim_url, got: %s", topIssuesQuery)
	}
	if !strings.Contains(topIssuesQuery, `SELECT
			u.url,`) {
		t.Fatalf("expected top issues query to select u.url, got: %s", topIssuesQuery)
	}
	if !strings.Contains(topIssuesQuery, `GROUP BY u.url`) {
		t.Fatalf("expected top issues query to group by u.url, got: %s", topIssuesQuery)
	}

	compareQuery := buildDeveloperIssueForURLQuery("site1", false)
	if !strings.Contains(compareQuery, `JOIN "site1_dim_url" u ON u.id = l.url_id`) {
		t.Fatalf("expected compare query to join dim_url, got: %s", compareQuery)
	}
	if !strings.Contains(compareQuery, `AND u.url = $4`) {
		t.Fatalf("expected compare query to filter by u.url, got: %s", compareQuery)
	}
	if !strings.Contains(compareQuery, `GROUP BY u.url`) {
		t.Fatalf("expected compare query to group by u.url, got: %s", compareQuery)
	}
}

func TestDeveloperIssueQueriesSupportURLFilter(t *testing.T) {
	topIssuesQuery := buildDeveloperTopIssuesQuery("site1", true)
	if !strings.Contains(topIssuesQuery, `WHERE l.timestamp >= $2 AND l.timestamp < $3 AND u.url LIKE $4`) {
		t.Fatalf("expected top issues query to filter by URL pattern, got: %s", topIssuesQuery)
	}

	compareQuery := buildDeveloperIssueForURLQuery("site1", true)
	if !strings.Contains(compareQuery, `AND u.url = $4 AND u.url LIKE $5`) {
		t.Fatalf("expected compare query to include exact URL and URL pattern filters, got: %s", compareQuery)
	}
}
