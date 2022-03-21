package plugin

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/Masterminds/semver/v3"
	"github.com/go-semantic-release/plugin-registry/pkg/data"
	"github.com/google/go-github/v43/github"
)

type Plugin struct {
	Type string
	Name string
	Repo string
}

func (p *Plugin) GetName() string {
	return fmt.Sprintf("%s-%s", p.Type, p.Name)
}

func (p *Plugin) savePluginRelease(ctx context.Context, db *firestore.Client, pr *data.PluginRelease) error {
	_, err := db.Collection("plugins").Doc(p.GetName()).Collection("versions").Doc(pr.Version).Set(ctx, pr)
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

func (p *Plugin) toPlugin() *data.Plugin {
	return &data.Plugin{
		FullName: p.GetName(),
		Type:     p.Type,
		Name:     p.Name,
		URL:      fmt.Sprintf("https://github.com/%s", p.Repo),
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
	plugin.LatestReleaseRef = db.Doc(fmt.Sprintf("plugins/%s/versions/%s", p.GetName(), latestRelease))
	_, err = db.Collection("plugins").Doc(plugin.FullName).Set(ctx, plugin)
	return err
}

func (p *Plugin) Get(ctx context.Context, db *firestore.Client) (*data.Plugin, error) {
	pluginRef := db.Collection("plugins").Doc(p.GetName())
	res, err := pluginRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	var dp data.Plugin
	if err := res.DataTo(&dp); err != nil {
		return nil, err
	}
	res, err = dp.LatestReleaseRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	var lr data.PluginRelease
	if err := res.DataTo(&lr); err != nil {
		return nil, err
	}
	dp.LatestReleaseRef = nil
	dp.LatestRelease = &lr

	versionRefs, err := pluginRef.Collection("versions").DocumentRefs(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	versions := make([]string, len(versionRefs))
	for i, ref := range versionRefs {
		versions[i] = ref.ID
	}
	dp.Versions = versions
	return &dp, nil
}

func (p *Plugin) GetRelease(ctx context.Context, db *firestore.Client, version string) (*data.PluginRelease, error) {
	pluginRelease, err := db.Collection("plugins").Doc(p.GetName()).Collection("versions").Doc(version).Get(ctx)
	if err != nil {
		return nil, err
	}
	var pr data.PluginRelease
	if err := pluginRelease.DataTo(&pr); err != nil {
		return nil, err
	}
	return &pr, nil
}
