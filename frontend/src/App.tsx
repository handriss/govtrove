import { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import * as Sentry from '@sentry/react';
import { AuthKitProvider } from '@workos-inc/authkit-react';
import { AuthProvider, useAppAuth } from './contexts/AuthContext';
import SimpleSearchPage from './pages/SimpleSearchPage';
import WhatsNewPage from './pages/WhatsNewPage';
import OpportunityDetail from './pages/OpportunityDetail';
import ProfilePage from './pages/ProfilePage';
import SavedPage from './pages/SavedPage';
import NotificationsPage from './pages/NotificationsPage';
import NotFoundPage from './pages/NotFoundPage';
import PreviewBanner from './components/PreviewBanner';
import Footer from './components/Footer';

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
// from the URL before AuthKit can read it (its createClient runs in a
// setTimeout inside useEffect, which fires after child effects).
function AuthCallback() {
  const { isLoading } = useAppAuth();
  if (isLoading) {
    return <div className="min-h-screen flex items-center justify-center text-dark-500 text-sm">Signing in...</div>;
  }
  return <Navigate to="/" replace />;
}

function ErrorFallback() {
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

function AppRoutes() {
  return (
    <Sentry.ErrorBoundary fallback={<ErrorFallback />}>
      <PreviewBanner />
      <Routes>
        <Route path="/" element={<SimpleSearchPage />} />
        <Route path="/whats-new" element={<WhatsNewPage />} />
        <Route path="/opportunity/:id" element={<OpportunityDetail />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/saved" element={<SavedPage />} />
        <Route path="/notifications" element={<NotificationsPage />} />
        <Route path="/callback" element={<AuthCallback />} />
        <Route path="/terms" element={<ExternalRedirect to="https://govtrove.com/terms.html" />} />
        <Route path="/privacy" element={<ExternalRedirect to="https://govtrove.com/privacy.html" />} />
        <Route path="/contact" element={<ExternalRedirect to="https://govtrove.com/contact.html" />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      <Footer />
    </Sentry.ErrorBoundary>
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
