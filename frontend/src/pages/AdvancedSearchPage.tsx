import { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Search, SlidersHorizontal, X, Lightbulb, ChevronDown, Bookmark } from 'lucide-react';
import QueryBuilder, { buildQueryString, createEmptyGroup } from '../components/QueryBuilder';
import FilterPanel from '../components/FilterPanel';
import ResultsList from '../components/ResultsList';
import AuthButton from '../components/AuthButton';
import { useSearch } from '../hooks/useSearch';
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


function deadlineToDate(preset: string): string | undefined {
  const days = parseInt(preset, 10);
  if (!days) return undefined;
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().split('T')[0];
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
  const [department, setDepartment] = useState(searchParams.get('department') || '');
  const [titleFilter, setTitleFilter] = useState('');
  const [deadlinePreset, setDeadlinePreset] = useState(searchParams.get('deadline') || '');
  const [showFilters, setShowFilters] = useState(false);
  const { results, total, page, totalPages, loading, search } = useSearch();
  const [hasSearched, setHasSearched] = useState(false);
  const [sort, setSort] = useState(searchParams.get('sort') || '');
  const [order, setOrder] = useState(searchParams.get('order') || 'desc');
  const [savedSearches] = useState<SavedSearch[]>(loadSavedSearches);
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
    (query: string, currentFilters: AdvancedFilters, s?: string, o?: string, dept?: string, dl?: string) => {
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
      if (dept) params.set('department', dept);
      if (s) params.set('sort', s);
      if (o && o !== 'desc') params.set('order', o);
      if (dl) params.set('deadline', dl);
      setSearchParams(params, { replace: true });
    },
    [setSearchParams]
  );

  const buildSearchParams = useCallback(
    (pageNum = 1, s?: string, o?: string, dept?: string): SearchParams => {
      const q = buildQueryString(groups, groupOperator);
      const activeSort = s ?? sort;
      const activeOrder = o ?? order;
      const activeDept = dept ?? department;
      return {
        q: q || undefined,
        type: filters.types.length ? filters.types.join(',') : undefined,
        set_aside: filters.setAsides.length ? filters.setAsides.join(',') : undefined,
        naics: filters.naicsCodes.length ? filters.naicsCodes.join(',') : undefined,
        state: filters.states.length ? filters.states.join(',') : undefined,
        department: activeDept || undefined,
        posted_from: filters.postedFrom,
        posted_to: filters.postedTo,
        deadline_from: filters.deadlineFrom,
        deadline_to: filters.deadlineTo,
        sort: activeSort || (q ? 'relevance' : 'posted_date'),
        order: activeOrder || 'desc',
        page: pageNum,
        limit: 25,
      };
    },
    [groups, groupOperator, filters, sort, order, department]
  );

  const handleSearch = useCallback(() => {
    const q = buildQueryString(groups, groupOperator);
    updateURL(q, filters, sort, order, department, deadlinePreset);
    search(buildSearchParams(1));
    setHasSearched(true);
  }, [groups, groupOperator, filters, updateURL, search, buildSearchParams, sort, order, department, deadlinePreset]);

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

  const handleSortChange = useCallback((newSort: string, newOrder: string) => {
    setSort(newSort);
    setOrder(newOrder);
    const q = buildQueryString(groups, groupOperator);
    updateURL(q, filters, newSort, newOrder, department, deadlinePreset);
    search(buildSearchParams(1, newSort, newOrder));
  }, [groups, groupOperator, filters, updateURL, search, buildSearchParams, department, deadlinePreset]);

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
      sort: sort || (example.params.q ? 'relevance' : 'posted_date'),
      order: order || 'desc',
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



  const searchWithFilters = useCallback((newFilters: AdvancedFilters, dept?: string, dl?: string) => {
    const q = buildQueryString(groups, groupOperator);
    const activeDept = dept ?? department;
    const activeDl = dl ?? deadlinePreset;
    updateURL(q, newFilters, sort, order, activeDept, activeDl);
    const params: SearchParams = {
      q: q || undefined,
      type: newFilters.types.length ? newFilters.types.join(',') : undefined,
      set_aside: newFilters.setAsides.length ? newFilters.setAsides.join(',') : undefined,
      naics: newFilters.naicsCodes.length ? newFilters.naicsCodes.join(',') : undefined,
      state: newFilters.states.length ? newFilters.states.join(',') : undefined,
      department: activeDept || undefined,
      posted_from: newFilters.postedFrom,
      posted_to: newFilters.postedTo,
      deadline_from: newFilters.deadlineFrom,
      deadline_to: newFilters.deadlineTo,
      sort: sort || (q ? 'relevance' : 'posted_date'),
      order: order || 'desc',
      page: 1,
      limit: 25,
    };
    search(params);
  }, [groups, groupOperator, department, deadlinePreset, sort, order, updateURL, search]);

  const handleFilterChange = useCallback((field: string, value: string) => {
    if (field === 'title') {
      setTitleFilter(value);
      return;
    }
    if (field === 'deadline') {
      setDeadlinePreset(value);
      const newFilters = { ...filters, deadlineTo: deadlineToDate(value) };
      setFilters(newFilters);
      searchWithFilters(newFilters, undefined, value);
      return;
    }
    if (field === 'department') {
      setDepartment(value);
      searchWithFilters(filters, value);
      return;
    }
    if (field === 'set_aside') {
      const newFilters = { ...filters, setAsides: value ? value.split(',') : [] };
      setFilters(newFilters);
      searchWithFilters(newFilters);
      return;
    }
    if (field === 'state') {
      const newFilters = { ...filters, states: value ? value.split(',') : [] };
      setFilters(newFilters);
      searchWithFilters(newFilters);
      return;
    }
  }, [filters, searchWithFilters]);

  const handleNaicsChange = useCallback((codes: string[]) => {
    const newFilters = { ...filters, naicsCodes: codes };
    setFilters(newFilters);
    searchWithFilters(newFilters);
  }, [filters, searchWithFilters]);

  const agencies = useMemo(() =>
    [...new Set(results.map((o) => o.department).filter(Boolean) as string[])].sort(),
    [results],
  );

  const columnFilters: Record<string, string> = useMemo(() => ({
    title: titleFilter,
    department: department,
    set_aside: filters.setAsides.join(','),
    deadline: deadlinePreset,
    state: filters.states.join(','),
    naics: filters.naicsCodes.join(','),
  }), [titleFilter, department, filters.setAsides, deadlinePreset, filters.states, filters.naicsCodes]);

  const filteredResults = useMemo(() => {
    if (!titleFilter) return results;
    const lower = titleFilter.toLowerCase();
    return results.filter((r) => r.title.toLowerCase().includes(lower));
  }, [results, titleFilter]);

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
        <div className="flex-1 max-w-[1400px] mx-auto px-6 py-6">
          {!hasSearched ? (
            <div className="flex flex-col items-center justify-center min-h-[400px] py-12 text-center">
              {savedSearches.length > 0 ? (
                <div className="w-full max-w-2xl mb-10 opacity-50 pointer-events-none">
                  <div className="rounded-xl border border-fuchsia-500 bg-fuchsia-500/10 p-4">
                    <div className="flex items-center gap-2 mb-4">
                      <Bookmark size={16} className="text-fuchsia-400" strokeWidth={1.5} />
                      <h3 className="text-sm font-medium text-fuchsia-400">Saved Searches — coming soon</h3>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      {savedSearches.map((saved) => (
                        <div
                          key={saved.id}
                          className="text-left p-3 rounded-xl border border-dark-800/50 bg-dark-900/30"
                        >
                          <p className="text-sm font-medium text-dark-400 pr-6">
                            {saved.name}
                          </p>
                          <p className="text-[11px] text-dark-500 mt-1 font-mono truncate">
                            {saved.params.q || Object.keys(saved.params).join(', ')}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
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
                <span className="flex items-center gap-1.5 text-xs text-fuchsia-400 opacity-50">
                  <Bookmark size={14} strokeWidth={1.5} />
                  Save search — coming soon
                </span>
              </div>

              <ResultsList
                results={filteredResults}
                page={page}
                totalPages={totalPages}
                loading={loading}
                query={queryString}
                onPageChange={handlePageChange}
                sort={sort || (queryString ? 'relevance' : 'posted_date')}
                order={order}
                onSortChange={handleSortChange}
                filters={columnFilters}
                onFilterChange={handleFilterChange}
                agencies={agencies}
                selectedNaics={filters.naicsCodes}
                onNaicsChange={handleNaicsChange}
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
