import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { PostHogProvider } from '@posthog/react'
import type { PostHogInterface } from 'posthog-js'
import './index.css'
import App from './App'
import { capturedUTM } from './hooks/useUTMCapture'

const posthogKey = import.meta.env.VITE_POSTHOG_KEY
const posthogOptions = {
  api_host: 'https://k.govtrove.com',
  person_profiles: 'identified_only' as const,
  capture_pageview: true,
  capture_pageleave: true,
  autocapture: false,
  disable_toolbar: true,
  persistence: 'memory' as const,
  loaded: (ph: PostHogInterface) => {
    if (window.location.pathname.startsWith('/admin')) {
      ph.opt_out_capturing();
      return;
    }
    const props: Record<string, string> = {}
    if (capturedUTM.source) props.utm_source = capturedUTM.source
    if (capturedUTM.medium) props.utm_medium = capturedUTM.medium
    if (capturedUTM.campaign) props.utm_campaign = capturedUTM.campaign
    if (Object.keys(props).length > 0) ph.register(props)
  },
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {posthogKey ? (
      <PostHogProvider apiKey={posthogKey} options={posthogOptions}>
        <App />
      </PostHogProvider>
    ) : (
      <App />
    )}
  </StrictMode>,
)

import('./sentry').catch(() => {})
