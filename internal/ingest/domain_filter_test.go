package ingest

import "testing"

func TestDomainMatcherIncludesConfiguredDomain(t *testing.T) {
	matcher := newDomainMatcher([]string{"example.com", "api.example.net"})

	cases := []string{
		"example.com",
		"www.example.com",
		"EXAMPLE.COM:443",
		"https://api.example.net/path",
	}
	for _, host := range cases {
		if !matcher.includesHost(host) {
			t.Fatalf("expected host %q to match configured domains", host)
		}
	}
}

func TestDomainMatcherRejectsOtherDomains(t *testing.T) {
	matcher := newDomainMatcher([]string{"example.com"})

	if matcher.includesHost("other.com") {
		t.Fatal("expected other.com to be rejected")
	}
	if matcher.includesHost("badexample.com") {
		t.Fatal("expected suffix without domain boundary to be rejected")
	}
}

func TestDomainMatcherAllowsAggregateAndHostlessLogs(t *testing.T) {
	aggregate := newDomainMatcher(nil)
	if !aggregate.includesHost("other.com") {
		t.Fatal("expected empty domain config to include all hosts")
	}

	site := newDomainMatcher([]string{"example.com"})
	if !site.includesHost("") {
		t.Fatal("expected hostless logs to be kept for backward compatibility")
	}
}

func TestDomainMatcherNormalizesWildcardAndTrailingDot(t *testing.T) {
	matcher := newDomainMatcher([]string{"*.example.com."})

	if !matcher.includesHost("shop.example.com.") {
		t.Fatal("expected wildcard/trailing-dot domain to match")
	}
}
