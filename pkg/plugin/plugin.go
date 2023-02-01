package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

var CollectionPrefix = "dev"

type fsPluginData struct {
	*registry.Plugin
	LatestReleaseRef *firestore.DocumentRef
	// override fields from embedded struct and prevent them from being saved in firestore
	LatestRelease *struct{} `firestore:",omitempty"`
	Versions      *struct{} `firestore:",omitempty"`
	UpdatedAt     *struct{} `firestore:",omitempty"`
}

type fsPluginReleaseData struct {
	*registry.PluginRelease
	// override fields from embedded struct and prevent them from being saved in firestore
	UpdatedAt *struct{} `firestore:",omitempty"`
}

func (p *Plugin) GetFullName() string {
	return fmt.Sprintf("%s-%s", p.Type, p.Name)
}

func (p *Plugin) getDocRef(db *firestore.Client) *firestore.DocumentRef {
	return db.Collection(CollectionPrefix + "-plugins").Doc(p.GetFullName())
}

func (p *Plugin) getVersionsColRef(db *firestore.Client) *firestore.CollectionRef {
	return p.getDocRef(db).Collection("versions")
}

func (p *Plugin) getVersionDocRef(db *firestore.Client, version string) *firestore.DocumentRef {
	return p.getVersionsColRef(db).Doc(version)
}

func (p *Plugin) savePluginRelease(ctx context.Context, db *firestore.Client, pr *registry.PluginRelease) error {
	_, err := p.getVersionDocRef(db, pr.Version).Set(ctx, &fsPluginReleaseData{PluginRelease: pr})
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
	plugin.LatestReleaseRef = p.getVersionDocRef(db, latestRelease)
	_, err = p.getDocRef(db).Set(ctx, plugin)
	return err
}

func (p *Plugin) getVersions(ctx context.Context, db *firestore.Client) ([]string, error) {
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

func (p *Plugin) getPlugin(ctx context.Context, db *firestore.Client) (*registry.Plugin, error) {
	res, err := p.getDocRef(db).Get(ctx)
	if err != nil {
		return nil, err
	}
	pluginData := fsPluginData{Plugin: &registry.Plugin{}}
	if dErr := res.DataTo(&pluginData); dErr != nil {
		return nil, dErr
	}
	pluginData.Plugin.UpdatedAt = res.UpdateTime

	// resolve latest release
	res, err = pluginData.LatestReleaseRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	var latestPluginRelease registry.PluginRelease
	if dErr := res.DataTo(&latestPluginRelease); dErr != nil {
		return nil, dErr
	}
	latestPluginRelease.UpdatedAt = res.UpdateTime
	pluginData.Plugin.LatestRelease = &latestPluginRelease
	return pluginData.Plugin, nil
}

func (p *Plugin) Get(ctx context.Context, db *firestore.Client) (*registry.Plugin, error) {
	latestRelease, err := p.getPlugin(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	versions, err := p.getVersions(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}
	latestRelease.Versions = versions
	return latestRelease, nil
}

func findMatchingVersion(stringVersions []string, constraint *semver.Constraints) (string, error) {
	versions := make(semver.Collection, len(stringVersions))
	for i, v := range stringVersions {
		version, err := semver.NewVersion(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse version %s: %w", v, err)
		}
		versions[i] = version
	}
	sort.Sort(sort.Reverse(versions))
	for _, v := range versions {
		if constraint.Check(v) {
			return v.String(), nil
		}
	}
	return "", fmt.Errorf("no matching version found for constraint %s", constraint.String())
}

func (p *Plugin) GetReleaseWithVersionConstraint(ctx context.Context, db *firestore.Client, versionConstraint string) (*registry.PluginRelease, error) {
	if versionConstraint == "latest" {
		latestPlugin, err := p.getPlugin(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest release: %w", err)
		}
		return latestPlugin.LatestRelease, nil
	}
	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version constraint: %w", err)
	}

	versions, err := p.getVersions(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	matchingVersion, err := findMatchingVersion(versions, constraint)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching version: %w", err)
	}
	return p.GetRelease(ctx, db, matchingVersion)
}

func (p *Plugin) GetRelease(ctx context.Context, db *firestore.Client, version string) (*registry.PluginRelease, error) {
	pluginRelease, err := p.getVersionDocRef(db, version).Get(ctx)
	if err != nil {
		return nil, err
	}
	var pr registry.PluginRelease
	if dErr := pluginRelease.DataTo(&pr); dErr != nil {
		return nil, dErr
	}
	pr.UpdatedAt = pluginRelease.UpdateTime
	return &pr, nil
}

type Plugins []*Plugin

func (l Plugins) Find(name string) *Plugin {
	for _, p := range l {
		if p.GetFullName() == strings.ToLower(name) {
			return p
		}
	}
	return nil
}
