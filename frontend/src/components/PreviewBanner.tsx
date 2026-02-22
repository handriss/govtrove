import { useState } from 'react';
import { X } from 'lucide-react';

export default function PreviewBanner() {
  const [dismissed, setDismissed] = useState(false);

  if (dismissed) return null;

  return (
    <div className="bg-accent/10 border-b border-accent/20 px-4 py-2 text-center text-sm text-dark-300 relative">
      <span>
        <strong className="text-dark-100 font-semibold">Under Development</strong>
        {' '}&mdash; GovTrove is actively being built. Data is incomplete and search may not return all results.
      </span>
      <button
        onClick={() => setDismissed(true)}
        className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-dark-400 hover:text-dark-200 transition-colors"
        aria-label="Dismiss banner"
      >
        <X size={16} />
      </button>
    </div>
  );
}
