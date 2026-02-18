import { useState } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import type { AdvancedFilters } from '../types/api';

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
}

function MultiSelect({ options, selected, onChange }: MultiSelectProps) {
  const toggle = (code: string) => {
    if (selected.includes(code)) {
      onChange(selected.filter((s) => s !== code));
    } else {
      onChange([...selected, code]);
    }
  };

  return (
    <div className="max-h-40 overflow-y-auto space-y-1">
      {options.map((option) => (
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
    </div>
  );
}

const TYPE_OPTIONS = [
  { code: 'Solicitation', label: 'Solicitation' },
  { code: 'Presolicitation', label: 'Presolicitation' },
  { code: 'Combined Synopsis/Solicitation', label: 'Combined Synopsis/Solicitation' },
  { code: 'Sources Sought', label: 'Sources Sought' },
  { code: 'Special Notice', label: 'Special Notice' },
  { code: 'Award Notice', label: 'Award Notice' },
  { code: 'Intent to Bundle', label: 'Intent to Bundle' },
];

const DISABLED_FILTERS = ['Set-Aside', 'NAICS Code', 'State'];

export default function FilterPanel({ filters, onChange }: FilterPanelProps) {
  const updateFilter = <K extends keyof AdvancedFilters>(key: K, value: AdvancedFilters[K]) => {
    onChange({ ...filters, [key]: value });
  };

  const activeCount = [
    filters.types.length,
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

      <AccordionSection title="Type" defaultOpen>
        <MultiSelect
          options={TYPE_OPTIONS}
          selected={filters.types}
          onChange={(v) => updateFilter('types', v)}
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

      {/* Disabled filters */}
      <div className="mt-3 rounded-lg border border-fuchsia-500 bg-fuchsia-500/10 p-3 opacity-50">
        <p className="text-xs text-fuchsia-400 mb-2">Coming soon</p>
        <div className="space-y-2 text-sm text-dark-500">
          {DISABLED_FILTERS.map((name) => (
            <p key={name}>{name}</p>
          ))}
        </div>
      </div>
    </div>
  );
}
