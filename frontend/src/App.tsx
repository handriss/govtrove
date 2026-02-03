import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import SimpleSearchPage from './pages/SimpleSearchPage';
import AdvancedSearchPage from './pages/AdvancedSearchPage';
import OpportunityDetail from './pages/OpportunityDetail';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<SimpleSearchPage />} />
        <Route path="/advanced" element={<AdvancedSearchPage />} />
        <Route path="/opportunity/:id" element={<OpportunityDetail />} />
      </Routes>
    </Router>
  );
}

export default App;
