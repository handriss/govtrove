resource "aws_cloudwatch_log_group" "ingestion" {
  name              = "/govtrove/ingestion"
  retention_in_days = 30

  tags = {
    Name = "${var.project_name}-ingestion-logs"
  }
}

resource "aws_cloudwatch_log_metric_filter" "ingestion_completed" {
  name           = "${var.project_name}-ingestion-completed"
  pattern        = "{ $.msg = \"ingestion completed\" }"
  log_group_name = aws_cloudwatch_log_group.ingestion.name

  metric_transformation {
    name      = "IngestionCompleted"
    namespace = "GovTrove"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "ingestion_failed" {
  name           = "${var.project_name}-ingestion-failed"
  pattern        = "{ $.level = \"error\" }"
  log_group_name = aws_cloudwatch_log_group.ingestion.name

  metric_transformation {
    name      = "IngestionErrors"
    namespace = "GovTrove"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "ingestion_errors" {
  alarm_name          = "${var.project_name}-ingestion-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "IngestionErrors"
  namespace           = "GovTrove"
  period              = 300
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "Triggered when ingestion service logs errors"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-ingestion-errors-alarm"
  }
}
