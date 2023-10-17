
output "world" {
  value       = "Output: ${var.hello}"
  description = "Some sample output"
  sensitive   = true
}

output "secure_value" {
  value       = null_resource.changing.id
  description = "Some sample output"
  sensitive   = true
}
