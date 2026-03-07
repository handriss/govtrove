# CloudFront response headers policies for security headers

resource "aws_cloudfront_response_headers_policy" "frontend" {
  name    = "${var.project_name}-frontend-security-headers"
  comment = "Security headers for app.govtrove.com"

  security_headers_config {
    strict_transport_security {
      access_control_max_age_sec = 31536000
      include_subdomains         = true
      preload                    = true
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
      content_security_policy = "default-src 'self'; script-src 'self' https://cdn.counter.dev https://*.tawk.to https://*.i.posthog.com 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://*.tawk.to; font-src 'self' https://fonts.gstatic.com https://*.tawk.to; connect-src 'self' https://api.govtrove.com https://*.workos.com https://*.sentry.io https://t.counter.dev https://*.tawk.to wss://*.tawk.to https://*.i.posthog.com; img-src 'self' data: https://*.tawk.to; frame-src https://*.workos.com https://*.tawk.to; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
      override               = true
    }
  }

  custom_headers_config {
    items {
      header   = "Cross-Origin-Opener-Policy"
      value    = "same-origin"
      override = true
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
      preload                    = true
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
      content_security_policy = "default-src 'self'; script-src 'self' https://cdn.counter.dev https://*.tawk.to 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://*.tawk.to; font-src 'self' https://fonts.gstatic.com https://*.tawk.to; connect-src 'self' https://api.govtrove.com https://t.counter.dev https://*.tawk.to wss://*.tawk.to; img-src 'self' data: https://*.tawk.to; frame-src https://*.tawk.to; frame-ancestors 'none'; base-uri 'self'; form-action 'self' https://api.govtrove.com"
      override               = true
    }
  }

  custom_headers_config {
    items {
      header   = "Cross-Origin-Opener-Policy"
      value    = "same-origin"
      override = true
    }
  }
}
