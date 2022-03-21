package data

import (
	"time"

	"cloud.google.com/go/firestore"
)

type Plugin struct {
	FullName         string
	Type             string
	Name             string
	URL              string
	LatestReleaseRef *firestore.DocumentRef `json:",omitempty"`
	LatestRelease    *PluginRelease         `firestore:",omitempty"`
	Versions         []string               `firestore:",omitempty"`
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
