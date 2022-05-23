# Basic hello world sample
resource "null_resource" "sample" {
  provisioner "local-exec" {
    command     = "echo ${var.hello}"
    interpreter = ["/bin/bash", "-c"]
  }
}

resource "null_resource" "remove" {
  provisioner "local-exec" {
    command     = "echo 'remove me from state file'}"
    interpreter = ["/bin/bash", "-c"]
  }
}
