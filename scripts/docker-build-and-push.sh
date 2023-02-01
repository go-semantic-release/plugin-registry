#!/usr/bin/env bash

set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "version not set"
  exit 1
fi

gcloud auth configure-docker gcr.io -q

version=$1
image_name="gcr.io/go-semantic-release/plugin-registry"
image_name_version="$image_name:$version"

echo "building image..."
docker build --build-arg "VERSION=$version" -t "$image_name_version" .

echo "pushing image..."
docker tag "$image_name_version" "$image_name"
docker push "$image_name_version"
docker push $image_name
