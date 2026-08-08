import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Clock } from 'lucide-react';
import ResultsList from '../components/ResultsList';
import MobileFilterSheet from '../components/MobileFilterSheet';
import { searchOpportunities } from '../services/api';
import { getWhatsNewSync, getWhatsNew, whatsNewBaseParams } from '../services/whatsNewCache';
import type { OpportunityListItem } from '../types/api';
import { usePageMeta } from '../hooks/usePageMeta';

const PAGE_SIZE = 25;

const URL_FILTER_KEYS = ['title', 'department', 'set_aside', 'naics_prefix', 'state'] as const;

function compareValues(a: string | undefined | null, b: string | undefined | null, order: string): number {
  if (a == null && b == null) return 0;
  if (a == null) return 1;
  if (b == null) return -1;
  const cmp = a.localeCompare(b);
  return order === 'asc' ? cmp : -cmp;
}

function filtersFromParams(params: URLSearchParams): Record<string, string> {
  const f: Record<string, string> = {};
  for (const key of URL_FILTER_KEYS) {
    const v = params.get(key);
    if (v) f[key] = v;
  }
  return f;
}

export default function WhatsNewPage() {
  usePageMeta({
    title: "What's New in GovTrove — Product Updates and Changelog",
    description: "Recent GovTrove releases: new search features, filters, alerts, and data improvements for finding federal contract opportunities from SAM.gov.",
    canonicalPath: '/whats-new',
  });

  const [searchParams, setSearchParams] = useSearchParams();
  const allData = useRef<OpportunityListItem[]>([]);
  const [loading, setLoading] = useState(true);

  const sort = searchParams.get('sort') || 'posted_date';
  const order = searchParams.get('order') || 'desc';
  const filters = useMemo(() => filtersFromParams(searchParams), [searchParams]);
  const selectedNaics = useMemo(() => {
    const v = searchParams.get('naics');
    return v ? v.split(',') : [];
  }, [searchParams]);
  const page = Number(searchParams.get('page')) || 1;

  const updateParams = useCallback((updates: Record<string, string | null>) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      for (const [k, v] of Object.entries(updates)) {
        if (v) next.set(k, v);
        else next.delete(k);
      }
      return next;
    }, { replace: true });
  }, [setSearchParams]);

  useEffect(() => {
    const cached = getWhatsNewSync();
    if (cached) {
      allData.current = cached.opportunities || [];
      setLoading(false);
      return;
    }
    getWhatsNew()
      .then((data) => {
        allData.current = data.opportunities || [];
        setLoading(false);
      })
      .catch(() => {
        searchOpportunities(whatsNewBaseParams()).then((data) => {
          allData.current = data.opportunities || [];
          setLoading(false);
        }).catch(() => setLoading(false));
      });
  }, []);

  const agencies = useMemo(() => {
    if (loading) return [];
    return [...new Set(allData.current.map((o) => o.department).filter(Boolean) as string[])].sort();
  }, [loading]);

  const setAsideCodes = useMemo(() => {
    if (loading) return [];
    return [...new Set(allData.current.map((o) => o.set_aside_code).filter(Boolean) as string[])].sort();
  }, [loading]);

  const stateCodes = useMemo(() => {
    if (loading) return [];
    return [...new Set(allData.current.map((o) => o.pop_state).filter(Boolean) as string[])].sort();
  }, [loading]);

  const filtered = useMemo(() => {
    if (loading) return [];
    let items = allData.current;

    if (filters.title) {
      const q = filters.title.toLowerCase();
      items = items.filter((o) => o.title.toLowerCase().includes(q));
    }
    if (filters.set_aside) {
      const codes = new Set(filters.set_aside.split(','));
      items = items.filter((o) => o.set_aside_code && codes.has(o.set_aside_code));
    }
    if (filters.department) {
      const depts = new Set(filters.department.split(','));
      items = items.filter((o) => o.department && depts.has(o.department));
    }
    if (filters.naics_prefix) {
      const prefix = filters.naics_prefix;
      items = items.filter((o) => o.naics_code?.startsWith(prefix));
    }
    if (selectedNaics.length > 0) {
      items = items.filter((o) =>
        o.naics_code && selectedNaics.some((code) => o.naics_code!.startsWith(code)),
      );
    }
    if (filters.state) {
      const states = new Set(filters.state.split(','));
      items = items.filter((o) => o.pop_state && states.has(o.pop_state));
    }

    items = [...items].sort((a, b) => {
      const fieldMap: Record<string, keyof OpportunityListItem> = {
        title: 'title',
        department: 'department',
        posted_date: 'posted_date',
        set_aside_code: 'set_aside_code',
        deadline: 'response_deadline',
        naics_code: 'naics_code',
        pop_state: 'pop_state',
      };
      const key = fieldMap[sort] || 'posted_date';
      const av = a[key] as string | undefined;
      const bv = b[key] as string | undefined;
      return compareValues(av, bv, order);
    });

    return items;
  }, [loading, filters, sort, order, selectedNaics]);

  const totalFiltered = filtered.length;
  const totalPages = Math.max(1, Math.ceil(totalFiltered / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const pageResults = filtered.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);
  const hasActiveFilters = Object.keys(filters).length > 0 || selectedNaics.length > 0;

  const handlePageChange = useCallback((newPage: number) => {
    updateParams({ page: newPage > 1 ? String(newPage) : null });
  }, [updateParams]);

  const handleSortChange = useCallback((newSort: string, newOrder: string) => {
    const isDefault = newSort === 'posted_date' && newOrder === 'desc';
    updateParams({
      sort: isDefault ? null : newSort,
      order: isDefault ? null : newOrder,
      page: null,
    });
  }, [updateParams]);

  const handleFilterChange = useCallback((field: string, value: string) => {
    updateParams({ [field]: value || null, page: null });
  }, [updateParams]);

  const handleNaicsChange = useCallback((codes: string[]) => {
    updateParams({ naics: codes.length > 0 ? codes.join(',') : null, page: null });
  }, [updateParams]);

  const handleResetFilters = useCallback(() => {
    setSearchParams({}, { replace: true });
  }, [setSearchParams]);

  return (
    <div className="min-h-screen flex flex-col relative">
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      <div className="relative z-10 pt-10 px-6 mb-4">
        <div className="flex items-center gap-2 text-accent">
          <Clock size={18} strokeWidth={1.5} />
          <h2 className="text-lg font-medium">What's New Today</h2>
        </div>
      </div>

      <div className="relative z-10 flex-1 max-w-[1400px] w-full mx-auto px-6 pb-10">
        <div className="flex items-center justify-between mb-6 pb-4 border-b border-dark-800/50">
          <p className="text-sm text-dark-400">
            {loading ? (
              <span className="inline-block h-4 w-48 bg-dark-800/50 rounded animate-pulse" />
            ) : (
              <>
                <span className="text-dark-200 font-medium">{totalFiltered.toLocaleString()}</span>
                <span className="ml-1">new opportunities with open deadlines</span>
              </>
            )}
          </p>
          {hasActiveFilters && (
            <button
              onClick={handleResetFilters}
              className="text-xs text-dark-400 hover:text-accent transition-colors"
            >
              Reset filters
            </button>
          )}
        </div>
        <ResultsList
          results={pageResults}
          page={currentPage}
          totalPages={totalPages}
          loading={loading}
          query=""
          onPageChange={handlePageChange}
          sort={sort}
          order={order}
          onSortChange={handleSortChange}
          filters={filters}
          onFilterChange={handleFilterChange}
          agencies={agencies}
          availableSetAsides={setAsideCodes}
          availableStates={stateCodes}
          selectedNaics={selectedNaics}
          onNaicsChange={handleNaicsChange}
        />
        {!loading && (
          <MobileFilterSheet
            filters={filters}
            onFilterChange={handleFilterChange}
            sort={sort}
            order={order}
            onSortChange={handleSortChange}
            agencies={agencies}
            availableSetAsides={setAsideCodes}
            availableStates={stateCodes}
            selectedNaics={selectedNaics}
            onNaicsChange={handleNaicsChange}
            resultCount={totalFiltered}
            onReset={handleResetFilters}
          />
        )}
      </div>
    </div>
  );
}
