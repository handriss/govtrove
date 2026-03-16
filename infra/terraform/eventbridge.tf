resource "aws_scheduler_schedule" "pipeline_peak" {
  name       = "${var.project_name}-pipeline-peak"
  group_name = "default"

  state = var.pipeline_schedule_enabled ? "ENABLED" : "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  # SAM.gov uploads the daily CSV around 03:30-04:45 UTC.
  # Check every 10 min during a wide window to catch it quickly,
  # with margin for US DST shifts.
  schedule_expression          = "cron(0/10 2-6 * * ? *)"
  schedule_expression_timezone = "UTC"

  target {
    arn      = aws_sfn_state_machine.pipeline.arn
    role_arn = aws_iam_role.eventbridge_scheduler.arn
  }
}

resource "aws_scheduler_schedule" "pipeline_offpeak" {
  name       = "${var.project_name}-pipeline-offpeak"
  group_name = "default"

  state = var.pipeline_schedule_enabled ? "ENABLED" : "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  # Off-peak: check every 3 hours as a safety net
  schedule_expression          = "cron(0 8,11,14,17,20,23 * * ? *)"
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
