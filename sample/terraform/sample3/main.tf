# Sample that destroys and updates on second plan/apply
resource "null_resource" "sample" {
  triggers = {
    run_always = uuid()
  }
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
