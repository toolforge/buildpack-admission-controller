#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

if [[ $OSTYPE =~ darwin ]]; then
	# For MacOS
	CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64)
else
	# For Linux
	CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 |tr -d '\n')
fi

cat > $(dirname $0)/../values/local.yaml <<EOF
image:
  name: buildpack-admission
  tag: latest
  pullPolicy: Never

webhook:
  caBundle: ${CA_BUNDLE}

config:
  debug: true

  harborDomains:
    - host.minikube.internal
	- ${DEV_DOMAIN_IP}

  systemUsers:
    - "system:serviceaccount:tekton-pipelines:tekton-pipelines-controller"
	- "minikube-user"

EOF
