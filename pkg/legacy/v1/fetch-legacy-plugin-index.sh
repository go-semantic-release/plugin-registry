#!/usr/bin/env bash

set -euo pipefail

echo "fetching plugin index..."
curl -SL https://github.com/go-semantic-release/go-semantic-release.github.io/archive/refs/heads/plugin-index.tar.gz -o plugin-index.tgz

echo "extracting plugin index..."
rm -rf ./plugins plugins.json
tar -xvzf plugin-index.tgz --strip-components=3

echo "cleaning up..."
rm -f plugin-index.tgz
