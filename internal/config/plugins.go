package config

import "github.com/go-semantic-release/plugin-registry/internal/plugin"

var Plugins = plugin.Plugins{
	{
		Type: "provider",
		Name: "github",
		Repo: "go-semantic-release/provider-github",
	},
	{
		Type: "provider",
		Name: "gitlab",
		Repo: "go-semantic-release/provider-gitlab",
	},
	{
		Type: "changelog-generator",
		Name: "default",
		Repo: "go-semantic-release/changelog-generator-default",
	},
	{
		Type: "commit-analyzer",
		Name: "default",
		Repo: "go-semantic-release/commit-analyzer-cz",
	},
	{
		Type: "condition",
		Name: "default",
		Repo: "go-semantic-release/condition-default",
	},
	{
		Type: "condition",
		Name: "github",
		Repo: "go-semantic-release/condition-github",
	},
	{
		Type: "condition",
		Name: "gitlab",
		Repo: "go-semantic-release/condition-gitlab",
	},
	{
		Type: "files-updater",
		Name: "npm",
		Repo: "go-semantic-release/files-updater-npm",
	},
	{
		Type: "provider",
		Name: "git",
		Repo: "go-semantic-release/provider-git",
	},
	{
		Type: "condition",
		Name: "bitbucket",
		Repo: "go-semantic-release/condition-bitbucket",
	},
	{
		Type: "files-updater",
		Name: "helm",
		Repo: "go-semantic-release/files-updater-helm",
	},
	{
		Type: "hooks",
		Name: "goreleaser",
		Repo: "go-semantic-release/hooks-goreleaser",
	},
	{
		Type: "hooks",
		Name: "npm-binary-releaser",
		Repo: "go-semantic-release/hooks-npm-binary-releaser",
	},
	{
		Type: "hooks",
		Name: "plugin-registry-update",
		Repo: "go-semantic-release/hooks-plugin-registry-update",
	},
}
