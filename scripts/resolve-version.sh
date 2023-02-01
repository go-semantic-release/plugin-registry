#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${1:-}" ]]; then
  echo "version not set"
  exit 1
fi

version=$1

if [[ "$version" == "latest" ]]; then
  echo "resolving latest release..."
  latest_version=$(curl -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/go-semantic-release/plugin-registry/releases/latest | jq -r '.name')
  version="${latest_version:1}"
  echo "found release: $version"
fi


if [[ -n "${GITHUB_ENV:-}" ]]; then
  echo "writing version to $GITHUB_ENV"
  echo "VERSION=${version}" >> "$GITHUB_ENV"
fi
