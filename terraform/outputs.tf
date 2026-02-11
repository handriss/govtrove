output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "ecr_repository_url" {
  description = "ECR repository URL for ingestion service"
  value       = aws_ecr_repository.ingestion.repository_url
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "ecs_task_definition_arn" {
  description = "ECS task definition ARN"
  value       = aws_ecs_task_definition.ingestion.arn
}

output "cloudwatch_log_group" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.ingestion.name
}

output "sns_topic_arn" {
  description = "SNS topic ARN for notifications"
  value       = aws_sns_topic.notifications.arn
}

output "public_subnet_ids" {
  description = "Public subnet IDs (for running tasks)"
  value       = aws_subnet.public[*].id
}

output "security_group_id" {
  description = "Security group ID for ECS tasks"
  value       = aws_security_group.ecs_tasks.id
}

# API outputs
output "ecr_api_repository_url" {
  description = "ECR repository URL for API service"
  value       = aws_ecr_repository.api.repository_url
}

output "apprunner_service_url" {
  description = "App Runner service URL for API"
  value       = aws_apprunner_service.api.service_url
}

output "apprunner_service_arn" {
  description = "App Runner service ARN (for deployments)"
  value       = aws_apprunner_service.api.arn
}

# Frontend outputs
output "frontend_bucket_name" {
  description = "S3 bucket name for frontend static files"
  value       = aws_s3_bucket.frontend.id
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID"
  value       = aws_cloudfront_distribution.frontend.id
}

output "cloudfront_distribution_url" {
  description = "CloudFront distribution URL for frontend"
  value       = "https://${aws_cloudfront_distribution.frontend.domain_name}"
}

# Custom domain outputs
output "frontend_url" {
  description = "Frontend URL (custom domain or CloudFront)"
  value       = var.domain_name != "" ? "https://app.${var.domain_name}" : "https://${aws_cloudfront_distribution.frontend.domain_name}"
}

output "api_url" {
  description = "API URL (custom domain or App Runner)"
  value       = var.domain_name != "" ? "https://api.${var.domain_name}" : "https://${aws_apprunner_service.api.service_url}"
}

output "acm_validation_records" {
  description = "CNAME records to add in Cloudflare for ACM certificate validation"
  value = var.domain_name != "" ? {
    for dvo in aws_acm_certificate.main[0].domain_validation_options : dvo.domain_name => {
      type  = "CNAME"
      name  = dvo.resource_record_name
      value = dvo.resource_record_value
      note  = "Add in Cloudflare with proxy OFF (DNS only)"
    }
  } : {}
}

output "apprunner_custom_domain_records" {
  description = "CNAME records to add in Cloudflare for App Runner custom domain validation"
  value = var.domain_name != "" ? {
    for record in aws_apprunner_custom_domain_association.api[0].certificate_validation_records : record.name => {
      type  = "CNAME"
      name  = record.name
      value = record.value
      note  = "Add in Cloudflare with proxy OFF (DNS only)"
    }
  } : {}
}

output "apprunner_custom_domain_target" {
  description = "CNAME target for api.<domain> — add this in Cloudflare after validation"
  value       = var.domain_name != "" ? aws_apprunner_custom_domain_association.api[0].dns_target : ""
}

output "cloudfront_domain_name" {
  description = "CNAME target for app.<domain> — add this in Cloudflare after validation"
  value       = aws_cloudfront_distribution.frontend.domain_name
}

# Landing page outputs
output "landing_bucket_name" {
  description = "S3 bucket name for landing page"
  value       = var.domain_name != "" ? aws_s3_bucket.landing[0].id : ""
}

output "landing_distribution_id" {
  description = "CloudFront distribution ID for landing page"
  value       = var.domain_name != "" ? aws_cloudfront_distribution.landing[0].id : ""
}

output "landing_domain_name" {
  description = "CNAME target for <domain> — the landing page CloudFront distribution"
  value       = var.domain_name != "" ? aws_cloudfront_distribution.landing[0].domain_name : ""
}
