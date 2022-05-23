variable "hello" {
  default     = "Hello World"
  type        = string
  description = "A sample value"
}
# tflint-ignore: terraform_unused_declarations
variable "region" {
  type        = string
  description = "used for tests"
}

# tflint-ignore: terraform_unused_declarations
variable "resource_tags" {
  type        = string
  description = "used for tests"
  default     = "123"
}
# tflint-ignore: terraform_unused_declarations
variable "resource_group" {
  type        = string
  description = "used for tests"
  default     = "123"
}

# tflint-ignore: terraform_unused_declarations
variable "prefix" {
  type        = string
  description = "used for tests"
  default     = "123"
}
