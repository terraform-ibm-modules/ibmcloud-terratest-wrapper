# IBM Cloud Terratest wrapper
[![Build Status](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/ci.yml/badge.svg)](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/ci.yml)
[![semantic-release](https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg)](https://github.com/semantic-release/semantic-release)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)

This Go module provides helper functions as a wrapper around the [Terratest](https://terratest.gruntwork.io/) library so that tests can be created quickly and consistently.

## Contributing
To contribute to this project, read through the documentation: **To Be Added**

### Setting up your local development environment

This Go project uses submodules, pre-commit hooks, and some other tools that are common across all projects in this org. Before you start contributing to the project, follow these steps to set up your environment: **To Be Added**

### Running tests

To run unit tests for all the packages in this module, you can use the `go test` command, either for a single package or all packages.

```bash
# run single package tests
go test -v ./cloudinfo
```

```bash
# run all packages tests
go test -v ./...
```

### Publishing
Publishing is handled automatically by the merge pipeline and Semantic Versioning automation, which creates a new Github release.

<!-- BEGIN EXAMPLES HOOK -->

<!-- END EXAMPLES HOOK -->
