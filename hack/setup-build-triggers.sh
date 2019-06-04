#!/usr/bin/env bash

# Use this script to set up build triggers for the Cloud Shell custom image on
# Google Cloud Build. This script currently can be run only on Cloud Shell as it
# relies on GCB base image trigger which isn’t available publicly.

set -euo pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# GCP project ID that has the Cloud Shell image for button and GCB trigger
# For PROD, use PROJECT_ID="cloudrun".
PROJECT_ID="${PROJECT_ID:?PROJECT_ID environment variable is not set}"

# Cloud Source Repositories (CSR) repo name hosting the source code
# For PROD, point to the CSR repo at https://source.cloud.google.com/cloudrun
# mirroring the source code.
CSR_REPO_NAME="${CSR_REPO_NAME:?CSR_REPO_NAME environment variable is not set}"

# Image name for the custom Cloud Shell image
IMAGE="${IMAGE:-gcr.io/"${PROJECT_ID}"/button}"

self_trigger() {
    cat <<EOF
{
  "id": "GCR_REPO_URL-self",
  "description": "Triggers on pushes to CSR repo ${CSR_REPO_NAME}",
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
        sed "s,GCR_REPO_URL,${IMAGE},g"
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

    if [[ "$http_status" -eq 409 ]]; then
        echo >&2 "$(tput setaf 3)WARNING: Build trigger already exists, not updating it.$(tput sgr0)"
    elif [[ "$http_status" -ne 200 ]]; then
        echo >&2 "$(tput setaf 1)Failed to create build trigger, returned HTTP ${http_status}.$(tput sgr0)"
        false
    fi
    echo >&2 "$(tput setaf 2)Build trigger created.$(tput sgr0)"
}

main() {
    # TODO(ahmetb): Since we use baseimage trigger on GCB which is currently a
    # private API, this can be only invoked from a Cloud Shell VM with a
    # @google.com account. Once it's public API, this check would be obsolete.
    if ! command -v cloudshell >/dev/null 2>&1; then
        echo >&2 "error: this script should be executed from a Google Cloud Shell."
        exit 1
    fi

    echo >&2 "Setting up GCB triggers for image ${IMAGE} on project ${PROJECT_ID}"

    echo >&2 "$(tput setaf 3)Creating trigger for source code updates.$(tput sgr0)"
    self_trigger | create_trigger

    echo >&2 "$(tput setaf 3)Creating trigger for Cloud Shell base image updates.$(tput sgr0)"
    base_image_trigger | create_trigger
}

main "$@"
