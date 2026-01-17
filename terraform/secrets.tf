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
