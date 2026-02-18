import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft, Clock } from 'lucide-react';
import ResultsList from '../components/ResultsList';
import AuthButton from '../components/AuthButton';
import { useSearch } from '../hooks/useSearch';
import { getWhatsNewSync, getWhatsNew, whatsNewParams } from '../services/whatsNewCache';
import { searchOpportunities } from '../services/api';

export default function WhatsNewPage() {
  const { results, total, page, totalPages, loading, search, inject } = useSearch();
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    const cached = getWhatsNewSync();
    if (cached) {
      inject(cached);
      setInitialized(true);
    } else {
      getWhatsNew().then((data) => {
        inject(data);
        setInitialized(true);
      }).catch(() => {
        search(whatsNewParams());
        setInitialized(true);
      });
    }
  }, [inject, search]);

  const handlePageChange = useCallback((newPage: number) => {
    search(whatsNewParams(newPage));
  }, [search]);

  return (
    <div className="min-h-screen flex flex-col relative">
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      <div className="absolute top-4 right-6 z-20">
        <AuthButton />
      </div>

      <div className="relative z-10 flex flex-col items-center pt-10">
        <Link to="/" className="mb-6">
          <h1 className="text-2xl font-semibold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-dark-50 to-dark-200">
            GovTrove
          </h1>
        </Link>

        <div className="flex items-center gap-3 mb-8">
          <Link
            to="/"
            className="p-2 text-dark-400 hover:text-dark-100 rounded-lg hover:bg-dark-800/50 transition-all duration-200"
          >
            <ArrowLeft size={18} strokeWidth={1.5} />
          </Link>
          <div className="flex items-center gap-2 text-accent">
            <Clock size={18} strokeWidth={1.5} />
            <h2 className="text-lg font-medium">What's New Today</h2>
          </div>
        </div>
      </div>

      <div className="relative z-10 flex-1 max-w-6xl w-full mx-auto px-6 pb-10">
        <div className="flex items-center justify-between mb-6 pb-4 border-b border-dark-800/50">
          <p className="text-sm text-dark-400">
            {loading || !initialized ? (
              <span className="text-dark-500">Loading...</span>
            ) : (
              <>
                <span className="text-dark-200 font-medium">{total.toLocaleString()}</span>
                <span className="ml-1">new opportunities with open deadlines</span>
              </>
            )}
          </p>
        </div>
        <ResultsList
          results={results}
          page={page}
          totalPages={totalPages}
          loading={loading || !initialized}
          query=""
          onPageChange={handlePageChange}
        />
      </div>
    </div>
  );
}
