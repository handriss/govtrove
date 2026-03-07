import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Clock, Bookmark, X, ChevronDown } from 'lucide-react';
import SearchInput from '../components/SearchInput';
import QuickFilterChips from '../components/QuickFilterChips';
import { FilterBar, SearchResults } from '../components/search';
import SearchMobileFilters from '../components/search/SearchMobileFilters';
import SignupPromptModal, { type SignupPromptContext } from '../components/SignupPromptModal';
import SignupNudgeBanner, { incrementAnonSearchCount } from '../components/SignupNudgeBanner';
import { DEFAULT_NOTICE_TYPES } from '../components/filters/constants';
import { useFilterState } from '../hooks/useFilterState';
import { useFacetCounts } from '../hooks/useFacetCounts';
import { useSavedOpportunities } from '../hooks/useSavedOpportunities';
import { useSavedSearches } from '../hooks/useSavedSearches';
import { useDebounce } from '../hooks/useDebounce';
import { useSearch } from '../hooks/useSearch';
import { useAppAuth } from '../contexts/AuthContext';
import { usePostHog } from '@posthog/react';
import { getFacetCounts } from '../services/api';
import { trackSearch, trackSavedSearchCreated } from '../lib/analytics';

const PAGE_SIZE_KEY = 'govtrove_page_size';

