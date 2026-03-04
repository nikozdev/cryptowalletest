#!/bin/sh

IMAGE="gitlab.nikozdev.net/root/cryptowalletest"
TAG="${1:-latest}"

docker run --rm -p 8080:8080 "${IMAGE}:${TAG}"