package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	githubOwner             = "qianlima-666"
	githubRepo              = "nginxpulse"
	githubAPITimeout        = 2500 * time.Millisecond
	githubUpdateCacheTTL    = 6 * time.Hour
	githubNoUpdateCacheTTL  = 15 * time.Minute
	githubUpdateFailureTTL  = 45 * time.Minute
	githubLatestReleaseURL  = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases/latest"
	githubTagsAPIURL        = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/tags?per_page=100"
	githubReleasePagePrefix = "https://github.com/" + githubOwner + "/" + githubRepo + "/releases/tag/"
	githubTagsPageURL       = "https://github.com/" + githubOwner + "/" + githubRepo + "/tags"
)

type UpdateInfo struct {
	CurrentVersion   string
	LatestVersion    string
	LatestReleaseURL string
	UpdateAvailable  bool
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type githubTag struct {
	Name string `json:"name"`
}

type cachedLatestVersion struct {
	mu            sync.Mutex
	latestVersion string
	latestURL     string
	checkedAt     time.Time
}

type parsedVersion struct {
	major      int
	minor      int
	patch      int
	prerelease string
	valid      bool
}

var latestVersionCache cachedLatestVersion

func GetUpdateInfo() UpdateInfo {
	return buildUpdateInfo(false)
}

func RefreshUpdateInfo() UpdateInfo {
	return buildUpdateInfo(true)
}

func buildUpdateInfo(force bool) UpdateInfo {
	info := UpdateInfo{
		CurrentVersion: Version,
	}

	current := parseVersion(Version)
	if !current.valid {
		return info
	}

	latestVersion, latestURL := getLatestVersion(force)
	if latestVersion == "" {
		return info
	}

	info.LatestVersion = latestVersion
	info.LatestReleaseURL = latestURL
	info.UpdateAvailable = compareVersions(latestVersion, Version) > 0
	return info
}

func getLatestVersion(force bool) (string, string) {
	latestVersionCache.mu.Lock()
	ttl := getLatestVersionCacheTTL(latestVersionCache.latestVersion)
	if !force && !latestVersionCache.checkedAt.IsZero() && time.Since(latestVersionCache.checkedAt) < ttl {
		version := latestVersionCache.latestVersion
		url := latestVersionCache.latestURL
		latestVersionCache.mu.Unlock()
		return version, url
	}
	latestVersionCache.mu.Unlock()

	latestVersion, latestURL, err := fetchLatestVersion()

	latestVersionCache.mu.Lock()
	defer latestVersionCache.mu.Unlock()

	latestVersionCache.checkedAt = time.Now()
	if err == nil {
		latestVersionCache.latestVersion = latestVersion
		latestVersionCache.latestURL = latestURL
	}

	return latestVersionCache.latestVersion, latestVersionCache.latestURL
}

func getLatestVersionCacheTTL(latestVersion string) time.Duration {
	trimmed := strings.TrimSpace(latestVersion)
	switch {
	case trimmed == "":
		return githubUpdateFailureTTL
	case compareVersions(trimmed, Version) > 0:
		return githubUpdateCacheTTL
	default:
		return githubNoUpdateCacheTTL
	}
}

func fetchLatestVersion() (string, string, error) {
	client := &http.Client{Timeout: githubAPITimeout}

	if latestVersion, latestURL, err := fetchLatestRelease(client); err == nil {
		return latestVersion, latestURL, nil
	}

	return fetchLatestStableTag(client)
}

func fetchLatestRelease(client *http.Client) (string, string, error) {
	var release githubRelease
	if err := fetchGitHubJSON(client, githubLatestReleaseURL, &release); err != nil {
		return "", "", err
	}
	if release.Draft || release.Prerelease {
		return "", "", errors.New("latest release is not a stable release")
	}
	if !parseVersion(release.TagName).valid {
		return "", "", errors.New("latest release tag is not a valid semver")
	}
	latestURL := strings.TrimSpace(release.HTMLURL)
	if latestURL == "" {
		latestURL = githubReleasePagePrefix + strings.TrimSpace(release.TagName)
	}
	return strings.TrimSpace(release.TagName), latestURL, nil
}

func fetchLatestStableTag(client *http.Client) (string, string, error) {
	var tags []githubTag
	if err := fetchGitHubJSON(client, githubTagsAPIURL, &tags); err != nil {
		return "", "", err
	}

	best := ""
	for _, tag := range tags {
		if !isStableVersion(tag.Name) {
			continue
		}
		if best == "" || compareVersions(tag.Name, best) > 0 {
			best = strings.TrimSpace(tag.Name)
		}
	}
	if best == "" {
		return "", "", errors.New("no stable version tags found")
	}
	return best, githubTagsPageURL, nil
}

func fetchGitHubJSON(client *http.Client, url string, target any) error {
	ctx, cancel := context.WithTimeout(context.Background(), githubAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "NginxPulse-Version-Checker")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected GitHub API status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

func isStableVersion(raw string) bool {
	version := parseVersion(raw)
	return version.valid && version.prerelease == ""
}

func compareVersions(leftRaw, rightRaw string) int {
	left := parseVersion(leftRaw)
	right := parseVersion(rightRaw)
	if !left.valid || !right.valid {
		return 0
	}

	if left.major != right.major {
		return compareInts(left.major, right.major)
	}
	if left.minor != right.minor {
		return compareInts(left.minor, right.minor)
	}
	if left.patch != right.patch {
		return compareInts(left.patch, right.patch)
	}

	switch {
	case left.prerelease == "" && right.prerelease != "":
		return 1
	case left.prerelease != "" && right.prerelease == "":
		return -1
	default:
		return comparePrerelease(left.prerelease, right.prerelease)
	}
}

func compareInts(left, right int) int {
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func comparePrerelease(left, right string) int {
	if left == right {
		return 0
	}
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	limit := len(leftParts)
	if len(rightParts) > limit {
		limit = len(rightParts)
	}

	for i := 0; i < limit; i++ {
		if i >= len(leftParts) {
			return -1
		}
		if i >= len(rightParts) {
			return 1
		}
		leftPart := leftParts[i]
		rightPart := rightParts[i]
		if leftPart == rightPart {
			continue
		}

		leftNum, leftErr := strconv.Atoi(leftPart)
		rightNum, rightErr := strconv.Atoi(rightPart)
		switch {
		case leftErr == nil && rightErr == nil:
			return compareInts(leftNum, rightNum)
		case leftErr == nil:
			return -1
		case rightErr == nil:
			return 1
		case leftPart > rightPart:
			return 1
		default:
			return -1
		}
	}

	return 0
}

func parseVersion(raw string) parsedVersion {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return parsedVersion{}
	}

	mainPart := value
	prerelease := ""
	if hyphen := strings.IndexByte(value, '-'); hyphen >= 0 {
		mainPart = value[:hyphen]
		prerelease = value[hyphen+1:]
		if prerelease == "" {
			return parsedVersion{}
		}
	}

	if plus := strings.IndexByte(mainPart, '+'); plus >= 0 {
		mainPart = mainPart[:plus]
	}

	segments := strings.Split(mainPart, ".")
	if len(segments) != 3 {
		return parsedVersion{}
	}

	major, err := strconv.Atoi(segments[0])
	if err != nil {
		return parsedVersion{}
	}
	minor, err := strconv.Atoi(segments[1])
	if err != nil {
		return parsedVersion{}
	}
	patch, err := strconv.Atoi(segments[2])
	if err != nil {
		return parsedVersion{}
	}

	return parsedVersion{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: prerelease,
		valid:      true,
	}
}
