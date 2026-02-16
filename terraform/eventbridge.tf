resource "aws_scheduler_schedule" "ingestion" {
  name       = "${var.project_name}-ingestion-schedule"
  group_name = "default"
  state      = "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression          = var.schedule_expression
  schedule_expression_timezone = "America/New_York"

  target {
    arn      = aws_ecs_cluster.main.arn
    role_arn = aws_iam_role.eventbridge_scheduler.arn

    ecs_parameters {
      task_definition_arn = aws_ecs_task_definition.ingestion.arn_without_revision
      launch_type         = "FARGATE"
      task_count          = 1

      network_configuration {
        subnets          = aws_subnet.public[*].id
        security_groups  = [aws_security_group.ecs_tasks.id]
        assign_public_ip = true
      }
    }
  }
}

resource "aws_scheduler_schedule" "csvarchive" {
  name       = "${var.project_name}-csvarchive-schedule"
  group_name = "default"

  # TEMPORARY: Every 15 min to detect SAM.gov CSV refresh timing.
  # After ~2 weeks of data, change to once/twice daily.
  state = var.csvarchive_schedule_enabled ? "ENABLED" : "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression          = var.csvarchive_schedule_expression
  schedule_expression_timezone = "UTC"

  target {
    arn      = aws_ecs_cluster.main.arn
    role_arn = aws_iam_role.eventbridge_scheduler.arn

    ecs_parameters {
      task_definition_arn = aws_ecs_task_definition.csvarchive.arn_without_revision
      launch_type         = "FARGATE"
      task_count          = 1

      network_configuration {
        subnets          = aws_subnet.public[*].id
        security_groups  = [aws_security_group.ecs_tasks.id]
        assign_public_ip = true
      }
    }
  }
}

resource "aws_scheduler_schedule" "archivedcsv" {
  name       = "${var.project_name}-archivedcsv-schedule"
  group_name = "default"

  state = var.archivedcsv_schedule_enabled ? "ENABLED" : "DISABLED"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression          = var.archivedcsv_schedule_expression
  schedule_expression_timezone = "UTC"

  target {
    arn      = aws_ecs_cluster.main.arn
    role_arn = aws_iam_role.eventbridge_scheduler.arn

    ecs_parameters {
      task_definition_arn = aws_ecs_task_definition.archivedcsv.arn_without_revision
      launch_type         = "FARGATE"
      task_count          = 1

      network_configuration {
        subnets          = aws_subnet.public[*].id
        security_groups  = [aws_security_group.ecs_tasks.id]
        assign_public_ip = true
      }
    }
  }
}
