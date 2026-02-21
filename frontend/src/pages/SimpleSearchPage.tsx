import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Sparkles, Clock } from 'lucide-react';
import SearchInput from '../components/SearchInput';
import ResultsList from '../components/ResultsList';
import SetAsideChips from '../components/SetAsideChips';
import AuthButton from '../components/AuthButton';
import { useDebounce } from '../hooks/useDebounce';
import { useSearch } from '../hooks/useSearch';
import { preloadWhatsNew } from '../services/whatsNewCache';

const DEFAULT_TYPES = 'Solicitation,Presolicitation,Combined Synopsis/Solicitation,Sources Sought';

function today() {
  return new Date().toISOString().split('T')[0];
}

function deadlineToDate(preset: string): string | undefined {
  const days = parseInt(preset, 10);
  if (!days) return undefined;
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().split('T')[0];
}

export default function SimpleSearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialQuery = searchParams.get('q') || '';
  const initialPage = parseInt(searchParams.get('page') || '1', 10);
  const initialSetAsides = searchParams.get('set_aside')?.split(',').filter(Boolean) || [];
  const initialDepartment = searchParams.get('department') || '';
  const initialState = searchParams.get('state') || '';
  const initialNaics = searchParams.get('naics')?.split(',').filter(Boolean) || [];

  const initialDeadline = searchParams.get('deadline') || '';
  const initialSort = searchParams.get('sort') || '';
  const initialOrder = searchParams.get('order') || 'desc';

  const [query, setQuery] = useState(initialQuery);
  const [setAsides, setSetAsides] = useState<string[]>(initialSetAsides);
  const [department, setDepartment] = useState(initialDepartment);
  const [stateFilter, setStateFilter] = useState(initialState);
  const [selectedNaics, setSelectedNaics] = useState<string[]>(initialNaics);
  const [titleFilter, setTitleFilter] = useState('');
  const [deadlinePreset, setDeadlinePreset] = useState(initialDeadline);
  const [sort, setSort] = useState(initialSort);
  const [order, setOrder] = useState(initialOrder);
  const debouncedQuery = useDebounce(query, 300);
  const { results, total, page, totalPages, loading, search, reset } = useSearch();
  const hasAnyFilter = initialQuery.length >= 2 || initialSetAsides.length > 0 ||
    initialDepartment !== '' || initialState !== '' || initialNaics.length > 0;
  const [hasSearched, setHasSearched] = useState(hasAnyFilter);
  const inputRef = useRef<HTMLInputElement>(null);
  const isInitialMount = useRef(true);
  const hadQuerySearch = useRef(false);

  useEffect(() => { preloadWhatsNew(); }, []);

  const updateURL = useCallback((q: string, p: number, sa?: string[], s?: string, o?: string, dept?: string, st?: string, naics?: string[], dl?: string) => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (p > 1) params.set('page', String(p));
    if (sa && sa.length) params.set('set_aside', sa.join(','));
    if (s) params.set('sort', s);
    if (o && o !== 'desc') params.set('order', o);
    if (dept) params.set('department', dept);
    if (st) params.set('state', st);
    if (naics && naics.length) params.set('naics', naics.join(','));
    if (dl) params.set('deadline', dl);
    setSearchParams(params, { replace: true });
  }, [setSearchParams]);

  const buildParams = useCallback((q: string, pageNum: number, sa: string[], s?: string, o?: string, dept?: string, st?: string, naics?: string[], dl?: string) => ({
    q: q || undefined,
    type: DEFAULT_TYPES,
    set_aside: sa.length ? sa.join(',') : undefined,
    department: dept || undefined,
    state: st || undefined,
    naics: naics && naics.length ? naics.join(',') : undefined,
    deadline_from: today(),
    deadline_to: deadlineToDate(dl || ''),
    sort: s || (q ? 'relevance' : 'posted_date'),
    order: o || 'desc',
    page: pageNum,
    limit: 25,
  }), []);

  const hasActiveFilters = useCallback((q: string, sa: string[], dept: string, st: string, naics: string[]) => {
    return q.length >= 2 || sa.length > 0 || dept !== '' || st !== '' || naics.length > 0;
  }, []);

  const doSearch = useCallback((q: string, pg: number, sa: string[], s: string, o: string, dept: string, st: string, naics: string[], dl: string) => {
    search(buildParams(q, pg, sa, s, o, dept, st, naics, dl));
    updateURL(q, pg, sa, s, o, dept, st, naics, dl);
    setHasSearched(true);
  }, [search, buildParams, updateURL]);

  useEffect(() => {
    if (isInitialMount.current) {
      if (hasActiveFilters(initialQuery, setAsides, department, stateFilter, selectedNaics)) {
        search(buildParams(initialQuery, initialPage, setAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset));
        hadQuerySearch.current = initialQuery.length >= 2;
      }
      isInitialMount.current = false;
      return;
    }

    if (hasActiveFilters(debouncedQuery, setAsides, department, stateFilter, selectedNaics)) {
      doSearch(debouncedQuery, 1, setAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset);
      if (debouncedQuery.length >= 2) hadQuerySearch.current = true;
    } else if (debouncedQuery.length === 0 && hadQuerySearch.current) {
      reset();
      updateURL('', 1);
      hadQuerySearch.current = false;
      setHasSearched(false);
    }
  }, [debouncedQuery, search, reset, initialQuery, initialPage, updateURL, buildParams, setAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset, doSearch, hasActiveFilters]);

  const handlePageChange = (newPage: number) => {
    search(buildParams(debouncedQuery, newPage, setAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset));
    updateURL(debouncedQuery, newPage, setAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset);
  };

  const handleSubmit = () => {
    if (hasActiveFilters(query, setAsides, department, stateFilter, selectedNaics)) {
      doSearch(query, 1, setAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset);
    }
  };

  const handleSortChange = useCallback((newSort: string, newOrder: string) => {
    setSort(newSort);
    setOrder(newOrder);
    if (hasActiveFilters(debouncedQuery, setAsides, department, stateFilter, selectedNaics)) {
      search(buildParams(debouncedQuery, 1, setAsides, newSort, newOrder, department, stateFilter, selectedNaics, deadlinePreset));
      updateURL(debouncedQuery, 1, setAsides, newSort, newOrder, department, stateFilter, selectedNaics, deadlinePreset);
    }
  }, [search, buildParams, debouncedQuery, setAsides, updateURL, department, stateFilter, selectedNaics, deadlinePreset, hasActiveFilters]);

  const handleSetAsideChange = useCallback((newSetAsides: string[]) => {
    setSetAsides(newSetAsides);
    if (hasActiveFilters(debouncedQuery, newSetAsides, department, stateFilter, selectedNaics)) {
      doSearch(debouncedQuery, 1, newSetAsides, sort, order, department, stateFilter, selectedNaics, deadlinePreset);
    } else {
      reset();
      updateURL('', 1);
      setHasSearched(false);
      hadQuerySearch.current = false;
    }
  }, [debouncedQuery, doSearch, reset, updateURL, sort, order, department, stateFilter, selectedNaics, deadlinePreset, hasActiveFilters]);

  const handleFilterChange = useCallback((field: string, value: string) => {
    if (field === 'title') {
      setTitleFilter(value);
      return;
    }
    if (field === 'deadline') {
      setDeadlinePreset(value);
      if (hasActiveFilters(debouncedQuery, setAsides, department, stateFilter, selectedNaics)) {
        doSearch(debouncedQuery, 1, setAsides, sort, order, department, stateFilter, selectedNaics, value);
      }
      return;
    }
    if (field === 'department') {
      setDepartment(value);
      const sa = setAsides, st = stateFilter, n = selectedNaics;
      if (hasActiveFilters(debouncedQuery, sa, value, st, n)) {
        doSearch(debouncedQuery, 1, sa, sort, order, value, st, n, deadlinePreset);
      }
      return;
    }
    if (field === 'set_aside') {
      const newSa = value ? value.split(',') : [];
      setSetAsides(newSa);
      if (hasActiveFilters(debouncedQuery, newSa, department, stateFilter, selectedNaics)) {
        doSearch(debouncedQuery, 1, newSa, sort, order, department, stateFilter, selectedNaics, deadlinePreset);
      }
      return;
    }
    if (field === 'state') {
      setStateFilter(value);
      const sa = setAsides, n = selectedNaics;
      if (hasActiveFilters(debouncedQuery, sa, department, value, n)) {
        doSearch(debouncedQuery, 1, sa, sort, order, department, value, n, deadlinePreset);
      }
      return;
    }
  }, [debouncedQuery, setAsides, department, stateFilter, selectedNaics, deadlinePreset, sort, order, doSearch, hasActiveFilters]);

  const handleNaicsChange = useCallback((codes: string[]) => {
    setSelectedNaics(codes);
    if (hasActiveFilters(debouncedQuery, setAsides, department, stateFilter, codes)) {
      doSearch(debouncedQuery, 1, setAsides, sort, order, department, stateFilter, codes, deadlinePreset);
    }
  }, [debouncedQuery, setAsides, department, stateFilter, deadlinePreset, sort, order, doSearch, hasActiveFilters]);

  const agencies = useMemo(() =>
    [...new Set(results.map((o) => o.department).filter(Boolean) as string[])].sort(),
    [results],
  );

  const columnFilters: Record<string, string> = useMemo(() => ({
    title: titleFilter,
    department: department,
    set_aside: setAsides.join(','),
    deadline: deadlinePreset,
    state: stateFilter,
    naics: selectedNaics.join(','),
  }), [titleFilter, department, setAsides, deadlinePreset, stateFilter, selectedNaics]);

  const filteredResults = useMemo(() => {
    if (!titleFilter) return results;
    const lower = titleFilter.toLowerCase();
    return results.filter((r) => r.title.toLowerCase().includes(lower));
  }, [results, titleFilter]);

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
      </div>

      {/* Results */}
      {showResults && (
        <div className="relative z-10 flex-1 max-w-[1400px] w-full mx-auto px-6 pb-10">
          <div className="flex items-center justify-between mb-6 pb-4 border-b border-dark-800/50">
            <p className="text-sm text-dark-400">
              {loading ? (
                <span className="text-dark-500">Searching...</span>
              ) : (
                <>
                  <span className="text-dark-200 font-medium">{total.toLocaleString()}</span>
                  <span className="ml-1">results</span>
                </>
              )}
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
            results={filteredResults}
            page={page}
            totalPages={totalPages}
            loading={loading}
            query={query}
            onPageChange={handlePageChange}
            sort={sort || (debouncedQuery ? 'relevance' : 'posted_date')}
            order={order}
            onSortChange={handleSortChange}
            filters={columnFilters}
            onFilterChange={handleFilterChange}
            agencies={agencies}
            selectedNaics={selectedNaics}
            onNaicsChange={handleNaicsChange}
          />
        </div>
      )}
    </div>
  );
}
