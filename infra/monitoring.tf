locals {
  metric_namespace = "${var.project}/${var.environment}"
}

resource "aws_sns_topic" "alerts" {
  name = "${local.name}-alerts"
}

resource "aws_sns_topic_subscription" "alerts" {
  for_each = toset(var.alarm_email_addresses)

  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = each.value
}

resource "aws_cloudwatch_log_metric_filter" "application_errors" {
  name           = "${local.name}-application-errors"
  log_group_name = module.app_lambda.lambda_cloudwatch_log_group_name
  pattern        = "{ $.level = \"ERROR\" }"

  metric_transformation {
    name      = "ApplicationErrors"
    namespace = local.metric_namespace
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_metric_alarm" "application_errors" {
  alarm_name        = "${local.name}-application-errors"
  alarm_description = "The app logged a server error. The request that failed is in the ${module.app_lambda.lambda_cloudwatch_log_group_name} log group at level ERROR."

  namespace   = local.metric_namespace
  metric_name = aws_cloudwatch_log_metric_filter.application_errors.metric_transformation[0].name
  statistic   = "Sum"
  period      = 60

  comparison_operator = "GreaterThanThreshold"
  threshold           = 0
  evaluation_periods  = 1
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.alerts.arn]
  ok_actions    = [aws_sns_topic.alerts.arn]
}

resource "aws_cloudwatch_metric_alarm" "lambda_faults" {
  alarm_name        = "${local.name}-lambda-faults"
  alarm_description = "The function failed outside its handlers: a timeout, an out-of-memory, or a start-up that could not read its configuration."

  namespace   = "AWS/Lambda"
  metric_name = "Errors"
  dimensions  = { FunctionName = module.app_lambda.lambda_function_name }
  statistic   = "Sum"
  period      = 60

  comparison_operator = "GreaterThanThreshold"
  threshold           = 0
  evaluation_periods  = 1
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.alerts.arn]
  ok_actions    = [aws_sns_topic.alerts.arn]
}
