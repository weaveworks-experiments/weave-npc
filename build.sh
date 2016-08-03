#!/bin/bash

set -e

kubectl delete -f k8s/daemonset.yaml || true
CGO_ENABLED=0 go build
sudo docker build -t harrisonadamw/weave-npc .
sudo docker push harrisonadamw/weave-npc
