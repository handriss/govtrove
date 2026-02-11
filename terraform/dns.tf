# ACM certificate for custom domain (govtrove.com + *.govtrove.com)
resource "aws_acm_certificate" "main" {
  count = var.domain_name != "" ? 1 : 0

  domain_name               = var.domain_name
  subject_alternative_names = ["*.${var.domain_name}"]
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name        = "${var.project_name}-cert"
    Environment = var.environment
  }
}

# Gate that waits for ACM cert to be validated (DNS records must exist in Cloudflare first)
resource "aws_acm_certificate_validation" "main" {
  count = var.domain_name != "" ? 1 : 0

  certificate_arn = aws_acm_certificate.main[0].arn

  # Terraform will poll until the cert is validated.
  # You must add the DNS CNAME records in Cloudflare before this will succeed.
  timeouts {
    create = "30m"
  }
}

# App Runner custom domain association for api.<domain>
resource "aws_apprunner_custom_domain_association" "api" {
  count = var.domain_name != "" ? 1 : 0

  domain_name = "api.${var.domain_name}"
  service_arn = aws_apprunner_service.api.arn
}
