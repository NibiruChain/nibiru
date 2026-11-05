package main

import "testing"

func TestUpgradeHandlerForVersion(t *testing.T) {
	for _, tc := range []struct {
		name     string
		version  string
		handlers []string
		want     string
	}{
		{"minor release", "v2.21.0", []string{"v2.21.0"}, "v2.21.0"},
		{"major release", "v3.0.0", []string{"v3.0.0"}, "v3.0.0"},
		{"compatible patch", "v2.21.2", []string{"v2.21.0"}, "v2.21.0"},
		{"patch prerelease", "v2.21.2-rc.1", []string{"v2.21.0"}, "v2.21.0"},
		{"minor prerelease", "v2.21.0-rc.1", []string{"v2.21.0"}, "v2.21.0"},
		{"build metadata", "v2.21.2+build.01", []string{"v2.21.0"}, "v2.21.0"},
		{"prerelease and metadata", "v2.21.2-rc.1+build.01", []string{"v2.21.0"}, "v2.21.0"},
		{"alphanumeric prerelease", "v2.21.2-01a", []string{"v2.21.0"}, "v2.21.0"},
		{"legacy patch", "v2.18.1", []string{"v2.18.1"}, "v2.18.1"},
		{"exact handler wins", "v2.18.1", []string{"v2.18.0", "v2.18.1"}, "v2.18.1"},
		{"legacy patch suffix", "v2.18.1-rc.1+build.2", []string{"v2.18.1"}, "v2.18.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := upgradeHandlerForVersion(tc.version, tc.handlers)
			if err != nil || got != tc.want {
				t.Fatalf("got handler %q, error %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestUpgradeHandlerForVersionMissingHandler(t *testing.T) {
	for _, version := range []string{"v2.22.0", "v3.0.0", "v2.22.1", "v3.0.1", "v2.22.0-rc.1"} {
		t.Run(version, func(t *testing.T) {
			if _, err := upgradeHandlerForVersion(version, []string{"v2.21.0", "v2.22.2"}); err == nil {
				t.Fatal("accepted release without its exact or base handler")
			}
		})
	}
	if _, err := upgradeHandlerForVersion("v2.21.2", nil); err == nil {
		t.Fatal("accepted patch without any handlers")
	}
}

func TestUpgradeHandlerForVersionInvalid(t *testing.T) {
	for _, version := range []string{
		"", "2.21.2", "v2.21", "v2.21.2.1", "v02.21.2", "v2.021.2", "v2.21.02",
		"v2.21.2garbage", "v2.21.2-", "v2.21.2+", "v2.21.2-rc..1", "v2.21.2+build..1",
		"v2.21.2-01", "v2.21.2-rc.01", "v2.21.2-rc_1", " v2.21.2", "v2.21.2\n",
	} {
		t.Run(version, func(t *testing.T) {
			if _, err := upgradeHandlerForVersion(version, []string{"v2.21.0", "v2.21.2"}); err == nil {
				t.Fatal("accepted malformed version")
			}
		})
	}
}
