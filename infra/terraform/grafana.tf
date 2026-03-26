resource "grafana_data_source" "neon" {
  count = var.grafana_stack_slug != "" ? 1 : 0

  type = "postgres"
  name = "Neon PostgreSQL"

  url             = "${var.grafana_neon_host}:5432"
  username        = var.grafana_neon_user

  json_data_encoded = jsonencode({
    database        = "neondb"
    sslmode         = "require"
    timescaledb     = false
    postgresVersion = 1500
  })

  secure_json_data_encoded = jsonencode({
    password = var.grafana_neon_password
  })
}

resource "grafana_dashboard" "business_overview" {
  count = var.grafana_stack_slug != "" ? 1 : 0

  config_json = templatefile("${path.module}/grafana-dashboards/business-overview.json", {
    datasource_uid = grafana_data_source.neon[0].uid
  })
}
