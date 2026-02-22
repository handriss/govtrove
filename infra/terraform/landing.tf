# S3 bucket for landing page
resource "aws_s3_bucket" "landing" {
  count  = var.domain_name != "" ? 1 : 0
  bucket = "${var.project_name}-landing-${var.environment}"

  tags = {
    Name        = "${var.project_name}-landing"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_public_access_block" "landing" {
  count  = var.domain_name != "" ? 1 : 0
  bucket = aws_s3_bucket.landing[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# CloudFront Origin Access Control
resource "aws_cloudfront_origin_access_control" "landing" {
  count                             = var.domain_name != "" ? 1 : 0
  name                              = "${var.project_name}-landing-oac"
  description                       = "OAC for ${var.project_name} landing page"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# CloudFront Function for subdirectory index.html rewriting
resource "aws_cloudfront_function" "landing_url_rewrite" {
  count   = var.domain_name != "" ? 1 : 0
  name    = "${var.project_name}-landing-url-rewrite"
  runtime = "cloudfront-js-2.0"
  comment = "Rewrite directory paths to index.html for landing page"
  publish = true
  code    = file("${path.module}/../cf-functions/landing-url-rewrite.js")
}

# CloudFront distribution for landing page at govtrove.com
resource "aws_cloudfront_distribution" "landing" {
  count               = var.domain_name != "" ? 1 : 0
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  comment             = "${var.project_name} landing page"
  price_class         = "PriceClass_100"
  aliases             = [var.domain_name]

  origin {
    domain_name              = aws_s3_bucket.landing[0].bucket_regional_domain_name
    origin_id                = "S3-${aws_s3_bucket.landing[0].id}"
    origin_access_control_id = aws_cloudfront_origin_access_control.landing[0].id
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "S3-${aws_s3_bucket.landing[0].id}"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    min_ttl     = 0
    default_ttl = 3600
    max_ttl     = 86400

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.landing_url_rewrite[0].arn
    }
  }

  logging_config {
    bucket          = aws_s3_bucket.cloudfront_logs.bucket_domain_name
    prefix          = "landing/"
    include_cookies = false
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.main[0].certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = {
    Name        = "${var.project_name}-landing"
    Environment = var.environment
  }
}

# S3 bucket policy allowing CloudFront access
resource "aws_s3_bucket_policy" "landing" {
  count  = var.domain_name != "" ? 1 : 0
  bucket = aws_s3_bucket.landing[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowCloudFrontServicePrincipal"
        Effect = "Allow"
        Principal = {
          Service = "cloudfront.amazonaws.com"
        }
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.landing[0].arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.landing[0].arn
          }
        }
      }
    ]
  })
}
