import { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { AuthKitProvider } from '@workos-inc/authkit-react';
import { AuthProvider } from './contexts/AuthContext';
import SimpleSearchPage from './pages/SimpleSearchPage';
import AdvancedSearchPage from './pages/AdvancedSearchPage';
import OpportunityDetail from './pages/OpportunityDetail';
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

function AppRoutes() {
  return (
    <>
      <PreviewBanner />
      <Routes>
        <Route path="/" element={<SimpleSearchPage />} />
        <Route path="/advanced" element={<AdvancedSearchPage />} />
        <Route path="/opportunity/:id" element={<OpportunityDetail />} />
        <Route path="/callback" element={<SimpleSearchPage />} />
        <Route path="/terms" element={<ExternalRedirect to="https://govtrove.com/terms.html" />} />
        <Route path="/privacy" element={<ExternalRedirect to="https://govtrove.com/privacy.html" />} />
        <Route path="/contact" element={<ExternalRedirect to="https://govtrove.com/contact.html" />} />
      </Routes>
      <Footer />
    </>
  );
}

function App() {
  return (
    <AuthKitProvider clientId={WORKOS_CLIENT_ID} redirectUri={REDIRECT_URI}>
      <Router>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </Router>
    </AuthKitProvider>
  );
}

export default App;
