import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
  Loader2,
  SearchX,
  Star,
  X,
  ChevronLeft,
  ChevronRight,
  WifiOff,
  BookOpen,
  ArrowRight,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import OpportunityCard from './OpportunityCard';
import SortDropdown from './SortDropdown';
import SignupPromptModal, { type SignupPromptContext } from '../SignupPromptModal';
import type { OpportunityListItem, FacetResult, RescueResult, RescueSuggestion } from '../../types/api';
import type { FilterState } from '../../hooks/useFilterState';
import { usePostHog } from '@posthog/react';
import { trackCodeFinderZeroOpportunities } from '../../lib/analytics';

const PAGE_SIZE_KEY = 'govtrove_page_size';

// True when the whole query is a single quoted phrase, i.e. the strictest
// possible search and the most common reason for an unexpected zero.
function isExactPhraseQuery(q?: string) {
  const t = (q ?? '').trim();
  return t.length > 2 && t.startsWith('"') && t.endsWith('"') && t.indexOf('"', 1) === t.length - 1;
}

interface SearchResultsProps {
  results: OpportunityListItem[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  loading: boolean;
  keyword?: string;
  filters: FilterState;
  facets: FacetResult['facets'] | null;
  sort: string;
  sortDir: 'asc' | 'desc';
  onSortChange: (sort: string, dir: 'asc' | 'desc') => void;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  savedIds: Set<number>;
  onToggleSave: (id: number) => void;
  onSaveAll: (ids: number[]) => void;
  clearAllFilters: () => void;
  querySuggestion?: string | null;
  onQuerySuggestionClick?: (suggestion: string) => void;
  error?: string | null;
  onRetry?: () => void;
  isAuthenticated?: boolean;
  onCreateAlert?: () => void;
  rescue?: RescueResult | null;
  rescueLoading?: boolean;
  onApplyRescue?: (s: RescueSuggestion) => void;
}

export default function SearchResults({
  results,
  total,
  page,
  pageSize,
  totalPages,
  loading,
  keyword,
  filters,
  facets,
  sort,
  sortDir,
  onSortChange,
  onPageChange,
  onPageSizeChange,
  savedIds,
  onToggleSave,
  onSaveAll,
  querySuggestion,
  onQuerySuggestionClick,
  clearAllFilters,
  error,
  onRetry,
  isAuthenticated,
  onCreateAlert,
  rescue,
  rescueLoading,
  onApplyRescue,
}: SearchResultsProps) {
  const codeFilter = useMemo((): { type: 'naics' | 'psc'; code: string } | null => {
    const naics = ((filters.naics as string[] | undefined) || []);
    const psc = ((filters.psc as string[] | undefined) || []);
    if (naics.length === 1 && psc.length === 0) return { type: 'naics', code: naics[0] };
    if (psc.length === 1 && naics.length === 0) return { type: 'psc', code: psc[0] };
    return null;
  }, [filters.naics, filters.psc]);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [signupPrompt, setSignupPrompt] = useState<SignupPromptContext | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const anySelected = selectedIds.size > 0;
  // Show the query exactly as typed. Joining on ", " used to imply AND while
  // the backend was ORing, which described the results incorrectly.
  const displayKeyword = keyword;

  // Clear selection when results change
  useEffect(() => {
    setSelectedIds(new Set());
  }, [results]);

  // Update document title with result count
  useEffect(() => {
    if (loading) return;
    const count = total.toLocaleString();
    document.title = keyword
      ? `${count} results for "${displayKeyword}" — GovTrove`
      : `${count} results — GovTrove`;
    return () => { document.title = 'GovTrove'; };
  }, [total, keyword, loading]);

  const toggleSelect = useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handlePageChange = useCallback((newPage: number) => {
    onPageChange(newPage);
    containerRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, [onPageChange]);

  const handlePageSizeChange = useCallback((size: number) => {
    localStorage.setItem(PAGE_SIZE_KEY, String(size));
    onPageSizeChange(size);
  }, [onPageSizeChange]);

  const handleToggleSave = useCallback((id: number) => {
    if (!isAuthenticated) {
      setSignupPrompt('bookmark');
      return;
    }
    onToggleSave(id);
  }, [isAuthenticated, onToggleSave]);

  const handleSaveAll = useCallback(() => {
    if (!isAuthenticated) {
      setSignupPrompt('save-all');
      return;
    }
    onSaveAll([...selectedIds]);
    setSelectedIds(new Set());
  }, [selectedIds, onSaveAll, isAuthenticated]);

  const handleClearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  // Filter suggestion for zero results
  const filterSuggestion = useMemo(() => {
    if (results.length > 0 || !facets) return null;
    // An exact-phrase query is the likelier culprit, and the empty state offers
    // a one-click way out of it — don't send people chasing unrelated filters.
    if (isExactPhraseQuery(filters.keyword)) return null;
    const activeFilters: { key: string; label: string }[] = [];
    if (filters.naics.length) activeFilters.push({ key: 'naics', label: 'NAICS' });
    if (filters.setAside.length) activeFilters.push({ key: 'setAside', label: 'Set-Aside' });
    if (filters.agency.length) activeFilters.push({ key: 'agency', label: 'Agency' });
    if (filters.noticeType.length) activeFilters.push({ key: 'noticeType', label: 'Notice Type' });
    if (filters.deadlinePreset) activeFilters.push({ key: 'deadline', label: 'Deadline' });
    if (activeFilters.length > 0) {
      return `Try removing the ${activeFilters[0].label} filter to broaden your search.`;
    }
    return null;
  }, [results.length, facets, filters]);

  const isInitialLoad = loading && results.length === 0;
  const isOverlayLoad = loading && results.length > 0;
  const showZeroResults = !loading && results.length === 0;

  return (
    <div ref={containerRef}>
      {/* Toolbar: result count + sort */}
      <div className="flex items-center justify-between pb-3 border-b border-dark-800/50 mb-3">
        <p className="text-sm text-dark-400" aria-live="polite" aria-atomic="true">
          {loading ? (
            <span className="inline-block h-4 w-24 bg-dark-800/50 rounded animate-pulse" />
          ) : (
            <>
              <span className="text-dark-200 font-medium">{total.toLocaleString()}</span>
              <span className="ml-1">results</span>
              {keyword && (
                <span className="ml-1">
                  for &lsquo;<span className="text-dark-200">{displayKeyword}</span>&rsquo;
                </span>
              )}
            </>
          )}
        </p>
        <div className="flex items-center gap-2">
          <SortDropdown
            sort={sort}
            sortDir={sortDir}
            onSortChange={onSortChange}
            hasKeyword={!!keyword}
          />
        </div>
      </div>

      {/* Error state */}
      {error && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <div className="w-14 h-14 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mb-5">
            <WifiOff size={24} className="text-red-400" strokeWidth={1.5} />
          </div>
          <h3 className="text-base font-medium text-dark-200 mb-1.5">Something went wrong</h3>
          <p className="text-sm text-dark-400 mb-5 max-w-xs">{error}</p>
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              className="px-4 py-2 text-sm rounded-lg bg-accent/10 text-accent hover:bg-accent/20 transition-colors"
            >
              Try again
            </button>
          )}
        </div>
      )}

