import { useMemo, useState, useRef, useEffect, useCallback } from 'react';
import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, X } from 'lucide-react';
import ResultRow from './ResultRow';
import ResultCard from './ResultCard';
import NoResults from './NoResults';
import MultiSelectFilter from './MultiSelectFilter';
import DeadlineFilter from './DeadlineFilter';
import NaicsTreeSelector from './filters/NaicsTreeSelector';
import type { FilterOption } from './MultiSelectFilter';
import type { OpportunityListItem } from '../types/api';

interface ResultsListProps {
  results: OpportunityListItem[];
  page: number;
  totalPages: number;
  loading: boolean;
  query?: string;
  onPageChange: (page: number) => void;
  sort?: string;
  order?: string;
  onSortChange?: (sort: string, order: string) => void;
  filters?: Record<string, string>;
  onFilterChange?: (field: string, value: string) => void;
  agencies?: string[];
  availableSetAsides?: string[];
  availableStates?: string[];
  selectedNaics?: string[];
  onNaicsChange?: (codes: string[]) => void;
}

export const SORTABLE_COLUMNS: Record<string, string> = {
  'Title': 'title',
  'Agency': 'department',
  'Posted': 'posted_date',
  'Set-Aside': 'set_aside_code',
  'Due': 'deadline',
  'NAICS': 'naics_code',
  'State': 'pop_state',
};

const STATE_NAMES: Record<string, string> = {
  'AL': 'Alabama', 'AK': 'Alaska', 'AZ': 'Arizona', 'AR': 'Arkansas',
  'CA': 'California', 'CO': 'Colorado', 'CT': 'Connecticut', 'DE': 'Delaware',
  'DC': 'District of Columbia', 'FL': 'Florida', 'GA': 'Georgia', 'HI': 'Hawaii',
  'ID': 'Idaho', 'IL': 'Illinois', 'IN': 'Indiana', 'IA': 'Iowa',
  'KS': 'Kansas', 'KY': 'Kentucky', 'LA': 'Louisiana', 'ME': 'Maine',
  'MD': 'Maryland', 'MA': 'Massachusetts', 'MI': 'Michigan', 'MN': 'Minnesota',
  'MS': 'Mississippi', 'MO': 'Missouri', 'MT': 'Montana', 'NE': 'Nebraska',
  'NV': 'Nevada', 'NH': 'New Hampshire', 'NJ': 'New Jersey', 'NM': 'New Mexico',
  'NY': 'New York', 'NC': 'North Carolina', 'ND': 'North Dakota', 'OH': 'Ohio',
  'OK': 'Oklahoma', 'OR': 'Oregon', 'PA': 'Pennsylvania', 'RI': 'Rhode Island',
  'SC': 'South Carolina', 'SD': 'South Dakota', 'TN': 'Tennessee', 'TX': 'Texas',
  'UT': 'Utah', 'VT': 'Vermont', 'VA': 'Virginia', 'WA': 'Washington',
  'WV': 'West Virginia', 'WI': 'Wisconsin', 'WY': 'Wyoming',
  'AS': 'American Samoa', 'GU': 'Guam', 'MP': 'Northern Mariana Islands',
  'PR': 'Puerto Rico', 'VI': 'U.S. Virgin Islands',
};

const US_STATE_CODES = [
  'AL','AK','AZ','AR','CA','CO','CT','DE','DC','FL','GA','HI','ID','IL','IN','IA',
  'KS','KY','LA','ME','MD','MA','MI','MN','MS','MO','MT','NE','NV','NH','NJ','NM',
  'NY','NC','ND','OH','OK','OR','PA','RI','SC','SD','TN','TX','UT','VT','VA','WA',
  'WV','WI','WY','AS','GU','MP','PR','VI',
];

