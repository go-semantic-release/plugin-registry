package plugin

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/Masterminds/semver/v3"
	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"github.com/google/go-github/v50/github"
)

type Plugin struct {
	Type string
	Name string
	Repo string
}

type fsPluginData struct {
	*registry.Plugin
	LatestReleaseRef *firestore.DocumentRef
	// override fields from embedded struct and prevent them from being saved in firestore
	LatestRelease *struct{} `firestore:",omitempty"`
	Versions      *struct{} `firestore:",omitempty"`
}

func (p *Plugin) GetFullName() string {
	return fmt.Sprintf("%s-%s", p.Type, p.Name)
}

func (p *Plugin) getDocRef(db *firestore.Client) *firestore.DocumentRef {
	return db.Collection("plugins").Doc(p.GetFullName())
}

func (p *Plugin) getVersionsColRef(db *firestore.Client) *firestore.CollectionRef {
	return p.getDocRef(db).Collection("versions")
}

func (p *Plugin) getVersionDocRef(db *firestore.Client, version string) *firestore.DocumentRef {
	return p.getVersionsColRef(db).Doc(version)
}

func (p *Plugin) savePluginRelease(ctx context.Context, db *firestore.Client, pr *registry.PluginRelease) error {
	_, err := p.getVersionDocRef(db, pr.Version).Set(ctx, pr)
	return err
}

func (p *Plugin) updateReleaseFromGitHub(ctx context.Context, db *firestore.Client, ghClient *github.Client, version string) error {
	release, err := getGitHubRelease(ctx, ghClient, p.Repo, fmt.Sprintf("v%s", version))
	if err != nil {
		return err
	}
	pr, err := toPluginRelease(ctx, release)
	if err != nil {
		return err
	}
	return p.savePluginRelease(ctx, db, pr)
}

func (p *Plugin) updateAllReleasesFromGitHub(ctx context.Context, db *firestore.Client, ghClient *github.Client) error {
	releases, err := getAllGitHubReleases(ctx, ghClient, p.Repo)
	if err != nil {
		return err
	}
	for _, release := range releases {
		pr, err := toPluginRelease(ctx, release)
		if err != nil {
			return err
		}
		err = p.savePluginRelease(ctx, db, pr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugin) getLatestReleaseFromGitHub(ctx context.Context, ghClient *github.Client) (string, error) {
	owner, repo := getOwnerRepo(p.Repo)
	release, _, err := ghClient.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	lrVersion, err := semver.NewVersion(release.GetTagName())
	if err != nil {
		return "", err
	}
	return lrVersion.String(), nil
}

func (p *Plugin) toPlugin() *fsPluginData {
	return &fsPluginData{
		Plugin: &registry.Plugin{
			FullName: p.GetFullName(),
			Type:     p.Type,
			Name:     p.Name,
			URL:      fmt.Sprintf("https://github.com/%s", p.Repo),
		},
	}
}

func (p *Plugin) Update(ctx context.Context, db *firestore.Client, ghClient *github.Client, version string) error {
	latestRelease, err := p.getLatestReleaseFromGitHub(ctx, ghClient)
	if err != nil {
		return err
	}

	updateMain := true
	if version == "" {
		err = p.updateAllReleasesFromGitHub(ctx, db, ghClient)
	} else {
		err = p.updateReleaseFromGitHub(ctx, db, ghClient, version)
		updateMain = version == latestRelease
	}
	if err != nil {
		return err
	}

	// do not update main entry if latest release has not been added to database
	if !updateMain {
		return nil
	}

	plugin := p.toPlugin()
	plugin.LatestReleaseRef = db.Doc(fmt.Sprintf("plugins/%s/versions/%s", p.GetFullName(), latestRelease))
	_, err = p.getDocRef(db).Set(ctx, plugin)
	return err
}

func (p *Plugin) GetVersions(ctx context.Context, db *firestore.Client) ([]string, error) {
	versionRefs, err := p.getVersionsColRef(db).DocumentRefs(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	versions := make([]string, len(versionRefs))
	for i, ref := range versionRefs {
		versions[i] = ref.ID
	}
	return versions, nil
}

func (p *Plugin) Get(ctx context.Context, db *firestore.Client) (*registry.Plugin, error) {
	pluginRef := p.getDocRef(db)
	res, err := pluginRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	dp := fsPluginData{Plugin: &registry.Plugin{}}
	if dErr := res.DataTo(&dp); dErr != nil {
		return nil, dErr
	}
	res, err = dp.LatestReleaseRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var lr registry.PluginRelease
	if dErr := res.DataTo(&lr); dErr != nil {
		return nil, dErr
	}
	dp.Plugin.LatestRelease = &lr

	versions, err := p.GetVersions(ctx, db)
	if err != nil {
		return nil, err
	}
	dp.Plugin.Versions = versions
	return dp.Plugin, nil
}

func (p *Plugin) GetRelease(ctx context.Context, db *firestore.Client, version string) (*registry.PluginRelease, error) {
	pluginRelease, err := p.getVersionDocRef(db, version).Get(ctx)
	if err != nil {
		return nil, err
	}
	var pr registry.PluginRelease
	if err := pluginRelease.DataTo(&pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

type Plugins []*Plugin

func (l Plugins) Find(name string) *Plugin {
	for _, p := range l {
		if p.GetFullName() == name {
			return p
		}
	}
	return nil
}
