# IBM Cloud Terratest Wrapper
[![Build Status](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/main.yml/badge.svg)](https://github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/actions/workflows/main.yml)

- [Overview](#overview)
- [Contributing](#contributing)
    + [Local Development Setup](#local-development-setup)
    + [Running Tests](#running-tests)
    + [Publishing](#publishing)

## Overview
This Go module provides helper functions that wrap terratest so tests can be created quickly and consistently.

## Contributing
If you would like to contribute to this project, please read through the documentation: **To Be Added**

### Local Development Setup
This Go project uses submodules, pre-commit hooks, and some other tools that are common across all projects in this org. Before you start contributing to the project, please follow the following guide on setting up your environment: **To Be Added**

### Running Tests
If you would like to run unit tests for all of the packages in this module, you can use the `go test` command, either for a single package or all packages.
```bash
# run single package tests
go test -v ./cloudinfo
```

```bash
# run all packages tests
go test -v ./...
```

### Publishing
Publishing is handled automatically via merge pipeline and the Semantic Versioning automation. This creates a new Github release
