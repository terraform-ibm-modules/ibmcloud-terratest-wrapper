variable "var1" {
  description = "Region"
  type        = string
  default     = "us-east"
}

variable "env" {
  description = "The environment name (e.g., dev, staging, prod)"
  type        = string
  default     = "dev"
}