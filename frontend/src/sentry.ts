import * as Sentry from '@sentry/react'

Sentry.init({
  dsn: import.meta.env.VITE_SENTRY_DSN || '',
  environment: import.meta.env.PROD ? 'production' : 'development',
  enabled: !!import.meta.env.VITE_SENTRY_DSN,
  // Drop noise from the Tawk.to live-chat widget: it's third-party code we don't own,
  // affects 0 users, and its events can carry chat-user PII (name/email) we don't want in Sentry.
  denyUrls: [/tawk\.to/i],
  ignoreErrors: [
    /Tawk/i,
    /i18next is not a function/i,
    /Socket server did not execute the callback/i,
  ],
  beforeBreadcrumb(breadcrumb) {
    if (breadcrumb.category === 'fetch' || breadcrumb.category === 'xhr') {
      const headers = breadcrumb.data?.headers;
      if (headers) {
        delete headers['Authorization'];
        delete headers['authorization'];
      }
    }
    return breadcrumb;
  },
})
