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

# Patch argo-server to run without auth & HTTPS
kubectl -n argo patch deploy argo-server \
  --type=json \
  -p='[
    {"op": "replace", "path": "/spec/template/spec/containers/0/args",
     "value": ["server", "--auth-mode=server", "--secure=false"]},
    {"op": "replace", "path": "/spec/template/spec/containers/0/readinessProbe",
     "value": {
       "httpGet": {
         "path": "/",
         "port": 2746,
         "scheme": "HTTP"
       },
       "initialDelaySeconds": 5,
       "periodSeconds": 5
     }}
  ]'
