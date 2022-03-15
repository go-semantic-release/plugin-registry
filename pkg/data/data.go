package data

import (
	"context"
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

func SavePlugin(ctx context.Context, db *firestore.Client, p *Plugin) error {
	_, err := db.Collection("plugins").Doc(p.FullName).Set(ctx, p)
	return err
}

func SavePluginRelease(ctx context.Context, db *firestore.Client, name string, pr *PluginRelease) error {
	_, err := db.Collection("plugins").Doc(name).Collection("versions").Doc(pr.Version).Set(ctx, pr)
	return err
}

func GetPlugin(ctx context.Context, db *firestore.Client, name string) (*Plugin, error) {
	pluginRef := db.Collection("plugins").Doc(name)
	res, err := pluginRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	var p Plugin
	if err := res.DataTo(&p); err != nil {
		return nil, err
	}
	res, err = p.LatestReleaseRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	var lr PluginRelease
	if err := res.DataTo(&lr); err != nil {
		return nil, err
	}
	p.LatestReleaseRef = nil
	p.LatestRelease = &lr

	versionRefs, err := pluginRef.Collection("versions").DocumentRefs(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	versions := make([]string, len(versionRefs))
	for i, ref := range versionRefs {
		versions[i] = ref.ID
	}
	p.Versions = versions
	return &p, nil
}

func GetPluginRelease(ctx context.Context, db *firestore.Client, name, version string) (*PluginRelease, error) {
	pluginRelease, err := db.Collection("plugins").Doc(name).Collection("versions").Doc(version).Get(ctx)
	if err != nil {
		return nil, err
	}
	var pr PluginRelease
	if err := pluginRelease.DataTo(&pr); err != nil {
		return nil, err
	}
	return &pr, nil
}
