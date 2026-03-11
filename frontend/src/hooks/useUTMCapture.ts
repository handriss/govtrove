import { useEffect } from 'react';

const API_BASE = import.meta.env.VITE_API_URL || '/api';

// Capture UTM params at module load time — before any React effects can
// strip them from the URL (useFilterState in SimpleSearchPage fires before
// parent effects and cleans unrecognized query params).
// PostHog super properties are registered in the `loaded` callback in main.tsx
// so they attach to the very first $pageview.
const initialParams = new URLSearchParams(window.location.search);
const capturedSource = initialParams.get('utm_source');
const capturedMedium = initialParams.get('utm_medium');
const capturedCampaign = initialParams.get('utm_campaign');

export const capturedUTM = {
  source: capturedSource,
  medium: capturedMedium,
  campaign: capturedCampaign,
};

export default function useUTMCapture() {
  useEffect(() => {
    if (!capturedSource && !capturedMedium && !capturedCampaign) return;

    if (capturedCampaign) {
      sessionStorage.setItem('govtrove_utm_campaign', capturedCampaign);
    }

    if (!sessionStorage.getItem('govtrove_utm_tracked')) {
      sessionStorage.setItem('govtrove_utm_tracked', '1');
      const payload = {
        utm_source: capturedSource,
        utm_medium: capturedMedium,
        utm_campaign: capturedCampaign,
        landing_page: window.location.pathname,
        origin: 'app',
        referrer: document.referrer || null,
      };
      fetch(`${API_BASE}/utm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }).catch(() => {});
    }

    // Strip UTM params from URL (may already be stripped by useFilterState)
    const params = new URLSearchParams(window.location.search);
    if (params.has('utm_source') || params.has('utm_medium') || params.has('utm_campaign')) {
      params.delete('utm_source');
      params.delete('utm_medium');
      params.delete('utm_campaign');
      const remaining = params.toString();
      const clean = window.location.pathname + (remaining ? '?' + remaining : '');
      window.history.replaceState({}, '', clean);
    }
  }, []);
}
