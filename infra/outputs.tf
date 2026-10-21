output "exchanges_bucket" {
  description = "Bucket holding the prompt/response history. Passed to the app as S3_BUCKET_NAME."
  value       = module.exchanges_s3_bucket.s3_bucket_id
}

output "exchanges_prefix" {
  description = "Key prefix for exchange objects. Passed to the app as S3_BUCKET_PATH."
  value       = local.bucket_path
}

output "app_role_arn" {
  description = "Execution role the Lambda function runs as."
  value       = module.app_lambda.lambda_role_arn
}

output "app_url" {
  description = "Public HTTPS endpoint for the chat app."
  value       = module.app_lambda.lambda_function_url
}

output "function_name" {
  description = "Lambda function name, for log tailing and manual invokes."
  value       = module.app_lambda.lambda_function_name
}
