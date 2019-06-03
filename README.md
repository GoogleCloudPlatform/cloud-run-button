# Cloud Run Button

If you have a public repository, you can add this button to your `README.md` and
let anyone deploy your application to [Google Cloud Run][run] with a single
click.

[run]: https://cloud.google.com/run

Try it out:

[![Run on Google
Cloud](https://storage.googleapis.com/cloudrun/button.svg)](https://console.cloud.google.com/cloudshell/editor?shellonly=true&cloudshell_image=gcr.io/ahmetb-public/button&cloudshell_git_repo=https://github.com/jamesward/cloud-run-button.git)

### Instructions

1. Copy & paste this markdown:

    ```md
    [![Run on Google Cloud](https://storage.googleapis.com/cloudrun/button.svg)](https://console.cloud.google.com/cloudshell/editor?shellonly=true&cloudshell_image=gcr.io/ahmetb-public/button&cloudshell_git_repo=YOUR_HTTP_GIT_URL)
    ```

1. Replace `YOUR_HTTP_GIT_URL` with your HTTP git URL, like:
   `https://github.com/jamesward/hello-kotlin-ktor.git`

1. Make sure the repository has a Dockerfile, so it can be built using the
   `docker build` command.

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
  - `required`, _(optional, default: `true`)_ indicaes if they user must provide
    a value for this variable.

### Notes

- Disclaimer: This is not an official Google project.
- See [LICENSE](./LICENSE) for the licensing information.
- See [Contribution Guidelines](./CONTRIBUTING.md) on how to contribute.
