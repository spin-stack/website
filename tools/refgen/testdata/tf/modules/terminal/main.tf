variable "domain" {
  type = string
}

locals {
  host = "term.${var.domain}"
}

resource "aws_route53_record" "terminal" {
  name = local.host
}
