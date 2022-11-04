#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail


help() {
    cat <<EOH
    Usage: $0 [OPTIONS] <ENVIRONMENT>

    Options:
      -h        Show this help.
      -c        Force refresh the certificates (only for new installations or
                if they are expired).
      -b        Also build the container image (locally only).
      -v        Show verbose output.

EOH
}


check_environment() {
    # verify that the proper environment is passed

    if [[ ! -d "deploy/$environment" || "$environment" =~ ^(base)$ ]]; then
        echo "Unknown environment $environment, use one of:"
        ls deploy/ | egrep -v '^(base)'
        exit 1
    fi
}


build_image() {
    # build container image for dev or prod environments

    local environment="${1?No environment passed}"
    local git_hash="$(git rev-parse --short HEAD)"
    local timestamp="$(date +%Y%m%d%H%M%S)"

    if [[ "$environment" == "devel" ]]
    then
        # creates the image on minikube's docker daemon
        eval $(minikube docker-env)
        docker build -t buildpack-admission:latest .
    else
        # Build the container image on the docker-builder host (currently tools-docker-imagebuilder-01.tools.eqiad1.wikimedia.cloud).
        docker build . -f Dockerfile -t "docker-registry.tools.wmflabs.org/buildpack-admission:${timestamp}_${git_hash}"
        # Push the image to the internal repo
        docker push "docker-registry.tools.wmflabs.org/buildpack-admission:${timestamp}_${git_hash}"
        echo "Successfully built container image for tools/toolsbeta environments with exit code 0. \
        To deploy,log into k8s control node with repository checked out and run './deploy.sh (tools or toolsbeta)'"
    fi
}


regenerate_certs() {
    # creates a new certificate, a CSR to sign it, and a k8s secret with the signed cert and key

    local refresh_certs="${1?No refresh_certs argument passed}"
    if [[ "$refresh_certs" == "yes" ]]; then
        ./utils/regenerate_certs.sh
    fi
}


deploy_generic() {
    # deploy buildpack-admission-controller image to either dev or prod environments

    local environment="${1?No environment passed}"
    local refresh_certs="${2?No refresh_certs argument passed}"

    ./utils/realize_patch.sh
    if [[ "$environment" == "devel" ]]
    then
        # generate the patch to override the ca bundle with the k8s secret we just created
        # Deploy the dev environment
        kubectl apply -k deploy/devel
    else
        if [[ "$refresh_certs" == "yes" ]]
        then
            # do rolling restart to ensure the new certs created by regenerate_certs is being used
            kubectl rollout restart -n buildpack-admission deployment/buildpack-admission
        else
            kubectl --as=admin --as-group=system:masters apply -k "deploy/${environment}"
        fi
    fi
}


main () {
    local do_build="no"
    local refresh_certs="no"

    while getopts "hvcb" option; do
        case "${option}" in
        h)
            help
            exit 0
            ;;
        b) do_build="yes";;
        c) refresh_certs="yes";;
        v) set -x;;
        *)
            echo "Wrong option $option"
            help
            exit 1
            ;;
        esac
    done
    shift $((OPTIND-1))

    # default to prod, avoid deploying dev in prod if there's any issue
    local environment="tools"
    if [[ "${1:-}" == "" ]]; then
        if [[ -f /etc/wmcs-project ]]; then
            environment="$(cat /etc/wmcs-project)"
        fi
    else
        environment="${1:-}"
    fi

    check_environment "$environment"

    if [[ "$environment" == "devel" ]];then
        refresh_certs="yes"
    fi

    if [[ "$do_build" == "yes" ]];then
        build_image "$environment"
        if [[ "$environment" != "devel" ]];then
            exit 0
        fi
    fi

    regenerate_certs "$refresh_certs"
    deploy_generic "$environment" "$refresh_certs"
}

main "$@"
