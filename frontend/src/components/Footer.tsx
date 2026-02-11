import { useEffect, useState } from 'react';
import { getStatus } from '../services/api';

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

  useEffect(() => {
    getStatus()
      .then((data) => {
        if (data.last_synced_at) setLastSynced(data.last_synced_at);
      })
      .catch(() => {});
  }, []);

  return (
    <footer className="bg-dark-950 border-t border-dark-800 px-4 py-4 text-xs text-dark-500">
      <div className="max-w-7xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-2">
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
          <a href="mailto:privacy@govtrove.com" className="hover:text-dark-300">
            privacy@govtrove.com
          </a>
        </div>
      </div>
    </footer>
  );
}
