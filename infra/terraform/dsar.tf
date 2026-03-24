resource "aws_s3_bucket" "dsar_exports" {
  bucket = "${var.project_name}-dsar-exports"

  tags = {
    Name        = "${var.project_name}-dsar-exports"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_public_access_block" "dsar_exports" {
  bucket = aws_s3_bucket.dsar_exports.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "dsar_exports" {
  bucket = aws_s3_bucket.dsar_exports.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "dsar_exports" {
  bucket = aws_s3_bucket.dsar_exports.id

  rule {
    id     = "expire-exports"
    status = "Enabled"

    filter {}

    expiration {
      days = 7
    }
  }
}

resource "aws_iam_role_policy" "apprunner_dsar_s3" {
  name = "${var.project_name}-apprunner-dsar-s3"
  role = aws_iam_role.apprunner_instance.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject"
        ]
        Resource = "${aws_s3_bucket.dsar_exports.arn}/*"
      }
    ]
  })
}
