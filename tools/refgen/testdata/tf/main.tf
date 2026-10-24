variable "domain" {
  type = string
}

variable "spin_version" {
  type = string
}

variable "decommission" {
  type    = bool
  default = false
}

module "controlplane" {
  source = "./modules/controlplane"
  domain = var.domain
}

module "runners" {
  source = "./modules/runners"
}

output "update_status" {
  value = "aws autoscaling describe-instance-refreshes"
}
