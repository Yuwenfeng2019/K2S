#!/bin/bash
set -e

cd $(dirname $0)/..

if [ -z "$K2S_ARM64_HOST" ]; then
    echo K2S_ARM_HOST must be set
    exit 1
fi

if [ -z "$K2S_ARM64_HOST_USER" ]; then
    echo K2S_ARM_HOST_USER must be set
    exit 1
fi

rm -rf dist
mkdir -p build

DAPPER_HOST_ARCH=arm64 DOCKER_HOST="ssh://${K2S_ARM64_HOST_USER}@${K2S_ARM64_HOST}" make release-arm

ls -la dist
echo Done
