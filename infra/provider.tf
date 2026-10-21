provider "aws" {
  region = var.aws_region
  assume_role {
    role_arn     = var.aws_iam_role
    session_name = "go-chat"
  }

  default_tags {
    tags = {
      Project     = var.project
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}