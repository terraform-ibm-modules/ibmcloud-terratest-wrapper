# Sample that destroys and updates on second plan/apply
resource "null_resource" "sample" {
  triggers = {
    run_always = uuid()
  }
  provisioner "local-exec" {
    command     = "${path.module}/../../scripts/hello.sh"
    interpreter = ["/bin/bash", "-c"]
  }
}
