#!/bin/sh
set -e

export dockerfile="Dockerfile"
export arch=$(uname -m)
export TAG="latest"

if [ "$arch" = "armv7l" ]; then
    dockerfile="Dockerfile.armhf"
    TAG="latest-armhf-dev"
fi

if [ "$1" ]; then
    TAG=$1
    if [ "$arch" = "armv7l" ]; then
        TAG="$1-armhf"
    fi
fi

if [ -z "$NAMESPACE" ]; then
    NAMESPACE="openfaas"
fi

docker build -t $NAMESPACE/vcenter-connector:$TAG . -f $dockerfile --no-cache