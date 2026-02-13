import { useState, useEffect, useRef, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Sparkles, Clock } from 'lucide-react';
import SearchInput from '../components/SearchInput';
import ResultsList from '../components/ResultsList';
import SetAsideChips from '../components/SetAsideChips';
import AuthButton from '../components/AuthButton';
import { useDebounce } from '../hooks/useDebounce';
import { useSearch } from '../hooks/useSearch';

const DEFAULT_TYPES = 'Solicitation,Presolicitation,Combined Synopsis/Solicitation,Sources Sought';

export default function SimpleSearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialQuery = searchParams.get('q') || '';
  const initialPage = parseInt(searchParams.get('page') || '1', 10);
  const initialSetAsides = searchParams.get('set_aside')?.split(',').filter(Boolean) || [];

  const [query, setQuery] = useState(initialQuery);
  const [setAsides, setSetAsides] = useState<string[]>(initialSetAsides);
  const debouncedQuery = useDebounce(query, 300);
  const { results, total, page, totalPages, loading, search, reset } = useSearch();
  const [hasSearched, setHasSearched] = useState(initialQuery.length >= 2);
  const inputRef = useRef<HTMLInputElement>(null);
  const isInitialMount = useRef(true);

  const updateURL = useCallback((q: string, p: number, sa?: string[]) => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (p > 1) params.set('page', String(p));
    if (sa && sa.length) params.set('set_aside', sa.join(','));
    setSearchParams(params, { replace: true });
  }, [setSearchParams]);

  const buildParams = useCallback((q: string, pageNum: number, sa: string[]) => ({
    q: q || undefined,
    type: DEFAULT_TYPES,
    set_aside: sa.length ? sa.join(',') : undefined,
    sort: q ? 'relevance' : 'posted_date' as const,
    order: 'desc' as const,
    page: pageNum,
    limit: 25,
  }), []);

  useEffect(() => {
    if (isInitialMount.current && initialQuery.length >= 2) {
      search(buildParams(initialQuery, initialPage, setAsides));
      isInitialMount.current = false;
      return;
    }
    isInitialMount.current = false;

    if (debouncedQuery.length >= 2) {
      search(buildParams(debouncedQuery, 1, setAsides));
      updateURL(debouncedQuery, 1, setAsides);
      setHasSearched(true);
    } else if (debouncedQuery.length === 0 && hasSearched) {
      reset();
      updateURL('', 1, setAsides);
      setHasSearched(false);
    }
  }, [debouncedQuery, search, reset, hasSearched, initialQuery, initialPage, updateURL, buildParams, setAsides]);

  const handlePageChange = (newPage: number) => {
    search(buildParams(debouncedQuery, newPage, setAsides));
    updateURL(debouncedQuery, newPage, setAsides);
  };

  const handleSubmit = () => {
    if (query.length >= 2) {
      search(buildParams(query, 1, setAsides));
      updateURL(query, 1, setAsides);
      setHasSearched(true);
    }
  };

  const handleSetAsideChange = useCallback((newSetAsides: string[]) => {
    setSetAsides(newSetAsides);
    if (hasSearched || results.length > 0) {
      search(buildParams(debouncedQuery, 1, newSetAsides));
      updateURL(debouncedQuery, 1, newSetAsides);
    }
  }, [hasSearched, results.length, search, buildParams, debouncedQuery, updateURL]);

  const handleWhatsNew = useCallback(() => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const postedFrom = yesterday.toISOString().split('T')[0];
    search({
      type: DEFAULT_TYPES,
      set_aside: setAsides.length ? setAsides.join(',') : undefined,
      posted_from: postedFrom,
      sort: 'posted_date',
      order: 'desc',
      page: 1,
      limit: 25,
    });
    setHasSearched(true);
  }, [search, setAsides]);

  const showResults = hasSearched || results.length > 0;

  return (
    <div className="min-h-screen flex flex-col relative">
      {/* Subtle gradient overlay */}
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      {/* Subtle radial glow */}
      <div className="fixed top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-accent/[0.02] rounded-full blur-3xl pointer-events-none" />

      <div className="absolute top-4 right-6 z-20">
        <AuthButton />
      </div>

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

        {/* Quick Filters */}
        {!showResults && (
          <div className="text-center space-y-5 animate-in fade-in duration-500">
            <div className="flex flex-wrap items-center justify-center gap-2">
              <button
                onClick={handleWhatsNew}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm
                           border border-accent/30 bg-accent/10 text-accent
                           hover:bg-accent/20 transition-all duration-200"
              >
                <Clock size={14} strokeWidth={1.5} />
                What's New Today?
              </button>
            </div>

            <div className="flex flex-wrap items-center justify-center gap-2">
              <SetAsideChips selected={setAsides} onChange={handleSetAsideChange} />
            </div>

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

        {/* Set-aside chips when results are showing */}
        {showResults && (
          <div className="w-full max-w-2xl px-6 mb-4">
            <div className="flex flex-wrap items-center justify-center gap-2">
              <SetAsideChips selected={setAsides} onChange={handleSetAsideChange} />
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
