locals {
  llm_api_key_ssm_name = var.llm_api_key_ssm_name != null ? var.llm_api_key_ssm_name : "/${var.project}/${var.environment}/llm-api-key"
  llm_api_key_ssm_arn  = "arn:${data.aws_partition.current.partition}:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${local.llm_api_key_ssm_name}"
  llm_api_key_ssm_ref  = var.create_llm_api_key_ssm_parameter ? aws_ssm_parameter.llm_api_key[0].name : local.llm_api_key_ssm_name
}

locals {
  app_auth_ssm_names = {
    user = var.app_user_ssm_name != null ? var.app_user_ssm_name : "/${var.project}/${var.environment}/app-user-name"
    pass = var.app_pass_ssm_name != null ? var.app_pass_ssm_name : "/${var.project}/${var.environment}/app-user-pass"
  }
  app_auth_ssm_arns = [
    for name in values(local.app_auth_ssm_names) :
    "arn:${data.aws_partition.current.partition}:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${name}"
  ]
}

resource "random_password" "app_user" {
  count   = var.create_app_auth_ssm_parameters ? 1 : 0
  length  = 16
  special = false
}

resource "random_password" "app_pass" {
  count   = var.create_app_auth_ssm_parameters ? 1 : 0
  length  = 32
  special = false
}

resource "aws_ssm_parameter" "app_auth" {
  for_each = var.create_app_auth_ssm_parameters ? local.app_auth_ssm_names : {}

  name        = each.value
  description = "Sign-in ${each.key} credential for ${local.name}"
  type        = "SecureString"
  key_id      = var.app_auth_kms_key_arn
  value       = each.key == "user" ? random_password.app_user[0].result : random_password.app_pass[0].result
}

resource "aws_ssm_parameter" "llm_api_key" {
  count = var.create_llm_api_key_ssm_parameter ? 1 : 0

  name        = local.llm_api_key_ssm_name
  description = "LLM API key for ${local.name}"
  type        = "SecureString"
  key_id      = var.llm_api_key_kms_key_arn
  value       = var.llm_api_key

  lifecycle {
    precondition {
      condition     = var.llm_api_key != null && try(trimspace(var.llm_api_key) != "", false)
      error_message = "llm_api_key must be provided and nonempty when create_llm_api_key_ssm_parameter is true."
    }
  }
}
