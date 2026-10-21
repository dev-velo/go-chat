output "app_url" {
  description = "Public HTTPS endpoint for the chat app."
  value       = module.app_lambda.lambda_function_url
}

output "app_user_name" {
  description = "Generated sign-in username. Null when using existing SSM credentials."
  value       = var.create_app_auth_ssm_parameters ? random_password.app_user[0].result : null
  sensitive   = true
}

output "app_user_pass" {
  description = "Generated sign-in password. Null when using existing SSM credentials."
  value       = var.create_app_auth_ssm_parameters ? random_password.app_pass[0].result : null
  sensitive   = true
}
