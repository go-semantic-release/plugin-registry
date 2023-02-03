# :electric_plug: plugin-registry
[![CI](https://github.com/go-semantic-release/plugin-registry/workflows/CI/badge.svg?branch=main)](https://github.com/go-semantic-release/plugin-registry/actions?query=workflow%3ACI+branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-semantic-release/plugin-registry)](https://goreportcard.com/report/github.com/go-semantic-release/plugin-registry)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/go-semantic-release/plugin-registry)](https://pkg.go.dev/github.com/go-semantic-release/plugin-registry)

The plugin registry service for the go-semantic-release CLI tool provides information about available plugins. It leverages the GitHub API to fetch the latest releases and assets, ensuring that users have access to up-to-date information. The registry also includes caching and batch request capabilities, which allow multiple plugin assets to be bundled into a single, compressed archive for efficient and streamlined delivery.

## API

### GET [/api/v2/plugins](https://registry.go-semantic-release.xyz/api/v2/plugins)
Returns a list of all available plugins.

<details>
<summary>Example response body</summary>

```json
[
  "provider-github",
  "provider-gitlab",
  "changelog-generator-default",
  "commit-analyzer-cz",
  "condition-default",
  "condition-github",
  "condition-gitlab",
  "files-updater-npm",
  "provider-git",
  "condition-bitbucket",
  "files-updater-helm",
  "hooks-goreleaser",
  "hooks-npm-binary-releaser",
  "hooks-plugin-registry-update"
]
```
</details>

### GET [/api/v2/plugins/:plugin](https://registry.go-semantic-release.xyz/api/v2/plugins/provider-github)
Returns information about a specific plugin.


<details>
<summary>Example response body</summary>

```json
{
  "FullName": "provider-github",
  "Type": "provider",
  "Name": "github",
  "URL": "https://github.com/go-semantic-release/provider-github",
  "LatestRelease": {
    "Version": "1.14.0",
    "Prerelease": false,
    "CreatedAt": "2023-02-03T15:14:47Z",
    "Assets": {
      "darwin/amd64": {
        "FileName": "provider-github_v1.14.0_darwin_amd64",
        "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_darwin_amd64",
        "OS": "darwin",
        "Arch": "amd64",
        "Checksum": "5f1bdc2eccc99e158c525033a64dd490e6dca8f020bf700a2edf6d3e1cbba3c4"
      },
      "darwin/arm64": {
        "FileName": "provider-github_v1.14.0_darwin_arm64",
        "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_darwin_arm64",
        "OS": "darwin",
        "Arch": "arm64",
        "Checksum": "ce6bd1e591621d005fe0840a92f2e751a83d8b2280573832c3b81eae3f7e751e"
      },
      "linux/amd64": {
        "FileName": "provider-github_v1.14.0_linux_amd64",
        "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_amd64",
        "OS": "linux",
        "Arch": "amd64",
        "Checksum": "2ed8f28aec663ad549875abb6257fe333f99ac23aa337d0d53df84bbc10f2930"
      },
      "linux/arm": {
        "FileName": "provider-github_v1.14.0_linux_arm",
        "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_arm",
        "OS": "linux",
        "Arch": "arm",
        "Checksum": "b3a823b4ebb30136c27b48bcf92002c20abf59c297c278283db00800762b5ab4"
      },
      "linux/arm64": {
        "FileName": "provider-github_v1.14.0_linux_arm64",
        "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_arm64",
        "OS": "linux",
        "Arch": "arm64",
        "Checksum": "1bae4ef206c1a849e33fdef49b4b8b21aa81b05ca77f4043e632c35694373fc6"
      },
      "windows/amd64": {
        "FileName": "provider-github_v1.14.0_windows_amd64.exe",
        "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_windows_amd64.exe",
        "OS": "windows",
        "Arch": "amd64",
        "Checksum": "b4d2e2e8a9b4f6b278920869dc9bb0ce2fb85a9e79da1f210f0f1bf4baac6a56"
      }
    },
    "UpdatedAt": "2023-02-03T15:22:18.198347Z"
  },
  "Versions": [
    "1.0.0",
    "1.1.0",
    "1.1.1",
    "1.10.0",
    "1.11.0",
    "1.12.0",
    "1.13.0",
    "1.14.0",
    "1.2.0",
    "1.3.0",
    "1.4.0",
    "1.4.1",
    "1.5.0",
    "1.5.1",
    "1.5.2",
    "1.6.0",
    "1.6.1",
    "1.7.0",
    "1.8.0",
    "1.9.0"
  ],
  "UpdatedAt": "2023-02-03T15:22:18.228101Z"
}
```
</details>

### GET [/api/v2/plugins/:plugin/versions](https://registry.go-semantic-release.xyz/api/v2/plugins/provider-github/versions)
Returns all plugin releases.


<details>
<summary>Example response body</summary>

```json
[
  "1.0.0",
  "1.1.0",
  "1.1.1",
  "1.10.0",
  "1.11.0",
  "1.12.0",
  "1.13.0",
  "1.14.0",
  "1.2.0",
  "1.3.0",
  "1.4.0",
  "1.4.1",
  "1.5.0",
  "1.5.1",
  "1.5.2",
  "1.6.0",
  "1.6.1",
  "1.7.0",
  "1.8.0",
  "1.9.0"
]
```
</details>

### GET [/api/v2/plugins/:plugin/versions/:version](https://registry.go-semantic-release.xyz/api/v2/plugins/provider-github/versions/1.14.0)
Returns information about a specific plugin release.


<details>
<summary>Example response body</summary>

```json
{
  "Version": "1.14.0",
  "Prerelease": false,
  "CreatedAt": "2023-02-03T15:14:47Z",
  "Assets": {
    "darwin/amd64": {
      "FileName": "provider-github_v1.14.0_darwin_amd64",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_darwin_amd64",
      "OS": "darwin",
      "Arch": "amd64",
      "Checksum": "5f1bdc2eccc99e158c525033a64dd490e6dca8f020bf700a2edf6d3e1cbba3c4"
    },
    "darwin/arm64": {
      "FileName": "provider-github_v1.14.0_darwin_arm64",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_darwin_arm64",
      "OS": "darwin",
      "Arch": "arm64",
      "Checksum": "ce6bd1e591621d005fe0840a92f2e751a83d8b2280573832c3b81eae3f7e751e"
    },
    "linux/amd64": {
      "FileName": "provider-github_v1.14.0_linux_amd64",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_amd64",
      "OS": "linux",
      "Arch": "amd64",
      "Checksum": "2ed8f28aec663ad549875abb6257fe333f99ac23aa337d0d53df84bbc10f2930"
    },
    "linux/arm": {
      "FileName": "provider-github_v1.14.0_linux_arm",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_arm",
      "OS": "linux",
      "Arch": "arm",
      "Checksum": "b3a823b4ebb30136c27b48bcf92002c20abf59c297c278283db00800762b5ab4"
    },
    "linux/arm64": {
      "FileName": "provider-github_v1.14.0_linux_arm64",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_arm64",
      "OS": "linux",
      "Arch": "arm64",
      "Checksum": "1bae4ef206c1a849e33fdef49b4b8b21aa81b05ca77f4043e632c35694373fc6"
    },
    "windows/amd64": {
      "FileName": "provider-github_v1.14.0_windows_amd64.exe",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_windows_amd64.exe",
      "OS": "windows",
      "Arch": "amd64",
      "Checksum": "b4d2e2e8a9b4f6b278920869dc9bb0ce2fb85a9e79da1f210f0f1bf4baac6a56"
    }
  },
  "UpdatedAt": "2023-02-03T15:22:18.198347Z"
}
```
</details>

### POST [/api/v2/plugins/_batch](https://registry.go-semantic-release.xyz/api/v2/plugins/_batch)
Returns information about multiple plugins and a download link to a compressed archive containing all plugins.


<details>
<summary>Example request body</summary>

```json
{
  "OS": "linux",
  "Arch": "amd64",
  "Plugins": [
    {
      "FullName": "provider-github",
      "VersionConstraint": "latest"
    },
    {
      "FullName": "condition-github",
      "VersionConstraint": "^1.0.0"
    }
  ]
}
```
</details>

<details>
<summary>Example response body</summary>

```json
{
  "OS": "linux",
  "Arch": "amd64",
  "Plugins": [
    {
      "FullName": "condition-github",
      "VersionConstraint": "^1.0.0",
      "Version": "1.8.0",
      "FileName": "condition-github_v1.8.0_linux_amd64",
      "URL": "https://github.com/go-semantic-release/condition-github/releases/download/v1.8.0/condition-github_v1.8.0_linux_amd64",
      "Checksum": "6274fd728cb95fdf6863a2ef18d9a37179285e00a24aef6bed96def67eda4fcd"
    },
    {
      "FullName": "provider-github",
      "VersionConstraint": "latest",
      "Version": "1.14.0",
      "FileName": "provider-github_v1.14.0_linux_amd64",
      "URL": "https://github.com/go-semantic-release/provider-github/releases/download/v1.14.0/provider-github_v1.14.0_linux_amd64",
      "Checksum": "2ed8f28aec663ad549875abb6257fe333f99ac23aa337d0d53df84bbc10f2930"
    }
  ],
  "DownloadHash": "5e1460e12232dbb785ca6774d0eb7fa6cf14a2212b72607e7c1070ffa8395a2a",
  "DownloadURL": "https://plugin-cache.go-semantic-release.xyz/archives/plugins-5e1460e12232dbb785ca6774d0eb7fa6cf14a2212b72607e7c1070ffa8395a2a.tar.gz",
  "DownloadChecksum": "900182d40199ca85c26ee707fbe5f8a5f8f219b7a1835bfa5e1623884b96af49"
}
```
</details>

## Licence

The [MIT License (MIT)](http://opensource.org/licenses/MIT)

Copyright © 2023 [Christoph Witzko](https://twitter.com/christophwitzko)
