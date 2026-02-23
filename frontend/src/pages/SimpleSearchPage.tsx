import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Clock, Bookmark, X } from 'lucide-react';
import SearchInput from '../components/SearchInput';
import QuickFilterChips from '../components/QuickFilterChips';
import AuthButton from '../components/AuthButton';
import { FilterBar, SearchResults } from '../components/search';
import SearchMobileFilters from '../components/search/SearchMobileFilters';
import { useFilterState } from '../hooks/useFilterState';
import { useFacetCounts } from '../hooks/useFacetCounts';
import { useSavedOpportunities } from '../hooks/useSavedOpportunities';
import { useSavedSearches } from '../hooks/useSavedSearches';
import { useDebounce } from '../hooks/useDebounce';
import { useSearch } from '../hooks/useSearch';
import { useAppAuth } from '../contexts/AuthContext';
import { preloadWhatsNew } from '../services/whatsNewCache';

const PAGE_SIZE_KEY = 'govtrove_page_size';

export default function SimpleSearchPage() {
  const fs = useFilterState();
  const { facets, total: facetTotal, isLoading: facetsLoading } = useFacetCounts(fs.toFacetParams());
  const { results, total, page, totalPages, loading, error, search, reset } = useSearch();
  const saved = useSavedOpportunities();
  const { isAuthenticated } = useAppAuth();
  const { savedSearches, saveCurrentSearch, deleteSearch } = useSavedSearches();
  const [hasSearched, setHasSearched] = useState(false);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);
  const [saveSearchOpen, setSaveSearchOpen] = useState(false);
  const [saveSearchName, setSaveSearchName] = useState('');
  const [savingSearch, setSavingSearch] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const isInitialSearch = useRef(true);
  const lastSearchedRef = useRef('');

  // Snapshot the first opportunity count so it stays stable in hero mode
  const [heroTotal, setHeroTotal] = useState(0);
  useEffect(() => {
    if (facetTotal > 0 && heroTotal === 0) setHeroTotal(facetTotal);
  }, [facetTotal, heroTotal]);

  const [pageSize, setPageSize] = useState(() => {
    const stored = localStorage.getItem(PAGE_SIZE_KEY);
    return stored ? parseInt(stored, 10) : 25;
  });

  useEffect(() => { preloadWhatsNew(); }, []);

  // Non-keyword search params — auto-search fires when these change
  const autoSearchKey = useMemo(() => {
    const p = fs.toSearchParams();
    delete p.q;
    return JSON.stringify(p);
  }, [fs.toSearchParams]);
  const debouncedAutoSearch = useDebounce(autoSearchKey, 300);

  // Explicit search trigger (Enter key, chip clicks)
  const [searchTrigger, setSearchTrigger] = useState(0);
  const triggerSearch = useCallback(() => {
    setSearchTrigger(t => t + 1);
  }, []);

  const hasActiveFilters = useMemo(() => {
    const f = fs.filters;
    return (
      f.keyword.length >= 2 ||
      f.setAside.length > 0 ||
      f.department !== '' ||
      f.state !== '' ||
      f.naics.length > 0 ||
      f.psc.length > 0 ||
      f.noticeType.length > 0 ||
      f.deadlinePreset !== '' ||
      f.postedFrom !== '' ||
      f.postedTo !== ''
    );
  }, [fs.filters]);

  // Reset results when all filters are cleared
  useEffect(() => {
    if (!hasActiveFilters && hasSearched) {
      reset();
      setHasSearched(false);
      lastSearchedRef.current = '';
    }
  }, [hasActiveFilters]); // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-search: fires on non-keyword filter changes or explicit trigger (Enter, chip click)
  useEffect(() => {
    if (!hasActiveFilters) return;

    const params = fs.toSearchParams();
    params.limit = pageSize;
    const key = JSON.stringify(params);
    if (key === lastSearchedRef.current) return;
    lastSearchedRef.current = key;

    search(params);
    setHasSearched(true);
    isInitialSearch.current = false;
  }, [debouncedAutoSearch, searchTrigger]); // eslint-disable-line react-hooks/exhaustive-deps

  // Initial search on mount if URL has filters
  useEffect(() => {
    if (!isInitialSearch.current) return;
    if (hasActiveFilters) {
      const params = fs.toSearchParams();
      params.limit = pageSize;
      lastSearchedRef.current = JSON.stringify(params);
      search(params);
      setHasSearched(true);
    }
    isInitialSearch.current = false;
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSubmit = useCallback(() => {
    if (hasActiveFilters) triggerSearch();
  }, [hasActiveFilters, triggerSearch]);

  const handlePageSizeChange = useCallback((size: number) => {
    setPageSize(size);
    localStorage.setItem(PAGE_SIZE_KEY, String(size));
    fs.setFilter('page', 1);
  }, [fs]);

  const handleSaveSearch = useCallback(async () => {
    if (!saveSearchName.trim()) return;
    setSavingSearch(true);
    try {
      const { sort, sortDir, page: _page, ...filterData } = fs.filters;
      await saveCurrentSearch(saveSearchName.trim(), filterData);
      setSaveSearchOpen(false);
      setSaveSearchName('');
    } catch {
      // ignore
    } finally {
      setSavingSearch(false);
    }
  }, [saveSearchName, fs.filters, saveCurrentSearch]);

  const handleSavedSearchClick = useCallback((filters: Record<string, unknown>) => {
    fs.setFilters(filters as Partial<typeof fs.filters>);
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
          <h1 className={`text-center font-semibold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-dark-50 to-dark-200 transition-all duration-500 ${showResults ? 'text-2xl' : 'text-5xl'}`}>
            GovTrove
          </h1>
          {!showResults && (
            <>
              <p className="text-center text-dark-400 text-sm mt-2 tracking-wide">
                Government Contract Intelligence
              </p>
              {heroTotal > 0 && (
                <p className="text-center text-dark-500 text-xs mt-1">
                  {heroTotal.toLocaleString()} active opportunities
                </p>
              )}
            </>
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
                showSubmitButton
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

              <QuickFilterChips filters={fs.filters} setFilter={fs.setFilter} onSearch={triggerSearch} />
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

          {/* Saved search pills + save button */}
          {(savedSearches.length > 0 || (isAuthenticated && hasActiveFilters)) && (
            <div className="flex items-center gap-2 mb-3">
              <div className="flex items-center gap-2 overflow-x-auto scrollbar-hide flex-1 min-w-0">
                {savedSearches.map((ss) => (
                  <button
                    key={ss.id}
                    onClick={() => handleSavedSearchClick(ss.filters)}
                    className="group inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs
                               border border-dark-700/50 bg-dark-800/30 text-dark-300
                               hover:border-dark-600/50 hover:text-dark-100 transition-all whitespace-nowrap shrink-0"
                  >
                    {ss.name}
                    <span
                      role="button"
                      onClick={(e) => { e.stopPropagation(); deleteSearch(ss.id); }}
                      className="opacity-0 group-hover:opacity-100 text-dark-500 hover:text-red-400 transition-opacity"
                    >
                      <X size={12} />
                    </span>
                  </button>
                ))}
              </div>
              {isAuthenticated && hasActiveFilters && (
                <div className="relative shrink-0">
                  <button
                    onClick={() => setSaveSearchOpen(!saveSearchOpen)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs
                               border border-dark-600/50 text-dark-400
                               hover:border-dark-500/50 hover:text-dark-200 transition-all whitespace-nowrap"
                    title="Save current search"
                  >
                    <Bookmark size={12} />
                    Save Search
                  </button>
                  {saveSearchOpen && (
                    <div className="absolute top-full right-0 mt-2 w-64 bg-dark-900 border border-dark-700/50 rounded-xl shadow-xl z-50 p-3">
                      <input
                        type="text"
                        value={saveSearchName}
                        onChange={(e) => setSaveSearchName(e.target.value)}
                        onKeyDown={(e) => { if (e.key === 'Enter') handleSaveSearch(); }}
                        placeholder="Name this search..."
                        className="w-full px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg
                                   text-dark-100 placeholder-dark-500 focus:outline-none focus:border-accent/50"
                        autoFocus
                      />
                      <div className="flex justify-end gap-2 mt-2">
                        <button
                          onClick={() => { setSaveSearchOpen(false); setSaveSearchName(''); }}
                          className="px-3 py-1.5 text-xs text-dark-400 hover:text-dark-200 transition-colors"
                        >
                          Cancel
                        </button>
                        <button
                          onClick={handleSaveSearch}
                          disabled={!saveSearchName.trim() || savingSearch}
                          className="px-3 py-1.5 text-xs bg-accent/20 text-accent rounded-lg
                                     hover:bg-accent/30 disabled:opacity-50 transition-all"
                        >
                          {savingSearch ? 'Saving...' : 'Save'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
              {!isAuthenticated && hasActiveFilters && (
                <span className="text-xs text-dark-600 whitespace-nowrap shrink-0" title="Sign in to save searches">
                  Sign in to save searches
                </span>
              )}
            </div>
          )}

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
