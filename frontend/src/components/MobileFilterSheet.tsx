import { useState, useCallback, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { SlidersHorizontal, X, ChevronDown, ChevronUp, Search } from 'lucide-react';
import MultiSelectFilter from './MultiSelectFilter';
import NaicsTreeSelector from './filters/NaicsTreeSelector';
import type { FilterOption } from './MultiSelectFilter';
import {
  SORTABLE_COLUMNS,
  SET_ASIDE_FILTER_OPTIONS,
  STATE_FILTER_OPTIONS,
  DEPT_ALIASES,
  parseMulti,
} from './ResultsList';

interface Props {
  filters: Record<string, string>;
  onFilterChange: (field: string, value: string) => void;
  sort: string;
  order: string;
  onSortChange: (sort: string, order: string) => void;
  agencies: string[];
  availableSetAsides: string[];
  availableStates: string[];
  selectedNaics: string[];
  onNaicsChange: (codes: string[]) => void;
  resultCount: number;
  onReset: () => void;
}

const SORT_LABELS: Record<string, string> = {};
for (const [label, field] of Object.entries(SORTABLE_COLUMNS)) {
  SORT_LABELS[field] = label;
}

export default function MobileFilterSheet({
  filters,
  onFilterChange,
  sort,
  order,
  onSortChange,
  agencies,
  availableSetAsides,
  availableStates,
  selectedNaics,
  onNaicsChange,
  resultCount,
  onReset,
}: Props) {
  const [open, setOpen] = useState(false);

  const agencyOptions: FilterOption[] = useMemo(() =>
    agencies.map((a) => ({
      value: a,
      label: a,
      searchTerms: DEPT_ALIASES[a],
    })),
    [agencies],
  );

  const setAsideOptions = useMemo(() => {
    const available = new Set(availableSetAsides);
    return SET_ASIDE_FILTER_OPTIONS.filter((o) => available.has(o.value));
  }, [availableSetAsides]);

  const stateOptions = useMemo(() => {
    const available = new Set(availableStates);
    return STATE_FILTER_OPTIONS.filter((o) => available.has(o.value));
  }, [availableStates]);

  const activeCount = useMemo(() => {
    let count = 0;
    if (filters.title) count++;
    if (filters.department) count++;
    if (filters.set_aside) count++;
    if (filters.state) count++;
    if (selectedNaics.length > 0) count++;
    if (sort !== 'posted_date' || order !== 'desc') count++;
    return count;
  }, [filters, selectedNaics, sort, order]);

  const handleReset = useCallback(() => {
    onReset();
    setOpen(false);
  }, [onReset]);

  return (
    <>
      {/* Sticky trigger button — mobile only */}
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="md:hidden fixed bottom-20 left-1/2 -translate-x-1/2 z-40
                   flex items-center gap-2 px-5 py-3 rounded-full
                   bg-dark-800 border border-dark-700/50 shadow-2xl shadow-black/50
                   text-sm text-dark-200 active:bg-dark-700 transition-colors"
      >
        <SlidersHorizontal size={16} strokeWidth={1.5} />
        <span>Filters</span>
        {activeCount > 0 && (
          <span className="flex items-center justify-center h-5 min-w-[20px] rounded-full bg-accent text-dark-950 text-[11px] font-semibold px-1.5">
            {activeCount}
          </span>
        )}
      </button>

      {/* Sheet overlay — mobile only */}
      {open && createPortal(
        <div className="md:hidden fixed inset-0 z-50 flex flex-col">
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/60"
            onClick={() => setOpen(false)}
          />

          {/* Sheet */}
          <div role="dialog" aria-modal="true" aria-labelledby="mobile-filters-title" className="relative mt-12 flex-1 flex flex-col bg-dark-900 rounded-t-2xl overflow-hidden animate-slide-up">
            {/* Header */}
            <div className="flex items-center justify-between px-5 py-4 border-b border-dark-800/50">
              <h3 id="mobile-filters-title" className="text-base font-medium text-dark-100">Filters</h3>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="p-1.5 text-dark-400 hover:text-dark-200 rounded-lg"
              >
                <X size={20} strokeWidth={1.5} />
              </button>
            </div>

            {/* Scrollable filter sections */}
            <div className="flex-1 overflow-y-auto px-5 py-4 space-y-6">
              {/* Title search */}
              <FilterSection label="Title">
                <div className="relative">
                  <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-500" />
                  <input
                    type="text"
                    placeholder="Search titles..."
                    value={filters.title ?? ''}
                    onChange={(e) => onFilterChange('title', e.target.value)}
                    className={`w-full text-sm bg-dark-800 border rounded-lg pl-9 pr-3 py-2.5
                               placeholder:text-dark-500 focus:outline-none focus:border-accent/50
                               ${filters.title ? 'border-accent/40 text-accent' : 'border-dark-700/50 text-dark-200'}`}
                  />
                  {filters.title && (
                    <button
                      onClick={() => onFilterChange('title', '')}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-500 hover:text-dark-300"
                    >
                      <X size={14} />
                    </button>
                  )}
                </div>
              </FilterSection>

              {/* Agency */}
              <FilterSection label="Agency">
                <MultiSelectFilter
                  options={agencyOptions}
                  selected={parseMulti(filters.department)}
                  onChange={(sel) => onFilterChange('department', sel.join(','))}
                  placeholder="All agencies"
                />
              </FilterSection>

              {/* Set-Aside */}
              <FilterSection label="Set-Aside">
                <MultiSelectFilter
                  options={setAsideOptions}
                  selected={parseMulti(filters.set_aside)}
                  onChange={(sel) => onFilterChange('set_aside', sel.join(','))}
                  placeholder="All set-asides"
                />
              </FilterSection>

              {/* NAICS */}
              <FilterSection label="NAICS">
                <NaicsTreeSelector
                  selected={selectedNaics}
                  onChange={onNaicsChange}
                  inline
                />
              </FilterSection>

              {/* State */}
              <FilterSection label="State">
                <MultiSelectFilter
                  options={stateOptions}
                  selected={parseMulti(filters.state)}
                  onChange={(sel) => onFilterChange('state', sel.join(','))}
                  placeholder="All states"
                />
              </FilterSection>

              {/* Sort */}
              <FilterSection label="Sort by">
                <div className="flex gap-2">
                  <select
                    value={sort}
                    onChange={(e) => onSortChange(e.target.value, order)}
                    className="flex-1 text-sm bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2.5
                             text-dark-200 focus:outline-none focus:border-accent/50 appearance-none"
                  >
                    {Object.entries(SORTABLE_COLUMNS).map(([label, field]) => (
                      <option key={field} value={field}>{label}</option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => onSortChange(sort, order === 'desc' ? 'asc' : 'desc')}
                    className="flex items-center gap-1 px-3 py-2.5 bg-dark-800 border border-dark-700/50 rounded-lg
                             text-sm text-dark-200"
                  >
                    {order === 'desc' ? (
                      <><ChevronDown size={14} strokeWidth={2} /><span>Desc</span></>
                    ) : (
                      <><ChevronUp size={14} strokeWidth={2} /><span>Asc</span></>
                    )}
                  </button>
                </div>
              </FilterSection>
            </div>

            {/* Footer */}
            <div className="px-5 py-4 border-t border-dark-800/50 flex items-center gap-3">
              {activeCount > 0 && (
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
                onClick={() => setOpen(false)}
                className="flex-1 py-3 rounded-xl bg-accent text-dark-950 text-sm font-semibold
                         active:opacity-90 transition-opacity"
              >
                Show {resultCount.toLocaleString()} result{resultCount !== 1 ? 's' : ''}
              </button>
            </div>
          </div>
        </div>,
        document.body,
      )}
    </>
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
