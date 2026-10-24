variable "domain" {
  type = string
}

resource "aws_route53_record" "issuer" {
  name = "id.${var.domain}"
}
