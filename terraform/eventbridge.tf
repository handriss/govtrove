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
