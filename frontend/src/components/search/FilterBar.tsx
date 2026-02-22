import { useState, useMemo, useRef, useCallback } from 'react';
import { Search, Filter, ChevronUp, ChevronDown } from 'lucide-react';
import SearchInput from '../SearchInput';
import {
  SearchableDropdownFilter,
  EnhancedDeadlineFilter,
  SimpleToggleFilter,
  FilterChipBar,
  NaicsTreeSelector,
  NOTICE_TYPE_OPTIONS,
} from '../filters';
import { SET_ASIDE_FILTER_OPTIONS } from '../ResultsList';
import AdvancedFilterBuilder from '../filters/AdvancedFilterBuilder';
import MoreFiltersPanel from './MoreFiltersPanel';
import type { FilterState, UseFilterStateReturn } from '../../hooks/useFilterState';
import type { FacetResult } from '../../types/api';

interface FilterBarProps {
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
  setFilters: UseFilterStateReturn['setFilters'];
  removeFilter: UseFilterStateReturn['removeFilter'];
  clearFilter: UseFilterStateReturn['clearFilter'];
  clearAllFilters: UseFilterStateReturn['clearAllFilters'];
  filterCount: number;
  facets: FacetResult['facets'] | null;
  facetsLoading: boolean;
  loading: boolean;
  onSearch: () => void;
  total: number;
  onMobileFiltersOpen?: () => void;
}

export default function FilterBar({
  filters,
  setFilter,
  setFilters,
  removeFilter,
  clearFilter,
  clearAllFilters,
  filterCount,
  facets,
  facetsLoading,
  loading,
  onSearch,
  total,
  onMobileFiltersOpen,
}: FilterBarProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Set-aside options with facet counts
  const setAsideOptions = useMemo(() => {
    const countMap = new Map(facets?.set_aside?.map((f) => [f.value, f.count]) ?? []);
    return SET_ASIDE_FILTER_OPTIONS.map((o) => ({
      ...o,
      count: countMap.get(o.value),
    }));
  }, [facets?.set_aside]);

  // Agency options from facets
  const agencyOptions = useMemo(() => {
    if (!facets?.agency) return [];
    return facets.agency.map((f) => ({
      value: f.value,
      label: f.label || f.value,
      count: f.count,
    }));
  }, [facets?.agency]);

  // Notice type options with facet counts
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

  const handleMoreFiltersChange = useCallback(
    (values: { postedFrom: string; postedTo: string }) => {
      setFilters({ postedFrom: values.postedFrom, postedTo: values.postedTo });
    },
    [setFilters],
  );

  const handleChipRemove = useCallback(
    (key: keyof FilterState, value?: string) => {
      if (key === 'activeOnly') {
        setFilter('activeOnly', true);
        return;
      }
      if (value && (key === 'naics' || key === 'setAside' || key === 'noticeType')) {
        removeFilter(key, value);
      } else {
        clearFilter(key);
      }
    },
    [setFilter, removeFilter, clearFilter],
  );

  return (
    <div role="search" aria-label="Search and filter opportunities" className="space-y-3">
      {/* ROW 1: Keyword search */}
      <div className="flex gap-2 items-center">
        <div className="flex-1">
          <SearchInput
            ref={inputRef}
            value={filters.keyword}
            onChange={(v) => setFilter('keyword', v)}
            onSubmit={onSearch}
            loading={loading && !!filters.keyword}
            placeholder="Search contracts, solicitations, awards..."
          />
        </div>
        <button
          type="button"
          onClick={onSearch}
          className="hidden md:flex items-center gap-1.5 px-4 py-3 rounded-xl bg-accent hover:bg-accent-hover
            text-white text-sm font-medium transition-colors shrink-0"
        >
          <Search size={16} />
          Search
        </button>
      </div>

      {/* ROW 2: Primary filter triggers (desktop) */}
      <div className="hidden md:flex flex-wrap items-center gap-2">
        <NaicsTreeSelector
          selected={filters.naics}
          onChange={(sel) => setFilter('naics', sel)}
          facets={facets?.naics}
          label="NAICS"
          loading={facetsLoading}
        />
        <SearchableDropdownFilter
          label="Set-Aside"
          options={setAsideOptions}
          selected={filters.setAside}
          onSelectionChange={(sel) => setFilter('setAside', sel)}
          searchPlaceholder="Search set-asides..."
          loading={facetsLoading}
        />
        <SearchableDropdownFilter
          label="Agency"
          options={agencyOptions}
          selected={filters.department ? [filters.department] : []}
          onSelectionChange={(sel) => setFilter('department', sel[0] || '')}
          searchPlaceholder="Search agencies..."
          loading={facetsLoading}
        />
        <EnhancedDeadlineFilter
          deadlinePreset={filters.deadlinePreset}
          deadlineFrom={filters.deadlineFrom}
          deadlineTo={filters.deadlineTo}
          onChange={handleDeadlineChange}
        />
        <SimpleToggleFilter
          label="Notice Type"
          options={noticeTypeOptions}
          selected={filters.noticeType}
          onSelectionChange={(sel) => setFilter('noticeType', sel)}
          loading={facetsLoading}
        />

        {/* Active Only toggle */}
        <button
          type="button"
          onClick={() => setFilter('activeOnly', !filters.activeOnly)}
          className={`inline-flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border cursor-pointer transition-colors
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

        <MoreFiltersPanel
          postedFrom={filters.postedFrom}
          postedTo={filters.postedTo}
          onChange={handleMoreFiltersChange}
        />

        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="inline-flex items-center gap-1 px-2 py-1.5 text-xs text-dark-400 hover:text-accent transition-colors"
        >
          Advanced filters
          {showAdvanced ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
        </button>
      </div>

      {/* Advanced filter builder (between ROW 2 and ROW 3) */}
      {showAdvanced && (
        <AdvancedFilterBuilder
          filters={filters}
          setFilter={setFilter}
          setFilters={setFilters}
          clearFilter={clearFilter}
          clearAllFilters={clearAllFilters}
          facets={facets}
          facetsLoading={facetsLoading}
          total={total}
        />
      )}

      {/* Mobile: filter button */}
      <div className="md:hidden flex items-center gap-2">
        <MobileFilterButton filterCount={filterCount} onClick={() => onMobileFiltersOpen?.()} />
      </div>

      {/* ROW 3: Filter chips */}
      <FilterChipBar
        filters={filters}
        onRemoveFilter={handleChipRemove}
        onClearAll={clearAllFilters}
      />
    </div>
  );
}

function MobileFilterButton({ filterCount, onClick }: { filterCount: number; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border
        bg-dark-800 border-dark-700/50 text-dark-200 hover:border-dark-600 transition-colors"
    >
      <Filter size={14} />
      <span>Filters</span>
      {filterCount > 0 && (
        <span className="inline-flex items-center justify-center w-4 h-4 rounded-full bg-accent text-[10px] font-medium text-white">
          {filterCount}
        </span>
      )}
    </button>
  );
}
