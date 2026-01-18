import { useState } from 'react';
import type { FilterOptions, SearchParams } from '../types/opportunity';

interface FilterPanelProps {
  filters: FilterOptions | undefined;
  params: SearchParams;
  onChange: (params: Partial<SearchParams>) => void;
  isLoading?: boolean;
}

export function FilterPanel({ filters, params, onChange, isLoading }: FilterPanelProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  const toggleFilter = (key: keyof Pick<SearchParams, 'type' | 'set_aside' | 'state'>, value: string) => {
    const current = params[key] || [];
    const updated = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value];
    onChange({ [key]: updated.length > 0 ? updated : undefined });
  };

  const clearFilters = () => {
    onChange({
      type: undefined,
      set_aside: undefined,
      state: undefined,
      posted_from: undefined,
      posted_to: undefined,
      deadline_from: undefined,
      deadline_to: undefined,
    });
  };

  const hasActiveFilters =
    (params.type?.length ?? 0) > 0 ||
    (params.set_aside?.length ?? 0) > 0 ||
    (params.state?.length ?? 0) > 0 ||
    params.posted_from ||
    params.posted_to ||
    params.deadline_from ||
    params.deadline_to;

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-2 font-medium text-gray-900"
        >
          <span className="text-sm">{isExpanded ? '▼' : '▶'}</span>
          Filters
        </button>
        {hasActiveFilters && (
          <button
            onClick={clearFilters}
            className="text-sm text-blue-600 hover:text-blue-800"
          >
            Clear all
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="space-y-6">
          {isLoading ? (
            <div className="text-gray-500 text-sm">Loading filters...</div>
          ) : (
            <>
              <FilterSection
                title="Opportunity Type"
                options={filters?.types || []}
                selected={params.type || []}
                onToggle={(value) => toggleFilter('type', value)}
              />

              <FilterSection
                title="Set-Aside"
                options={filters?.set_asides || []}
                selected={params.set_aside || []}
                onToggle={(value) => toggleFilter('set_aside', value)}
              />

              <FilterSection
                title="State"
                options={filters?.states?.slice(0, 20) || []}
                selected={params.state || []}
                onToggle={(value) => toggleFilter('state', value)}
              />

              <div className="space-y-2">
                <h4 className="text-sm font-medium text-gray-700">Posted Date</h4>
                <div className="flex gap-2">
                  <input
                    type="date"
                    value={params.posted_from || ''}
                    onChange={(e) => onChange({ posted_from: e.target.value || undefined })}
                    className="flex-1 px-2 py-1 text-sm border border-gray-300 rounded"
                  />
                  <span className="text-gray-500">to</span>
                  <input
                    type="date"
                    value={params.posted_to || ''}
                    onChange={(e) => onChange({ posted_to: e.target.value || undefined })}
                    className="flex-1 px-2 py-1 text-sm border border-gray-300 rounded"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <h4 className="text-sm font-medium text-gray-700">Response Deadline</h4>
                <div className="flex gap-2">
                  <input
                    type="date"
                    value={params.deadline_from || ''}
                    onChange={(e) => onChange({ deadline_from: e.target.value || undefined })}
                    className="flex-1 px-2 py-1 text-sm border border-gray-300 rounded"
                  />
                  <span className="text-gray-500">to</span>
                  <input
                    type="date"
                    value={params.deadline_to || ''}
                    onChange={(e) => onChange({ deadline_to: e.target.value || undefined })}
                    className="flex-1 px-2 py-1 text-sm border border-gray-300 rounded"
                  />
                </div>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

interface FilterSectionProps {
  title: string;
  options: { code: string; label?: string; count: number }[];
  selected: string[];
  onToggle: (value: string) => void;
}

function FilterSection({ title, options, selected, onToggle }: FilterSectionProps) {
  const [showAll, setShowAll] = useState(false);
  const displayOptions = showAll ? options : options.slice(0, 5);

  if (options.length === 0) return null;

  return (
    <div className="space-y-2">
      <h4 className="text-sm font-medium text-gray-700">{title}</h4>
      <div className="space-y-1">
        {displayOptions.map((option) => (
          <label
            key={option.code}
            className="flex items-center gap-2 text-sm cursor-pointer hover:bg-gray-50 p-1 rounded"
          >
            <input
              type="checkbox"
              checked={selected.includes(option.code)}
              onChange={() => onToggle(option.code)}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="flex-1 truncate" title={option.label || option.code}>
              {option.label || option.code}
            </span>
            <span className="text-gray-400 text-xs">({option.count})</span>
          </label>
        ))}
      </div>
      {options.length > 5 && (
        <button
          onClick={() => setShowAll(!showAll)}
          className="text-sm text-blue-600 hover:text-blue-800"
        >
          {showAll ? 'Show less' : `Show ${options.length - 5} more`}
        </button>
      )}
    </div>
  );
}
