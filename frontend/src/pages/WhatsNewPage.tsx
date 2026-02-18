import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft, Clock } from 'lucide-react';
import ResultsList from '../components/ResultsList';
import AuthButton from '../components/AuthButton';
import { searchOpportunities } from '../services/api';
import { getWhatsNewSync, getWhatsNew, whatsNewBaseParams } from '../services/whatsNewCache';
import type { OpportunityListItem } from '../types/api';

const PAGE_SIZE = 25;

function compareValues(a: string | undefined | null, b: string | undefined | null, order: string): number {
  if (a == null && b == null) return 0;
  if (a == null) return 1;
  if (b == null) return -1;
  const cmp = a.localeCompare(b);
  return order === 'asc' ? cmp : -cmp;
}

export default function WhatsNewPage() {
  const allData = useRef<OpportunityListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [sort, setSort] = useState('posted_date');
  const [order, setOrder] = useState('desc');
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [selectedNaics, setSelectedNaics] = useState<string[]>([]);
  const [page, setPage] = useState(1);

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
    setPage(newPage);
  }, []);

  const handleSortChange = useCallback((newSort: string, newOrder: string) => {
    setSort(newSort);
    setOrder(newOrder);
    setPage(1);
  }, []);

  const handleFilterChange = useCallback((field: string, value: string) => {
    setFilters((prev) => {
      const next = { ...prev };
      if (value) {
        next[field] = value;
      } else {
        delete next[field];
      }
      return next;
    });
    setPage(1);
  }, []);

  const handleResetFilters = useCallback(() => {
    setFilters({});
    setSelectedNaics([]);
    setPage(1);
  }, []);

  return (
    <div className="min-h-screen flex flex-col relative">
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      <div className="absolute top-4 right-6 z-20">
        <AuthButton />
      </div>

      <div className="relative z-10 flex flex-col items-center pt-10">
        <Link to="/" className="mb-6">
          <h1 className="text-2xl font-semibold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-dark-50 to-dark-200">
            GovTrove
          </h1>
        </Link>

        <div className="flex items-center gap-3 mb-8">
          <Link
            to="/"
            className="p-2 text-dark-400 hover:text-dark-100 rounded-lg hover:bg-dark-800/50 transition-all duration-200"
          >
            <ArrowLeft size={18} strokeWidth={1.5} />
          </Link>
          <div className="flex items-center gap-2 text-accent">
            <Clock size={18} strokeWidth={1.5} />
            <h2 className="text-lg font-medium">What's New Today</h2>
          </div>
        </div>
      </div>

      <div className="relative z-10 flex-1 max-w-[1400px] w-full mx-auto px-6 pb-10">
        <div className="flex items-center justify-between mb-6 pb-4 border-b border-dark-800/50">
          <p className="text-sm text-dark-400">
            {loading ? (
              <span className="text-dark-500">Loading...</span>
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
          onNaicsChange={(codes) => { setSelectedNaics(codes); setPage(1); }}
        />
      </div>
    </div>
  );
}
