import { useState, useEffect } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { getFilters } from '../services/api';
import type { AdvancedFilters, FilterOptions } from '../types/api';

interface FilterPanelProps {
  filters: AdvancedFilters;
  onChange: (filters: AdvancedFilters) => void;
}

interface AccordionSectionProps {
  title: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
}

function AccordionSection({ title, children, defaultOpen = false }: AccordionSectionProps) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div className="border-b border-dark-800/30 last:border-0">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between py-3 text-sm text-dark-300 hover:text-dark-100 transition-colors duration-200"
      >
        <span className="font-medium">{title}</span>
        {open ? (
          <ChevronUp size={16} strokeWidth={1.5} className="text-dark-500" />
        ) : (
          <ChevronDown size={16} strokeWidth={1.5} className="text-dark-500" />
        )}
      </button>
      {open && <div className="pb-4">{children}</div>}
    </div>
  );
}

interface MultiSelectProps {
  options: Array<{ code: string; label?: string; count?: number }>;
  selected: string[];
  onChange: (selected: string[]) => void;
  searchable?: boolean;
}

function MultiSelect({ options, selected, onChange, searchable }: MultiSelectProps) {
  const [search, setSearch] = useState('');

  const filtered = searchable && search
    ? options.filter((o) =>
        o.code.toLowerCase().includes(search.toLowerCase()) ||
        (o.label && o.label.toLowerCase().includes(search.toLowerCase()))
      )
    : options;

  const toggle = (code: string) => {
    if (selected.includes(code)) {
      onChange(selected.filter((s) => s !== code));
    } else {
      onChange([...selected, code]);
    }
  };

  return (
    <div className="space-y-2">
      {searchable && (
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search..."
          className="w-full py-2 px-3 text-xs bg-dark-850/50 border border-dark-700/50 rounded-lg
                     text-dark-100 placeholder:text-dark-500
                     focus:outline-none focus:border-accent/50 transition-colors duration-200"
        />
      )}
      <div className="max-h-40 overflow-y-auto space-y-1">
        {filtered.map((option) => (
          <label
            key={option.code}
            className="flex items-center gap-2.5 text-sm text-dark-400 hover:text-dark-200 cursor-pointer py-1.5 px-1 rounded-lg hover:bg-dark-800/30 transition-colors duration-150"
          >
            <input
              type="checkbox"
              checked={selected.includes(option.code)}
              onChange={() => toggle(option.code)}
              className="w-3.5 h-3.5 rounded border-dark-600 bg-dark-800 text-accent
                         focus:ring-1 focus:ring-accent/50 focus:ring-offset-0 cursor-pointer"
            />
            <span className="flex-1 truncate">{option.label || option.code}</span>
            {option.count !== undefined && (
              <span className="text-xs text-dark-600 tabular-nums">{option.count}</span>
            )}
          </label>
        ))}
        {filtered.length === 0 && (
          <p className="text-xs text-dark-500 py-2 text-center">No options match</p>
        )}
      </div>
    </div>
  );
}

const TYPE_OPTIONS = [
  { code: 'o', label: 'Solicitation' },
  { code: 'p', label: 'Pre-solicitation' },
  { code: 'k', label: 'Combined Synopsis/Solicitation' },
  { code: 'r', label: 'Sources Sought' },
  { code: 's', label: 'Special Notice' },
  { code: 'a', label: 'Award Notice' },
  { code: 'i', label: 'Intent to Bundle' },
];

const SET_ASIDE_OPTIONS = [
  { code: 'SBA', label: 'Small Business' },
  { code: '8A', label: '8(a)' },
  { code: 'SDVOSB', label: 'Service-Disabled Veteran-Owned' },
  { code: 'WOSB', label: 'Women-Owned' },
  { code: 'HUBZone', label: 'HUBZone' },
  { code: 'NONE', label: 'No Set-Aside' },
];

const US_STATES = [
  'AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'FL', 'GA',
  'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME', 'MD',
  'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH', 'NJ',
  'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI', 'SC',
  'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY', 'DC',
].map((s) => ({ code: s, label: s }));

