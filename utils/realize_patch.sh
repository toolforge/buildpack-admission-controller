#!/bin/bash -eu

set -o pipefail
function main() {
    if [[ ${1:-} == '-v' ]]; then
        shift
        set -x
    fi

    for template in $(find . -iname \*.tpl); do
        realize_template "$template"
    done
}



function realize_template() {
    local template_path="${1?No template_path passed}"

    if [[ -z $DEV_DOMAIN_IP ]]; then
        echo "DEV_DOMAIN_IP is undefined"
        exit 1
    fi

    if [[ $OSTYPE =~ darwin ]]; then
        # For MacOS
        CA_BUNDLE=$(
            kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' \
            | base64 \
        )
    else
        # For Linux
        CA_BUNDLE=$(
            kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' \
            | base64 \
            | tr -d '\n' \
        )
    fi

    local GIT_HASH="$(git rev-parse HEAD)"
    sed \
        -e "s/@@CA_BUNDLE@@/${CA_BUNDLE}/g" \
        -e "s/@@BUILD_ID@@/${GIT_HASH}-$(date +%Y%m%d_%H%M%S)/g" \
        -e "s/@@DEV_DOMAIN_IP@@/${DEV_DOMAIN_IP}/g" \
        "${template_path}" \
    > "${template_path%.tpl}"
    echo "Realized template ${template_path} into ${template_path%.tpl}"
}


main "$@"
