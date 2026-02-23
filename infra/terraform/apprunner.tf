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
        Resource = [
          aws_secretsmanager_secret.database_url.arn,
          aws_secretsmanager_secret.workos_client_id.arn,
          aws_secretsmanager_secret.workos_api_key.arn,
        ]
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

        runtime_environment_secrets = {
          DATABASE_URL     = aws_secretsmanager_secret.database_url.arn
          WORKOS_CLIENT_ID = aws_secretsmanager_secret.workos_client_id.arn
          WORKOS_API_KEY   = aws_secretsmanager_secret.workos_api_key.arn
        }

        runtime_environment_variables = {
          PORT            = tostring(var.api_port)
          LOG_LEVEL       = "info"
          ALLOWED_ORIGINS = var.domain_name != "" ? "https://app.${var.domain_name},https://${var.domain_name}" : "https://${aws_cloudfront_distribution.frontend.domain_name}"
          SNS_TOPIC_ARN   = aws_sns_topic.notifications.arn
          AWS_REGION      = var.aws_region
          ADMIN_EMAILS    = var.admin_emails
          SENTRY_DSN      = var.sentry_dsn
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
