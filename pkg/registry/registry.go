package registry

import (
	"time"
)

type Plugin struct {
	FullName      string
	Type          string
	Name          string
	URL           string
	LatestRelease *PluginRelease
	Versions      []string
}

type PluginRelease struct {
	Version    string
	Prerelease bool
	CreatedAt  time.Time
	Assets     map[string]*PluginAsset
}

type PluginAsset struct {
	FileName string
	URL      string
	OS       string
	Arch     string
	Checksum string
}

type BatchPluginConstraint struct {
	FullName          string
	VersionConstraint string
}

type BatchRequest struct {
	Plugins []BatchPluginConstraint
}

type BatchResponse struct {
	Plugins map[string]*Plugin
}
