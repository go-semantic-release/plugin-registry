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

func (p *Plugin) updateReleaseFromGitHub(ctx context.Context, db *firestore.Client, ghClient *github.Client, version string) error {
	release, err := getGitHubRelease(ctx, ghClient, p.Repo, fmt.Sprintf("v%s", version))
	if err != nil {
		return err
	}
	pr, err := toPluginRelease(ctx, release)
	if err != nil {
		return err
	}
	return data.SavePluginRelease(ctx, db, p.GetName(), pr)
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
		err = data.SavePluginRelease(ctx, db, p.GetName(), pr)
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
	return data.SavePlugin(ctx, db, plugin)
}
