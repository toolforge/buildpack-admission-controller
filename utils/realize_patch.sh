#!/bin/bash -eu

set -o pipefail


function main() {
    if [[ ${1:-} == '-v' ]]; then
        shift
        set -x
    fi

    local template_path="${1?No template_path passed}"

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

    sed "s/@@CA_BUNDLE@@/${CA_BUNDLE}/g" "${template_path}" > "${template_path%.tpl}"
    echo "Realized template ${template_path} into ${template_path%.tpl}."
}


main "$@"
