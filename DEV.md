# Dev Info

Follow these steps to try Cloud Run Button's underlying command:

## Create a Service Account

1. [Create a Service Account in a test account](https://console.cloud.google.com/iam-admin/serviceaccounts)
1. Download the key json file for the new service account

## Build the Container

```
docker build -f Dockerfile -t cloud-run-button .
```

## Test the Cloud Run Button

1. Set an env var pointing to the Service Account's JSON file:
    ```
    export KEY_FILE=PATH_TO_YOUR_SERVICE_ACCOUNT_KEY_FILE
    ```

1. Run Cloud Run Button via Docker:
    ```
    docker run -it -v /var/run/docker.sock:/var/run/docker.sock -v $KEY_FILE:/root/user.json --entrypoint=/bin/sh cloud-run-button -c "gcloud auth activate-service-account --key-file=/root/user.json --quiet && gcloud auth configure-docker --quiet && /bin/cloudshell_open --repo_url=https://github.com/GoogleCloudPlatform/cloud-run-hello.git"
    ```