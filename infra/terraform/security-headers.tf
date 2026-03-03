# CloudFront response headers policies for security headers

resource "aws_cloudfront_response_headers_policy" "frontend" {
  name    = "${var.project_name}-frontend-security-headers"
  comment = "Security headers for app.govtrove.com"

  security_headers_config {
    strict_transport_security {
      access_control_max_age_sec = 31536000
      include_subdomains         = true
      override                   = true
    }

    content_type_options {
      override = true
    }

    frame_options {
      frame_option = "DENY"
      override     = true
    }

    content_security_policy {
      content_security_policy = "default-src 'self'; script-src 'self' https://cdn.counter.dev 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://api.govtrove.com https://*.workos.com https://*.sentry.io https://t.counter.dev; img-src 'self' data:; frame-src https://*.workos.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
      override               = true
    }
  }
}

resource "aws_cloudfront_response_headers_policy" "landing" {
  count   = var.domain_name != "" ? 1 : 0
  name    = "${var.project_name}-landing-security-headers"
  comment = "Security headers for govtrove.com"

  security_headers_config {
    strict_transport_security {
      access_control_max_age_sec = 31536000
      include_subdomains         = true
      override                   = true
    }

    content_type_options {
      override = true
    }

    frame_options {
      frame_option = "DENY"
      override     = true
    }

    content_security_policy {
      content_security_policy = "default-src 'self'; script-src 'self' https://unpkg.com https://cdn.counter.dev 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://api.govtrove.com https://t.counter.dev; img-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self' https://api.govtrove.com"
      override               = true
    }
  }
}
