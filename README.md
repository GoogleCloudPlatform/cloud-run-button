# Cloud Run Button

If you have a public repository, you can add this button to your `README.md` and
let anyone deploy your application to [Google Cloud Run][run] with a single
click.

[run]: https://cloud.google.com/run

Try it out with a "hello, world" Go application ([source](https://github.com/GoogleCloudPlatform/cloud-run-hello)):

[![Run on Google
Cloud](https://storage.googleapis.com/cloudrun/button.svg)](https://deploy.cloud.run/?git_repo=https://github.com/GoogleCloudPlatform/cloud-run-hello.git)

### Demo
[![Cloud Run Button Demo](assets/cloud-run-button.png)](https://storage.googleapis.com/cloudrun/cloud-run-button.gif)

### Add the Cloud Run Button to Your Repo's README

1. Copy & paste this markdown:

    ```text
    [![Run on Google Cloud](https://storage.googleapis.com/cloudrun/button.svg)](https://deploy.cloud.run)
    ```

1. If the repo contains a `Dockerfile` it will be built using the `docker build` command. Otherwise, the [CNCF Buildpacks](https://buildpacks.io/) will be used to build the repo.

### Customizing source repository parameters

- When no parameters are passed, the referer is used to detect the git repo and branch
- To specify a git repo, add a `git_repo=URL` query parameter
- To specify a git branch, add a `revision=BRANCH_NAME` query parameter.
- To run the build in a subdirectory of the repo, add a `dir=SUBDIR` query parameter.


### Customizing deployment parameters

If you include an `app.json` at the root of your repository, it allows you
customize the experience such as defining an alternative service name, or
prompting for additional environment variables.

For example:

```json
{
  "name": "foo-app",
  "env": {
      "BACKGROUND_COLOR": {
          "description": "specify a css color",
          "value": "#fefefe",
          "required": false
      },
      "TITLE": {
          "description": "title for your site"
      }
  }
}
```

Reference:

- `name`: _(optional, default: repo name, or sub-directory name if specified)_
  Name of the Cloud Run service and the built container image. Not validated for
  naming restrictions.
- `env`: _(optional)_ Prompt user for environment variables.
  - `description`:  _(optional)_ short explanation of what the environment
    variable does, keep this short to make sure it fits into a line.
  - `value`: _(optional)_ default value for the variable, should be a string.
  - `required`, _(optional, default: `true`)_ indicates if they user must provide
    a value for this variable.

### Notes

- Disclaimer: This is not an officially supported Google product.
- See [LICENSE](./LICENSE) for the licensing information.
- See [Contribution Guidelines](./CONTRIBUTING.md) on how to contribute.
