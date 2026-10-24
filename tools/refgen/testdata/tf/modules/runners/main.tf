variable "runner_spot" {
  type    = bool
  default = true
}

locals {
  relay = "tunnel.${local.app}"
  app   = "app.${var.domain}"
}

resource "aws_route53_record" "relay" {
  name = local.relay
}
