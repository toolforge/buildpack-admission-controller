#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

cat > "$(dirname "$0")"/../values/local.yaml <<EOF
image:
  name: buildpack-admission
  tag: latest
  pullPolicy: Never

config:
  debug: true

  harborDomains:
    - host.minikube.internal
    - ${DEV_DOMAIN_IP}

  systemUsers:
    - "system:serviceaccount:tekton-pipelines:tekton-pipelines-controller"
    - "minikube-user"

EOF
