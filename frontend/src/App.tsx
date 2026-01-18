import { useState, useCallback, useRef, useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SearchBar } from './components/SearchBar';
import { FilterPanel } from './components/FilterPanel';
import { OpportunityList } from './components/OpportunityList';
import { OpportunityDetail } from './components/OpportunityDetail';
import { Pagination } from './components/Pagination';
import { AdminDashboard } from './components/AdminDashboard';
import { useOpportunitySearch, useFilters } from './hooks/useOpportunities';
import { useAnalytics } from './hooks/useAnalytics';
import type { SearchParams } from './types/opportunity';

type EventType = 'search' | 'filter' | 'page' | null;
type ViewType = 'search' | 'admin';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

interface SearchAppProps {
  onOpenAdmin: () => void;
}

function SearchApp({ onOpenAdmin }: SearchAppProps) {
  const [params, setParams] = useState<SearchParams>({
    page: 1,
    limit: 25,
    sort: 'posted_date',
    order: 'desc',
  });
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [showFilters, setShowFilters] = useState(true);
  const pendingEventRef = useRef<EventType>(null);
  const lastTrackedRef = useRef<string>('');

  const { data: searchResult, isLoading, error } = useOpportunitySearch(params);
  const { data: filters, isLoading: filtersLoading } = useFilters();
  const { trackSearch, trackFilter, trackPage, trackClick } = useAnalytics();

  useEffect(() => {
    if (!searchResult || isLoading) return;

    const trackKey = `${pendingEventRef.current}-${JSON.stringify(params)}-${searchResult.total}`;
    if (trackKey === lastTrackedRef.current) return;

    const eventType = pendingEventRef.current;
    if (eventType === 'search') {
      trackSearch(params, searchResult.total);
    } else if (eventType === 'filter') {
      trackFilter(params, searchResult.total);
    } else if (eventType === 'page') {
      trackPage(params.page || 1, params, searchResult.total);
    }

    lastTrackedRef.current = trackKey;
    pendingEventRef.current = null;
  }, [searchResult, isLoading, params, trackSearch, trackFilter, trackPage]);

  const handleSearch = useCallback(() => {
    pendingEventRef.current = 'search';
    setParams((prev) => ({ ...prev, page: 1 }));
  }, []);

  const handleQueryChange = useCallback((q: string) => {
    setParams((prev) => ({ ...prev, q: q || undefined }));
  }, []);

  const handleFilterChange = useCallback((updates: Partial<SearchParams>) => {
    pendingEventRef.current = 'filter';
    setParams((prev) => ({ ...prev, ...updates, page: 1 }));
  }, []);

  const handlePageChange = useCallback((page: number) => {
    pendingEventRef.current = 'page';
    setParams((prev) => ({ ...prev, page }));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }, []);

  const handleSortChange = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    pendingEventRef.current = 'filter';
    const [sort, order] = e.target.value.split(':');
    setParams((prev) => ({ ...prev, sort, order, page: 1 }));
  }, []);

  const handleSelect = useCallback(
    (id: number, index: number) => {
      const position = ((params.page || 1) - 1) * (params.limit || 25) + index + 1;
      trackClick(id, position, params);
      setSelectedId(id);
    },
    [params, trackClick]
  );

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-4">
              <h1 className="text-xl font-bold text-gray-900">OpScout</h1>
              <span className="text-sm text-gray-500">Federal Contract Opportunities</span>
            </div>
            <button
              onClick={onOpenAdmin}
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              Admin
            </button>
          </div>
          <SearchBar value={params.q || ''} onChange={handleQueryChange} onSearch={handleSearch} />
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-6">
        <div className="flex gap-6">
          <aside className={`w-64 flex-shrink-0 ${showFilters ? '' : 'hidden'} lg:block`}>
            <FilterPanel
              filters={filters}
              params={params}
              onChange={handleFilterChange}
              isLoading={filtersLoading}
            />
          </aside>

          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-4">
                <button
                  onClick={() => setShowFilters(!showFilters)}
                  className="lg:hidden px-3 py-1.5 text-sm border border-gray-300 rounded hover:bg-gray-50"
                >
                  {showFilters ? 'Hide Filters' : 'Show Filters'}
                </button>
                {searchResult && (
                  <span className="text-sm text-gray-600">
                    {searchResult.total.toLocaleString()} opportunities found
                  </span>
                )}
              </div>

              <div className="flex items-center gap-2">
                <label htmlFor="sort" className="text-sm text-gray-600">
                  Sort by:
                </label>
                <select
                  id="sort"
                  value={`${params.sort || 'posted_date'}:${params.order || 'desc'}`}
                  onChange={handleSortChange}
                  className="px-3 py-1.5 text-sm border border-gray-300 rounded bg-white"
                >
                  <option value="posted_date:desc">Posted Date (Newest)</option>
                  <option value="posted_date:asc">Posted Date (Oldest)</option>
                  <option value="deadline:asc">Deadline (Soonest)</option>
                  <option value="deadline:desc">Deadline (Latest)</option>
                  {params.q && <option value="relevance:desc">Relevance</option>}
                </select>
              </div>
            </div>

            <OpportunityList
              opportunities={searchResult?.opportunities}
              isLoading={isLoading}
              error={error}
              onSelect={handleSelect}
            />

            {searchResult && (
              <Pagination
                page={searchResult.page}
                totalPages={searchResult.total_pages}
                total={searchResult.total}
                limit={searchResult.limit}
                onPageChange={handlePageChange}
              />
            )}
          </div>
        </div>
      </main>

      {selectedId !== null && (
        <OpportunityDetail id={selectedId} onClose={() => setSelectedId(null)} />
      )}
    </div>
  );
}

export default function App() {
  const [view, setView] = useState<ViewType>('search');

  return (
    <QueryClientProvider client={queryClient}>
      {view === 'admin' ? (
        <AdminDashboard onClose={() => setView('search')} />
      ) : (
        <SearchApp onOpenAdmin={() => setView('admin')} />
      )}
    </QueryClientProvider>
  );
}
