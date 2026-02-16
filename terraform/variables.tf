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

# ECS Task
variable "task_cpu" {
  description = "CPU units for the Fargate task (256 = 0.25 vCPU)"
  type        = number
  default     = 256
}

variable "task_memory" {
  description = "Memory for the Fargate task in MB"
  type        = number
  default     = 1024
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

# CSV Archive Schedule
variable "csvarchive_schedule_expression" {
  description = "EventBridge schedule expression for CSV archive"
  type        = string
  default     = "cron(0/15 * * * ? *)"
}

variable "csvarchive_schedule_enabled" {
  description = "Whether the CSV archive schedule is enabled"
  type        = bool
  default     = true
}

# Archived CSV Schedule
variable "archivedcsv_schedule_expression" {
  description = "EventBridge schedule expression for archived CSV download"
  type        = string
  default     = "cron(0 0,6,12,18 * * ? *)"
}

variable "archivedcsv_schedule_enabled" {
  description = "Whether the archived CSV schedule is enabled"
  type        = bool
  default     = true
}

# API
variable "api_port" {
  description = "Port the API service listens on"
  type        = number
  default     = 8080
}

# Domain (optional, for future custom domain support)
variable "domain_name" {
  description = "Custom domain name for the application (optional)"
  type        = string
  default     = ""
}
