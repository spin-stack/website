variable "domain" {
  type = string
}

variable "log_retention_days" {
  type    = number
  default = 30
}

resource "aws_kms_key" "encryption" {}

resource "tls_private_key" "ca" {}

resource "aws_route53_record" "app" {
  for_each = toset(["app.${var.domain}", "*.ws.${var.domain}"])
  name     = each.value
}

locals {
  term = "term.${var.domain}"
}

resource "aws_route53_record" "terminal" {
  name = local.term
}

resource "aws_route53_record" "issuer" {
  name = "id.${var.domain}"
}

# Named by the certificate AWS issues, which no page can say.
resource "aws_route53_record" "validation" {
  for_each = { for o in aws_acm_certificate.app.domain_validation_options : o.domain_name => o }
  name     = each.value.resource_record_name
}