      {/* Initial loading: skeleton cards */}
      {isInitialLoad && !error && <SkeletonCards count={5} />}

      {/* Zero results */}
      {showZeroResults && !error && (
        <ZeroResults
          keyword={keyword}
          suggestion={filterSuggestion}
          querySuggestion={querySuggestion}
          onQuerySuggestionClick={onQuerySuggestionClick}
          onClearAll={clearAllFilters}
          codeFilter={codeFilter}
          isAuthenticated={isAuthenticated}
          onCreateAlert={onCreateAlert}
          rescue={rescue}
          rescueLoading={rescueLoading}
          onApplyRescue={onApplyRescue}
        />
      )}

      {/* Card list with optional loading overlay */}
      {results.length > 0 && (
        <div className="relative">
          {isOverlayLoad && (
            <div className="absolute inset-0 bg-dark-950/40 z-10 flex items-center justify-center rounded-xl">
              <Loader2 size={28} className="text-accent animate-spin" />
            </div>
          )}
          <div className="space-y-2">
            {results.map((opp) => (
              <OpportunityCard
                key={opp.id}
                opportunity={opp}
                keyword={keyword}
                isSaved={savedIds.has(opp.id)}
                isSelected={selectedIds.has(opp.id)}
                anySelected={anySelected}
                onToggleSave={handleToggleSave}
                onToggleSelect={toggleSelect}
              />
            ))}
          </div>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && !isInitialLoad && (
        <Pagination
          page={page}
          totalPages={totalPages}
          pageSize={pageSize}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      )}

      {/* Bulk action bar */}
      {anySelected && (
        <BulkActionBar
          count={selectedIds.size}
          onSaveAll={handleSaveAll}
          onClear={handleClearSelection}
        />
      )}

      {signupPrompt && (
        <SignupPromptModal context={signupPrompt} onClose={() => setSignupPrompt(null)} />
      )}
    </div>
  );
}

