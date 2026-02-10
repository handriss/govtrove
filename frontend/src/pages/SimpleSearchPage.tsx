import { useState, useEffect, useRef, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Sparkles } from 'lucide-react';
import SearchInput from '../components/SearchInput';
import ResultsList from '../components/ResultsList';
import { useDebounce } from '../hooks/useDebounce';
import { useSearch } from '../hooks/useSearch';

export default function SimpleSearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialQuery = searchParams.get('q') || '';
  const initialPage = parseInt(searchParams.get('page') || '1', 10);

  const [query, setQuery] = useState(initialQuery);
  const debouncedQuery = useDebounce(query, 300);
  const { results, total, page, totalPages, loading, search, reset } = useSearch();
  const [hasSearched, setHasSearched] = useState(initialQuery.length >= 2);
  const inputRef = useRef<HTMLInputElement>(null);
  const isInitialMount = useRef(true);

  const updateURL = useCallback((q: string, p: number) => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (p > 1) params.set('page', String(p));
    setSearchParams(params, { replace: true });
  }, [setSearchParams]);

  useEffect(() => {
    if (isInitialMount.current && initialQuery.length >= 2) {
      search({ q: initialQuery, sort: 'relevance', page: initialPage, limit: 25 });
      isInitialMount.current = false;
      return;
    }
    isInitialMount.current = false;

    if (debouncedQuery.length >= 2) {
      search({ q: debouncedQuery, sort: 'relevance', page: 1, limit: 25 });
      updateURL(debouncedQuery, 1);
      setHasSearched(true);
    } else if (debouncedQuery.length === 0 && hasSearched) {
      reset();
      updateURL('', 1);
      setHasSearched(false);
    }
  }, [debouncedQuery, search, reset, hasSearched, initialQuery, initialPage, updateURL]);

  const handlePageChange = (newPage: number) => {
    search({ q: debouncedQuery, sort: 'relevance', page: newPage, limit: 25 });
    updateURL(debouncedQuery, newPage);
  };

  const handleSubmit = () => {
    if (query.length >= 2) {
      search({ q: query, sort: 'relevance', page: 1, limit: 25 });
      updateURL(query, 1);
      setHasSearched(true);
    }
  };

  const showResults = hasSearched || results.length > 0;

  return (
    <div className="min-h-screen flex flex-col relative">
      {/* Subtle gradient overlay */}
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      {/* Subtle radial glow */}
      <div className="fixed top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-accent/[0.02] rounded-full blur-3xl pointer-events-none" />

      <div className={`relative z-10 flex flex-col items-center transition-all duration-500 ease-out ${showResults ? 'pt-10' : 'pt-[25vh]'}`}>
        {/* Logo */}
        <Link to="/" className={`mb-8 transition-all duration-500 ${showResults ? 'mb-6' : 'mb-10'}`}>
          <h1 className={`font-semibold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-dark-50 to-dark-200 transition-all duration-500 ${showResults ? 'text-2xl' : 'text-5xl'}`}>
            GovTrove
          </h1>
          {!showResults && (
            <p className="text-center text-dark-400 text-sm mt-2 tracking-wide">
              Government Contract Intelligence
            </p>
          )}
        </Link>

        {/* Search Input */}
        <div className={`w-full max-w-2xl px-6 transition-all duration-500 ${showResults ? 'mb-6' : 'mb-6'}`}>
          <SearchInput
            ref={inputRef}
            value={query}
            onChange={setQuery}
            onSubmit={handleSubmit}
            loading={loading && query.length >= 2}
            size={showResults ? 'default' : 'large'}
            placeholder="Search contracts, solicitations, awards..."
            autoFocus
          />
        </div>

        {/* Hints and Advanced Link */}
        {!showResults && (
          <div className="text-center space-y-6 animate-in fade-in duration-500">
            <Link
              to="/advanced"
              className="inline-flex items-center gap-2 text-sm text-dark-400 hover:text-accent transition-colors duration-200"
            >
              <Sparkles size={14} strokeWidth={1.5} />
              Advanced Search
            </Link>

            <div className="flex items-center gap-6 text-xs text-dark-500">
              <span className="flex items-center gap-2">
                <kbd className="px-2 py-0.5 bg-dark-800/50 rounded text-dark-400 font-mono">"quotes"</kbd>
                <span>exact phrase</span>
              </span>
              <span className="flex items-center gap-2">
                <kbd className="px-2 py-0.5 bg-dark-800/50 rounded text-dark-400 font-mono">OR</kbd>
                <span>alternatives</span>
              </span>
              <span className="flex items-center gap-2">
                <kbd className="px-2 py-0.5 bg-dark-800/50 rounded text-dark-400 font-mono">-minus</kbd>
                <span>exclude</span>
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Results */}
      {showResults && (
        <div className="relative z-10 flex-1 max-w-6xl w-full mx-auto px-6 pb-10">
          <div className="flex items-center justify-between mb-6 pb-4 border-b border-dark-800/50">
            <p className="text-sm text-dark-400">
              <span className="text-dark-200 font-medium">{total.toLocaleString()}</span>
              <span className="ml-1">results</span>
            </p>
            <Link
              to="/advanced"
              className="text-sm text-dark-400 hover:text-accent transition-colors duration-200 flex items-center gap-1.5"
            >
              <Sparkles size={14} strokeWidth={1.5} />
              Advanced
            </Link>
          </div>
          <ResultsList
            results={results}
            page={page}
            totalPages={totalPages}
            loading={loading}
            query={query}
            onPageChange={handlePageChange}
          />
        </div>
      )}
    </div>
  );
}
