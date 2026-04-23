package ingest

import "testing"

func TestBuildLogLineFingerprintUsesSourceAndOffset(t *testing.T) {
	line := `127.0.0.1 - - [14/Apr/2026:04:24:18 +0800] "GET /video.mkv HTTP/1.1" 206 1024 "-" "curl/8"`
	source := parseSourceContext{
		sourceKey: "source-a:/var/log/nginx/access.log",
		hasOffset: true,
	}

	first := buildLogLineFingerprint(source, 128, line)
	second := buildLogLineFingerprint(source, 128, line)
	if first == "" {
		t.Fatal("fingerprint is empty")
	}
	if first != second {
		t.Fatalf("fingerprint is not stable: %q != %q", first, second)
	}

	if got := buildLogLineFingerprint(source, 256, line); got == first {
		t.Fatal("fingerprint should change when the source offset changes")
	}

	otherSource := source
	otherSource.sourceKey = "source-b:/var/log/nginx/access.log"
	if got := buildLogLineFingerprint(otherSource, 128, line); got == first {
		t.Fatal("fingerprint should change when the source key changes")
	}
}

func TestStreamLogLineFingerprintUsesSource(t *testing.T) {
	line := `127.0.0.1 - - [14/Apr/2026:04:24:18 +0800] "GET /video.mkv HTTP/1.1" 206 1024 "-" "curl/8"`

	first := streamLogLineFingerprint("agent-a", line)
	second := streamLogLineFingerprint("agent-a", line)
	if first == "" {
		t.Fatal("fingerprint is empty")
	}
	if first != second {
		t.Fatalf("stream fingerprint is not stable: %q != %q", first, second)
	}
	if got := streamLogLineFingerprint("agent-b", line); got == first {
		t.Fatal("stream fingerprint should change when source id changes")
	}
}
