#!/bin/bash -eu

set -o pipefail

function help() {
    cat <<EOH
    Usage: $0 [-v]

    This script prepares k8s for the deployment of the buildpack-admission
    controller, creating the namespace and recreating the certs needed for
    it.

    Options:
        -v Enables verbose output.
EOH
}


function generate_cert() {
    local dst_dir="${1?No dst_dir passed}"
    echo "Creating certs in dir ${dst_dir} "
    openssl genrsa -out "${dst_dir}/server-key.pem" 2048
}


function create_certificate_signing_request() {
    local dst_dir="${1?No dst_dir passed}"
    local app_name="${2?No app_name passed}"
    echo "Creating certificate signing request (csr) at $dst_dir}/server.csr"

    cat <<EOF >> "${dst_dir}/csr.conf"
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${app_name}
DNS.2 = ${app_name}.${app_name}
DNS.3 = ${app_name}.${app_name}.svc
EOF

    openssl req -new \
        -key "${dst_dir}/server-key.pem" \
        -subj "/O=system:nodes/CN=system:node:${app_name}.${app_name}.svc" \
        -out "${dst_dir}/server.csr" \
        -config "${dst_dir}/csr.conf"
}


function create_k8s_csr() {
    local server_csr_path="${1?No server_csr_path passed}"
    local csr_name="${2?No csr_name passed}"
    local total_tries=30
    local retry_num
    # clean-up any previously created CSR for our service. Ignore errors if not present.
    kubectl delete csr "${csr_name}" 2>/dev/null || true

    # create  server cert/key CSR and  send to k8s API
    cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
    name: ${csr_name}
spec:
    signerName: kubernetes.io/kubelet-serving
    groups:
        - system:authenticated
    request: $(< "${server_csr_path}" base64 | tr -d '\n')
    usages:
        - digital signature
        - key encipherment
        - server auth
EOF

    echo "Verifying that the csr ${csr_name} was created..."
    for retry_num in $(seq $total_tries); do
        if kubectl get csr "${csr_name}"; then
            return 0
        fi
        echo "try ${retry_num}/${total_tries} ..."
        sleep 1
    done
    echo "Failed to verify that csr ${csr_name} was created ('kubectl get csr ${csr_name}' failed to execute)."
    return 1
}


function approve_csr() {
    local csr_name="${1?No csr_name passed}"
    local total_tries=30
    local retry_num
    local server_cert

    kubectl certificate approve "${csr_name}"

    echo "Verifying that the csr ${csr_name} approval went through..."
    for retry_num in $(seq $total_tries); do
        server_cert=$(kubectl get csr "${csr_name}" -o jsonpath='{.status.certificate}')
        if [[ $server_cert != '' ]]; then
            break
        fi
        if [[ $(kubectl get csr "${csr_name}" -o jsonpath='{.status.conditions[-1].type}') == "Failed" ]]; then
            echo "Failed generating the signed cert: $(kubectl get csr "${csr_name}" -o jsonpath='{.status.conditions[-1].message}')"
            return 1
        fi
        echo "try ${retry_num}/${total_tries} ..."
        sleep 1
    done
    if [[ $server_cert == '' ]]; then
        echo "ERROR: After approving csr ${csr_name}, the signed certificate did not appear on the resource. Giving up after ${total_tries} attempts." >&2
        return 1
    fi
}

function download_signed_pem() {
    local csr_name="${1?No csr_name}"
    local out_pem_path="${2?No out_pem_path}"
    kubectl get csr "${csr_name}" -o jsonpath='{.status.certificate}' \
    | openssl base64 -d -A -out "${out_pem_path}"
}


function upload_signed_crts_to_k8s() {
    # TODO: we might want to move this to kustomize secretGenerator config
    local certs_dir="${1?No certs_dir passed}"
    local app_name="${2?No app_name passed}"
    # create the secret with CA cert and server cert/key
    kubectl create secret generic "${app_name}-certs" \
            --from-file=key.pem="${certs_dir}/server-key.pem" \
            --from-file=cert.pem="${certs_dir}/server-cert.pem" \
            --dry-run=client\
             -o yaml \
    | kubectl -n "${app_name}" apply -f -
}


function main() {
    local tmpdir
    local app_name="buildpack-admission"
    local csr_name=${app_name}.${app_name}
    tmpdir=$(mktemp -d)

    if [[ ${1:-} == '-h' ]]; then
        help
        exit 0
    fi

    if [[ ${1:-} == '-v' ]]; then
        shift
        set -x
    fi

    kubectl create namespace "$app_name" || true
    generate_cert "$tmpdir" "$app_name"
    create_certificate_signing_request "$tmpdir" "$app_name"
    create_k8s_csr "${tmpdir}/server.csr" "$csr_name"
    approve_csr "$csr_name"
    download_signed_pem "$csr_name" "${tmpdir}/server-cert.pem"
    upload_signed_crts_to_k8s "$tmpdir" "$app_name"
}


main "$@"
