variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "govtrove"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "prod"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "govtrove"
}

# Networking
variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "Availability zones to use"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

# Secrets (sensitive - provide via tfvars or environment)
variable "database_url" {
  description = "Neon PostgreSQL connection string"
  type        = string
  sensitive   = true
}

variable "sam_api_key" {
  description = "SAM.gov API key"
  type        = string
  sensitive   = true
}

# WorkOS Auth
variable "workos_client_id" {
  description = "WorkOS AuthKit client ID"
  type        = string
  sensitive   = true
}

variable "workos_api_key" {
  description = "WorkOS API key"
  type        = string
  sensitive   = true
}

# Notifications
variable "notification_email" {
  description = "Email address for ingestion notifications"
  type        = string
}

# Schedule
variable "schedule_expression" {
  description = "EventBridge schedule expression for ingestion"
  type        = string
  default     = "cron(0 11 * * ? *)" # 6 AM ET = 11 AM UTC
}

variable "bulkcsv_schedule_expression" {
  description = "EventBridge schedule expression for pipeline (active + archived)"
  type        = string
  default     = "cron(0/15 * * * ? *)"
}

variable "pipeline_schedule_enabled" {
  description = "Whether the pipeline schedule is enabled"
  type        = bool
  default     = true
}

# Lambda
variable "lambda_memory" {
  description = "Memory for pipeline Lambda functions in MB"
  type        = number
  default     = 1024
}

variable "lambda_timeout" {
  description = "Timeout for pipeline Lambda functions in seconds"
  type        = number
  default     = 900
}

# Admin
variable "admin_emails" {
  description = "Comma-separated list of admin email addresses"
  type        = string
  default     = ""
}

# API
variable "api_port" {
  description = "Port the API service listens on"
  type        = number
  default     = 8080
}

# Sentry
variable "sentry_dsn" {
  description = "Sentry DSN for API error monitoring (optional)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "sentry_pipeline_dsn" {
  description = "Sentry DSN for pipeline Lambda error monitoring (optional)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "sentry_frontend_dsn" {
  description = "Sentry DSN for frontend error monitoring (optional, not sensitive — baked into JS bundle)"
  type        = string
  default     = ""
}

# Email (Resend)
variable "resend_api_key" {
  description = "Resend API key for transactional emails"
  type        = string
  sensitive   = true
}

variable "resend_webhook_secret" {
  description = "Resend webhook secret (also used for unsubscribe HMAC)"
  type        = string
  sensitive   = true
}

# Domain (optional, for future custom domain support)
variable "domain_name" {
  description = "Custom domain name for the application (optional)"
  type        = string
  default     = ""
}
