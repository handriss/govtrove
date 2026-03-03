import { useEffect } from 'react';

const API_BASE = import.meta.env.VITE_API_URL || '/api';

export default function useUTMCapture() {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const source = params.get('utm_source');
    const medium = params.get('utm_medium');
    const campaign = params.get('utm_campaign');

    if (!source && !medium && !campaign) return;

    if (campaign) {
      sessionStorage.setItem('govtrove_utm_campaign', campaign);
    }

    if (!sessionStorage.getItem('govtrove_utm_tracked')) {
      sessionStorage.setItem('govtrove_utm_tracked', '1');
      const payload = {
        utm_source: source,
        utm_medium: medium,
        utm_campaign: campaign,
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

    // Strip UTM params from URL
    params.delete('utm_source');
    params.delete('utm_medium');
    params.delete('utm_campaign');
    const remaining = params.toString();
    const clean = window.location.pathname + (remaining ? '?' + remaining : '');
    window.history.replaceState({}, '', clean);
  }, []);
}
