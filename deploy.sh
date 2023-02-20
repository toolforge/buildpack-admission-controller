#!/bin/bash
# standard deploy.sh for applications using Helmfile
set -eu

# explicitly find and specify path to helmfile to allow invoking
# this script without having to cd to the deployment directory
BASE_DIR=$(dirname "$(realpath -s "$0")")
cd "$BASE_DIR"

DEPLOY_ENVIRONMENT=${1:-}
PROJECT=$(cat /etc/wmcs-project 2>/dev/null || echo "local")

if [ -z "$DEPLOY_ENVIRONMENT" ]; then
	DEPLOY_ENVIRONMENT="$PROJECT"
fi

kubectl create namespace buildpack-admission --dry-run=client -o yaml | kubectl apply -f -

if [ "$DEPLOY_ENVIRONMENT" = "local" ]; then
	DEV_DOMAIN_IP="${DEV_DOMAIN_IP}" ./deployment/utils/local-values.sh
fi

./deployment/utils/regenerate-certs.sh

# use -i (interactive) to ask for confirmation for changing
# live cluster state if stdin is a tty
if [ -t 0 ]; then
	INTERACTIVE_PARAM="-i"
else
	INTERACTIVE_PARAM=""
fi

# helmfile apply will show a diff before doing changes
helmfile -e "$DEPLOY_ENVIRONMENT" --file "./deployment/helmfile.yaml" $INTERACTIVE_PARAM apply
