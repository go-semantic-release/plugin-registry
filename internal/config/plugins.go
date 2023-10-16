package config

import "github.com/go-semantic-release/plugin-registry/internal/plugin"

var Plugins = plugin.Plugins{
	{
		Type:        "provider",
		Name:        "github",
		Repo:        "go-semantic-release/provider-github",
		Description: "A provider plugin that uses the GitHub API to publish releases.",
	},
	{
		Type:        "provider",
		Name:        "gitlab",
		Repo:        "go-semantic-release/provider-gitlab",
		Description: "A provider plugin that uses the GitLab API to publish releases.",
	},
	{
		Type:        "changelog-generator",
		Name:        "default",
		Repo:        "go-semantic-release/changelog-generator-default",
		Description: "A changelog generator plugin that generates a changelog based on the commit messages since the last release.",
	},
	{
		Type:        "commit-analyzer",
		Name:        "cz",
		Aliases:     []string{"default"},
		Repo:        "go-semantic-release/commit-analyzer-cz",
		Description: "A commit analyzer plugin that uses the Conventional Commits specification to determine the type of release to create based on the commit messages since the last release.",
	},
	{
		Type:        "condition",
		Name:        "default",
		Repo:        "go-semantic-release/condition-default",
		Description: "The fallback CI condition plugin that detects the current git branch and does not prevent any release.",
	},
	{
		Type:        "condition",
		Name:        "github",
		Repo:        "go-semantic-release/condition-github",
		Description: "A CI condition plugin for GitHub Actions that checks if the current branch should trigger a new release.",
	},
	{
		Type:        "condition",
		Name:        "gitlab",
		Repo:        "go-semantic-release/condition-gitlab",
		Description: "A CI condition plugin for GitLab CI that checks if the current branch should trigger a new release.",
	},
	{
		Type:        "files-updater",
		Name:        "npm",
		Repo:        "go-semantic-release/files-updater-npm",
		Description: "A files updater plugin that updates the version in the package.json file.",
	},
	{
		Type:        "provider",
		Name:        "git",
		Repo:        "go-semantic-release/provider-git",
		Description: "A provider plugin that uses git tags directly to publish releases. This works with any git repository.",
	},
	{
		Type:        "condition",
		Name:        "bitbucket",
		Repo:        "go-semantic-release/condition-bitbucket",
		Description: "A CI condition plugin for Bitbucket Pipelines that checks if the current branch should trigger a new release.",
	},
	{
		Type:        "files-updater",
		Name:        "helm",
		Repo:        "go-semantic-release/files-updater-helm",
		Description: "A files updater plugin that updates the version in the Chart.yaml file.",
	},
	{
		Type:        "hooks",
		Name:        "goreleaser",
		Repo:        "go-semantic-release/hooks-goreleaser",
		Description: "A hooks plugin that runs GoReleaser to publish releases. The GoReleaser binary is bundled with the plugin.",
	},
	{
		Type:        "hooks",
		Name:        "npm-binary-releaser",
		Repo:        "go-semantic-release/hooks-npm-binary-releaser",
		Description: "A hooks plugin that runs npm-binary-releaser to publish the released binaries to npm.",
	},
	{
		Type:        "hooks",
		Name:        "plugin-registry-update",
		Repo:        "go-semantic-release/hooks-plugin-registry-update",
		Description: "A hooks plugin that updates the plugin registry after a new release.",
	},
	{
		Type:        "hooks",
		Name:        "exec",
		Repo:        "go-semantic-release/hooks-exec",
		Description: "A hooks plugin that executes commands after a new release.",
	},
}
