resource "aws_scheduler_schedule" "pipeline" {
  name       = "${var.project_name}-pipeline-schedule"
  group_name = "default"

  state = var.pipeline_schedule_enabled ? "ENABLED" : "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression          = var.bulkcsv_schedule_expression
  schedule_expression_timezone = "UTC"

  target {
    arn      = aws_sfn_state_machine.pipeline.arn
    role_arn = aws_iam_role.eventbridge_scheduler.arn
  }
}

resource "aws_scheduler_schedule" "ingest_api_fallback" {
  name       = "${var.project_name}-ingest-api-fallback"
  group_name = "default"

  state = var.pipeline_schedule_enabled ? "ENABLED" : "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression          = "rate(24 hours)"
  schedule_expression_timezone = "UTC"

  target {
    arn      = aws_lambda_function.pipeline["ingest-api"].arn
    role_arn = aws_iam_role.eventbridge_scheduler.arn
    input    = jsonencode({ source = "fallback" })
  }
}

resource "aws_lambda_permission" "ingest_api_fallback" {
  statement_id  = "AllowEventBridgeScheduler"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.pipeline["ingest-api"].function_name
  principal     = "scheduler.amazonaws.com"
  source_arn    = aws_scheduler_schedule.ingest_api_fallback.arn
}
