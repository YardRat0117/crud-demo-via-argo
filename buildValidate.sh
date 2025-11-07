#!/bin/bash
set -e

IMAGE_NAME="my-validate"
IMAGE_TAG="latest"
DOCKERFILE_NAME="validate.Dockerfile"

docker build -t "$IMAGE_NAME:$IMAGE_TAG" -f $DOCKERFILE_NAME .

echo "✅ Built image: $IMAGE_NAME:$IMAGE_TAG"
