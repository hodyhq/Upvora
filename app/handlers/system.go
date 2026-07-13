package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/web"
)

const releasesURL = "https://api.github.com/repos/hodyhq/Upvora/releases/latest"

// validTag keeps anything GitHub returns from ever reaching the UI unless it
// looks like one of our version tags.
var validTag = regexp.MustCompile(`^v[0-9]+(\.[0-9]+)*$`)

type latestRelease struct {
	Tag       string    `json:"tag"`
	URL       string    `json:"url"`
	CheckedAt time.Time `json:"checkedAt"`
}

var releaseCache struct {
	sync.Mutex
	latest *latestRelease
}

// fetchLatestRelease queries GitHub, caching the result for a day unless forced.
func fetchLatestRelease(c *web.Context, force bool) *latestRelease {
	releaseCache.Lock()
	defer releaseCache.Unlock()

	if !force && releaseCache.latest != nil && time.Since(releaseCache.latest.CheckedAt) < 24*time.Hour {
		return releaseCache.latest
	}

	req := &cmd.HTTPRequest{
		URL:    releasesURL,
		Method: "GET",
		Headers: map[string]string{
			"Accept":     "application/vnd.github+json",
			"User-Agent": "Upvora/" + env.Version(),
		},
	}
	if err := bus.Dispatch(c, req); err != nil || req.ResponseStatusCode != http.StatusOK {
		return releaseCache.latest // keep whatever we had
	}

	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(req.ResponseBody, &body); err != nil {
		return releaseCache.latest
	}
	if !validTag.MatchString(body.TagName) || !strings.HasPrefix(body.HTMLURL, "https://github.com/") {
		return releaseCache.latest
	}

	releaseCache.latest = &latestRelease{Tag: body.TagName, URL: body.HTMLURL, CheckedAt: time.Now()}
	return releaseCache.latest
}

// compareVersions reports where current stands against latest: "update-available",
// "up-to-date", "ahead" or "unknown". Hash suffixes (v1.2.3-abc123) are ignored.
func compareVersions(current, latest string) string {
	cur := versionSegments(current)
	lat := versionSegments(latest)
	if cur == nil || lat == nil {
		return "unknown"
	}
	for i := 0; i < len(cur) || i < len(lat); i++ {
		var c, l int
		if i < len(cur) {
			c = cur[i]
		}
		if i < len(lat) {
			l = lat[i]
		}
		if c < l {
			return "update-available"
		}
		if c > l {
			return "ahead"
		}
	}
	return "up-to-date"
}

func versionSegments(v string) []int {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		v = v[:idx]
	}
	if v == "" {
		return nil
	}
	segs := []int{}
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		segs = append(segs, n)
	}
	return segs
}

// updaterDir is where the updater sidecar shares its trigger/status/log files.
// Empty (env var unset) means no sidecar: the one-click update is unavailable.
func updaterDir() string {
	return os.Getenv("UPDATER_SHARED_DIR")
}

func readUpdaterFile(name string, max int64) string {
	dir := updaterDir()
	if dir == "" {
		return ""
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, max)
	n, _ := f.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}

// SystemStatus returns version/update information for the admin System box.
func SystemStatus() web.HandlerFunc {
	return func(c *web.Context) error {
		force := c.QueryParam("force") == "true"
		latest := fetchLatestRelease(c, force)

		state := "unknown"
		if latest != nil {
			state = compareVersions(env.Version(), latest.Tag)
		}

		updateStatus := "idle"
		if updaterDir() != "" {
			if _, err := os.Stat(filepath.Join(updaterDir(), "update-requested")); err == nil {
				updateStatus = "requested"
			} else if s := readUpdaterFile("status", 32); s != "" {
				updateStatus = s // running | done | error
			}
		}

		return c.Ok(web.Map{
			"currentVersion":    env.Version(),
			"latest":            latest,
			"state":             state,
			"updaterConfigured": updaterDir() != "",
			"updateStatus":      updateStatus,
			"updateLog":         readUpdaterFile("log", 4096),
		})
	}
}

// SystemTriggerUpdate rings the updater sidecar's doorbell. The trigger file is
// deliberately empty — the sidecar takes no instructions from it, so there is
// nothing an attacker could inject even with an admin session.
func SystemTriggerUpdate() web.HandlerFunc {
	return func(c *web.Context) error {
		dir := updaterDir()
		if dir == "" {
			return c.BadRequest(web.Map{"message": "The updater sidecar is not configured on this host (UPDATER_SHARED_DIR is unset)."})
		}
		if err := os.WriteFile(filepath.Join(dir, "update-requested"), nil, 0644); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}
