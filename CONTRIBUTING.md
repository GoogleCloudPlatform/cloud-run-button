# How to Contribute

## Test Cloud Run Button's Underlying Command Locally with Local Go

1. Download Go 1.14.x
1. Run the tests:
    ```
    go test ./cmd/cloudshell_open
    ```
1. Build the command:
    ```
    go build -o /tmp/cloudshell_open ./cmd/cloudshell_open
    ```
1. To test the command:
    1. [Enable the cloudresourcemanager API](https://console.developers.google.com/apis/api/cloudresourcemanager.googleapis.com/overview)
    1. [Enable the billing API](https://console.developers.google.com/apis/api/cloudbilling.googleapis.com/overview)
    1. Create a Service Account with the *Cloud Run Admin*, *ServiceEnabler*, *Service Account User*, and *Storage Admin* roles
    1. Download the JSON key
    1. Authenticate gcloud as the service account:
        ```
        export GOOGLE_APPLICATION_CREDENTIALS=PATH_TO_YOUR_SERVICE_ACCOUNT_KEY_FILE
        gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS
        ```
    1. Run the button:
        ```
        (cd /tmp; ./cloudshell_open --repo_url=https://github.com/GoogleCloudPlatform/cloud-run-hello.git; rm -rf cloud-run-hello)
        ```

## Test Cloud Run Button's Underlying Command Locally in a Container

1. [Create a Service Account in a test account](https://console.cloud.google.com/iam-admin/serviceaccounts)
1. Download the key json file for the new service account
1. Build the Container

    ```
    docker build -f Dockerfile -t cloud-run-button .
    ```

1. Set an env var pointing to the Service Account's JSON file:

    ```
    export KEY_FILE=PATH_TO_YOUR_SERVICE_ACCOUNT_KEY_FILE
    ```

1. Run Cloud Run Button via Docker:
    ```
    docker run -it -v /var/run/docker.sock:/var/run/docker.sock \
      -v $KEY_FILE:/root/user.json \
      -e GOOGLE_APPLICATION_CREDENTIALS=/root/user.json \
      --entrypoint=/bin/sh cloud-run-button -c \
      "gcloud auth activate-service-account --key-file=/root/user.json \
      --quiet && gcloud auth configure-docker --quiet && \
      /bin/cloudshell_open \
      --repo_url=https://github.com/GoogleCloudPlatform/cloud-run-hello.git"
    ```

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution;
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## Community Guidelines

This project follows [Google's Open Source Community
Guidelines](https://opensource.google.com/conduct/).
