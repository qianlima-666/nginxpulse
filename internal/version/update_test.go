package version

import "testing"

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "same stable", left: "v1.6.17", right: "v1.6.17", want: 0},
		{name: "higher patch", left: "v1.6.18", right: "v1.6.17", want: 1},
		{name: "lower minor", left: "v1.5.9", right: "v1.6.0", want: -1},
		{name: "stable beats prerelease", left: "v1.6.17", right: "v1.6.17-beta.1", want: 1},
		{name: "higher prerelease core wins", left: "v1.6.18-beta.1", right: "v1.6.17", want: 1},
		{name: "numeric prerelease compare", left: "v1.6.17-beta.2", right: "v1.6.17-beta.1", want: 1},
		{name: "string prerelease compare", left: "v1.6.17-rc.1", right: "v1.6.17-beta.4", want: 1},
		{name: "invalid version", left: "dev", right: "v1.6.17", want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := compareVersions(tt.left, tt.right); got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestIsStableVersion(t *testing.T) {
	t.Parallel()

	if !isStableVersion("v1.6.17") {
		t.Fatal("expected v1.6.17 to be treated as stable")
	}
	if isStableVersion("v1.6.17-beta.1") {
		t.Fatal("expected prerelease tag to be excluded from stable update checks")
	}
	if isStableVersion("dev") {
		t.Fatal("expected dev version to be excluded from stable update checks")
	}
}
