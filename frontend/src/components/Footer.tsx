import { useEffect, useState } from 'react';
import { usePostHog, useFeatureFlagVariantKey } from '@posthog/react';
import { getStatus } from '../services/api';
import { trackFounderCtaClicked } from '../lib/analytics';

const FOUNDER_CTA_COPY: Record<string, { prefix: string; suffix: string }> = {
  'read-every-email': { prefix: 'Feedback? I read every email', suffix: '' },
  'something-off': { prefix: 'Something off? Tell me directly', suffix: '' },
  'i-reply': { prefix: 'Questions?', suffix: '— I reply to everything' },
};

function formatSyncTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    timeZone: 'UTC',
    timeZoneName: 'short',
  });
}

export default function Footer() {
  const [lastSynced, setLastSynced] = useState<string | null>(null);
  const posthog = usePostHog();
  const ctaVariant = useFeatureFlagVariantKey('founder-cta-variant');
  const variant = typeof ctaVariant === 'string' ? ctaVariant : 'read-every-email';
  const cta = FOUNDER_CTA_COPY[variant] || FOUNDER_CTA_COPY['read-every-email'];

  useEffect(() => {
    getStatus()
      .then((data) => {
        if (data.last_synced_at) setLastSynced(data.last_synced_at);
      })
      .catch(() => {});
  }, []);

  return (
    <footer className="bg-dark-950 border-t border-dark-800 px-4 py-4 text-xs text-dark-500">
      <div className="max-w-7xl mx-auto flex flex-col items-center gap-2">
        <span className="text-dark-400">
          {cta.prefix}{' '}
          <a
            href="mailto:andrew@govtrove.com"
            className="underline hover:text-dark-200"
            onClick={() => trackFounderCtaClicked(posthog, variant)}
          >
            {variant === 'i-reply' ? 'andrew@govtrove.com' : '\u2192 andrew@govtrove.com'}
          </a>
          {cta.suffix && ` ${cta.suffix}`}
        </span>
        <div className="w-full flex flex-col sm:flex-row items-center justify-between gap-2">
          <div className="flex flex-col sm:flex-row items-center gap-1 sm:gap-3">
            <span>
              Data sourced from{' '}
              <a href="https://sam.gov" target="_blank" rel="noopener noreferrer" className="underline hover:text-dark-300">
                SAM.gov
              </a>
              . GovTrove is not affiliated with the U.S. Government.
            </span>
            {lastSynced && (
              <span className="text-dark-600">Last updated: {formatSyncTime(lastSynced)}</span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <a href="https://govtrove.com/terms.html" target="_blank" rel="noopener noreferrer" className="hover:text-dark-300">
              Terms
            </a>
            <a href="https://govtrove.com/privacy.html" target="_blank" rel="noopener noreferrer" className="hover:text-dark-300">
              Privacy
            </a>
            <a href="https://govtrove.com/contact.html" target="_blank" rel="noopener noreferrer" className="hover:text-dark-300">
              Contact
            </a>
            <a href="https://govtrove.com/contact.html?subject=bug" target="_blank" rel="noopener noreferrer" className="hover:text-dark-300">
              Report a Problem
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
