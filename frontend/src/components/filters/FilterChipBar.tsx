import { useMemo, useState, useCallback, useRef, useEffect } from 'react';
import { X } from 'lucide-react';
import type { FilterState } from '../../hooks/useFilterState';
import {
  NOTICE_TYPE_LABELS,
  DEFAULT_NOTICE_TYPES,
  DEADLINE_PRESET_LABELS,
  SET_ASIDE_LABELS,
  STATE_NAMES,
} from './constants';

interface FilterChipBarProps {
  filters: FilterState;
  onRemoveFilter: (key: keyof FilterState, value?: string) => void;
  onClearAll: () => void;
}

interface Chip {
  id: string;
  filterKey: keyof FilterState;
  value?: string;
  label: string;
}

function formatDateShort(iso: string): string {
  const d = new Date(iso + 'T00:00:00');
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export default function FilterChipBar({ filters, onRemoveFilter, onClearAll }: FilterChipBarProps) {
  const [exiting, setExiting] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const [overflowCount, setOverflowCount] = useState(0);

  const chips = useMemo<Chip[]>(() => {
    const result: Chip[] = [];

    if (filters.keyword) {
      result.push({
        id: 'keyword',
        filterKey: 'keyword',
        label: `"${filters.keyword}"`,
      });
    }

    for (const code of filters.naics) {
      result.push({
        id: `naics:${code}`,
        filterKey: 'naics',
        value: code,
        label: `NAICS ${code}`,
      });
    }

    for (const code of filters.psc) {
      result.push({
        id: `psc:${code}`,
        filterKey: 'psc',
        value: code,
        label: `PSC ${code}`,
      });
    }

    for (const code of filters.setAside) {
      result.push({
        id: `setAside:${code}`,
        filterKey: 'setAside',
        value: code,
        label: SET_ASIDE_LABELS[code] || code,
      });
    }

    const isDefaultTypes = filters.noticeType.length === DEFAULT_NOTICE_TYPES.length
      && filters.noticeType.slice().sort().every((v, i) => v === DEFAULT_NOTICE_TYPES.slice().sort()[i]);
    if (!isDefaultTypes) {
      for (const type of filters.noticeType) {
        result.push({
          id: `noticeType:${type}`,
          filterKey: 'noticeType',
          value: type,
          label: NOTICE_TYPE_LABELS[type] || type,
        });
      }
    }

    for (const path of filters.agency) {
      const name = path.split('.').pop() || path;
      result.push({
        id: `agency:${path}`,
        filterKey: 'agency',
        value: path,
        label: name,
      });
    }

    if (filters.state) {
      result.push({
        id: 'state',
        filterKey: 'state',
        label: STATE_NAMES[filters.state] || filters.state,
      });
    }

    if (filters.deadlinePreset) {
      result.push({
        id: 'deadlinePreset',
        filterKey: 'deadlinePreset',
        label: `Due: ${DEADLINE_PRESET_LABELS[filters.deadlinePreset] || filters.deadlinePreset}`,
      });
    }

    if (filters.postedFrom || filters.postedTo) {
      const from = filters.postedFrom ? formatDateShort(filters.postedFrom) : '';
      const to = filters.postedTo ? formatDateShort(filters.postedTo) : '';
      let label = 'Posted: ';
      if (from && to) label += `${from} \u2013 ${to}`;
      else if (from) label += `from ${from}`;
      else label += `until ${to}`;
      result.push({ id: 'posted', filterKey: 'postedFrom', label });
    }

    if (!filters.deadlinePreset && (filters.deadlineFrom || filters.deadlineTo)) {
      const from = filters.deadlineFrom ? formatDateShort(filters.deadlineFrom) : '';
      const to = filters.deadlineTo ? formatDateShort(filters.deadlineTo) : '';
      let label = 'Due: ';
      if (from && to) label += `${from} \u2013 ${to}`;
      else if (from) label += `from ${from}`;
      else label += `until ${to}`;
      result.push({ id: 'deadline', filterKey: 'deadlineFrom', label });
    }

    if (filters.exactMatch) {
      result.push({ id: 'exactMatch', filterKey: 'exactMatch', label: 'Exact match' });
    }

    if (filters.activeOnly === false) {
      result.push({ id: 'activeOnly', filterKey: 'activeOnly', label: 'Including inactive' });
    }

    return result;
  }, [filters]);

  // Detect overflow for 2-row cap
  useEffect(() => {
    if (!containerRef.current || expanded) {
      setOverflowCount(0);
      return;
    }
    const ro = new ResizeObserver(() => {
      const el = containerRef.current;
      if (!el) return;
      const children = Array.from(el.children) as HTMLElement[];
      const maxBottom = el.getBoundingClientRect().top + 72; // 4.5rem
      let hidden = 0;
      for (const child of children) {
        if (child.dataset.chip && child.getBoundingClientRect().top >= maxBottom) hidden++;
      }
      setOverflowCount(hidden);
    });
    ro.observe(containerRef.current);
    return () => ro.disconnect();
  }, [chips.length, expanded]);

  const dismiss = useCallback(
    (chip: Chip) => {
      setExiting((prev) => new Set(prev).add(chip.id));
      setTimeout(() => {
        onRemoveFilter(chip.filterKey, chip.value);
        setExiting((prev) => {
          const next = new Set(prev);
          next.delete(chip.id);
          return next;
        });
      }, 150);
    },
    [onRemoveFilter],
  );

  if (chips.length === 0) return null;

  return (
    <div className="sticky top-0 z-20 bg-dark-950/90 backdrop-blur-sm py-2">
      <div className="flex items-start gap-2">
        <div
          ref={containerRef}
          role="list"
          className={`flex flex-wrap gap-1.5 flex-1 ${expanded ? '' : 'max-h-[4.5rem] overflow-hidden'}`}
        >
          {chips.map((chip) => (
            <span
              key={chip.id}
              role="listitem"
              data-chip="true"
              className={`inline-flex items-center gap-1 px-2.5 py-2 md:py-1 rounded-full text-xs min-h-[44px] md:min-h-0
                bg-accent/10 border border-accent/20 text-accent
                ${exiting.has(chip.id) ? 'animate-chip-exit' : ''}`}
            >
              {chip.label}
              <button
                type="button"
                onClick={() => dismiss(chip)}
                aria-label={`Remove ${chip.label} filter`}
                className="hover:text-white transition-colors rounded focus-visible:ring-2 focus-visible:ring-accent/50 focus-visible:ring-offset-1 focus-visible:ring-offset-dark-950"
              >
                <X size={12} />
              </button>
            </span>
          ))}
        </div>
        <div className="flex items-center gap-2 shrink-0 pt-1">
          {overflowCount > 0 && !expanded && (
            <button
              type="button"
              onClick={() => setExpanded(true)}
              className="text-[11px] text-dark-400 hover:text-dark-200 whitespace-nowrap"
            >
              +{overflowCount} more
            </button>
          )}
          {expanded && (
            <button
              type="button"
              onClick={() => setExpanded(false)}
              className="text-[11px] text-dark-400 hover:text-dark-200 whitespace-nowrap"
            >
              Show less
            </button>
          )}
          {chips.length >= 2 && (
            <button
              type="button"
              onClick={onClearAll}
              className="text-[11px] text-dark-400 hover:text-dark-200 whitespace-nowrap"
            >
              Clear all
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
