// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package version

import (
	"fmt"
	"runtime/debug"
	"time"
)

// VersionInfo contains version information about the binary
type VersionInfo struct {
	Commit    string    `json:"commit"`
	BuildTime time.Time `json:"build_time"`
	Modified  bool      `json:"modified"`
	GoVersion string    `json:"go_version"`
}

// GetInfo retrieves version information from the binary
func GetInfo() VersionInfo {
	var buildTime time.Time
	var modified bool
	commit := "unknown"
	goVersion := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		goVersion = info.GoVersion

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if setting.Value != "" {
					commit = setting.Value
				}
			case "vcs.time":
				if setting.Value != "" {
					// vcs.time is always in RFC3339 format
					t, err := time.Parse(time.RFC3339, setting.Value)
					if err != nil {
						continue
					}
					buildTime = t
				}
			case "vcs.modified":
				if setting.Value == "true" {
					modified = true
				}
			}
		}
	}

	return VersionInfo{
		Commit:    commit,
		BuildTime: buildTime,
		Modified:  modified,
		GoVersion: goVersion,
	}
}

// String returns a formatted string with version information
func (v VersionInfo) String() string {
	buildTimeStr := "unknown"
	if !v.BuildTime.IsZero() {
		buildTimeStr = v.BuildTime.Format("2006-01-02 15:04:05")
	}

	modifiedStr := ""
	if v.Modified {
		modifiedStr = " (modified)"
	}

	return fmt.Sprintf("Commit: %s%s\nBuild Time: %s\nGo Version: %s", v.Commit, modifiedStr, buildTimeStr, v.GoVersion)
}
