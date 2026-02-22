import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Clock } from 'lucide-react';
import SearchInput from '../components/SearchInput';
import SetAsideChips from '../components/SetAsideChips';
import AuthButton from '../components/AuthButton';
import { FilterBar, SearchResults } from '../components/search';
import SearchMobileFilters from '../components/search/SearchMobileFilters';
import { useFilterState } from '../hooks/useFilterState';
import { useFacetCounts } from '../hooks/useFacetCounts';
import { useSavedOpportunities } from '../hooks/useSavedOpportunities';
import { useDebounce } from '../hooks/useDebounce';
import { useSearch } from '../hooks/useSearch';
import { preloadWhatsNew } from '../services/whatsNewCache';

const PAGE_SIZE_KEY = 'govtrove_page_size';

export default function SimpleSearchPage() {
  const fs = useFilterState();
  const { facets, total: facetTotal, isLoading: facetsLoading } = useFacetCounts(fs.toFacetParams());
  const { results, total, page, totalPages, loading, error, search, reset } = useSearch();
  const saved = useSavedOpportunities();
  const [hasSearched, setHasSearched] = useState(false);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const isInitialSearch = useRef(true);

  const [pageSize, setPageSize] = useState(() => {
    const stored = localStorage.getItem(PAGE_SIZE_KEY);
    return stored ? parseInt(stored, 10) : 25;
  });

  useEffect(() => { preloadWhatsNew(); }, []);

  const searchParamsSerialized = useMemo(
    () => JSON.stringify(fs.toSearchParams()),
    [fs.toSearchParams],
  );
  const debouncedParams = useDebounce(searchParamsSerialized, 300);

  const hasActiveFilters = useMemo(() => {
    const f = fs.filters;
    return (
      f.keyword.length >= 2 ||
      f.setAside.length > 0 ||
      f.department !== '' ||
      f.state !== '' ||
      f.naics.length > 0 ||
      f.noticeType.length > 0 ||
      f.deadlinePreset !== '' ||
      f.postedFrom !== '' ||
      f.postedTo !== ''
    );
  }, [fs.filters]);

  // Debounced auto-search
  useEffect(() => {
    if (!hasActiveFilters) {
      if (hasSearched && !fs.filters.keyword) {
        reset();
        setHasSearched(false);
      }
      return;
    }

    const params = JSON.parse(debouncedParams);
    params.limit = pageSize;
    search(params);
    setHasSearched(true);
    isInitialSearch.current = false;
  }, [debouncedParams]); // eslint-disable-line react-hooks/exhaustive-deps

  // Initial search on mount if URL has filters
  useEffect(() => {
    if (!isInitialSearch.current) return;
    if (hasActiveFilters) {
      const params = fs.toSearchParams();
      params.limit = pageSize;
      search(params);
      setHasSearched(true);
    }
    isInitialSearch.current = false;
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSubmit = useCallback(() => {
    if (hasActiveFilters) {
      const params = fs.toSearchParams();
      params.limit = pageSize;
      search(params);
      setHasSearched(true);
    }
  }, [hasActiveFilters, fs, search, pageSize]);

  const handleSetAsideChange = useCallback((newSetAsides: string[]) => {
    fs.setFilter('setAside', newSetAsides);
  }, [fs]);

  const handlePageSizeChange = useCallback((size: number) => {
    setPageSize(size);
    localStorage.setItem(PAGE_SIZE_KEY, String(size));
    fs.setFilter('page', 1);
  }, [fs]);

  const showResults = hasSearched || results.length > 0;

  return (
    <div className="min-h-screen flex flex-col relative">
      <a href="#search-results" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:top-4 focus:left-4 focus:px-4 focus:py-2 focus:bg-accent focus:text-white focus:rounded-lg">Skip to search results</a>
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

        {/* Hero mode: large search + quick filters */}
        {!showResults && (
          <>
            <div className="w-full max-w-2xl px-6 mb-6">
              <SearchInput
                ref={inputRef}
                value={fs.filters.keyword}
                onChange={(v) => fs.setFilter('keyword', v)}
                onSubmit={handleSubmit}
                loading={loading && fs.filters.keyword.length >= 2}
                size="large"
                placeholder="Search contracts, solicitations, awards..."
                autoFocus
              />
            </div>

            <div className="text-center space-y-5 animate-in fade-in duration-500">
              <div className="flex flex-wrap items-center justify-center gap-2">
                <Link
                  to="/whats-new"
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm
                             border border-accent/30 bg-accent/10 text-accent
                             hover:bg-accent/20 transition-all duration-200"
                >
                  <Clock size={14} strokeWidth={1.5} />
                  What's New Today?
                </Link>
              </div>

              <div className="flex flex-wrap items-center justify-center gap-2">
                <SetAsideChips selected={fs.filters.setAside} onChange={handleSetAsideChange} />
              </div>

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
          </>
        )}
      </div>

      {/* Results mode: compact FilterBar + SearchResults */}
      {showResults && (
        <div id="search-results" className="relative z-10 flex-1 max-w-[1400px] w-full mx-auto px-6 pb-10">
          <FilterBar
            filters={fs.filters}
            setFilter={fs.setFilter}
            setFilters={fs.setFilters}
            removeFilter={fs.removeFilter}
            clearFilter={fs.clearFilter}
            clearAllFilters={fs.clearAllFilters}
            filterCount={fs.filterCount}
            facets={facets}
            facetsLoading={facetsLoading}
            loading={loading}
            onSearch={handleSubmit}
            total={facetTotal || total}
            onMobileFiltersOpen={() => setMobileFiltersOpen(true)}
          />
          <SearchResults
            results={results}
            total={facetTotal || total}
            page={page}
            pageSize={pageSize}
            totalPages={totalPages}
            loading={loading}
            keyword={fs.filters.keyword}
            filters={fs.filters}
            facets={facets}
            sort={fs.filters.sort}
            sortDir={fs.filters.sortDir}
            onSortChange={(s, d) => fs.setFilters({ sort: s, sortDir: d })}
            onPageChange={(p) => fs.setFilter('page', p)}
            onPageSizeChange={handlePageSizeChange}
            savedIds={saved.savedIds}
            onToggleSave={saved.toggleSave}
            onSaveAll={saved.saveAll}
            clearAllFilters={fs.clearAllFilters}
            error={error}
            onRetry={handleSubmit}
          />
          <SearchMobileFilters
            open={mobileFiltersOpen}
            onClose={() => setMobileFiltersOpen(false)}
            filters={fs.filters}
            setFilter={fs.setFilter}
            setFilters={fs.setFilters}
            clearAllFilters={fs.clearAllFilters}
            filterCount={fs.filterCount}
            facets={facets}
            facetsLoading={facetsLoading}
            total={facetTotal || total}
          />
        </div>
      )}
    </div>
  );
}
