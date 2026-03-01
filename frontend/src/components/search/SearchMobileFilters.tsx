import { useEffect, useCallback, useRef } from 'react';
import { createPortal } from 'react-dom';
import { X } from 'lucide-react';
import {
  NaicsTreeSelector,
  PscTreeSelector,
  SearchableDropdownFilter,
  EnhancedDeadlineFilter,
  SimpleToggleFilter,
  AgencyFilter,
  NOTICE_TYPE_OPTIONS,
} from '../filters';
import { SET_ASIDE_FILTER_OPTIONS } from '../ResultsList';
import type { FilterState, UseFilterStateReturn } from '../../hooks/useFilterState';
import type { FacetResult } from '../../types/api';
import { useMemo } from 'react';

interface SearchMobileFiltersProps {
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
  setFilters: UseFilterStateReturn['setFilters'];
  clearAllFilters: UseFilterStateReturn['clearAllFilters'];
  filterCount: number;
  facets: FacetResult['facets'] | null;
  facetsLoading: boolean;
  total: number;
  open: boolean;
  onClose: () => void;
}

export default function SearchMobileFilters({
  filters,
  setFilter,
  setFilters,
  clearAllFilters,
  filterCount,
  facets,
  facetsLoading,
  total,
  open,
  onClose,
}: SearchMobileFiltersProps) {
  const sheetRef = useRef<HTMLDivElement>(null);

  // Body scroll lock
  useEffect(() => {
    if (!open) return;
    document.body.style.overflow = 'hidden';
    return () => { document.body.style.overflow = ''; };
  }, [open]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  // Focus first interactive element on open
  useEffect(() => {
    if (!open) return;
    setTimeout(() => {
      const el = sheetRef.current?.querySelector<HTMLElement>('input, button, select');
      el?.focus();
    }, 100);
  }, [open]);

  const setAsideOptions = useMemo(() => {
    const countMap = new Map(facets?.set_aside?.map((f) => [f.value, f.count]) ?? []);
    return SET_ASIDE_FILTER_OPTIONS.map((o) => ({
      ...o,
      count: countMap.get(o.value),
    }));
  }, [facets?.set_aside]);

  const noticeTypeOptions = useMemo(() => {
    const countMap = new Map(facets?.notice_type?.map((f) => [f.value, f.count]) ?? []);
    return NOTICE_TYPE_OPTIONS.map((o) => ({
      ...o,
      count: countMap.get(o.value),
    }));
  }, [facets?.notice_type]);

  const handleDeadlineChange = useCallback(
    (values: { deadlinePreset: string; deadlineFrom: string; deadlineTo: string }) => {
      setFilters({
        deadlinePreset: values.deadlinePreset,
        deadlineFrom: values.deadlineFrom,
        deadlineTo: values.deadlineTo,
      });
    },
    [setFilters],
  );

  const handleReset = useCallback(() => {
    clearAllFilters();
  }, [clearAllFilters]);

  if (!open) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex flex-col">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60"
        onClick={onClose}
      />

      {/* Sheet */}
      <div
        ref={sheetRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="mobile-search-filters-title"
        className="relative mt-12 flex-1 flex flex-col bg-dark-900 rounded-t-2xl overflow-hidden animate-slide-up"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-dark-800/50">
          <h3 id="mobile-search-filters-title" className="text-base font-medium text-dark-100">
            Filters
            {filterCount > 0 && (
              <span className="ml-2 inline-flex items-center justify-center h-5 min-w-[20px] rounded-full bg-accent text-[11px] font-semibold text-white px-1.5">
                {filterCount}
              </span>
            )}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 text-dark-400 hover:text-dark-200 rounded-lg"
          >
            <X size={20} strokeWidth={1.5} />
          </button>
        </div>

        {/* Scrollable filter sections */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-6">
          {/* NAICS */}
          <FilterSection label="NAICS">
            <NaicsTreeSelector
              selected={filters.naics}
              onChange={(sel) => setFilter('naics', sel)}
              facets={facets?.naics}
              inline
              loading={facetsLoading}
            />
          </FilterSection>

          {/* PSC */}
          <FilterSection label="PSC">
            <PscTreeSelector
              selected={filters.psc}
              onChange={(sel) => setFilter('psc', sel)}
              facets={facets?.psc}
              inline
              loading={facetsLoading}
            />
          </FilterSection>

          {/* Set-Aside */}
          <FilterSection label="Set-Aside">
            <SearchableDropdownFilter
              label="Set-Aside"
              options={setAsideOptions}
              selected={filters.setAside}
              onSelectionChange={(sel) => setFilter('setAside', sel)}
              searchPlaceholder="Search set-asides..."
              loading={facetsLoading}
            />
          </FilterSection>


          {/* Agency */}
          <FilterSection label="Agency">
            <AgencyFilter
              selected={filters.agency}
              onChange={(sel) => setFilter('agency', sel)}
            />
          </FilterSection>

          {/* Deadline */}
          <FilterSection label="Deadline">
            <EnhancedDeadlineFilter
              deadlinePreset={filters.deadlinePreset}
              deadlineFrom={filters.deadlineFrom}
              deadlineTo={filters.deadlineTo}
              onChange={handleDeadlineChange}
            />
          </FilterSection>

          {/* Notice Type */}
          <FilterSection label="Notice Type">
            <SimpleToggleFilter
              label="Notice Type"
              options={noticeTypeOptions}
              selected={filters.noticeType}
              onSelectionChange={(sel) => setFilter('noticeType', sel)}
              loading={facetsLoading}
            />
          </FilterSection>

          {/* Exact Match */}
          <FilterSection label="Exact Match">
            <button
              type="button"
              onClick={() => setFilter('exactMatch', !filters.exactMatch)}
              className={`inline-flex items-center gap-2 px-3 py-2.5 text-sm rounded-lg border cursor-pointer transition-colors
                ${filters.exactMatch
                  ? 'bg-accent/10 border-accent/40 text-accent'
                  : 'bg-dark-800 border-dark-700/50 text-dark-300 hover:border-dark-600'
                }`}
            >
              <span className={`relative inline-flex h-4 w-7 items-center rounded-full transition-colors
                ${filters.exactMatch ? 'bg-accent' : 'bg-dark-600'}`}
              >
                <span className={`inline-block h-3 w-3 rounded-full bg-white transition-transform
                  ${filters.exactMatch ? 'translate-x-3.5' : 'translate-x-0.5'}`}
                />
              </span>
              Exact Match
            </button>
          </FilterSection>

          {/* Active Only */}
          <FilterSection label="Active Only">
            <button
              type="button"
              onClick={() => setFilter('activeOnly', !filters.activeOnly)}
              className={`inline-flex items-center gap-2 px-3 py-2.5 text-sm rounded-lg border cursor-pointer transition-colors
                ${filters.activeOnly
                  ? 'bg-accent/10 border-accent/40 text-accent'
                  : 'bg-dark-800 border-dark-700/50 text-dark-300 hover:border-dark-600'
                }`}
            >
              <span className={`relative inline-flex h-4 w-7 items-center rounded-full transition-colors
                ${filters.activeOnly ? 'bg-accent' : 'bg-dark-600'}`}
              >
                <span className={`inline-block h-3 w-3 rounded-full bg-white transition-transform
                  ${filters.activeOnly ? 'translate-x-3.5' : 'translate-x-0.5'}`}
                />
              </span>
              Active Only
            </button>
          </FilterSection>

          {/* Posted Date */}
          <FilterSection label="Posted Date">
            <div className="flex gap-2 items-center">
              <div className="flex-1">
                <span className="text-[11px] text-dark-500 block mb-1">From</span>
                <input
                  type="date"
                  value={filters.postedFrom}
                  onChange={(e) => setFilter('postedFrom', e.target.value)}
                  className="w-full text-sm bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2.5 text-dark-200
                    focus:outline-none focus:border-accent/50"
                />
              </div>
              <div className="flex-1">
                <span className="text-[11px] text-dark-500 block mb-1">To</span>
                <input
                  type="date"
                  value={filters.postedTo}
                  min={filters.postedFrom || undefined}
                  onChange={(e) => setFilter('postedTo', e.target.value)}
                  className="w-full text-sm bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2.5 text-dark-200
                    focus:outline-none focus:border-accent/50"
                />
              </div>
            </div>
          </FilterSection>
        </div>

        {/* Footer */}
        <div className="px-5 py-4 border-t border-dark-800/50 flex items-center gap-3">
          {filterCount > 0 && (
            <button
              type="button"
              onClick={handleReset}
              className="text-sm text-dark-400 hover:text-dark-200 transition-colors"
            >
              Reset
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            className="flex-1 py-3 rounded-xl bg-accent text-dark-950 text-sm font-semibold
                     active:opacity-90 transition-opacity"
          >
            Show {total.toLocaleString()} result{total !== 1 ? 's' : ''}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}

function FilterSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-xs text-dark-400 uppercase tracking-wider mb-2">{label}</label>
      {children}
    </div>
  );
}
