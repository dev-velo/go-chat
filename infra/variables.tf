variable "aws_region" {
  description = "Region to deploy into. No default: the region belongs to the operator, not the repo."
  type        = string
}

variable "aws_iam_role" {
  description = "ARN of the IAM role to assume for deployment."
  type        = string
}

variable "environment" {
  description = "Deployment environment, used in resource names and tags."
  type        = string
  default     = "dev"

  validation {
    condition     = can(regex("^[a-z0-9-]{1,12}$", var.environment))
    error_message = "environment must be 1-12 lowercase letters, digits or hyphens."
  }
}

variable "project" {
  description = "Name prefix for every resource. Keep it short: it is part of a globally unique bucket name."
  type        = string
  default     = "go-chat"
}

variable "exchange_retention_days" {
  description = "Days to keep prompt/response history before S3 deletes it."
  type        = number
  default     = 30

  validation {
    condition     = var.exchange_retention_days >= 1
    error_message = "exchange_retention_days must be at least 1."
  }
}

variable "lambda_package" {
  description = <<-EOT
    Path to the zipped Lambda package, built by scripts/terraform.sh before
    apply. Not built by Terraform: a plan should not require a Go toolchain,
    and the artifact that was reviewed should be the one that ships.
  EOT
  type        = string
  default     = "../app/build/bootstrap.zip"
}


variable "create_llm_api_key_ssm_parameter" {
  description = "Create and manage the API key parameter. False uses an existing parameter without reading or changing its value."
  type        = bool
  default     = false
  nullable    = false
}

variable "llm_api_key_ssm_name" {
  description = "SSM parameter name in the deployment account and region, starting with /. Null defaults to /<project>/<environment>/llm-api-key. Used for both existing and newly created parameters."
  type        = string
  default     = null

  validation {
    condition     = var.llm_api_key_ssm_name == null || can(regex("^/[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*$", var.llm_api_key_ssm_name))
    error_message = "llm_api_key_ssm_name must be an absolute SSM parameter path, such as /shared/openai/api-key, or null."
  }
}

variable "llm_api_key" {
  description = "API key value, required only when creating the SSM parameter. Stored in Terraform state; supply via a secure TF_VAR_llm_api_key environment variable, never a committed file."
  type        = string
  sensitive   = true
  default     = null
}

variable "llm_api_key_kms_key_arn" {
  description = "Customer-managed KMS key encrypting the SSM API key parameter. Null uses the AWS-managed SSM key."
  type        = string
  default     = null
}

variable "create_app_auth_ssm_parameters" {
  description = "Generate Basic Auth credentials and create their SecureString parameters. False uses existing parameters. Generated secrets enter state and saved plans."
  type        = bool
  default     = true
  nullable    = false
}

variable "app_user_ssm_name" {
  description = "Existing or created username parameter path. Null uses /<project>/<environment>/app-user-name."
  type        = string
  default     = null
  validation {
    condition     = var.app_user_ssm_name == null || can(regex("^/[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*$", var.app_user_ssm_name))
    error_message = "app_user_ssm_name must be an absolute SSM parameter path or null."
  }
}

variable "app_pass_ssm_name" {
  description = "Existing or created password parameter path. Null uses /<project>/<environment>/app-user-pass."
  type        = string
  default     = null
  validation {
    condition     = var.app_pass_ssm_name == null || can(regex("^/[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*$", var.app_pass_ssm_name))
    error_message = "app_pass_ssm_name must be an absolute SSM parameter path or null."
  }
}

variable "app_auth_kms_key_arn" {
  description = "Customer-managed KMS key for both Basic Auth parameters. Null uses the AWS-managed SSM key."
  type        = string
  default     = null
}

variable "llm_provider" {
  description = "Model backend the function runs with: fake, anthropic or openai."
  type        = string
  default     = "openai"
}

variable "llm_model" {
  description = "Provider-specific model id. Required by every provider; no defaults."
  type        = string
  default     = "gpt-4o-mini"
}
