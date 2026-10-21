module "app_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "8.1.0"

  architectures              = ["arm64"]
  attach_policy_json         = true
  authorization_type         = "NONE"
  create_lambda_function_url = true
  create_package             = false
  description                = "Go-Chat"
  function_name              = "${local.name}-app"
  handler                    = "bootstrap"
  runtime                    = "provided.al2023"

  cloudwatch_logs_retention_in_days = 14
  memory_size                       = 128
  policy_json                       = data.aws_iam_policy_document.iam_policy.json
  publish                           = true
  timeout                           = 60

  local_existing_package = var.lambda_package
  environment_variables = {
    STORE_BACKEND     = "s3"
    S3_BUCKET_NAME    = module.exchanges_s3_bucket.s3_bucket_id
    S3_BUCKET_PATH    = local.bucket_path
    GIN_MODE          = "release"
    LLM_PROVIDER      = var.llm_provider
    LLM_MODEL         = var.llm_model
    SSM_KEY_NAME      = local.llm_api_key_ssm_ref
    SSM_USER_KEY_NAME = var.create_app_auth_ssm_parameters ? aws_ssm_parameter.app_auth["user"].name : local.app_auth_ssm_names.user
    SSM_PASS_KEY_NAME = var.create_app_auth_ssm_parameters ? aws_ssm_parameter.app_auth["pass"].name : local.app_auth_ssm_names.pass
  }
}
