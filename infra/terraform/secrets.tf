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

resource "aws_secretsmanager_secret" "stripe_price_monthly" {
  count       = var.stripe_price_monthly != "" ? 1 : 0
  name        = "${var.project_name}/stripe-price-monthly"
  description = "Stripe Price ID for the monthly Pro plan"
}

resource "aws_secretsmanager_secret_version" "stripe_price_monthly" {
  count         = var.stripe_price_monthly != "" ? 1 : 0
  secret_id     = aws_secretsmanager_secret.stripe_price_monthly[0].id
  secret_string = var.stripe_price_monthly
}

resource "aws_secretsmanager_secret" "stripe_promo_coupon_id" {
  count       = var.stripe_promo_coupon_id != "" ? 1 : 0
  name        = "${var.project_name}/stripe-promo-coupon-id"
  description = "Stripe Coupon ID for promo code generation"
}

resource "aws_secretsmanager_secret_version" "stripe_promo_coupon_id" {
  count         = var.stripe_promo_coupon_id != "" ? 1 : 0
  secret_id     = aws_secretsmanager_secret.stripe_promo_coupon_id[0].id
  secret_string = var.stripe_promo_coupon_id
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

# Shared token for API -> MCP internal calls (/embed). Generated here rather than
# supplied via tfvars so it never lands in a file on disk.
resource "random_password" "internal_api_token" {
  length  = 48
  special = false
}

resource "aws_secretsmanager_secret" "internal_api_token" {
  name        = "${var.project_name}/internal-api-token"
  description = "Shared bearer token for API -> MCP service-to-service calls"
}

resource "aws_secretsmanager_secret_version" "internal_api_token" {
  secret_id     = aws_secretsmanager_secret.internal_api_token.id
  secret_string = random_password.internal_api_token.result
}

# Shared secret between the Cloudflare Transform Rule and the API's origin-verify
# middleware. Empty by default: the API only enforces once this is set, so the
# Cloudflare rule can be created and confirmed first without risking an outage.
resource "aws_secretsmanager_secret" "origin_verify" {
  count       = var.origin_verify_secret != "" ? 1 : 0
  name        = "${var.project_name}/origin-verify-secret"
  description = "Value Cloudflare injects as X-Origin-Verify on proxied requests"
}

resource "aws_secretsmanager_secret_version" "origin_verify" {
  count         = var.origin_verify_secret != "" ? 1 : 0
  secret_id     = aws_secretsmanager_secret.origin_verify[0].id
  secret_string = var.origin_verify_secret
}
