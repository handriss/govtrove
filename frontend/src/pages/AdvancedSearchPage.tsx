import { useState, useCallback, useEffect, useRef } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Search, SlidersHorizontal, X, Lightbulb, ChevronDown, Bookmark, Trash2, Clock } from 'lucide-react';
import QueryBuilder, { buildQueryString, createEmptyGroup } from '../components/QueryBuilder';
import FilterPanel from '../components/FilterPanel';
import ResultsList from '../components/ResultsList';
import AuthButton from '../components/AuthButton';
import { useSearch } from '../hooks/useSearch';
import SetAsideChips from '../components/SetAsideChips';
import type { QueryGroup, AdvancedFilters, SearchParams } from '../types/api';

const SAVED_SEARCHES_KEY = 'govtrove_saved_searches';

const DEFAULT_TYPES = ['Solicitation', 'Presolicitation', 'Combined Synopsis/Solicitation', 'Sources Sought'];

interface SavedSearch {
  id: string;
  name: string;
  params: Record<string, string>;
  createdAt: number;
}

interface ExampleSearch {
  title: string;
  description: string;
  params: Record<string, string>;
}

const exampleSearches: ExampleSearch[] = [
  {
    title: 'Active Solicitations',
    description: 'All open solicitations accepting bids',
    params: { type: 'o' },
  },
  {
    title: 'Small Business Set-Asides',
    description: 'Opportunities reserved for small businesses',
    params: { set_aside: 'SBA,SB' },
  },
  {
    title: 'IT Services in California',
    description: 'Information technology contracts in CA',
    params: { q: 'IT services', state: 'CA', naics: '541512' },
  },
  {
    title: '8(a) Program Opportunities',
    description: 'Set-asides for 8(a) certified businesses',
    params: { set_aside: '8A' },
  },
  {
    title: 'Cybersecurity Contracts',
    description: 'Search for cybersecurity related opportunities',
    params: { q: 'cybersecurity OR "cyber security" OR "information security"' },
  },
  {
    title: 'Construction in Texas',
    description: 'Construction and building contracts in TX',
    params: { q: 'construction', state: 'TX', naics: '236220' },
  },
  {
    title: 'Service-Disabled Veteran Owned',
    description: 'SDVOSB set-aside opportunities',
    params: { set_aside: 'SDVOSB' },
  },
  {
    title: 'Healthcare & Medical',
    description: 'Medical supplies and healthcare services',
    params: { q: 'medical OR healthcare OR "health care"', naics: '621999,339112' },
  },
  {
    title: 'Sources Sought Notices',
    description: 'Market research - agencies seeking information',
    params: { type: 'r' },
  },
  {
    title: 'Award Notices',
    description: 'Recently awarded contracts',
    params: { type: 'a' },
  },
  {
    title: 'Complex: Defense IT in Multiple States',
    description: 'Defense-related IT services in CA, TX, or VA',
    params: {
      q: 'defense IT OR "defense information technology"',
      state: 'CA,TX,VA',
      type: 'o,k',
      naics: '541512,541519',
    },
  },
  {
    title: 'Complex: Small Business Construction',
    description: 'Small business construction set-asides excluding maintenance',
    params: {
      q: 'construction -maintenance -janitorial',
      set_aside: 'SBA,SB',
      type: 'o,k',
    },
  },
];

