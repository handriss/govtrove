# --- Pipeline Step Functions alarm ---

resource "aws_cloudwatch_metric_alarm" "pipeline_failures" {
  alarm_name          = "${var.project_name}-pipeline-failures"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ExecutionsFailed"
  namespace           = "AWS/States"
  period              = 300
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "Step Functions pipeline execution failed"
  treat_missing_data  = "notBreaching"

  dimensions = {
    StateMachineArn = aws_sfn_state_machine.pipeline.arn
  }

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-pipeline-failures-alarm"
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
