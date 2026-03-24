provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile

  default_tags {
    tags = {
      Project     = "govtrove"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

provider "grafana" {
  cloud_access_policy_token = var.grafana_cloud_access_token
  url                       = "https://${var.grafana_stack_slug}.grafana.net"
  auth                      = var.grafana_sa_token
}
