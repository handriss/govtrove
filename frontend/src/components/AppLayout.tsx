import { useEffect, useState } from 'react';
import { Outlet } from 'react-router-dom';
import { AlertTriangle, ExternalLink, Gift, X } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { getStatus } from '../services/api';
import { SidebarProvider, useSidebar } from '../contexts/SidebarContext';
import { PRO_FEATURES_FREE_FOR_ALL } from '../lib/billing';
import AppSidebar from './AppSidebar';
import BottomTabBar from './BottomTabBar';
import AuthButton from './AuthButton';
import Footer from './Footer';

function PromoBanner() {
  const { isAuthenticated, signUp, govtroveUser } = useAppAuth();
  const hasPromo = sessionStorage.getItem('govtrove_promo_code');
  const hasPro = PRO_FEATURES_FREE_FOR_ALL || govtroveUser?.plan === 'pro' || govtroveUser?.free_forever || (govtroveUser?.gift_expires_at && new Date(govtroveUser.gift_expires_at) > new Date());

  if (!hasPromo || hasPro) return null;

  if (!isAuthenticated) {
    return (
      <div className="bg-accent/10 border-b border-accent/30 px-4 py-3">
        <div className="max-w-4xl mx-auto flex items-center justify-center gap-3 flex-wrap">
          <Gift size={16} className="text-accent shrink-0" />
          <p className="text-sm text-dark-200">
            You have a special offer! Sign up to claim your discount on GovTrove Pro.
          </p>
          <button
            onClick={() => signUp()}
            className="px-4 py-1.5 text-xs font-semibold rounded-lg bg-accent text-dark-950 hover:bg-accent/90 transition-colors"
          >
            Sign Up Now
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-accent/10 border-b border-accent/30 px-4 py-3">
      <div className="max-w-4xl mx-auto flex items-center justify-center gap-3 flex-wrap">
        <Gift size={16} className="text-accent shrink-0" />
        <p className="text-sm text-dark-200">
          You have a special offer for GovTrove Pro!
        </p>
        <a
          href={`/profile?promo=${hasPromo}`}
          className="px-4 py-1.5 text-xs font-semibold rounded-lg bg-accent text-dark-950 hover:bg-accent/90 transition-colors"
        >
          Claim Offer
        </a>
      </div>
    </div>
  );
}

// Data-freshness notice driven by the same last_synced_at signal as the staleness
// watcher: shown whenever opportunity data hasn't refreshed within STALE_THRESHOLD_HOURS.
// The cause is intentionally hedged (SAM.gov outage vs. a problem on our side) since the
// banner can't know which, and it discloses the last-updated date (same format as the
// footer). Threshold-based, so it covers any future staleness and auto-hides once a fresh
// sync lands. 48h tolerates SAM.gov's daily cadence + weekends without false alarms.
const STALE_THRESHOLD_HOURS = 48;
const STALE_DISMISS_KEY = 'govtrove_stale_dismissed';

function formatSyncTime(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    timeZone: 'UTC',
    timeZoneName: 'short',
  });
}

function StaleDataBanner() {
  const { isAuthenticated } = useAppAuth();
  const [lastSynced, setLastSynced] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    getStatus()
      .then((data) => {
        setLastSynced(data.last_synced_at);
        // Stay dismissed only for the exact stale state the user dismissed; if a
        // later/different stale state occurs (last_synced_at changes), show again.
        if (sessionStorage.getItem(STALE_DISMISS_KEY) === (data.last_synced_at ?? 'null')) {
          setDismissed(true);
        }
      })
      .catch(() => {})
      .finally(() => setLoaded(true));
  }, []);

  if (!loaded || dismissed) return null;
  const staleMs = STALE_THRESHOLD_HOURS * 60 * 60 * 1000;
  const isStale = !lastSynced || Date.now() - new Date(lastSynced).getTime() > staleMs;
  if (!isStale) return null;

  const handleDismiss = () => {
    sessionStorage.setItem(STALE_DISMISS_KEY, lastSynced ?? 'null');
    setDismissed(true);
  };

  return (
    <div
      className={`pointer-events-none fixed inset-x-0 z-30 flex justify-center px-3 ${
        isAuthenticated ? 'bottom-20 md:bottom-4' : 'bottom-4'
      }`}
    >
      <div className="pointer-events-auto flex w-full max-w-xl items-start gap-2.5 rounded-xl border border-amber-500/30 bg-dark-900/95 px-3.5 py-3 shadow-xl shadow-black/40 backdrop-blur-sm">
        <AlertTriangle size={16} className="mt-0.5 shrink-0 text-amber-400" />
        <p className="min-w-0 flex-1 text-[13px] leading-relaxed text-amber-100/90">
          Opportunity data hasn&apos;t updated
          {lastSynced ? (
            <> since <span className="font-semibold text-amber-50">{formatSyncTime(lastSynced)}</span></>
          ) : (
            ' recently'
          )}
          . Possibly a temporary SAM.gov outage or an issue on our end.{' '}
          <a
            href="https://sam.gov/alerts"
            target="_blank"
            rel="noopener noreferrer"
            className="whitespace-nowrap font-medium text-amber-300 underline decoration-amber-400/40 underline-offset-2 transition-colors hover:text-amber-200 hover:decoration-amber-300"
          >
            Check SAM.gov status
            <ExternalLink size={11} className="ml-0.5 inline align-baseline" />
          </a>
        </p>
        <button
          type="button"
          onClick={handleDismiss}
          aria-label="Dismiss notice"
          className="-mr-1 -mt-0.5 shrink-0 rounded p-1 text-amber-400/70 transition-colors hover:bg-amber-400/10 hover:text-amber-200"
        >
          <X size={15} />
        </button>
      </div>
    </div>
  );
}

function AppLayoutInner() {
  const { isAuthenticated } = useAppAuth();
  const { collapsed } = useSidebar();
  const marginLeft = isAuthenticated ? (collapsed ? 'md:ml-14' : 'md:ml-56') : '';

  return (
    <>
      <StaleDataBanner />
      <PromoBanner />
      {isAuthenticated && (
        <>
          <AppSidebar />
          <BottomTabBar />
        </>
      )}
      <div className="fixed top-4 right-6 z-20">
        <AuthButton />
      </div>
      <main className={`${marginLeft} transition-all duration-200 min-h-screen flex flex-col ${isAuthenticated ? 'pb-20 md:pb-0' : ''}`}>
        <div className="flex-1 flex flex-col">
          <Outlet />
        </div>
        <Footer />
      </main>
    </>
  );
}

export default function AppLayout() {
  return (
    <SidebarProvider>
      <AppLayoutInner />
    </SidebarProvider>
  );
}
