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

# --- App Runner error rate alarms ---

resource "aws_cloudwatch_metric_alarm" "api_5xx" {
  alarm_name          = "${var.project_name}-api-5xx-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "5xxStatusResponses"
  namespace           = "AWS/AppRunner"
  period              = 300
  statistic           = "Sum"
  threshold           = 5
  alarm_description   = "API returning >5 server errors per 5 minutes"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ServiceName = aws_apprunner_service.api.service_name
    ServiceId   = aws_apprunner_service.api.service_id
  }

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-api-5xx-errors"
  }
}

resource "aws_cloudwatch_metric_alarm" "api_4xx" {
  alarm_name          = "${var.project_name}-api-4xx-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "4xxStatusResponses"
  namespace           = "AWS/AppRunner"
  period              = 300
  statistic           = "Sum"
  threshold           = 50
  alarm_description   = "API returning >50 client errors per 5 minutes for 2 consecutive periods"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ServiceName = aws_apprunner_service.api.service_name
    ServiceId   = aws_apprunner_service.api.service_id
  }

  alarm_actions = [aws_sns_topic.notifications.arn]
  ok_actions    = [aws_sns_topic.notifications.arn]

  tags = {
    Name = "${var.project_name}-api-4xx-errors"
  }
}

# --- Log group retention ---

resource "aws_cloudwatch_log_group" "lambda_pipeline" {
  for_each = toset(local.lambda_functions)

  name              = "/aws/lambda/${var.project_name}-${each.key}"
  retention_in_days = 14

  tags = {
    Name = "${var.project_name}-${each.key}-logs"
  }
}

# --- Dashboard ---

resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "${var.project_name}-dashboard"

  dashboard_body = jsonencode({
    widgets = [
      # Row 1: API Health
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 8
        height = 6
        properties = {
          title  = "API Request Count"
          region = var.aws_region
          period = 300
          stat   = "Sum"
          metrics = [
            ["AWS/AppRunner", "RequestCount", "ServiceName", aws_apprunner_service.api.service_name, "ServiceId", aws_apprunner_service.api.service_id]
          ]
        }
      },
      {
        type   = "metric"
        x      = 8
        y      = 0
        width  = 8
        height = 6
        properties = {
          title  = "API Error Rates"
          region = var.aws_region
          period = 300
          stat   = "Sum"
          metrics = [
            ["AWS/AppRunner", "4xxStatusResponses", "ServiceName", aws_apprunner_service.api.service_name, "ServiceId", aws_apprunner_service.api.service_id, { label = "4xx" }],
            ["AWS/AppRunner", "5xxStatusResponses", "ServiceName", aws_apprunner_service.api.service_name, "ServiceId", aws_apprunner_service.api.service_id, { label = "5xx" }]
          ]
        }
      },
      {
        type   = "metric"
        x      = 16
        y      = 0
        width  = 8
        height = 6
        properties = {
          title  = "Active Instances"
          region = var.aws_region
          period = 300
          stat   = "Average"
          metrics = [
            ["AWS/AppRunner", "ActiveInstances", "ServiceName", aws_apprunner_service.api.service_name, "ServiceId", aws_apprunner_service.api.service_id]
          ]
        }
      },

      # Row 2: Pipeline Health
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 8
        height = 6
        properties = {
          title  = "Pipeline Executions"
          region = var.aws_region
          period = 300
          stat   = "Sum"
          metrics = [
            ["AWS/States", "ExecutionsSucceeded", "StateMachineArn", aws_sfn_state_machine.pipeline.arn, { label = "Succeeded" }],
            ["AWS/States", "ExecutionsFailed", "StateMachineArn", aws_sfn_state_machine.pipeline.arn, { label = "Failed", color = "#d62728" }]
          ]
        }
      },
      {
        type   = "metric"
        x      = 8
        y      = 6
        width  = 8
        height = 6
        properties = {
          title  = "Pipeline Execution Time"
          region = var.aws_region
          period = 300
          stat   = "Average"
          metrics = [
            ["AWS/States", "ExecutionTime", "StateMachineArn", aws_sfn_state_machine.pipeline.arn]
          ]
          yAxis = { left = { label = "ms" } }
        }
      },
      {
        type   = "metric"
        x      = 16
        y      = 6
        width  = 8
        height = 6
        properties = {
          title  = "DLQ Messages"
          region = var.aws_region
          period = 300
          stat   = "Maximum"
          view   = "singleValue"
          metrics = [
            ["AWS/SQS", "ApproximateNumberOfMessagesVisible", "QueueName", "${var.project_name}-pipeline-dlq"]
          ]
        }
      },

      # Row 3: Lambda Performance
      {
        type   = "metric"
        x      = 0
        y      = 12
        width  = 12
        height = 6
        properties = {
          title  = "Lambda Duration"
          region = var.aws_region
          period = 300
          stat   = "Average"
          metrics = [
            for fn in local.lambda_functions : [
              "AWS/Lambda", "Duration", "FunctionName", "${var.project_name}-${fn}", { label = fn }
            ]
          ]
          yAxis = { left = { label = "ms" } }
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 12
        width  = 12
        height = 6
        properties = {
          title  = "Lambda Errors"
          region = var.aws_region
          period = 300
          stat   = "Sum"
          metrics = [
            for fn in local.lambda_functions : [
              "AWS/Lambda", "Errors", "FunctionName", "${var.project_name}-${fn}", { label = fn }
            ]
          ]
        }
      },

      # Row 4: Alarms & Cost
      {
        type   = "alarm"
        x      = 0
        y      = 18
        width  = 12
        height = 4
        properties = {
          title  = "Alarm Status"
          alarms = [
            aws_cloudwatch_metric_alarm.pipeline_failures.arn,
            aws_cloudwatch_metric_alarm.api_5xx.arn,
            aws_cloudwatch_metric_alarm.api_4xx.arn,
            aws_cloudwatch_metric_alarm.api_request_spike.arn,
            aws_cloudwatch_metric_alarm.billing_30.arn,
            aws_cloudwatch_metric_alarm.billing_50.arn,
          ]
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 18
        width  = 12
        height = 4
        properties = {
          title  = "Estimated Charges (USD)"
          region = "us-east-1"
          period = 21600
          stat   = "Maximum"
          view   = "singleValue"
          metrics = [
            ["AWS/Billing", "EstimatedCharges", "Currency", "USD"]
          ]
        }
      }
    ]
  })
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
