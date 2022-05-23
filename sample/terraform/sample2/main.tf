# Sample that calls a module in a relative path
module "sample1" {
  source         = "../sample1"
  hello          = var.hello
  region         = var.region
  resource_group = var.resource_group
  prefix         = var.prefix
  resource_tags  = var.resource_tags
}
