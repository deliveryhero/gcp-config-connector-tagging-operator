# Contributing to gcp-config-connector-tagging-operator

Contributions are welcome ❤️

## Development

### Running E2E Tests

To run an E2E test, you will need to have [GCP application default credentials](https://cloud.google.com/docs/authentication/provide-credentials-adc) setup locally.
Then you can create a cluster using [kind](https://kind.sigs.k8s.io/) and run the tests:

```shell
kind create cluster
make test-e2e
```

## Submitting Changes

### Opening a PR

Follow these steps:

1. Fork this repo
2. Make desired changes
3. Open pull request
