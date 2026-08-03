import { useEffect, useState } from 'react';
import { Outlet } from 'react-router-dom';
import { AlertTriangle, Gift } from 'lucide-react';
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
  const [lastSynced, setLastSynced] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    getStatus()
      .then((data) => setLastSynced(data.last_synced_at))
      .catch(() => {})
      .finally(() => setLoaded(true));
  }, []);

  if (!loaded) return null;
  const staleMs = STALE_THRESHOLD_HOURS * 60 * 60 * 1000;
  const isStale = !lastSynced || Date.now() - new Date(lastSynced).getTime() > staleMs;
  if (!isStale) return null;

  return (
    <div className="bg-amber-500/10 border-b border-amber-500/30 px-4 py-3">
      <div className="max-w-4xl mx-auto flex items-center justify-center gap-2 flex-wrap text-center">
        <AlertTriangle size={16} className="text-amber-400 shrink-0" />
        <p className="text-sm text-dark-200">
          Opportunity data hasn&apos;t updated
          {lastSynced ? (
            <> since <span className="text-dark-100 font-medium">{formatSyncTime(lastSynced)}</span></>
          ) : (
            ' recently'
          )}
          . This could be a temporary SAM.gov data outage or an issue on our end &mdash; we&apos;re looking into it. Existing opportunities remain fully searchable.{' '}
          <a
            href="https://sam.gov/alerts"
            target="_blank"
            rel="noopener noreferrer"
            className="text-amber-400 hover:text-amber-300 underline underline-offset-2 font-medium"
          >
            Check SAM.gov status
          </a>
        </p>
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
