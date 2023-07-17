terraform {
  required_version = ">= 1.0.0"

  required_providers {
    source  = "hashicorp/null"
    null = {
      version = ">= 3.2.1"
    }
  }
}
