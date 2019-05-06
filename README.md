# Cloud Run Button

If you have a public repository, you can add this button to your `README.md` and
let anyone deploy your application to [Google Cloud Run][run] with a single
click.

[run]: https://cloud.google.com/run

Try it out:

[![Run on Google
Cloud](https://storage.googleapis.com/cloudrun/button.png)](https://console.cloud.google.com/cloudshell/editor?shellonly=true&cloudshell_image=gcr.io/ahmetb-public/button&cloudshell_git_repo=https://github.com/jamesward/cloud-run-button.git)

### Instructions

1. Copy & paste this markdown:

    ```md
    [![Run on Google Cloud](https://storage.googleapis.com/cloudrun/button.png)](https://console.cloud.google.com/cloudshell/editor?shellonly=true&cloudshell_image=gcr.io/ahmetb-public/button&cloudshell_git_repo=YOUR_HTTP_GIT_URL)
    ```

1. Replace `YOUR_HTTP_GIT_URL` with your HTTP git URL, like:
   `https://github.com/jamesward/hello-kotlin-ktor.git`

1. Make sure the repository has a Dockerfile, so it can be built using the
   `docker build` command.

### Notes

- Disclaimer: This is not an official Google project.
- See [LICENSE](./LICENSE) for the licensing information.
- See [Contribution Guidelines](./CONTRIBUTING.md) on how to contribute.
