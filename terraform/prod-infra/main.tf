# main.tf

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "rmalisetti-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "ap-south-1"
    profile        = "terraform-dev"
    dynamodb_table = "terraform-state-lock"
    encrypt        = true
  }
}

provider "aws" {
  region  = var.aws_region
  profile = "terraform-dev"
}
