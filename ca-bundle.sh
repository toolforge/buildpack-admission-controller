#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

# For Linux
export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 |tr -d '\n')
sed -i "s/caBundle: --CA_BUNDLE--/caBundle: ${CA_BUNDLE}/g" service.yaml

# For MacOS use the following
# export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 )

# sed -i "" "s/caBundle: --CA_BUNDLE--/caBundle: ${CA_BUNDLE}/g" service.yaml
