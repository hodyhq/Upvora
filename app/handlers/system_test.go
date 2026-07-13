package handlers

import (
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
)

func TestCompareVersions(t *testing.T) {
	RegisterT(t)

	testCases := []struct {
		current  string
		latest   string
		expected string
	}{
		{"v0.36.1.2.13", "v0.36.1.2.13", "up-to-date"},
		{"v0.36.1.2.13-abc1234", "v0.36.1.2.13", "up-to-date"},
		{"v0.36.1.2.12", "v0.36.1.2.13", "update-available"},
		{"v0.36.1.2.68-03b7d81b", "v0.36.1.2.13", "ahead"},
		{"v0.36.1.2", "v0.36.1.2.1", "update-available"},
		{"v0.36.1.3", "v0.36.1.2.99", "ahead"},
		{"dev", "v0.36.1.2.13", "unknown"},
		{"v0.36.1.2.13", "", "unknown"},
	}

	for _, tc := range testCases {
		Expect(compareVersions(tc.current, tc.latest)).Equals(tc.expected)
	}
}