export const STATE_FILTER_OPTIONS: FilterOption[] = US_STATE_CODES.map((s) => ({
  value: s,
  label: `${s} — ${STATE_NAMES[s] || s}`,
  searchTerms: STATE_NAMES[s] ? [STATE_NAMES[s]] : undefined,
}));

export const SET_ASIDE_FILTER_OPTIONS: FilterOption[] = [
  { value: 'SBA', label: 'SBA' },
  { value: 'SBP', label: 'Small Business', searchTerms: ['sbp', 'small business set-aside'] },
  { value: '8A', label: '8(a)', searchTerms: ['8a', 'minority', 'disadvantaged'] },
  { value: '8AN', label: '8(a) Sole Source', searchTerms: ['8a', 'minority', 'sole source'] },
  { value: 'SDVOSBC', label: 'SDVOSB', searchTerms: ['service disabled', 'veteran'] },
  { value: 'SDVOSBS', label: 'SDVOSB Sole Source', searchTerms: ['service disabled', 'veteran', 'sole source'] },
  { value: 'WOSB', label: 'WOSB', searchTerms: ['women', 'woman owned'] },
  { value: 'WOSBSS', label: 'WOSB Sole Source', searchTerms: ['women', 'woman owned', 'sole source'] },
  { value: 'EDWOSB', label: 'EDWOSB', searchTerms: ['economically disadvantaged', 'women'] },
  { value: 'EDWOSBSS', label: 'EDWOSB Sole Source', searchTerms: ['economically disadvantaged', 'women', 'sole source'] },
  { value: 'HZC', label: 'HUBZone', searchTerms: ['historically underutilized'] },
  { value: 'HZS', label: 'HUBZone Sole Source', searchTerms: ['historically underutilized', 'sole source'] },
  { value: 'VSA', label: 'VOSB', searchTerms: ['veteran owned'] },
  { value: 'VSS', label: 'VOSB Sole Source', searchTerms: ['veteran owned', 'sole source'] },
];

export const DEPT_ALIASES: Record<string, string[]> = {
  'DEPT OF DEFENSE': ['dod', 'military', 'pentagon'],
  'DEPT OF THE ARMY': ['army'],
  'DEPT OF THE NAVY': ['navy', 'usn', 'marines', 'usmc'],
  'DEPT OF THE AIR FORCE': ['usaf', 'space force', 'ussf', 'daf'],
  'DEPT OF AGRICULTURE': ['usda'],
  'DEPT OF COMMERCE': ['doc'],
  'DEPT OF EDUCATION': ['ed'],
  'DEPT OF ENERGY': ['doe'],
  'DEPT OF HEALTH AND HUMAN SERVICES': ['hhs'],
  'DEPT OF HOMELAND SECURITY': ['dhs'],
  'DEPT OF HOUSING AND URBAN DEVELOPMENT': ['hud'],
  'DEPT OF THE INTERIOR': ['doi'],
  'DEPT OF JUSTICE': ['doj'],
  'DEPT OF LABOR': ['dol'],
  'DEPT OF STATE': ['dos', 'state department'],
  'DEPT OF TRANSPORTATION': ['dot'],
  'DEPT OF THE TREASURY': ['treasury', 'irs'],
  'DEPT OF VETERANS AFFAIRS': ['va', 'veterans'],
  'ENVIRONMENTAL PROTECTION AGENCY': ['epa'],
  'GENERAL SERVICES ADMINISTRATION': ['gsa'],
  'NATIONAL AERONAUTICS AND SPACE ADMINISTRATION': ['nasa'],
  'NATIONAL SCIENCE FOUNDATION': ['nsf'],
  'NUCLEAR REGULATORY COMMISSION': ['nrc'],
  'SMALL BUSINESS ADMINISTRATION': ['sba admin'],
  'SOCIAL SECURITY ADMINISTRATION': ['ssa'],
  'AGENCY FOR INTERNATIONAL DEVELOPMENT': ['usaid'],
  'OFFICE OF PERSONNEL MANAGEMENT': ['opm'],
  'FEDERAL COMMUNICATIONS COMMISSION': ['fcc'],
  'FEDERAL EMERGENCY MANAGEMENT AGENCY': ['fema'],
};