export default function FilterPanel({ filters, onChange }: FilterPanelProps) {
  const [apiFilters, setApiFilters] = useState<FilterOptions | null>(null);

  useEffect(() => {
    getFilters().then(setApiFilters).catch(() => {});
  }, []);

  const updateFilter = <K extends keyof AdvancedFilters>(key: K, value: AdvancedFilters[K]) => {
    onChange({ ...filters, [key]: value });
  };

  const typeOptions = apiFilters?.types.length
    ? apiFilters.types.map((t) => ({
        code: t.code,
        label: TYPE_OPTIONS.find((o) => o.code === t.code)?.label || t.code,
        count: t.count,
      }))
    : TYPE_OPTIONS;

  const setAsideOptions = apiFilters?.set_asides.length
    ? apiFilters.set_asides.map((s) => ({
        code: s.code,
        label: SET_ASIDE_OPTIONS.find((o) => o.code === s.code)?.label || s.code,
        count: s.count,
      }))
    : SET_ASIDE_OPTIONS;

  const stateOptions = apiFilters?.states.length
    ? apiFilters.states.map((s) => ({ code: s.code, label: s.code, count: s.count }))
    : US_STATES;

  const activeCount = [
    filters.types.length,
    filters.setAsides.length,
    filters.naicsCodes.length,
    filters.states.length,
    filters.postedFrom ? 1 : 0,
    filters.postedTo ? 1 : 0,
    filters.deadlineFrom ? 1 : 0,
    filters.deadlineTo ? 1 : 0,
  ].reduce((a, b) => a + b, 0);

  return (
    <div className="rounded-xl border border-dark-800/50 bg-dark-900/30 p-4">
      {activeCount > 0 && (
        <div className="flex items-center justify-end mb-2 -mt-1">
          <button
            onClick={() =>
              onChange({
                types: [],
                setAsides: [],
                naicsCodes: [],
                states: [],
                postedFrom: undefined,
                postedTo: undefined,
                deadlineFrom: undefined,
                deadlineTo: undefined,
              })
            }
            className="text-xs text-accent hover:text-accent-hover transition-colors duration-200"
          >
            Clear all ({activeCount})
          </button>
        </div>
      )}

      <AccordionSection title="Type">
        <MultiSelect
          options={typeOptions}
          selected={filters.types}
          onChange={(v) => updateFilter('types', v)}
        />
      </AccordionSection>

      <AccordionSection title="Set-Aside">
        <MultiSelect
          options={setAsideOptions}
          selected={filters.setAsides}
          onChange={(v) => updateFilter('setAsides', v)}
        />
      </AccordionSection>

      <AccordionSection title="NAICS Code">
        <input
          type="text"
          value={filters.naicsCodes.join(', ')}
          onChange={(e) =>
            updateFilter(
              'naicsCodes',
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean)
            )
          }
          placeholder="e.g., 541512, 518210"
          className="w-full py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                     text-dark-100 placeholder:text-dark-500
                     focus:outline-none focus:border-accent/50 transition-colors duration-200"
        />
        <p className="text-[10px] text-dark-500 mt-1.5">Comma-separated codes</p>
      </AccordionSection>

      <AccordionSection title="State">
        <MultiSelect
          options={stateOptions}
          selected={filters.states}
          onChange={(v) => updateFilter('states', v)}
          searchable
        />
      </AccordionSection>

      <AccordionSection title="Posted Date">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-[10px] text-dark-500 block mb-1.5 uppercase tracking-wider">From</label>
            <input
              type="date"
              value={filters.postedFrom || ''}
              onChange={(e) => updateFilter('postedFrom', e.target.value || undefined)}
              className="w-full py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                         text-dark-100 focus:outline-none focus:border-accent/50 transition-colors duration-200"
            />
          </div>
          <div>
            <label className="text-[10px] text-dark-500 block mb-1.5 uppercase tracking-wider">To</label>
            <input
              type="date"
              value={filters.postedTo || ''}
              onChange={(e) => updateFilter('postedTo', e.target.value || undefined)}
              className="w-full py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                         text-dark-100 focus:outline-none focus:border-accent/50 transition-colors duration-200"
            />
          </div>
        </div>
      </AccordionSection>

      <AccordionSection title="Response Deadline">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-[10px] text-dark-500 block mb-1.5 uppercase tracking-wider">From</label>
            <input
              type="date"
              value={filters.deadlineFrom || ''}
              onChange={(e) => updateFilter('deadlineFrom', e.target.value || undefined)}
              className="w-full py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                         text-dark-100 focus:outline-none focus:border-accent/50 transition-colors duration-200"
            />
          </div>
          <div>
            <label className="text-[10px] text-dark-500 block mb-1.5 uppercase tracking-wider">To</label>
            <input
              type="date"
              value={filters.deadlineTo || ''}
              onChange={(e) => updateFilter('deadlineTo', e.target.value || undefined)}
              className="w-full py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                         text-dark-100 focus:outline-none focus:border-accent/50 transition-colors duration-200"
            />
          </div>
        </div>
      </AccordionSection>
    </div>
  );
}
