# tflint-ignore: all
variable "var1" {
  description = "Region"
  type        = string
  default     = "us-east"
}

# tflint-ignore: all
variable "env" {
  description = "The environment name (e.g., dev, staging, prod)"
  type        = string
  default     = "dev"
}


# tflint-ignore: all
variable "var2" {
  description = "test variable"
  type        = string
  default     = "abcd"
}
