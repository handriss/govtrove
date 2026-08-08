resource "aws_sns_topic" "notifications" {
  name = "${var.project_name}-notifications"

  tags = {
    Name = "${var.project_name}-notifications"
  }
}

resource "aws_sns_topic_policy" "notifications" {
  arn = aws_sns_topic.notifications.arn

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCostAnomalyDetection"
        Effect    = "Allow"
        Principal = { Service = "costalerts.amazonaws.com" }
        Action    = "SNS:Publish"
        Resource  = aws_sns_topic.notifications.arn
      }
    ]
  })
}

# Email subscriptions are managed by hand, not Terraform. SNS deletes an
# unconfirmed email subscription after 3 days, so a Terraform-created one that
# nobody clicks disappears and reappears in every plan forever. Current
# subscribers: info@govtrove.com (notifications), andrew@govtrove.com (bounces).
