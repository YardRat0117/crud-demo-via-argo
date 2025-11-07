#!/bin/zsh

# Create k8s cluster locally with kind
kind create cluster --name argo-demo

# Copy kubeconfig here
cp ~/.kube/config ./kubeconfig

# Create namespace for argo
kubectl create ns argo

# Create service account and grant permission
kubectl create sa argo -n argo
kubectl create clusterrolebinding argo-workflow-binding \
  --clusterrole=admin \
  --serviceaccount=argo:argo

# Install Argo Workflows
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml

# build docker images
docker build -t my-create .
docker build -t my-read .
docker build -t my-update .
docker build -t my-delete .

# load images
kind load docker-image my-create:latest --name argo-demo
kind load docker-image my-read:latest --name argo-demo
kind load docker-image my-update:latest --name argo-demo
kind load docker-image my-delete:latest --name argo-demo
