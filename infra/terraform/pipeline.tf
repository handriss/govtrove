# =============================================================================
# Pipeline — Lambda Functions + Step Functions State Machine
# =============================================================================

locals {
  lambda_functions = ["download-csvs", "ingest-active", "ingest-archived", "reconcile", "generate-alerts"]
}

# --- Lambda Zip Archives ---

data "archive_file" "lambda" {
  for_each = toset(local.lambda_functions)

  type        = "zip"
  source_file = "${path.module}/../../bin/lambda/${each.key}/bootstrap"
  output_path = "${path.module}/../../bin/lambda/${each.key}.zip"
}

# --- Lambda Functions ---

resource "aws_lambda_function" "pipeline" {
  for_each = toset(local.lambda_functions)

  function_name    = "${var.project_name}-${each.key}"
  role             = aws_iam_role.lambda_pipeline.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  timeout          = var.lambda_timeout
  memory_size      = var.lambda_memory
  filename         = data.archive_file.lambda[each.key].output_path
  source_code_hash = data.archive_file.lambda[each.key].output_base64sha256

  environment {
    variables = {
      DATABASE_URL_SECRET_ARN = aws_secretsmanager_secret.database_url.arn
      S3_BUCKET               = aws_s3_bucket.data.id
      AWS_REGION_NAME         = var.aws_region
      SENTRY_DSN              = var.sentry_pipeline_dsn
    }
  }

  tags = {
    Name = "${var.project_name}-${each.key}"
  }
}

# --- Lambda IAM Role (shared) ---

resource "aws_iam_role" "lambda_pipeline" {
  name = "${var.project_name}-lambda-pipeline"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_pipeline_base" {
  name = "${var.project_name}-lambda-pipeline-base"
  role = aws_iam_role.lambda_pipeline.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Logs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      },
      {
        Sid      = "SecretsManager"
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = [aws_secretsmanager_secret.database_url.arn]
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_pipeline_download_csvs" {
  name = "${var.project_name}-lambda-download-csvs"
  role = aws_iam_role.lambda_pipeline.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "S3ReadWrite"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject"]
        Resource = ["${aws_s3_bucket.data.arn}/raw/*"]
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_pipeline_ingest" {
  name = "${var.project_name}-lambda-ingest"
  role = aws_iam_role.lambda_pipeline.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "S3Read"
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = ["${aws_s3_bucket.data.arn}/raw/*"]
      }
    ]
  })
}

# --- SQS DLQ ---

resource "aws_sqs_queue" "pipeline_dlq" {
  name                      = "${var.project_name}-pipeline-dlq"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name = "${var.project_name}-pipeline-dlq"
  }
}

# --- Step Functions State Machine ---

resource "aws_sfn_state_machine" "pipeline" {
  name     = "${var.project_name}-pipeline"
  role_arn = aws_iam_role.sfn_pipeline.arn

  definition = templatefile("${path.module}/step-functions.asl.json", {
    download_csvs_arn      = aws_lambda_function.pipeline["download-csvs"].arn
    ingest_active_arn      = aws_lambda_function.pipeline["ingest-active"].arn
    ingest_archived_arn    = aws_lambda_function.pipeline["ingest-archived"].arn
    reconcile_arn          = aws_lambda_function.pipeline["reconcile"].arn
    generate_alerts_arn    = aws_lambda_function.pipeline["generate-alerts"].arn
    sns_notifications_arn  = aws_sns_topic.notifications.arn
    sqs_dlq_url            = aws_sqs_queue.pipeline_dlq.url
  })

  tags = {
    Name = "${var.project_name}-pipeline"
  }
}

# --- Step Functions IAM Role ---

resource "aws_iam_role" "sfn_pipeline" {
  name = "${var.project_name}-sfn-pipeline"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "states.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "sfn_pipeline" {
  name = "${var.project_name}-sfn-pipeline"
  role = aws_iam_role.sfn_pipeline.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "InvokeLambda"
        Effect   = "Allow"
        Action   = ["lambda:InvokeFunction"]
        Resource = [for fn in local.lambda_functions : aws_lambda_function.pipeline[fn].arn]
      },
      {
        Sid      = "SNSPublish"
        Effect   = "Allow"
        Action   = ["sns:Publish"]
        Resource = [aws_sns_topic.notifications.arn]
      },
      {
        Sid      = "SQSSend"
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = [aws_sqs_queue.pipeline_dlq.arn]
      }
    ]
  })
}
