#!/usr/bin/env bash

# Use this script to set up build triggers for the Cloud Shell custom image on
# Google Cloud Build. This script currently can be run only on Cloud Shell as it
# relies on GCB base image trigger which isn’t available publicly.

set -euo pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# GCP project ID hosting the custom Cloud Shell image and GCB trigger
PROJECT_ID="${PROJECT_ID:-ahmetb-samples-playground}"

# Cloud Source Repositories (CSR) repo name hosting the source code
CSR_REPO_NAME="${CSR_REPO_NAME:-cloud-shell-button}"

# Image name for the custom Cloud Shell image
CUSTOM_IMAGE="${CUSTOM_IMAGE:-gcr.io/"${PROJECT_ID}"/button}"

self_trigger() {
    cat <<EOF
{
  "trigger_template": {
    "repoName": "${CSR_REPO_NAME}",
    "branchName": "master"
  },
  "filename": "cloudbuild.yaml"
}
EOF
}

base_image_trigger() {
        sed "s,CSR_REPO_NAME,${CSR_REPO_NAME},g" "${SCRIPT_DIR}/trigger-baseimage.json" | \
        sed "s,GCR_REPO_URL,${CUSTOM_IMAGE},g"
}

create_trigger() {
    local tempfile
    tempfile="$(mktemp)"
    cat > "${tempfile}" < /dev/stdin

    local access_token
    access_token="$(gcloud auth print-access-token -q)"

    local http_status
    http_status="$(
        curl --silent --output /dev/stderr --write-out "%{http_code}" \
            -X POST -T "${tempfile}" \
            -H "Authorization: Bearer ${access_token}" \
            "https://cloudbuild.googleapis.com/v1/projects/${PROJECT_ID}/triggers"
        )"
    rm -rf "$tempfile"

    if [[ "$http_status" -ne 200 ]]; then
        echo "Failed to create build trigger, returned HTTP ${http_status}."
        false
    fi
}

main() {
    echo >&2 "Setting up GCB triggers for image ${CUSTOM_IMAGE} on project ${PROJECT_ID}"

    echo >&2 "$(tput setaf 3)Creating trigger for Cloud Shell base image updates.$(tput sgr0)"
    base_image_trigger | create_trigger
    echo >&2 "$(tput setaf 2)Done.$(tput sgr0)"

    echo >&2 "$(tput setaf 3)Creating trigger for source code updates.$(tput sgr0)"
    self_trigger | create_trigger
    echo >&2 "$(tput setaf 2)Done.$(tput sgr0)"
}

main "$@"
