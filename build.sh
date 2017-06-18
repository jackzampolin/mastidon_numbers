#!/bin/bash

GOARCH=amd64 GOOS=linux go build -o mastidon main.go
docker build -t gcr.io/jackzampolin-web/mastidon:latest .
gcloud docker -- push gcr.io/jackzampolin-web/mastidon:latest
rm mastidon
kubectl apply -f deployment.yaml