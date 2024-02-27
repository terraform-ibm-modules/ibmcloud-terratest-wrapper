terraform {
  required_version = ">= 1.0.0, <1.7.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = ">= 3.2.1"
    }
  }
}
