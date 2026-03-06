import { useState } from 'react';
import { X } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';

const SEARCH_COUNT_KEY = 'govtrove_anon_search_count';
const DISMISSED_KEY = 'govtrove_nudge_dismissed';
const THRESHOLD = 5;

export function incrementAnonSearchCount() {
  try {
    const count = parseInt(localStorage.getItem(SEARCH_COUNT_KEY) || '0', 10);
    localStorage.setItem(SEARCH_COUNT_KEY, String(count + 1));
  } catch { /* ignore */ }
}

export default function SignupNudgeBanner() {
  const { isAuthenticated, signUp } = useAppAuth();
  const [dismissed, setDismissed] = useState(() => localStorage.getItem(DISMISSED_KEY) === '1');

  if (isAuthenticated || dismissed) return null;

  const count = parseInt(localStorage.getItem(SEARCH_COUNT_KEY) || '0', 10);
  if (count < THRESHOLD) return null;

  const handleDismiss = () => {
    localStorage.setItem(DISMISSED_KEY, '1');
    setDismissed(true);
  };

  return (
    <div className="mb-3 flex items-center gap-3 px-4 py-3 rounded-xl border border-accent/20 bg-accent/5">
      <p className="flex-1 text-sm text-dark-300">
        Sign up free to save your searches and get notified when new matches appear.
      </p>
      <button
        onClick={() => signUp()}
        className="shrink-0 px-3 py-1.5 text-xs font-medium bg-accent hover:bg-accent-hover text-white rounded-lg transition-colors"
      >
        Sign Up Free
      </button>
      <button
        onClick={handleDismiss}
        className="shrink-0 p-1 text-dark-500 hover:text-dark-300 transition-colors"
        aria-label="Dismiss"
      >
        <X size={14} />
      </button>
    </div>
  );
}
