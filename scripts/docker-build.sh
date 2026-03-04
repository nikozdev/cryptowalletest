#!/bin/sh

IMAGE="gitlab.nikozdev.net/root/cryptowalletest"
TAG="${1:-latest}"

docker build -t "${IMAGE}:${TAG}" .