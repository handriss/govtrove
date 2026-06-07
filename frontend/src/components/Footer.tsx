import { useEffect, useState } from 'react';
import { Heart } from 'lucide-react';
import { getStatus } from '../services/api';
import { DONATION_URL, trackDonationClick } from '../lib/billing';
import { useAppAuth } from '../contexts/AuthContext';

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
  const { isAuthenticated, getAccessToken } = useAppAuth();

  async function handleDonationClick() {
    const token = isAuthenticated ? await getAccessToken().catch(() => undefined) : undefined;
    trackDonationClick('footer', token);
  }

  useEffect(() => {
    getStatus()
      .then((data) => {
        if (data.last_synced_at) setLastSynced(data.last_synced_at);
      })
      .catch(() => {});
  }, []);

  return (
    <footer className="bg-dark-950 border-t border-dark-800">
      <div className="max-w-7xl mx-auto px-6">
        <div className="py-6 flex flex-col items-center gap-2.5">
          <div className="flex flex-wrap items-center justify-center gap-3">
            <span className="text-sm text-dark-200">GovTrove is independent and free to use.</span>
            <a
              href={DONATION_URL}
              target="_blank"
              rel="noopener noreferrer"
              onClick={handleDonationClick}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-accent bg-accent/10 hover:bg-accent/20 border border-accent/20 transition-colors text-sm font-medium"
            >
              <Heart size={14} strokeWidth={2} />
              Support GovTrove
            </a>
          </div>
          <span className="text-xs text-dark-500">
            Built by Andrew &middot;{' '}
            <a href="mailto:andrew@govtrove.com" className="underline hover:text-dark-300">
              andrew@govtrove.com
            </a>
          </span>
        </div>

        <div className="py-3 border-t border-dark-800/50 flex flex-col sm:flex-row items-center justify-between gap-2 text-[11px] text-dark-600">
          <span>
            Data sourced from{' '}
            <a href="https://sam.gov" target="_blank" rel="noopener noreferrer" className="hover:text-dark-400">
              SAM.gov
            </a>
            {' \u00b7 '}Not affiliated with the U.S. Government
            {lastSynced && <> {' \u00b7 '}Updated {formatSyncTime(lastSynced)}</>}
          </span>
          <div className="flex items-center gap-3">
            <a href="https://govtrove.com/terms.html" target="_blank" rel="noopener noreferrer" className="hover:text-dark-400">
              Terms
            </a>
            <a href="https://govtrove.com/privacy.html" target="_blank" rel="noopener noreferrer" className="hover:text-dark-400">
              Privacy
            </a>
            <a href="https://govtrove.com/contact.html" target="_blank" rel="noopener noreferrer" className="hover:text-dark-400">
              Contact
            </a>
            <a href="https://govtrove.com/contact.html?subject=bug" target="_blank" rel="noopener noreferrer" className="hover:text-dark-400">
              Report
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
