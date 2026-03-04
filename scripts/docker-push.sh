#!/bin/sh

IMAGE="gitlab.nikozdev.net/root/cryptowalletest"
TAG="${1:-latest}"

docker push "${IMAGE}:${TAG}"