export function parseMulti(v: string | undefined): string[] {
  return v ? v.split(',') : [];
}

function SearchInput({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder: string }) {
  return (
    <div className="relative">
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={`w-full text-xs bg-dark-800 border rounded px-1.5 py-1 pr-5
                   placeholder:text-dark-500 focus:outline-none focus:border-accent/50
                   ${value ? 'border-accent/40 text-accent' : 'border-dark-700/50 text-dark-200'}`}
      />
      {value && (
        <button
          onClick={() => onChange('')}
          className="absolute right-1 top-1/2 -translate-y-1/2 text-dark-500 hover:text-dark-300"
        >
          <X size={10} />
        </button>
      )}
    </div>
  );
}

function TableColGroup() {
  return (
    <colgroup>
      <col className="w-10" />
      <col style={{ width: '40%' }} />
      <col style={{ width: '15%' }} />
      <col style={{ width: '8%' }} />
      <col style={{ width: '8%' }} />
      <col style={{ width: '6%' }} />
      <col style={{ width: '8%' }} />
      <col style={{ width: '6%' }} />
      <col className="w-12" />
    </colgroup>
  );
}

export default function ResultsList({
  results,
  page,
  totalPages,
  loading,
  query,
  onPageChange,
  sort,
  order,
  onSortChange,
  filters,
  onFilterChange,
  agencies,
  availableSetAsides,
  availableStates,
  selectedNaics,
  onNaicsChange,
}: ResultsListProps) {
  const agencyOptions: FilterOption[] = useMemo(() =>
    (agencies || []).map((a) => ({
      value: a,
      label: a,
      searchTerms: DEPT_ALIASES[a],
    })),
    [agencies],
  );

  const setAsideOptions = useMemo(() => {
    if (!availableSetAsides) return SET_ASIDE_FILTER_OPTIONS;
    const available = new Set(availableSetAsides);
    return SET_ASIDE_FILTER_OPTIONS.filter((o) => available.has(o.value));
  }, [availableSetAsides]);

  const stateOptions = useMemo(() => {
    if (!availableStates) return STATE_FILTER_OPTIONS;
    const available = new Set(availableStates);
    return STATE_FILTER_OPTIONS.filter((o) => available.has(o.value));
  }, [availableStates]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(false);

  const updateScrollState = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    setCanScrollLeft(el.scrollLeft > 0);
    setCanScrollRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 1);
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    updateScrollState();
    el.addEventListener('scroll', updateScrollState, { passive: true });
    const ro = new ResizeObserver(updateScrollState);
    ro.observe(el);
    return () => {
      el.removeEventListener('scroll', updateScrollState);
      ro.disconnect();
    };
  }, [updateScrollState, loading, results]);

  if (loading) {
    const skeletonRows = Array.from({ length: 10 });
    return (
      <div>
        {/* Mobile skeleton */}
        <div className="md:hidden space-y-3">
          {skeletonRows.map((_, i) => (
            <div key={i} className="rounded-xl border border-dark-800/50 bg-dark-900/30 p-4 space-y-3 animate-pulse">
              <div className="h-4 bg-dark-800/60 rounded w-3/4" />
              <div className="h-3 bg-dark-800/40 rounded w-1/2" />
              <div className="h-3 bg-dark-800/30 rounded w-full" />
              <div className="flex gap-3 mt-2">
                <div className="h-5 bg-dark-800/40 rounded-full w-12" />
                <div className="h-5 bg-dark-800/40 rounded-full w-10" />
                <div className="h-5 bg-dark-800/40 rounded-full w-8" />
              </div>
            </div>
          ))}
        </div>

        {/* Desktop skeleton */}
        <div className="hidden md:block rounded-xl border border-dark-800/50 overflow-hidden bg-dark-900/30 backdrop-blur-sm">
          <table className="w-full min-w-[800px]">
            <TableColGroup />
            <thead>
              <tr className="text-[11px] text-dark-400 uppercase tracking-wider bg-dark-850/50">
                <th className="px-4 py-3" />
                {['Title', 'Agency', 'Posted', 'Set-Aside', 'Due', 'NAICS', 'State'].map((label) => (
                  <th key={label} className="px-4 py-3 text-left font-medium">{label}</th>
                ))}
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/30">
              {skeletonRows.map((_, i) => (
                <tr key={i} className="animate-pulse">
                  <td className="px-4 py-4"><div className="h-2.5 w-2.5 rounded-full bg-dark-800/50" /></td>
                  <td className="px-4 py-4 space-y-2">
                    <div className="h-3.5 bg-dark-800/50 rounded" style={{ width: `${55 + (i * 7) % 30}%` }} />
                    <div className="h-2.5 bg-dark-800/30 rounded w-1/3" />
                  </td>
                  <td className="px-4 py-4"><div className="h-3 bg-dark-800/40 rounded" style={{ width: `${50 + (i * 13) % 40}%` }} /></td>
                  <td className="px-4 py-4"><div className="h-3 bg-dark-800/40 rounded w-14" /></td>
                  <td className="px-4 py-4"><div className="h-5 bg-dark-800/40 rounded-full w-10" /></td>
                  <td className="px-4 py-4"><div className="h-3 bg-dark-800/40 rounded w-8" /></td>
                  <td className="px-4 py-4"><div className="h-3 bg-dark-800/40 rounded w-14" /></td>
                  <td className="px-4 py-4"><div className="h-3 bg-dark-800/40 rounded w-6" /></td>
                  <td className="px-4 py-4" />
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }

  if (results.length === 0 && !onFilterChange) {
    return <NoResults query={query} />;
  }

  return (
    <div>
      {/* Mobile: card layout */}
      <div className="md:hidden space-y-3">
        {results.length === 0 ? (
          <p className="text-center text-dark-400 text-sm py-12">No matching opportunities. Try adjusting your filters.</p>
        ) : (
          results.map((opp, index) => (
            <ResultCard key={opp.id} opportunity={opp} index={index} />
          ))
        )}
      </div>

      {/* Desktop: table layout */}
      <div className="hidden md:block relative">
        <div ref={scrollRef} className="rounded-xl border border-dark-800/50 overflow-x-auto bg-dark-900/30 backdrop-blur-sm">
        <table className="w-full min-w-[800px]">
          <TableColGroup />
          <thead>
            <tr className="text-[11px] text-dark-400 uppercase tracking-wider bg-dark-850/50">
              <th className="px-4 py-3"></th>
              {['Title', 'Agency', 'Posted', 'Set-Aside', 'Due', 'NAICS', 'State'].map((label) => {
                const sortField = SORTABLE_COLUMNS[label];
                const isSortable = onSortChange && sortField;
                const isActive = sortField === sort;
                return (
                  <th
                    key={label}
                    className={`px-4 py-3 text-left font-medium ${isSortable ? 'cursor-pointer select-none hover:text-dark-200 transition-colors' : ''}`}
                    onClick={isSortable ? () => {
                      const newOrder = isActive && order === 'desc' ? 'asc' : 'desc';
                      onSortChange(sortField, isActive ? newOrder : 'desc');
                    } : undefined}
                  >
                    <span className="inline-flex items-center gap-1">
                      {label}
                      {isSortable && isActive && (
                        order === 'asc'
                          ? <ChevronUp size={12} strokeWidth={2} className="text-accent" />
                          : <ChevronDown size={12} strokeWidth={2} className="text-accent" />
                      )}
                    </span>
                  </th>
                );
              })}
              <th className="px-4 py-3"></th>
            </tr>
            {onFilterChange && (
              <tr className="bg-dark-850/30">
                <th className="px-4 py-2"></th>
                {/* Title */}
                <th className="px-4 py-2">
                  <SearchInput value={filters?.title ?? ''} onChange={(v) => onFilterChange('title', v)} placeholder="Search titles..." />
                </th>
                {/* Agency */}
                <th className="px-4 py-2">
                  <MultiSelectFilter
                    options={agencyOptions}
                    selected={parseMulti(filters?.department)}
                    onChange={(sel) => onFilterChange('department', sel.join(','))}
                  />
                </th>
                {/* Posted — skip */}
                <th className="px-4 py-2"></th>
                {/* Set-Aside */}
                <th className="px-4 py-2">
                  <MultiSelectFilter
                    options={setAsideOptions}
                    selected={parseMulti(filters?.set_aside)}
                    onChange={(sel) => onFilterChange('set_aside', sel.join(','))}
                  />
                </th>
                {/* Due */}
                <th className="px-4 py-2">
                  <DeadlineFilter
                    value={filters?.deadline ?? ''}
                    onChange={(v) => onFilterChange('deadline', v)}
                  />
                </th>
                {/* NAICS */}
                <th className="px-4 py-2">
                  {onNaicsChange ? (
                    <NaicsTreeSelector
                      selected={selectedNaics || []}
                      onChange={onNaicsChange}
                      compact
                    />
                  ) : (
                    <SearchInput value={filters?.naics_prefix ?? ''} onChange={(v) => onFilterChange('naics_prefix', v)} placeholder="e.g. 33" />
                  )}
                </th>
                {/* State */}
                <th className="px-4 py-2">
                  <MultiSelectFilter
                    options={stateOptions}
                    selected={parseMulti(filters?.state)}
                    onChange={(sel) => onFilterChange('state', sel.join(','))}
                  />
                </th>
                <th className="px-4 py-2"></th>
              </tr>
            )}
          </thead>
          <tbody className="divide-y divide-dark-800/30">
            {results.length === 0 ? (
              <tr>
                <td colSpan={9} className="px-4 py-12 text-center text-dark-400 text-sm">
                  No matching opportunities. Try adjusting your filters.
                </td>
              </tr>
            ) : (
              results.map((opp, index) => (
                <ResultRow key={opp.id} opportunity={opp} index={index} />
              ))
            )}
          </tbody>
        </table>
        </div>
        {canScrollLeft && (
          <div className="absolute left-0 top-0 bottom-0 w-6 rounded-l-xl bg-gradient-to-r from-dark-950/80 to-transparent pointer-events-none z-10" />
        )}
        {canScrollRight && (
          <div className="absolute right-0 top-0 bottom-0 w-6 rounded-r-xl bg-gradient-to-l from-dark-950/80 to-transparent pointer-events-none z-10" />
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 mt-6">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="p-2 rounded-lg border border-dark-700/50 bg-dark-900/50
                       text-dark-400 hover:text-dark-100 hover:border-dark-600/50 hover:bg-dark-800/50
                       disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:bg-dark-900/50
                       transition-all duration-200"
          >
            <ChevronLeft size={18} strokeWidth={1.5} />
          </button>
          <span className="text-sm text-dark-400 px-4 tabular-nums">
            <span className="text-dark-200">{page}</span>
            <span className="mx-2 text-dark-600">/</span>
            <span>{totalPages}</span>
          </span>
          <button
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="p-2 rounded-lg border border-dark-700/50 bg-dark-900/50
                       text-dark-400 hover:text-dark-100 hover:border-dark-600/50 hover:bg-dark-800/50
                       disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:bg-dark-900/50
                       transition-all duration-200"
          >
            <ChevronRight size={18} strokeWidth={1.5} />
          </button>
        </div>
      )}
    </div>
  );
}
