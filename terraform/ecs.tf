resource "aws_ecs_cluster" "main" {
  name = var.project_name

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = {
    Name = "${var.project_name}-cluster"
  }
}

resource "aws_ecs_task_definition" "ingestion" {
  family                   = "${var.project_name}-ingestion"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "ingestion"
      image     = "${aws_ecr_repository.ingestion.repository_url}:latest"
      essential = true

      command = ["ingest"]

      environment = [
        {
          name  = "AWS_REGION"
          value = var.aws_region
        },
        {
          name  = "SNS_TOPIC_ARN"
          value = aws_sns_topic.notifications.arn
        },
        {
          name  = "LOG_LEVEL"
          value = "info"
        }
      ]

      secrets = [
        {
          name      = "DATABASE_URL"
          valueFrom = aws_secretsmanager_secret.database_url.arn
        },
        {
          name      = "SAM_API_KEY"
          valueFrom = aws_secretsmanager_secret.sam_api_key.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ingestion.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name = "${var.project_name}-ingestion-task"
  }
}

resource "aws_ecs_task_definition" "csvarchive" {
  family                   = "${var.project_name}-csvarchive"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "csvarchive"
      image     = "${aws_ecr_repository.ingestion.repository_url}:latest"
      essential = true
      command   = ["archive-active"]

      environment = [
        {
          name  = "AWS_REGION"
          value = var.aws_region
        },
        {
          name  = "SNS_TOPIC_ARN"
          value = aws_sns_topic.notifications.arn
        },
        {
          name  = "LOG_LEVEL"
          value = "info"
        },
        {
          name  = "S3_BUCKET"
          value = aws_s3_bucket.data.id
        }
      ]

      secrets = [
        {
          name      = "DATABASE_URL"
          valueFrom = aws_secretsmanager_secret.database_url.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.csvarchive.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name = "${var.project_name}-csvarchive-task"
  }
}

resource "aws_ecs_task_definition" "archivedcsv" {
  family                   = "${var.project_name}-archivedcsv"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 256
  memory                   = 2048
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "archivedcsv"
      image     = "${aws_ecr_repository.ingestion.repository_url}:latest"
      essential = true
      command   = ["archive-historical"]

      environment = [
        {
          name  = "AWS_REGION"
          value = var.aws_region
        },
        {
          name  = "SNS_TOPIC_ARN"
          value = aws_sns_topic.notifications.arn
        },
        {
          name  = "LOG_LEVEL"
          value = "info"
        },
        {
          name  = "S3_BUCKET"
          value = aws_s3_bucket.data.id
        }
      ]

      secrets = [
        {
          name      = "DATABASE_URL"
          valueFrom = aws_secretsmanager_secret.database_url.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.archivedcsv.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name = "${var.project_name}-archivedcsv-task"
  }
}
