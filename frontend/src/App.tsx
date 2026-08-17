import { Component, Suspense, lazy, useEffect } from 'react';
import type { ReactNode, ErrorInfo } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthKitProvider } from '@workos-inc/authkit-react';
import { AuthProvider, useAppAuth } from './contexts/AuthContext';
import SimpleSearchPage from './pages/SimpleSearchPage';
import NotFoundPage from './pages/NotFoundPage';
import useUTMCapture from './hooks/useUTMCapture';
import useTawk from './hooks/useTawk';
import AppLayout from './components/AppLayout';

const CHUNK_RELOAD_KEY = 'govtrove_chunk_reload';

// A deploy replaces the hashed asset files, so a tab that loaded the old index 404s
// on the next lazy route and renders a blank page. Reload once to pick up the new
// index; the sentinel keeps a genuinely unreachable chunk from looping forever, and
// is cleared as soon as any chunk loads.
function lazyWithReload(factory: Parameters<typeof lazy>[0]): ReturnType<typeof lazy> {
  return lazy(() =>
    factory()
      .then((mod) => {
        sessionStorage.removeItem(CHUNK_RELOAD_KEY);
        return mod;
      })
      .catch((err) => {
        if (sessionStorage.getItem(CHUNK_RELOAD_KEY)) throw err;
        sessionStorage.setItem(CHUNK_RELOAD_KEY, '1');
        window.location.reload();
        return new Promise<never>(() => {}); // the reload replaces the page
      }),
  );
}

const WhatsNewPage = lazyWithReload(() => import('./pages/WhatsNewPage'));
const OpportunityDetail = lazyWithReload(() => import('./pages/OpportunityDetail'));
const ProfilePage = lazyWithReload(() => import('./pages/ProfilePage'));
const SavedPage = lazyWithReload(() => import('./pages/SavedPage'));
const NotificationsPage = lazyWithReload(() => import('./pages/NotificationsPage'));
const AdminPage = lazyWithReload(() => import('./pages/AdminPage'));
const AdminPipelineRunDetailPage = lazyWithReload(() => import('./pages/AdminPipelineRunDetailPage'));
const AdminDataQualityPage = lazyWithReload(() => import('./pages/AdminDataQualityPage'));
const AdminReconcileDQPage = lazyWithReload(() => import('./pages/AdminReconcileDQPage'));
const AdminCampaignsPage = lazyWithReload(() => import('./pages/AdminCampaignsPage'));
const AdminObservabilityPage = lazyWithReload(() => import('./pages/AdminObservabilityPage'));
const AdminDSARPage = lazyWithReload(() => import('./pages/AdminDSARPage'));
const SnapDetailPage = lazyWithReload(() => import('./pages/AdminSnapDetailPage'));
const RedeemPage = lazyWithReload(() => import('./pages/RedeemPage'));
const SearchHistoryPage = lazyWithReload(() => import('./pages/SearchHistoryPage'));
const SearchGuidePage = lazyWithReload(() => import('./pages/SearchGuidePage'));
const SearchGuideInteractivePage = lazyWithReload(() => import('./pages/SearchGuideInteractivePage'));
const GlossaryPage = lazyWithReload(() => import('./pages/GlossaryPage'));
const ThankYouPage = lazyWithReload(() => import('./pages/ThankYouPage'));

const WORKOS_CLIENT_ID = import.meta.env.VITE_WORKOS_CLIENT_ID || '';
const REDIRECT_URI = `${window.location.origin}/callback`;

function ExternalRedirect({ to }: { to: string }) {
  useEffect(() => {
    window.location.href = to;
  }, [to]);
  return null;
}

// Lightweight callback page that doesn't interfere with URL params.
// AuthKit's createClient reads window.location.search to extract the OAuth
// code. If we render SimpleSearchPage here, useFilterState strips the code
// from the URL befo redirect is a act as a callbackre AuthKit can read it (its createClient runs in a
// setTimeout inside useEffect, which fires after child effects).
function AuthCallback() {
  const { isLoading } = useAppAuth();
  if (isLoading) {
    return <div className="min-h-screen flex items-center justify-center text-dark-500 text-sm">Signing in...</div>;
  }
  const redeemCode = sessionStorage.getItem('govtrove_redeem_code');
  if (redeemCode) {
    sessionStorage.removeItem('govtrove_redeem_code');
    return <Navigate to={`/redeem/${encodeURIComponent(redeemCode)}`} replace />;
  }
  const promo = sessionStorage.getItem('govtrove_promo_code');
  if (promo) {
    return <Navigate to={`/profile?promo=${encodeURIComponent(promo)}`} replace />;
  }
  return <Navigate to="/" replace />;
}