export default function SimpleSearchPage() {
  const fs = useFilterState();
  const { facets, total: facetTotal, isLoading: facetsLoading } = useFacetCounts(fs.toFacetParams());
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const authOptions = useMemo(() => ({ getAccessToken }), [getAccessToken]);
  const { results, total, page, totalPages, loading, error, suggestion, search, reset } = useSearch(authOptions);
  const posthog = usePostHog();
  const saved = useSavedOpportunities();
  const { savedSearches, saveCurrentSearch, deleteSearch } = useSavedSearches();
  const [hasSearched, setHasSearched] = useState(false);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);
  const [signupPrompt, setSignupPrompt] = useState<SignupPromptContext | null>(null);
  const [savedSearchesOpen, setSavedSearchesOpen] = useState(false);
  const [saveSearchOpen, setSaveSearchOpen] = useState(false);
  const savedSearchesRef = useRef<HTMLDivElement>(null);
  const [saveSearchName, setSaveSearchName] = useState('');
  const [savingSearch, setSavingSearch] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const isInitialSearch = useRef(true);
  const lastSearchedRef = useRef('');

  // Fetch unfiltered active count once for the hero badge
  const [heroTotal, setHeroTotal] = useState(0);
  useEffect(() => {
    getFacetCounts({})
      .then((r) => setHeroTotal(r.total))
      .catch(() => {});
  }, []);

  const [pageSize, setPageSize] = useState(() => {
    const stored = localStorage.getItem(PAGE_SIZE_KEY);
    return stored ? parseInt(stored, 10) : 25;
  });

  // Close saved searches dropdown on click outside
  useEffect(() => {
    if (!savedSearchesOpen) return;
    function handleClick(e: MouseEvent) {
      if (savedSearchesRef.current && !savedSearchesRef.current.contains(e.target as Node)) {
        setSavedSearchesOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [savedSearchesOpen]);

  // "/" keyboard shortcut to focus search
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const tag = (e.target as HTMLElement).tagName;
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

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
    const nonDefaultNoticeType =
      f.noticeType.length !== DEFAULT_NOTICE_TYPES.length ||
      f.noticeType.slice().sort().join(',') !== DEFAULT_NOTICE_TYPES.slice().sort().join(',');
    return (
      f.keyword.length >= 2 ||
      f.setAside.length > 0 ||
      f.agency.length > 0 ||
      f.state !== '' ||
      f.naics.length > 0 ||
      f.psc.length > 0 ||
      nonDefaultNoticeType ||
      f.deadlinePreset !== '' ||
      f.postedFrom !== '' ||
      f.postedTo !== ''
    );
  }, [fs.filters]);

  // Track search events in PostHog after results arrive
  const lastTrackedRef = useRef('');
  useEffect(() => {
    if (!hasSearched || loading || total === undefined) return;
    const key = JSON.stringify(fs.toSearchParams()) + ':' + total;
    if (key === lastTrackedRef.current) return;
    lastTrackedRef.current = key;
    const params = fs.toSearchParams();
    trackSearch(posthog, params.q || '', { ...params }, total);
  }, [hasSearched, loading, total]); // eslint-disable-line react-hooks/exhaustive-deps

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
    if (!isAuthenticated) incrementAnonSearchCount();
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

  const handleQuerySuggestion = useCallback((suggested: string) => {
    fs.setFilter('keyword', suggested);
    triggerSearch();
  }, [fs, triggerSearch]);

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
      trackSavedSearchCreated(posthog, filterData);
      setSaveSearchOpen(false);
      setSaveSearchName('');
    } catch {
      // ignore
    } finally {
      setSavingSearch(false);
    }
  }, [saveSearchName, fs.filters, saveCurrentSearch]);

  const handleSavedSearchClick = useCallback((filters: Record<string, unknown>) => {
    const normalized = { ...filters };
    // Legacy saved searches stored agency as `department` string — normalize to `agency` array
    if (!normalized.agency && normalized.department) {
      normalized.agency = [String(normalized.department)];
      delete normalized.department;
    }
    fs.setFilters(normalized as Partial<typeof fs.filters>);
  }, [fs]);

  const showResults = hasSearched || results.length > 0;

  return (
    <div className="min-h-screen flex flex-col relative">
      <a href="#search-results" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:top-4 focus:left-4 focus:px-4 focus:py-2 focus:bg-accent focus:text-white focus:rounded-lg">Skip to search results</a>
      {/* Ambient glow */}
      <div className="fixed top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-accent/[0.03] rounded-full blur-3xl pointer-events-none" />


      <div className={`relative z-10 flex flex-col items-center transition-all duration-500 ease-out ${showResults ? 'pt-10' : 'flex-1 justify-center pb-24'}`}>
        {/* Logo */}
        <Link to="/" className={`transition-all duration-500 ${showResults ? 'mb-6' : 'mb-8'}`}>
          <h1 className={`text-center font-semibold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-dark-50 to-dark-300 transition-all duration-500 ${showResults ? 'text-2xl' : 'text-5xl'}`}>
            GovTrove
          </h1>
          {!showResults && (
            <>
              <p className="text-center text-dark-400 text-sm mt-2 tracking-wide">
                Government Contract Intelligence
              </p>
              {heroTotal > 0 && (
                <div className="flex items-center justify-center gap-1.5 mt-3">
                  <span className="inline-flex items-center gap-1.5 text-dark-300 text-xs bg-dark-800/60 border border-dark-700/30 px-3 py-1 rounded-full">
                    <span className="w-1.5 h-1.5 rounded-full bg-success" />
                    {heroTotal.toLocaleString()} active opportunities
                  </span>
                </div>
              )}
            </>
          )}
        </Link>

        {/* Hero mode: large search + quick filters */}
        {!showResults && (
          <>
            <div className="w-full max-w-2xl px-6 mb-8">
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

            <div className="text-center space-y-4 animate-in fade-in duration-500">
              <div className="flex flex-wrap items-center justify-center gap-2">
                <button
                  onClick={() => {
                    const yesterday = new Date();
                    yesterday.setDate(yesterday.getDate() - 1);
                    fs.setFilter('postedFrom', yesterday.toISOString().split('T')[0]);
                  }}
                  className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full text-sm font-medium
                             border border-accent/40 bg-accent/20 text-accent
                             hover:bg-accent/30 hover:border-accent/50 transition-all duration-200
                             shadow-md shadow-accent/15"
                >
                  <Clock size={14} strokeWidth={1.5} />
                  What's New Today?
                </button>
              </div>

              <QuickFilterChips filters={fs.filters} setFilter={fs.setFilter} onSearch={triggerSearch} />
            </div>
          </>
        )}
      </div>

      {/* Empty state footer hint */}
      {!showResults && (
        <div className="absolute bottom-20 md:bottom-6 left-0 right-0 z-10 text-center">
          <p className="text-dark-600 text-xs tracking-wide">
            Data sourced from SAM.gov &middot; Press <kbd className="px-1.5 py-0.5 rounded bg-dark-800/60 border border-dark-700/30 text-dark-500 font-mono text-[10px]">/</kbd> to search
          </p>
          <Link to="/guide" className="inline-block mt-2 text-dark-500 hover:text-dark-300 text-xs transition-colors">
            New here? Check out the Search Guide &rarr;
          </Link>
        </div>
      )}

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

          {/* Saved searches dropdown + save button */}
          {(savedSearches.length > 0 || hasActiveFilters) && (
            <div className="flex items-center gap-2 mb-3">
              {savedSearches.length > 0 && (
                <div className="relative" ref={savedSearchesRef}>
                  <button
                    onClick={() => setSavedSearchesOpen(!savedSearchesOpen)}
                    className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs
                               border transition-all whitespace-nowrap ${
                               savedSearchesOpen
                                 ? 'border-dark-600/50 bg-dark-800/50 text-dark-100'
                                 : 'border-dark-700/50 bg-dark-800/30 text-dark-300 hover:border-dark-600/50 hover:text-dark-100'
                               }`}
                  >
                    <Bookmark size={12} />
                    Saved Searches
                    <span className="text-dark-500 tabular-nums">({savedSearches.length})</span>
                    <ChevronDown size={12} className={`transition-transform ${savedSearchesOpen ? 'rotate-180' : ''}`} />
                  </button>
                  {savedSearchesOpen && (
                    <div className="absolute left-0 top-full mt-2 w-72 bg-dark-900 border border-dark-700/50 rounded-xl shadow-xl z-50 overflow-hidden">
                      <div className="max-h-64 overflow-y-auto py-1">
                        {savedSearches.map((ss) => (
                          <button
                            key={ss.id}
                            onClick={() => { handleSavedSearchClick(ss.filters); setSavedSearchesOpen(false); }}
                            className="group w-full flex items-center gap-2 px-3 py-2 text-left
                                       hover:bg-dark-800/50 transition-colors"
                          >
                            <span className="flex-1 min-w-0 text-sm text-dark-300 group-hover:text-dark-100 truncate">
                              {ss.name}
                            </span>
                            {ss.last_match_count > 0 && (
                              <span className="text-[10px] font-medium text-accent bg-accent/10 px-1.5 py-0.5 rounded-full shrink-0">
                                +{ss.last_match_count}
                              </span>
                            )}
                            <span
                              role="button"
                              onClick={(e) => { e.stopPropagation(); deleteSearch(ss.id); }}
                              className="opacity-0 group-hover:opacity-100 text-dark-600 hover:text-red-400 transition-all p-0.5 shrink-0"
                            >
                              <X size={12} />
                            </span>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
              {isAuthenticated && hasActiveFilters && (
                <div className="relative shrink-0">
                  <button
                    onClick={() => setSaveSearchOpen(!saveSearchOpen)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs
                               border border-accent/30 bg-accent/10 text-accent
                               hover:bg-accent/20 transition-all whitespace-nowrap"
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
                <button
                  onClick={() => setSignupPrompt('save-search')}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs shrink-0
                             border border-accent/30 bg-accent/10 text-accent
                             hover:bg-accent/20 transition-all whitespace-nowrap"
                >
                  <Bookmark size={12} />
                  Save Search
                </button>
              )}
            </div>
          )}

          <SignupNudgeBanner />

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
            querySuggestion={suggestion}
            onQuerySuggestionClick={handleQuerySuggestion}
            clearAllFilters={fs.clearAllFilters}
            error={error}
            onRetry={handleSubmit}
            isAuthenticated={isAuthenticated}
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

      {signupPrompt && (
        <SignupPromptModal context={signupPrompt} onClose={() => setSignupPrompt(null)} />
      )}
    </div>
  );
}
