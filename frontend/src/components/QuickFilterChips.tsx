import { Search, ShieldCheck, Clock } from 'lucide-react';
import type { FilterState, UseFilterStateReturn } from '../hooks/useFilterState';

type QuickFilter =
  | { type: 'keyword'; label: string; value: string }
  | { type: 'setAside'; label: string; value: string }
  | { type: 'deadline'; label: string; preset: string };

const CHIPS: QuickFilter[] = [
  { type: 'keyword', label: 'IT Services', value: 'IT services' },
  { type: 'keyword', label: 'Cybersecurity', value: 'cybersecurity' },
  { type: 'keyword', label: 'Construction', value: 'construction' },
  { type: 'keyword', label: 'Professional Services', value: 'professional services' },
  { type: 'setAside', label: 'Small Business', value: 'SBA' },
  { type: 'deadline', label: 'Closing This Week', preset: '7' },
];

const ICONS = {
  keyword: Search,
  setAside: ShieldCheck,
  deadline: Clock,
} as const;

function isActive(chip: QuickFilter, filters: FilterState): boolean {
  switch (chip.type) {
    case 'keyword':
      return filters.keyword.toLowerCase() === chip.value.toLowerCase();
    case 'setAside':
      return filters.setAside.includes(chip.value);
    case 'deadline':
      return filters.deadlinePreset === chip.preset;
  }
}

interface QuickFilterChipsProps {
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
  onSearch?: () => void;
}

export default function QuickFilterChips({ filters, setFilter, onSearch }: QuickFilterChipsProps) {
  const toggle = (chip: QuickFilter) => {
    const active = isActive(chip, filters);
    switch (chip.type) {
      case 'keyword':
        setFilter('keyword', active ? '' : chip.value);
        break;
      case 'setAside':
        setFilter(
          'setAside',
          active
            ? filters.setAside.filter((v) => v !== chip.value)
            : [...filters.setAside, chip.value],
        );
        break;
      case 'deadline':
        setFilter('deadlinePreset', active ? '' : chip.preset);
        break;
    }
    onSearch?.();
  };

  return (
    <div className="flex flex-wrap items-center justify-center gap-2">
      {CHIPS.map((chip) => {
        const active = isActive(chip, filters);
        const Icon = ICONS[chip.type];
        return (
          <button
            key={chip.label}
            type="button"
            onClick={() => toggle(chip)}
            className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm border transition-all duration-150 cursor-pointer
              ${active
                ? 'border-accent/30 bg-accent/10 text-accent'
                : 'border-dark-700/50 bg-dark-800/30 text-dark-400 hover:border-dark-600 hover:text-dark-300'
              }`}
          >
            <Icon size={13} strokeWidth={1.5} />
            {chip.label}
          </button>
        );
      })}
    </div>
  );
}
