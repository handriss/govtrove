# ECR Repository for MCP server
resource "aws_ecr_repository" "mcp" {
  name                 = "${var.project_name}-mcp"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name = "${var.project_name}-mcp"
  }
}

resource "aws_ecr_lifecycle_policy" "mcp" {
  repository = aws_ecr_repository.mcp.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep last 10 images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = 10
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}

# IAM role for MCP App Runner instance (runtime)
resource "aws_iam_role" "mcp_instance" {
  name = "${var.project_name}-mcp-instance"

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

# MCP instance only needs to read the WorkOS client ID secret
resource "aws_iam_role_policy" "mcp_secrets" {
  name = "${var.project_name}-mcp-secrets"
  role = aws_iam_role.mcp_instance.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          aws_secretsmanager_secret.workos_client_id.arn,
          aws_secretsmanager_secret.database_url.arn,
        ]
      }
    ]
  })
}

# Auto-scaling for MCP service
resource "aws_apprunner_auto_scaling_configuration_version" "mcp" {
  auto_scaling_configuration_name = "${var.project_name}-mcp"

  min_size = 1
  max_size = 2

  tags = {
    Name = "${var.project_name}-mcp-autoscaling"
  }
}

# App Runner service for MCP
resource "aws_apprunner_service" "mcp" {
  service_name = "${var.project_name}-mcp"

  source_configuration {
    authentication_configuration {
      access_role_arn = aws_iam_role.apprunner_ecr_access.arn
    }

    image_repository {
      image_identifier      = "${aws_ecr_repository.mcp.repository_url}:latest"
      image_repository_type = "ECR"

      image_configuration {
        port = "3000"

        runtime_environment_secrets = {
          WORKOS_CLIENT_ID = aws_secretsmanager_secret.workos_client_id.arn
          DATABASE_URL     = aws_secretsmanager_secret.database_url.arn
        }

        runtime_environment_variables = {
          PORT             = "3000"
          NODE_ENV         = "production"
          MCP_RESOURCE_URL = "https://mcp.${var.domain_name}"
          AUTHKIT_DOMAIN   = var.authkit_domain
        }
      }
    }

    auto_deployments_enabled = false
  }

  instance_configuration {
    cpu               = "256"
    memory            = "512"
    instance_role_arn = aws_iam_role.mcp_instance.arn
  }

  auto_scaling_configuration_arn = aws_apprunner_auto_scaling_configuration_version.mcp.arn

  health_check_configuration {
    protocol            = "HTTP"
    path                = "/health"
    interval            = 10
    timeout             = 5
    healthy_threshold   = 1
    unhealthy_threshold = 5
  }

  tags = {
    Name        = "${var.project_name}-mcp"
    Environment = var.environment
  }
}

# Custom domain for mcp.<domain>
resource "aws_apprunner_custom_domain_association" "mcp" {
  count = var.domain_name != "" ? 1 : 0

  domain_name = "mcp.${var.domain_name}"
  service_arn = aws_apprunner_service.mcp.arn
}
