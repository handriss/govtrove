import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import * as Sentry from '@sentry/react'
import './index.css'
import App from './App'

Sentry.init({
  dsn: import.meta.env.VITE_SENTRY_DSN || '',
  environment: import.meta.env.PROD ? 'production' : 'development',
  enabled: !!import.meta.env.VITE_SENTRY_DSN,
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

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
