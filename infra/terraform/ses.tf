# SES domain identity with DKIM
resource "aws_sesv2_email_identity" "domain" {
  email_identity = var.domain_name

  dkim_signing_attributes {
    next_signing_key_length = "RSA_2048_BIT"
  }
}

# Configuration set for tracking bounces, complaints, and reputation
resource "aws_sesv2_configuration_set" "main" {
  configuration_set_name = "${var.project_name}-emails"

  reputation_options {
    reputation_metrics_enabled = true
  }

  sending_options {
    sending_enabled = true
  }

  suppression_options {
    suppressed_reasons = ["BOUNCE", "COMPLAINT"]
  }
}

# SNS topic for bounce/complaint notifications
resource "aws_sns_topic" "ses_bounces" {
  name = "${var.project_name}-ses-bounces"

  tags = {
    Name = "${var.project_name}-ses-bounces"
  }
}

resource "aws_sns_topic_subscription" "ses_bounces_email" {
  topic_arn = aws_sns_topic.ses_bounces.arn
  protocol  = "email"
  endpoint  = var.notification_email
}

# Route bounce and complaint events to SNS
resource "aws_sesv2_configuration_set_event_destination" "bounces" {
  configuration_set_name = aws_sesv2_configuration_set.main.configuration_set_name
  event_destination_name = "bounces-complaints"

  event_destination {
    enabled              = true
    matching_event_types = ["BOUNCE", "COMPLAINT"]

    sns_destination {
      topic_arn = aws_sns_topic.ses_bounces.arn
    }
  }
}
