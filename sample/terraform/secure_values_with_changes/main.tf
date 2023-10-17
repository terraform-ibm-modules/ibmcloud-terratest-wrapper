# Basic hello world sample
resource "null_resource" "sample" {
  provisioner "local-exec" {
    command     = "echo ${var.hello}"
    interpreter = ["/bin/bash", "-c"]
  }
}

resource "null_resource" "changing" {

  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = "echo \"${var.hello} ${timestamp()}\""
  }
}