// --- Skeleton Cards ---

function SkeletonCards({ count }: { count: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className="rounded-xl border border-dark-800/50 bg-dark-900/30 p-4 space-y-3 animate-pulse"
        >
          <div className="flex items-start gap-2">
            <div className="w-4 h-4 rounded bg-dark-800/40 shrink-0 mt-0.5" />
            <div className="w-4 h-4 rounded bg-dark-800/40 shrink-0 mt-0.5" />
            <div className="flex-1 space-y-2">
              <div className="h-4 bg-dark-800/60 rounded" style={{ width: `${55 + (i * 11) % 30}%` }} />
              <div className="flex gap-2">
                <div className="h-3 bg-dark-800/40 rounded w-24" />
                <div className="h-5 bg-dark-800/40 rounded-full w-16" />
                <div className="h-5 bg-dark-800/40 rounded-full w-14" />
              </div>
              <div className="h-3 bg-dark-800/30 rounded w-36" />
              <div className="h-3 bg-dark-800/30 rounded" style={{ width: `${70 + (i * 7) % 20}%` }} />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// --- Zero Results ---

function ZeroResults({
  keyword,
  suggestion,
  querySuggestion,
  onQuerySuggestionClick,
  onClearAll,
  codeFilter,
  isAuthenticated,
  onCreateAlert,
  rescue,
  rescueLoading,
  onApplyRescue,
}: {
  keyword?: string;
  suggestion: string | null;
  querySuggestion?: string | null;
  onQuerySuggestionClick?: (suggestion: string) => void;
  onClearAll: () => void;
  codeFilter?: { type: 'naics' | 'psc'; code: string } | null;
  isAuthenticated?: boolean;
  onCreateAlert?: () => void;
  rescue?: RescueResult | null;
  rescueLoading?: boolean;
  onApplyRescue?: (s: RescueSuggestion) => void;
}) {
  const posthog = usePostHog();
  const [alertDone, setAlertDone] = useState(false);
  useEffect(() => {
    if (codeFilter) trackCodeFinderZeroOpportunities(posthog, codeFilter.type, codeFilter.code);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [codeFilter?.type, codeFilter?.code]);
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="w-16 h-16 rounded-2xl bg-dark-800/50 border border-dark-700/30 flex items-center justify-center mb-6">
        <SearchX size={28} className="text-dark-500" strokeWidth={1.5} />
      </div>
      <h3 className="text-lg font-medium text-dark-200 mb-2">
        No opportunities match your filters
      </h3>
      {keyword && (
        <p className="text-dark-400 text-sm mb-2">
          No results for &ldquo;<span className="text-dark-300">{keyword}</span>&rdquo;
        </p>
      )}
      {isExactPhraseQuery(keyword) && onQuerySuggestionClick && (
        <p className="text-dark-400 text-sm mb-4">
          That searched for the exact phrase.{' '}
          <button
            type="button"
            onClick={() => onQuerySuggestionClick((keyword ?? '').trim().slice(1, -1))}
            className="text-accent font-medium underline underline-offset-4 decoration-accent/40
              hover:decoration-accent transition-colors"
          >
            Match all these words instead
          </button>
        </p>
      )}
      {querySuggestion && onQuerySuggestionClick && (
        <p className="text-dark-400 text-xl mb-4 mt-2">
          Did you mean{' '}
          <button
            type="button"
            onClick={() => onQuerySuggestionClick(querySuggestion)}
            className="text-accent font-semibold underline underline-offset-4 decoration-accent/40 hover:decoration-accent transition-colors"
          >
            {querySuggestion}
          </button>
          ?
        </p>
      )}
      {suggestion && (
        <p className="text-dark-400 text-sm mb-4">{suggestion}</p>
      )}

      {rescueLoading && (
        <p className="text-dark-500 text-xs mb-4 inline-flex items-center gap-2">
          <Loader2 size={12} className="animate-spin" />
          Checking why this found nothing…
        </p>
      )}

      {rescue && rescue.suggestions.length > 0 && onApplyRescue && (
        <div className="w-full max-w-md mb-5 rounded-xl border border-accent/25 bg-accent/5 p-5 text-left">
          {rescue.explanation && (
            <p className="text-sm text-dark-300 mb-3">{rescue.explanation}</p>
          )}
          <div className="flex flex-col gap-2">
            {rescue.suggestions.map((s) => (
              <button
                key={s.rule + s.label}
                type="button"
                onClick={() => onApplyRescue(s)}
                className="flex items-center justify-between gap-3 px-3 py-2 text-sm rounded-lg
                  bg-accent/10 text-accent hover:bg-accent/20 transition-colors text-left"
              >
                <span>{s.label}</span>
                <span className="shrink-0 text-xs font-medium tabular-nums">
                  {s.verified_total.toLocaleString()} results
                </span>
              </button>
            ))}
          </div>
        </div>
      )}

      {codeFilter && (
        <div className="w-full max-w-md mb-5 rounded-xl border border-accent/25 bg-accent/5 p-5">
          <p className="text-sm text-dark-300 mb-3">
            No open opportunities under{' '}
            <span className="font-mono text-dark-100">{codeFilter.type.toUpperCase()} {codeFilter.code}</span>{' '}
            right now — but new ones are posted daily. Get an email the moment one appears.
          </p>
          {alertDone ? (
            <p className="text-sm text-accent font-medium">✓ Alert created — we&rsquo;ll email you when new opportunities match.</p>
          ) : (
            <button
              type="button"
              onClick={() => { onCreateAlert?.(); if (isAuthenticated) setAlertDone(true); }}
              className="px-4 py-2 text-sm rounded-lg bg-accent text-white hover:bg-accent-hover transition-colors font-medium"
            >
              {isAuthenticated ? 'Notify me when opportunities appear' : 'Sign in to set an alert'}
            </button>
          )}
        </div>
      )}

      <button
        type="button"
        onClick={onClearAll}
        className="px-4 py-2 text-sm rounded-lg bg-accent/10 text-accent
          hover:bg-accent/20 transition-colors"
      >
        {codeFilter ? 'Or clear all filters' : 'Clear all filters'}
      </button>

      <div className="mt-8 text-left bg-dark-900/50 border border-dark-800/50 rounded-xl p-5 max-w-sm">
        <p className="text-xs font-medium text-dark-300 mb-3 uppercase tracking-wider">
          Search tips
        </p>
        <ul className="space-y-2.5 text-sm text-dark-400">
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">
              space
            </kbd>
            <span>All words must appear &mdash; more words, fewer results</span>
          </li>
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">
              &quot;...&quot;
            </kbd>
            <span>Match an exact phrase (e.g. &ldquo;zero trust&rdquo;)</span>
          </li>
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">
              OR
            </kbd>
            <span>Either word (e.g. cyber OR cloud)</span>
          </li>
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">
              -
            </kbd>
            <span>Exclude unwanted terms</span>
          </li>
        </ul>
      </div>

      <Link
        to="/guide"
        className="mt-4 inline-flex items-center gap-1.5 text-xs text-dark-500 hover:text-dark-300 transition-colors"
      >
        <BookOpen size={12} />
        Learn how to use all search filters
        <ArrowRight size={12} />
      </Link>
    </div>
  );
}

// --- Pagination ---

function Pagination({
  page,
  totalPages,
  pageSize,
  onPageChange,
  onPageSizeChange,
}: {
  page: number;
  totalPages: number;
  pageSize: number;
  onPageChange: (p: number) => void;
  onPageSizeChange: (s: number) => void;
}) {
  const pages = useMemo(() => {
    const items: (number | 'ellipsis')[] = [];
    if (totalPages <= 7) {
      for (let i = 1; i <= totalPages; i++) items.push(i);
      return items;
    }
    items.push(1);
    if (page > 3) items.push('ellipsis');
    const start = Math.max(2, page - 1);
    const end = Math.min(totalPages - 1, page + 1);
    for (let i = start; i <= end; i++) items.push(i);
    if (page < totalPages - 2) items.push('ellipsis');
    items.push(totalPages);
    return items;
  }, [page, totalPages]);

  return (
    <div className="flex flex-wrap items-center justify-center gap-3 mt-6">
      {/* Previous */}
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="hidden md:inline-flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg border
          border-dark-700/50 bg-dark-900/50 text-dark-400
          hover:text-dark-100 hover:border-dark-600/50
          disabled:opacity-30 disabled:cursor-not-allowed transition-all"
      >
        <ChevronLeft size={14} />
        Previous
      </button>

      {/* Mobile: simple nav */}
      <div className="flex md:hidden items-center gap-3">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="p-3 rounded-lg border border-dark-700/50 bg-dark-900/50 text-dark-400
            disabled:opacity-30 disabled:cursor-not-allowed transition-all
            focus-visible:ring-2 focus-visible:ring-accent/50 focus-visible:ring-offset-1 focus-visible:ring-offset-dark-950"
        >
          <ChevronLeft size={16} />
        </button>
        <span className="text-sm text-dark-400 tabular-nums">
          <span className="text-dark-200">{page}</span>
          <span className="mx-1 text-dark-600">/</span>
          <span>{totalPages}</span>
        </span>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="p-3 rounded-lg border border-dark-700/50 bg-dark-900/50 text-dark-400
            disabled:opacity-30 disabled:cursor-not-allowed transition-all
            focus-visible:ring-2 focus-visible:ring-accent/50 focus-visible:ring-offset-1 focus-visible:ring-offset-dark-950"
        >
          <ChevronRight size={16} />
        </button>
      </div>

      {/* Desktop: page numbers */}
      <div className="hidden md:flex items-center gap-1">
        {pages.map((item, i) =>
          item === 'ellipsis' ? (
            <span key={`e${i}`} className="px-2 text-dark-500">
              ...
            </span>
          ) : (
            <button
              key={item}
              onClick={() => onPageChange(item)}
              className={`min-w-[32px] h-8 px-2 text-sm rounded-lg transition-colors
                ${item === page
                  ? 'bg-accent text-white font-medium'
                  : 'text-dark-400 hover:text-dark-200 hover:bg-dark-800/50'
                }`}
            >
              {item}
            </button>
          ),
        )}
      </div>

      {/* Next */}
      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="hidden md:inline-flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg border
          border-dark-700/50 bg-dark-900/50 text-dark-400
          hover:text-dark-100 hover:border-dark-600/50
          disabled:opacity-30 disabled:cursor-not-allowed transition-all"
      >
        Next
        <ChevronRight size={14} />
      </button>

      {/* Page size selector */}
      <select
        value={pageSize}
        onChange={(e) => onPageSizeChange(parseInt(e.target.value, 10))}
        className="text-xs text-dark-400 bg-dark-800 border border-dark-700/50 rounded-lg
          px-2 py-1.5 cursor-pointer hover:border-dark-600 transition-colors
          focus:outline-none focus:border-accent/50"
      >
        {[25, 50, 100].map((size) => (
          <option key={size} value={size}>
            {size} per page
          </option>
        ))}
      </select>
    </div>
  );
}

// --- Bulk Action Bar ---

function BulkActionBar({
  count,
  onSaveAll,
  onClear,
}: {
  count: number;
  onSaveAll: () => void;
  onClear: () => void;
}) {
  return (
    <div
      role="toolbar"
      aria-label="Bulk actions for selected opportunities"
      className="fixed bottom-16 md:bottom-0 left-0 right-0 z-50 bg-dark-900/95 border-t border-dark-700/50
        backdrop-blur-sm shadow-2xl shadow-black/50 animate-in slide-in-from-bottom duration-200"
    >
      <div className="max-w-[1400px] mx-auto px-6 py-3 flex items-center justify-between gap-4">
        <span className="text-sm text-dark-200 font-medium tabular-nums">
          {count} selected
        </span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onSaveAll}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg
              bg-accent/10 text-accent hover:bg-accent/20 transition-colors"
          >
            <Star size={14} />
            Save All
          </button>
          <button
            type="button"
            onClick={onClear}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg
              text-dark-400 hover:text-dark-200 hover:bg-dark-800 transition-colors"
          >
            <X size={14} />
            Clear
          </button>
        </div>
      </div>
    </div>
  );
}
