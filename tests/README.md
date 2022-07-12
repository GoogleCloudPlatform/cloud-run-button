# Integration tests

The service account that runs these tests (either the default Cloud Build Service Account, or a custom account) requires the same permissions as in the [CONTRIBUTING](../CONTRIBUTING.md) guide. 

Run `../integration.cloudbuild.yaml` to test variations on deployments. Open any test folder in GitHub in the browser and click the Run on Google Cloud to test in isolation. 

When adding new features, you should:

 * create a new folder in `test/`
 * append the test to the `run tests` step in `../integration.cloudbuild.yaml`