function loadSavedSearches(): SavedSearch[] {
  try {
    const stored = localStorage.getItem(SAVED_SEARCHES_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
}

function saveSavedSearches(searches: SavedSearch[]) {
  localStorage.setItem(SAVED_SEARCHES_KEY, JSON.stringify(searches));
}

function parseFiltersFromParams(searchParams: URLSearchParams): AdvancedFilters {
  const hasTypeParam = searchParams.has('type');
  return {
    types: hasTypeParam ? (searchParams.get('type')?.split(',').filter(Boolean) || []) : DEFAULT_TYPES,
    setAsides: searchParams.get('set_aside')?.split(',').filter(Boolean) || [],
    naicsCodes: searchParams.get('naics')?.split(',').filter(Boolean) || [],
    states: searchParams.get('state')?.split(',').filter(Boolean) || [],
    postedFrom: searchParams.get('posted_from') || undefined,
    postedTo: searchParams.get('posted_to') || undefined,
    deadlineFrom: searchParams.get('deadline_from') || undefined,
    deadlineTo: searchParams.get('deadline_to') || undefined,
  };
}

function createGroupFromQuery(query: string): QueryGroup[] {
  if (!query) return [createEmptyGroup()];
  return [
    {
      id: Math.random().toString(36).substring(2, 9),
      operator: 'AND',
      terms: [{ id: Math.random().toString(36).substring(2, 9), value: query, type: 'include' }],
    },
  ];
}

export default function AdvancedSearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialQuery = searchParams.get('q') || '';
  const initialFilters = parseFiltersFromParams(searchParams);

  const [groups, setGroups] = useState<QueryGroup[]>(createGroupFromQuery(initialQuery));
  const [groupOperator, setGroupOperator] = useState<'AND' | 'OR'>('AND');
  const [filters, setFilters] = useState<AdvancedFilters>(initialFilters);
  const [showFilters, setShowFilters] = useState(false);
  const { results, total, page, totalPages, loading, search } = useSearch();
  const [hasSearched, setHasSearched] = useState(false);
  const [savedSearches, setSavedSearches] = useState<SavedSearch[]>(loadSavedSearches);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [saveName, setSaveName] = useState('');
  const examplesRef = useRef<HTMLDivElement>(null);
  const isInitialMount = useRef(true);

  const isDefaultTypes = filters.types.length === DEFAULT_TYPES.length &&
    DEFAULT_TYPES.every((t) => filters.types.includes(t));
  const activeFilterCount = [
    isDefaultTypes ? 0 : filters.types.length,
    filters.setAsides.length,
    filters.naicsCodes.length,
    filters.states.length,
    filters.postedFrom ? 1 : 0,
    filters.postedTo ? 1 : 0,
    filters.deadlineFrom ? 1 : 0,
    filters.deadlineTo ? 1 : 0,
  ].reduce((a, b) => a + b, 0);

  const updateURL = useCallback(
    (query: string, currentFilters: AdvancedFilters) => {
      const params = new URLSearchParams();
      if (query) params.set('q', query);
      if (currentFilters.types.length) params.set('type', currentFilters.types.join(','));
      if (currentFilters.setAsides.length) params.set('set_aside', currentFilters.setAsides.join(','));
      if (currentFilters.naicsCodes.length) params.set('naics', currentFilters.naicsCodes.join(','));
      if (currentFilters.states.length) params.set('state', currentFilters.states.join(','));
      if (currentFilters.postedFrom) params.set('posted_from', currentFilters.postedFrom);
      if (currentFilters.postedTo) params.set('posted_to', currentFilters.postedTo);
      if (currentFilters.deadlineFrom) params.set('deadline_from', currentFilters.deadlineFrom);
      if (currentFilters.deadlineTo) params.set('deadline_to', currentFilters.deadlineTo);
      setSearchParams(params, { replace: true });
    },
    [setSearchParams]
  );

  const buildSearchParams = useCallback(
    (pageNum = 1): SearchParams => {
      const q = buildQueryString(groups, groupOperator);
      return {
        q: q || undefined,
        type: filters.types.length ? filters.types.join(',') : undefined,
        set_aside: filters.setAsides.length ? filters.setAsides.join(',') : undefined,
        naics: filters.naicsCodes.length ? filters.naicsCodes.join(',') : undefined,
        state: filters.states.length ? filters.states.join(',') : undefined,
        posted_from: filters.postedFrom,
        posted_to: filters.postedTo,
        deadline_from: filters.deadlineFrom,
        deadline_to: filters.deadlineTo,
        sort: q ? 'relevance' : 'posted_date',
        order: 'desc',
        page: pageNum,
        limit: 25,
      };
    },
    [groups, groupOperator, filters]
  );

  const handleSearch = useCallback(() => {
    const q = buildQueryString(groups, groupOperator);
    updateURL(q, filters);
    search(buildSearchParams(1));
    setHasSearched(true);
  }, [groups, groupOperator, filters, updateURL, search, buildSearchParams]);

  const handleWhatsNew = useCallback(() => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const postedFrom = yesterday.toISOString().split('T')[0];
    const newFilters = { ...filters, postedFrom, postedTo: undefined };
    setFilters(newFilters);
    const q = buildQueryString(groups, groupOperator);
    updateURL(q, newFilters);
    search({
      q: q || undefined,
      type: newFilters.types.length ? newFilters.types.join(',') : undefined,
      set_aside: newFilters.setAsides.length ? newFilters.setAsides.join(',') : undefined,
      naics: newFilters.naicsCodes.length ? newFilters.naicsCodes.join(',') : undefined,
      state: newFilters.states.length ? newFilters.states.join(',') : undefined,
      posted_from: postedFrom,
      sort: 'posted_date',
      order: 'desc',
      page: 1,
      limit: 25,
    });
    setHasSearched(true);
  }, [filters, groups, groupOperator, updateURL, search]);

  // Run initial search if URL has explicit params (default types don't count)
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      const hasExplicitParams = initialQuery || searchParams.toString().length > 0;
      if (hasExplicitParams) {
        search(buildSearchParams(1));
        setHasSearched(true);
      }
    }
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSearch();
    }
  };

  const handlePageChange = (newPage: number) => {
    search(buildSearchParams(newPage));
  };

  const handleExampleClick = (example: ExampleSearch) => {
    const newParams = new URLSearchParams(example.params);
    setSearchParams(newParams);

    // Update local state
    const newQuery = example.params.q || '';
    setGroups(createGroupFromQuery(newQuery));
    setFilters(parseFiltersFromParams(newParams));

    // Execute search
    const searchParamsObj: SearchParams = {
      q: example.params.q || undefined,
      type: example.params.type || undefined,
      set_aside: example.params.set_aside || undefined,
      naics: example.params.naics || undefined,
      state: example.params.state || undefined,
      posted_from: example.params.posted_from || undefined,
      posted_to: example.params.posted_to || undefined,
      deadline_from: example.params.deadline_from || undefined,
      deadline_to: example.params.deadline_to || undefined,
      sort: example.params.q ? 'relevance' : 'posted_date',
      order: 'desc',
      page: 1,
      limit: 25,
    };
    search(searchParamsObj);
    setHasSearched(true);

    // Scroll to top
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const scrollToExamples = () => {
    examplesRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const getCurrentParams = (): Record<string, string> => {
    const params: Record<string, string> = {};
    const q = buildQueryString(groups, groupOperator);
    if (q) params.q = q;
    if (filters.types.length) params.type = filters.types.join(',');
    if (filters.setAsides.length) params.set_aside = filters.setAsides.join(',');
    if (filters.naicsCodes.length) params.naics = filters.naicsCodes.join(',');
    if (filters.states.length) params.state = filters.states.join(',');
    if (filters.postedFrom) params.posted_from = filters.postedFrom;
    if (filters.postedTo) params.posted_to = filters.postedTo;
    if (filters.deadlineFrom) params.deadline_from = filters.deadlineFrom;
    if (filters.deadlineTo) params.deadline_to = filters.deadlineTo;
    return params;
  };

  const handleSaveSearch = () => {
    if (!saveName.trim()) return;
    const newSearch: SavedSearch = {
      id: Math.random().toString(36).substring(2, 9),
      name: saveName.trim(),
      params: getCurrentParams(),
      createdAt: Date.now(),
    };
    const updated = [newSearch, ...savedSearches];
    setSavedSearches(updated);
    saveSavedSearches(updated);
    setSaveName('');
    setShowSaveDialog(false);
  };

  const handleDeleteSaved = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const updated = savedSearches.filter((s) => s.id !== id);
    setSavedSearches(updated);
    saveSavedSearches(updated);
  };

  const handleLoadSaved = (saved: SavedSearch) => {
    const newParams = new URLSearchParams(saved.params);
    setSearchParams(newParams);

    const newQuery = saved.params.q || '';
    setGroups(createGroupFromQuery(newQuery));
    setFilters(parseFiltersFromParams(newParams));

    const searchParamsObj: SearchParams = {
      q: saved.params.q || undefined,
      type: saved.params.type || undefined,
      set_aside: saved.params.set_aside || undefined,
      naics: saved.params.naics || undefined,
      state: saved.params.state || undefined,
      posted_from: saved.params.posted_from || undefined,
      posted_to: saved.params.posted_to || undefined,
      deadline_from: saved.params.deadline_from || undefined,
      deadline_to: saved.params.deadline_to || undefined,
      sort: saved.params.q ? 'relevance' : 'posted_date',
      order: 'desc',
      page: 1,
      limit: 25,
    };
    search(searchParamsObj);
    setHasSearched(true);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const queryString = buildQueryString(groups, groupOperator);

  return (
    <div className="min-h-screen relative flex flex-col">
      {/* Background */}
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      {/* Header */}
      <header className="relative z-20 border-b border-dark-800/50 bg-dark-950/80 backdrop-blur-md sticky top-0">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-5">
            <Link
              to="/"
              className="p-2 -ml-2 text-dark-400 hover:text-dark-100 rounded-lg hover:bg-dark-800/50 transition-all duration-200"
            >
              <ArrowLeft size={20} strokeWidth={1.5} />
            </Link>
            <Link to="/" className="text-xl font-semibold tracking-tight text-dark-50">
              GovTrove
            </Link>
          </div>
          <div className="flex items-center gap-4">
            <button
              onClick={scrollToExamples}
              className="hidden sm:flex items-center gap-1.5 text-xs text-dark-500 hover:text-dark-300 transition-colors"
            >
              <Lightbulb size={14} strokeWidth={1.5} />
              Examples
            </button>
            <span className="text-sm text-dark-500 tracking-wide">Advanced Search</span>
            <AuthButton />
          </div>
        </div>
      </header>

      {/* Search Bar - Always Visible */}
      <div className="relative z-10 border-b border-dark-800/30 bg-dark-950/50 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center gap-3">
            {/* Query Builder Compact View */}
            <div className="flex-1" onKeyDown={handleKeyDown}>
              <QueryBuilder
                groups={groups}
                onChange={setGroups}
                groupOperator={groupOperator}
                onGroupOperatorChange={setGroupOperator}
              />
            </div>

            {/* Filter Toggle */}
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`relative flex items-center gap-2 px-4 py-2.5 rounded-xl border text-sm font-medium transition-all duration-200 ${
                showFilters || activeFilterCount > 0
                  ? 'bg-accent/10 border-accent/30 text-accent hover:bg-accent/20'
                  : 'border-dark-700/50 bg-dark-800/30 text-dark-300 hover:text-dark-100 hover:border-dark-600/50'
              }`}
            >
              <SlidersHorizontal size={16} strokeWidth={1.5} />
              <span className="hidden sm:inline">Filters</span>
              {activeFilterCount > 0 && (
                <span className="absolute -top-1.5 -right-1.5 w-5 h-5 bg-accent text-white text-[10px] font-bold rounded-full flex items-center justify-center">
                  {activeFilterCount}
                </span>
              )}
            </button>

            {/* Search Button */}
            <button
              onClick={handleSearch}
              disabled={loading}
              className="flex items-center gap-2 px-6 py-2.5 bg-accent hover:bg-accent-hover
                         text-white font-medium rounded-xl
                         transition-all duration-200
                         hover:shadow-lg hover:shadow-accent/20
                         disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Search size={18} strokeWidth={1.5} />
              <span className="hidden sm:inline">Search</span>
            </button>
          </div>

          {/* Query Preview */}
          {queryString && (
            <div className="mt-3 flex items-center gap-2">
              <span className="text-[10px] text-dark-500 uppercase tracking-wider">Query:</span>
              <code className="text-xs text-accent font-mono">{queryString}</code>
            </div>
          )}

          {/* Quick Filters Row */}
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <button
              onClick={handleWhatsNew}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm
                         border border-accent/30 bg-accent/10 text-accent
                         hover:bg-accent/20 transition-all duration-200"
            >
              <Clock size={14} strokeWidth={1.5} />
              What's New Today?
            </button>
            <div className="w-px h-5 bg-dark-800/50 mx-1 hidden sm:block" />
            <SetAsideChips
              selected={filters.setAsides}
              onChange={(setAsides) => setFilters({ ...filters, setAsides })}
            />
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="relative z-10 flex-1 flex">
        {/* Filter Panel - Slide Out */}
        {showFilters && (
          <>
            {/* Backdrop for mobile */}
            <div
              className="fixed inset-0 bg-dark-950/50 backdrop-blur-sm lg:hidden z-30"
              onClick={() => setShowFilters(false)}
            />

            {/* Filter Panel */}
            <div className="fixed lg:relative right-0 top-0 lg:top-auto h-full lg:h-auto w-80 lg:w-72 bg-dark-950 lg:bg-transparent border-l border-dark-800/50 lg:border-0 z-40 lg:z-auto overflow-y-auto">
              <div className="p-4 lg:p-6 lg:pt-8">
                <div className="flex items-center justify-between mb-4 lg:hidden">
                  <h3 className="text-sm font-medium text-dark-200">Filters</h3>
                  <button
                    onClick={() => setShowFilters(false)}
                    className="p-1.5 text-dark-400 hover:text-dark-100 rounded-lg hover:bg-dark-800/50"
                  >
                    <X size={18} strokeWidth={1.5} />
                  </button>
                </div>
                <FilterPanel filters={filters} onChange={setFilters} />
              </div>
            </div>
          </>
        )}

        {/* Results Area */}
        <div className="flex-1 max-w-7xl mx-auto px-6 py-6">
          {!hasSearched ? (
            <div className="flex flex-col items-center justify-center min-h-[400px] py-12 text-center">
              {savedSearches.length > 0 ? (
                <>
                  <div className="w-full max-w-2xl mb-10">
                    <div className="flex items-center gap-2 mb-4">
                      <Bookmark size={16} className="text-accent" strokeWidth={1.5} />
                      <h3 className="text-sm font-medium text-dark-200">Saved Searches</h3>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      {savedSearches.map((saved) => (
                        <button
                          key={saved.id}
                          onClick={() => handleLoadSaved(saved)}
                          className="group relative text-left p-3 rounded-xl border border-dark-800/50 bg-dark-900/30
                                     hover:border-dark-700/50 hover:bg-dark-800/30 transition-all duration-200"
                        >
                          <p className="text-sm font-medium text-dark-200 group-hover:text-dark-50 pr-6">
                            {saved.name}
                          </p>
                          <p className="text-[11px] text-dark-500 mt-1 font-mono truncate">
                            {saved.params.q || Object.keys(saved.params).join(', ')}
                          </p>
                          <button
                            onClick={(e) => handleDeleteSaved(saved.id, e)}
                            className="absolute top-3 right-3 p-1 text-dark-600 hover:text-red-400
                                       opacity-0 group-hover:opacity-100 transition-all duration-150"
                            title="Delete saved search"
                          >
                            <Trash2 size={14} strokeWidth={1.5} />
                          </button>
                        </button>
                      ))}
                    </div>
                  </div>
                  <div className="w-full max-w-2xl border-t border-dark-800/30 pt-8">
                    <p className="text-dark-500 text-sm mb-6">Or start a new search</p>
                  </div>
                </>
              ) : (
                <>
                  <div className="w-16 h-16 rounded-2xl bg-dark-800/30 border border-dark-700/20 flex items-center justify-center mb-6">
                    <Search size={28} className="text-dark-600" strokeWidth={1.5} />
                  </div>
                  <h3 className="text-lg font-medium text-dark-300 mb-2">Ready to search</h3>
                  <p className="text-dark-500 text-sm mb-8 max-w-sm leading-relaxed">
                    Build your query using the search builder above, add filters to narrow results, then click Search.
                  </p>
                </>
              )}

              <div className="grid grid-cols-3 gap-6 mb-10 text-center">
                <div className="flex flex-col items-center gap-2">
                  <div className="w-10 h-10 rounded-xl bg-dark-800/20 border border-dark-700/20 flex items-center justify-center">
                    <span className="text-dark-500 text-sm font-mono">" "</span>
                  </div>
                  <span className="text-[11px] text-dark-500">Exact phrases</span>
                </div>
                <div className="flex flex-col items-center gap-2">
                  <div className="w-10 h-10 rounded-xl bg-dark-800/20 border border-dark-700/20 flex items-center justify-center">
                    <span className="text-dark-500 text-xs font-medium">OR</span>
                  </div>
                  <span className="text-[11px] text-dark-500">Alternatives</span>
                </div>
                <div className="flex flex-col items-center gap-2">
                  <div className="w-10 h-10 rounded-xl bg-dark-800/20 border border-dark-700/20 flex items-center justify-center">
                    <span className="text-dark-500 text-sm font-mono">-</span>
                  </div>
                  <span className="text-[11px] text-dark-500">Exclusions</span>
                </div>
              </div>

              <button
                onClick={scrollToExamples}
                className="flex items-center gap-2 text-xs text-accent hover:text-accent-hover transition-colors"
              >
                <ChevronDown size={14} strokeWidth={1.5} />
                View example searches for inspiration
              </button>
            </div>
          ) : (
            <>
              {/* Results Header */}
              <div className="flex items-center justify-between mb-4 pb-3 border-b border-dark-800/30">
                <p className="text-sm text-dark-400">
                  <span className="text-dark-200 font-medium">{total.toLocaleString()}</span> results
                  {queryString && (
                    <span className="ml-2 text-dark-500">
                      for <span className="text-accent font-mono text-xs">{queryString}</span>
                    </span>
                  )}
                </p>
                <div className="relative">
                  <button
                    onClick={() => setShowSaveDialog(!showSaveDialog)}
                    className="flex items-center gap-1.5 text-xs text-dark-400 hover:text-dark-200 transition-colors"
                  >
                    <Bookmark size={14} strokeWidth={1.5} />
                    Save search
                  </button>
                  {showSaveDialog && (
                    <div className="absolute right-0 top-full mt-2 w-64 p-3 bg-dark-900 border border-dark-700/50 rounded-xl shadow-xl z-50">
                      <input
                        type="text"
                        value={saveName}
                        onChange={(e) => setSaveName(e.target.value)}
                        placeholder="Name this search..."
                        className="w-full py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                                   text-dark-100 placeholder:text-dark-500
                                   focus:outline-none focus:border-accent/50 transition-colors duration-200"
                        autoFocus
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') handleSaveSearch();
                          if (e.key === 'Escape') setShowSaveDialog(false);
                        }}
                      />
                      <div className="flex gap-2 mt-2">
                        <button
                          onClick={() => setShowSaveDialog(false)}
                          className="flex-1 py-1.5 text-xs text-dark-400 hover:text-dark-200 transition-colors"
                        >
                          Cancel
                        </button>
                        <button
                          onClick={handleSaveSearch}
                          disabled={!saveName.trim()}
                          className="flex-1 py-1.5 text-xs bg-accent hover:bg-accent-hover text-white rounded-lg
                                     disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                          Save
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              </div>

              <ResultsList
                results={results}
                page={page}
                totalPages={totalPages}
                loading={loading}
                query={queryString}
                onPageChange={handlePageChange}
              />
            </>
          )}
        </div>
      </div>

      {/* Examples Section */}
      <div ref={examplesRef} className="relative z-10 border-t border-dark-800/50 bg-dark-950/30">
        <div className="max-w-7xl mx-auto px-6 py-12">
          <div className="flex items-center gap-3 mb-8">
            <Lightbulb size={20} className="text-accent" strokeWidth={1.5} />
            <h2 className="text-lg font-semibold text-dark-100">Example Searches</h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {exampleSearches.map((example, idx) => (
              <button
                key={idx}
                onClick={() => handleExampleClick(example)}
                className="text-left p-4 rounded-xl border border-dark-800/50 bg-dark-900/30
                           hover:border-dark-700/50 hover:bg-dark-800/30 transition-all duration-200 group"
              >
                <h3 className="text-sm font-medium text-dark-200 group-hover:text-dark-50 mb-1">
                  {example.title}
                </h3>
                <p className="text-xs text-dark-500 leading-relaxed">{example.description}</p>
              </button>
            ))}
          </div>

          <p className="mt-8 text-xs text-dark-600 text-center">
            Click any example to load it into the search. You can modify the query and filters before searching.
          </p>
        </div>
      </div>
    </div>
  );
}