function LoadingFallback() {
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="h-6 w-6 border-2 border-primary-600 border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

class ErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean }> {
  state = { hasError: false };
  static getDerivedStateFromError() { return { hasError: true }; }
  componentDidCatch(error: Error, info: ErrorInfo) {
    import('@sentry/react').then(Sentry => Sentry.captureException(error, { extra: { componentStack: info.componentStack } })).catch(() => {});
  }
  render() {
    if (!this.state.hasError) return this.props.children;
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center max-w-md px-4">
          <h1 className="text-2xl font-bold text-dark-900 mb-2">Something went wrong</h1>
          <p className="text-dark-500 mb-4">An unexpected error occurred. Please try refreshing the page.</p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
          >
            Refresh page
          </button>
        </div>
      </div>
    );
  }
}

function AppRoutes() {
  useUTMCapture();
  useTawk();

  // Persist promo code across auth redirects
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const promo = params.get('promo');
    if (promo) {
      sessionStorage.setItem('govtrove_promo_code', promo);
    }
  }, []);

  return (
    <ErrorBoundary>
      <Suspense fallback={<LoadingFallback />}>
        <Routes>
          {/* App routes with sidebar/tab bar layout */}
          <Route element={<AppLayout />}>
            <Route path="/" element={<SimpleSearchPage />} />
            <Route path="/whats-new" element={<WhatsNewPage />} />
            <Route path="/opportunity/:id" element={<OpportunityDetail />} />
            <Route path="/profile" element={<ProfilePage />} />
            <Route path="/saved" element={<SavedPage />} />
            <Route path="/notifications" element={<NotificationsPage />} />
            <Route path="/search/:searchId/history" element={<SearchHistoryPage />} />
            <Route path="/guide" element={<SearchGuidePage />} />
            <Route path="/guide/interactive" element={<SearchGuideInteractivePage />} />
            <Route path="/glossary" element={<GlossaryPage />} />
            <Route path="/thanks" element={<ThankYouPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
          {/* Admin routes — keep AdminLayout */}
          <Route path="/admin" element={<AdminPage />} />
          <Route path="/admin/pipeline/:id" element={<AdminPipelineRunDetailPage />} />
          <Route path="/admin/data-quality" element={<AdminDataQualityPage />} />
          <Route path="/admin/reconcile-dq" element={<AdminReconcileDQPage />} />
          <Route path="/admin/campaigns" element={<AdminCampaignsPage />} />
          <Route path="/admin/observability" element={<AdminObservabilityPage />} />
          <Route path="/admin/dsar" element={<AdminDSARPage />} />
          <Route path="/snap/csv/:id" element={<SnapDetailPage source="csv" />} />
          <Route path="/snap/archived-csv/:id" element={<SnapDetailPage source="archived-csv" />} />
          <Route path="/snap/api/:id" element={<SnapDetailPage source="api" />} />
          {/* Standalone routes — no layout */}
          <Route path="/redeem/:code" element={<RedeemPage />} />
          <Route path="/callback" element={<AuthCallback />} />
          <Route path="/terms" element={<ExternalRedirect to="https://govtrove.com/terms.html" />} />
          <Route path="/privacy" element={<ExternalRedirect to="https://govtrove.com/privacy.html" />} />
          <Route path="/contact" element={<ExternalRedirect to="https://govtrove.com/contact.html" />} />
        </Routes>
      </Suspense>
    </ErrorBoundary>
  );
}

function App() {
  return (
    <AuthKitProvider clientId={WORKOS_CLIENT_ID} redirectUri={REDIRECT_URI} devMode>
      <Router>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </Router>
    </AuthKitProvider>
  );
}

export default App;
