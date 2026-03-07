resource "aws_secretsmanager_secret" "database_url" {
  name        = "${var.project_name}/database-url"
  description = "Neon PostgreSQL connection string"
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id     = aws_secretsmanager_secret.database_url.id
  secret_string = var.database_url
}

resource "aws_secretsmanager_secret" "sam_api_key" {
  name        = "${var.project_name}/sam-api-key"
  description = "SAM.gov API key"
}

resource "aws_secretsmanager_secret_version" "sam_api_key" {
  secret_id     = aws_secretsmanager_secret.sam_api_key.id
  secret_string = var.sam_api_key
}

resource "aws_secretsmanager_secret" "workos_client_id" {
  name        = "${var.project_name}/workos-client-id"
  description = "WorkOS AuthKit client ID"
}

resource "aws_secretsmanager_secret_version" "workos_client_id" {
  secret_id     = aws_secretsmanager_secret.workos_client_id.id
  secret_string = var.workos_client_id
}

resource "aws_secretsmanager_secret" "workos_api_key" {
  name        = "${var.project_name}/workos-api-key"
  description = "WorkOS API key"
}

resource "aws_secretsmanager_secret_version" "workos_api_key" {
  secret_id     = aws_secretsmanager_secret.workos_api_key.id
  secret_string = var.workos_api_key
}

resource "aws_secretsmanager_secret" "resend_api_key" {
  name        = "${var.project_name}/resend-api-key"
  description = "Resend API key for transactional emails"
}

resource "aws_secretsmanager_secret_version" "resend_api_key" {
  secret_id     = aws_secretsmanager_secret.resend_api_key.id
  secret_string = var.resend_api_key
}

resource "aws_secretsmanager_secret" "resend_webhook_secret" {
  name        = "${var.project_name}/resend-webhook-secret"
  description = "Resend webhook secret (also used for unsubscribe HMAC)"
}

resource "aws_secretsmanager_secret_version" "resend_webhook_secret" {
  secret_id     = aws_secretsmanager_secret.resend_webhook_secret.id
  secret_string = var.resend_webhook_secret
}

resource "aws_secretsmanager_secret" "stripe_secret_key" {
  count       = var.stripe_secret_key != "" ? 1 : 0
  name        = "${var.project_name}/stripe-secret-key"
  description = "Stripe secret API key"
}

resource "aws_secretsmanager_secret_version" "stripe_secret_key" {
  count         = var.stripe_secret_key != "" ? 1 : 0
  secret_id     = aws_secretsmanager_secret.stripe_secret_key[0].id
  secret_string = var.stripe_secret_key
}

resource "aws_secretsmanager_secret" "stripe_webhook_secret" {
  count       = var.stripe_webhook_secret != "" ? 1 : 0
  name        = "${var.project_name}/stripe-webhook-secret"
  description = "Stripe webhook endpoint signing secret"
}

resource "aws_secretsmanager_secret_version" "stripe_webhook_secret" {
  count         = var.stripe_webhook_secret != "" ? 1 : 0
  secret_id     = aws_secretsmanager_secret.stripe_webhook_secret[0].id
  secret_string = var.stripe_webhook_secret
}

resource "aws_secretsmanager_secret" "posthog_key" {
  count       = var.posthog_key != "" ? 1 : 0
  name        = "${var.project_name}/posthog-key"
  description = "PostHog project API key"
}

resource "aws_secretsmanager_secret_version" "posthog_key" {
  count         = var.posthog_key != "" ? 1 : 0
  secret_id     = aws_secretsmanager_secret.posthog_key[0].id
  secret_string = var.posthog_key
}
