terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket         = "govtrove-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    profile        = "govtrove"
    dynamodb_table = "govtrove-terraform-lock"
    encrypt        = true
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    grafana = {
      source  = "grafana/grafana"
      version = "~> 3.0"
    }
  }
}
