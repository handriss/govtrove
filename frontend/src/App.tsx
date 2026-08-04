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

const WhatsNewPage = lazy(() => import('./pages/WhatsNewPage'));
const OpportunityDetail = lazy(() => import('./pages/OpportunityDetail'));
const ProfilePage = lazy(() => import('./pages/ProfilePage'));
const SavedPage = lazy(() => import('./pages/SavedPage'));
const NotificationsPage = lazy(() => import('./pages/NotificationsPage'));
const AdminPage = lazy(() => import('./pages/AdminPage'));
const AdminPipelineRunDetailPage = lazy(() => import('./pages/AdminPipelineRunDetailPage'));
const AdminDataQualityPage = lazy(() => import('./pages/AdminDataQualityPage'));
const AdminReconcileDQPage = lazy(() => import('./pages/AdminReconcileDQPage'));
const AdminCampaignsPage = lazy(() => import('./pages/AdminCampaignsPage'));
const AdminObservabilityPage = lazy(() => import('./pages/AdminObservabilityPage'));
const AdminDSARPage = lazy(() => import('./pages/AdminDSARPage'));
const SnapDetailPage = lazy(() => import('./pages/AdminSnapDetailPage'));
const RedeemPage = lazy(() => import('./pages/RedeemPage'));
const SearchHistoryPage = lazy(() => import('./pages/SearchHistoryPage'));
const SearchGuidePage = lazy(() => import('./pages/SearchGuidePage'));
const SearchGuideInteractivePage = lazy(() => import('./pages/SearchGuideInteractivePage'));
const GlossaryPage = lazy(() => import('./pages/GlossaryPage'));
const ThankYouPage = lazy(() => import('./pages/ThankYouPage'));

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
