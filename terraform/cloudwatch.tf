resource "aws_cloudwatch_log_group" "ingestion" {
  name              = "/govtrove/ingestion"
  retention_in_days = 30

  tags = {
    Name = "${var.project_name}-ingestion-logs"
  }
}

resource "aws_cloudwatch_log_group" "csvarchive" {
  name              = "/govtrove/csvarchive"
  retention_in_days = 30

  tags = {
    Name = "${var.project_name}-csvarchive-logs"
  }
}

resource "aws_cloudwatch_log_group" "archivedcsv" {
  name              = "/govtrove/archivedcsv"
  retention_in_days = 30

  tags = {
    Name = "${var.project_name}-archivedcsv-logs"
  }
}

resource "aws_cloudwatch_log_metric_filter" "archivedcsv_failed" {
  name           = "${var.project_name}-archivedcsv-failed"
  pattern        = "{ $.level = \"error\" }"
  log_group_name = aws_cloudwatch_log_group.archivedcsv.name

  metric_transformation {
    name      = "ArchivedCSVErrors"
    namespace = "GovTrove"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "archivedcsv_errors" {
  alarm_name          = "${var.project_name}-archivedcsv-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ArchivedCSVErrors"
  namespace           = "GovTrove"
  period              = 300
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "Triggered when archived CSV service logs errors"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-archivedcsv-errors-alarm"
  }
}

resource "aws_cloudwatch_log_metric_filter" "csvarchive_failed" {
  name           = "${var.project_name}-csvarchive-failed"
  pattern        = "{ $.level = \"error\" }"
  log_group_name = aws_cloudwatch_log_group.csvarchive.name

  metric_transformation {
    name      = "CSVArchiveErrors"
    namespace = "GovTrove"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "csvarchive_errors" {
  alarm_name          = "${var.project_name}-csvarchive-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "CSVArchiveErrors"
  namespace           = "GovTrove"
  period              = 300
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "Triggered when CSV archive service logs errors"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-csvarchive-errors-alarm"
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

# Fires when no successful ingestion completion is detected in 26 hours.
# Catches silent failures (OOM kills, storage limit crashes) that don't log errors.
resource "aws_cloudwatch_metric_alarm" "ingestion_missing" {
  alarm_name          = "${var.project_name}-ingestion-missing"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 1
  metric_name         = "IngestionCompleted"
  namespace           = "GovTrove"
  period              = 93600 # 26 hours — allows buffer beyond the daily 24h schedule
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "No successful ingestion in 26 hours — task may be crashing silently"
  treat_missing_data  = "breaching"

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-ingestion-missing-alarm"
  }
}

# --- Billing alarms (tiered) ---

resource "aws_cloudwatch_metric_alarm" "billing_15" {
  alarm_name          = "${var.project_name}-billing-15"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "EstimatedCharges"
  namespace           = "AWS/Billing"
  period              = 21600
  statistic           = "Maximum"
  threshold           = 15
  alarm_description   = "Estimated monthly charges exceed $15"
  treat_missing_data  = "notBreaching"

  dimensions = {
    Currency = "USD"
  }

  alarm_actions = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-billing-15"
  }
}

resource "aws_cloudwatch_metric_alarm" "billing_30" {
  alarm_name          = "${var.project_name}-billing-30"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "EstimatedCharges"
  namespace           = "AWS/Billing"
  period              = 21600
  statistic           = "Maximum"
  threshold           = 30
  alarm_description   = "Estimated monthly charges exceed $30"
  treat_missing_data  = "notBreaching"

  dimensions = {
    Currency = "USD"
  }

  alarm_actions = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-billing-30"
  }
}

resource "aws_cloudwatch_metric_alarm" "billing_50" {
  alarm_name          = "${var.project_name}-billing-50"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "EstimatedCharges"
  namespace           = "AWS/Billing"
  period              = 21600
  statistic           = "Maximum"
  threshold           = 50
  alarm_description   = "Estimated monthly charges exceed $50 — investigate immediately"
  treat_missing_data  = "notBreaching"

  dimensions = {
    Currency = "USD"
  }

  alarm_actions = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-billing-50"
  }
}

# --- App Runner request spike alarm ---

resource "aws_cloudwatch_metric_alarm" "api_request_spike" {
  alarm_name          = "${var.project_name}-api-request-spike"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "RequestCount"
  namespace           = "AWS/AppRunner"
  period              = 300
  statistic           = "Sum"
  threshold           = 1000
  alarm_description   = "API receiving >1000 requests per 5 minutes for 2 consecutive periods — possible bot traffic"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ServiceName = aws_apprunner_service.api.service_name
    ServiceId   = aws_apprunner_service.api.service_id
  }

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-api-request-spike"
  }
}

# --- Cost Anomaly Detection ---

resource "aws_ce_anomaly_monitor" "cost" {
  name              = "Default-Services-Monitor"
  monitor_type      = "DIMENSIONAL"
  monitor_dimension = "SERVICE"
}

resource "aws_ce_anomaly_subscription" "cost" {
  name      = "${var.project_name}-cost-alerts"
  frequency = "IMMEDIATE"

  monitor_arn_list = [aws_ce_anomaly_monitor.cost.arn]

  subscriber {
    type    = "SNS"
    address = aws_sns_topic.notifications.arn
  }

  threshold_expression {
    dimension {
      key           = "ANOMALY_TOTAL_IMPACT_ABSOLUTE"
      values        = ["5"]
      match_options = ["GREATER_THAN_OR_EQUAL"]
    }
  }
}
