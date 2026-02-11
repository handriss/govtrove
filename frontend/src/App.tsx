import { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import SimpleSearchPage from './pages/SimpleSearchPage';
import AdvancedSearchPage from './pages/AdvancedSearchPage';
import OpportunityDetail from './pages/OpportunityDetail';
import PreviewBanner from './components/PreviewBanner';
import Footer from './components/Footer';

function ExternalRedirect({ to }: { to: string }) {
  useEffect(() => {
    window.location.href = to;
  }, [to]);
  return null;
}

function App() {
  return (
    <Router>
      <PreviewBanner />
      <Routes>
        <Route path="/" element={<SimpleSearchPage />} />
        <Route path="/advanced" element={<AdvancedSearchPage />} />
        <Route path="/opportunity/:id" element={<OpportunityDetail />} />
        <Route path="/terms" element={<ExternalRedirect to="https://govtrove.com/terms.html" />} />
        <Route path="/privacy" element={<ExternalRedirect to="https://govtrove.com/privacy.html" />} />
      </Routes>
      <Footer />
    </Router>
  );
}

export default App;
