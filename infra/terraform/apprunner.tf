# IAM role for App Runner to access ECR
resource "aws_iam_role" "apprunner_ecr_access" {
  name = "${var.project_name}-apprunner-ecr-access"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "build.apprunner.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "apprunner_ecr_access" {
  role       = aws_iam_role.apprunner_ecr_access.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSAppRunnerServicePolicyForECRAccess"
}

# IAM role for App Runner instance (runtime)
resource "aws_iam_role" "apprunner_instance" {
  name = "${var.project_name}-apprunner-instance"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "tasks.apprunner.amazonaws.com"
        }
      }
    ]
  })
}

# Allow App Runner instance to read secrets
resource "aws_iam_role_policy" "apprunner_secrets" {
  name = "${var.project_name}-apprunner-secrets"
  role = aws_iam_role.apprunner_instance.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = concat([
          aws_secretsmanager_secret.database_url.arn,
          aws_secretsmanager_secret.workos_client_id.arn,
          aws_secretsmanager_secret.workos_api_key.arn,
          aws_secretsmanager_secret.resend_api_key.arn,
          aws_secretsmanager_secret.resend_webhook_secret.arn,
          aws_secretsmanager_secret.internal_api_token.arn,
          ],
          var.stripe_secret_key != "" ? [aws_secretsmanager_secret.stripe_secret_key[0].arn] : [],
          var.stripe_webhook_secret != "" ? [aws_secretsmanager_secret.stripe_webhook_secret[0].arn] : [],
          var.stripe_price_monthly != "" ? [aws_secretsmanager_secret.stripe_price_monthly[0].arn] : [],
          var.stripe_promo_coupon_id != "" ? [aws_secretsmanager_secret.stripe_promo_coupon_id[0].arn] : [],
          var.posthog_key != "" ? [aws_secretsmanager_secret.posthog_key[0].arn] : [],
          var.origin_verify_secret != "" ? [aws_secretsmanager_secret.origin_verify[0].arn] : [],
        )
      }
    ]
  })
}

# Allow App Runner instance to send SES emails
resource "aws_iam_role_policy" "apprunner_ses" {
  name = "${var.project_name}-apprunner-ses"
  role = aws_iam_role.apprunner_instance.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "sesv2:SendEmail"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "ses:FromAddress" = "noreply@${var.domain_name}"
          }
        }
      }
    ]
  })
}

# Allow App Runner instance to publish SNS notifications
resource "aws_iam_role_policy" "apprunner_sns" {
  name = "${var.project_name}-apprunner-sns"
  role = aws_iam_role.apprunner_instance.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sns:Publish"
        ]
        Resource = [
          aws_sns_topic.notifications.arn
        ]
      }
    ]
  })
}

# App Runner auto-scaling configuration
resource "aws_apprunner_auto_scaling_configuration_version" "api" {
  auto_scaling_configuration_name = "${var.project_name}-api"

  min_size = 1
  max_size = 2

  tags = {
    Name = "${var.project_name}-api-autoscaling"
  }
}

# App Runner service
resource "aws_apprunner_service" "api" {
  service_name = "${var.project_name}-api"

  source_configuration {
    authentication_configuration {
      access_role_arn = aws_iam_role.apprunner_ecr_access.arn
    }

    image_repository {
      image_identifier      = "${aws_ecr_repository.api.repository_url}:latest"
      image_repository_type = "ECR"

      image_configuration {
        port = tostring(var.api_port)

        runtime_environment_secrets = merge({
          DATABASE_URL          = aws_secretsmanager_secret.database_url.arn
          WORKOS_CLIENT_ID      = aws_secretsmanager_secret.workos_client_id.arn
          WORKOS_API_KEY        = aws_secretsmanager_secret.workos_api_key.arn
          RESEND_API_KEY        = aws_secretsmanager_secret.resend_api_key.arn
          RESEND_WEBHOOK_SECRET = aws_secretsmanager_secret.resend_webhook_secret.arn
          INTERNAL_API_TOKEN    = aws_secretsmanager_secret.internal_api_token.arn
          },
          var.stripe_secret_key != "" ? { STRIPE_SECRET_KEY = aws_secretsmanager_secret.stripe_secret_key[0].arn } : {},
          var.stripe_webhook_secret != "" ? { STRIPE_WEBHOOK_SECRET = aws_secretsmanager_secret.stripe_webhook_secret[0].arn } : {},
          var.stripe_price_monthly != "" ? { STRIPE_PRICE_MONTHLY = aws_secretsmanager_secret.stripe_price_monthly[0].arn } : {},
          var.stripe_promo_coupon_id != "" ? { STRIPE_PROMO_COUPON_ID = aws_secretsmanager_secret.stripe_promo_coupon_id[0].arn } : {},
          var.posthog_key != "" ? { POSTHOG_KEY = aws_secretsmanager_secret.posthog_key[0].arn } : {},
          var.origin_verify_secret != "" ? { ORIGIN_VERIFY_SECRET = aws_secretsmanager_secret.origin_verify[0].arn } : {},
        )

        runtime_environment_variables = {
          PORT              = tostring(var.api_port)
          LOG_LEVEL         = "info"
          ALLOWED_ORIGINS   = var.domain_name != "" ? "https://app.${var.domain_name},https://${var.domain_name}" : "https://${aws_cloudfront_distribution.frontend.domain_name}"
          SNS_TOPIC_ARN     = aws_sns_topic.notifications.arn
          AWS_REGION        = var.aws_region
          ADMIN_EMAILS      = var.admin_emails
          SENTRY_DSN        = var.sentry_dsn
          RESEND_FROM_EMAIL = "GovTrove <notifications@govtrove.com>"
          SES_FROM_EMAIL    = var.domain_name != "" ? "noreply@${var.domain_name}" : ""
          SES_CONFIG_SET    = aws_sesv2_configuration_set.main.configuration_set_name
          POSTHOG_HOST      = var.posthog_host
          MCP_INTERNAL_URL  = "https://${aws_apprunner_service.mcp.service_url}"
          DSAR_S3_BUCKET    = aws_s3_bucket.dsar_exports.bucket
          # Deterministic tiers only. The LLM tier stays dark until
          # OPENROUTER_API_KEY is set, which also needs the OpenRouter/Anthropic
          # subprocessor disclosed in the privacy policy first.
          SEARCH_RESCUE_ENABLED = "true"
        }
      }
    }

    auto_deployments_enabled = false
  }

  instance_configuration {
    cpu               = "256"
    memory            = "512"
    instance_role_arn = aws_iam_role.apprunner_instance.arn
  }

  auto_scaling_configuration_arn = aws_apprunner_auto_scaling_configuration_version.api.arn

  health_check_configuration {
    protocol            = "HTTP"
    path                = "/health"
    interval            = 10
    timeout             = 5
    healthy_threshold   = 1
    unhealthy_threshold = 5
  }

  tags = {
    Name        = "${var.project_name}-api"
    Environment = var.environment
  }
}
