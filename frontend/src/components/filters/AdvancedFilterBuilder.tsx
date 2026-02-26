import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { X, Plus, RotateCcw, ChevronDown } from 'lucide-react';
import SearchableDropdownFilter from './SearchableDropdownFilter';
import SimpleToggleFilter from './SimpleToggleFilter';
import NaicsTreeSelector from './NaicsTreeSelector';
import PscTreeSelector from './PscTreeSelector';
import AgencyFilter from './AgencyFilter';
import { NOTICE_TYPE_OPTIONS, STATE_NAMES } from './constants';
import { SET_ASIDE_FILTER_OPTIONS } from '../ResultsList';
import type { FilterState, UseFilterStateReturn } from '../../hooks/useFilterState';
import type { FacetResult, FacetValue } from '../../types/api';

interface FieldDef {
  key: string;
  label: string;
  filterKeys: (keyof FilterState)[];
  type: 'multi-value' | 'single-value' | 'date-range' | 'boolean';
  operators: { value: string; label: string }[];
  defaultOperator: string;
  supported: boolean;
}

interface BuilderRow {
  id: string;
  field: string;
  operator: string;
}

interface AdvancedFilterBuilderProps {
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
  setFilters: UseFilterStateReturn['setFilters'];
  clearFilter: UseFilterStateReturn['clearFilter'];
  clearAllFilters: UseFilterStateReturn['clearAllFilters'];
  facets: FacetResult['facets'] | null;
  facetsLoading: boolean;
  total: number;
}

const MAX_ROWS = 10;

const FIELD_DEFS: FieldDef[] = [
  {
    key: 'naics',
    label: 'NAICS Code',
    filterKeys: ['naics'],
    type: 'multi-value',
    operators: [{ value: 'is_any_of', label: 'is any of' }],
    defaultOperator: 'is_any_of',
    supported: true,
  },
  {
    key: 'psc',
    label: 'PSC Code',
    filterKeys: ['psc'],
    type: 'multi-value',
    operators: [{ value: 'is_any_of', label: 'is any of' }],
    defaultOperator: 'is_any_of',
    supported: true,
  },
  {
    key: 'setAside',
    label: 'Set-Aside Type',
    filterKeys: ['setAside'],
    type: 'multi-value',
    operators: [{ value: 'is_any_of', label: 'is any of' }],
    defaultOperator: 'is_any_of',
    supported: true,
  },
  {
    key: 'agency',
    label: 'Agency',
    filterKeys: ['agency'],
    type: 'multi-value',
    operators: [{ value: 'is_any_of', label: 'is any of' }],
    defaultOperator: 'is_any_of',
    supported: true,
  },
  {
    key: 'noticeType',
    label: 'Notice Type',
    filterKeys: ['noticeType'],
    type: 'multi-value',
    operators: [{ value: 'is_any_of', label: 'is any of' }],
    defaultOperator: 'is_any_of',
    supported: true,
  },
  {
    key: 'deadline',
    label: 'Response Deadline',
    filterKeys: ['deadlineFrom', 'deadlineTo', 'deadlinePreset'],
    type: 'date-range',
    operators: [
      { value: 'is_after', label: 'is after' },
      { value: 'is_before', label: 'is before' },
      { value: 'is_between', label: 'is between' },
      { value: 'in_next_x_days', label: 'in the next X days' },
    ],
    defaultOperator: 'in_next_x_days',
    supported: true,
  },
  {
    key: 'postedDate',
    label: 'Posted Date',
    filterKeys: ['postedFrom', 'postedTo'],
    type: 'date-range',
    operators: [
      { value: 'is_after', label: 'is after' },
      { value: 'is_before', label: 'is before' },
      { value: 'is_between', label: 'is between' },
    ],
    defaultOperator: 'is_after',
    supported: true,
  },
  {
    key: 'state',
    label: 'State',
    filterKeys: ['state'],
    type: 'single-value',
    operators: [{ value: 'is', label: 'is' }],
    defaultOperator: 'is',
    supported: true,
  },
  {
    key: 'activeOnly',
    label: 'Active Status',
    filterKeys: ['activeOnly'],
    type: 'boolean',
    operators: [
      { value: 'is_true', label: 'is true' },
      { value: 'is_false', label: 'is false' },
    ],
    defaultOperator: 'is_true',
    supported: true,
  },
  {
    key: 'awardValue',
    label: 'Award Value',
    filterKeys: [],
    type: 'multi-value',
    operators: [
      { value: 'greater_than', label: 'greater than' },
      { value: 'less_than', label: 'less than' },
      { value: 'between', label: 'between' },
    ],
    defaultOperator: 'greater_than',
    supported: false,
  },
  {
    key: 'solicitationNumber',
    label: 'Solicitation Number',
    filterKeys: [],
    type: 'single-value',
    operators: [
      { value: 'contains', label: 'contains' },
      { value: 'is_exactly', label: 'is exactly' },
    ],
    defaultOperator: 'contains',
    supported: false,
  },
];

const FIELD_DEF_MAP = new Map(FIELD_DEFS.map((d) => [d.key, d]));

let rowIdCounter = 0;
function genRowId(): string {
  return `brow-${++rowIdCounter}`;
}

const STATE_OPTIONS = Object.entries(STATE_NAMES).map(([code, name]) => ({
  value: code,
  label: `${code} — ${name}`,
  searchTerms: [name],
}));

function inferDeadlineOperator(filters: FilterState): string {
  if (filters.deadlinePreset) return 'in_next_x_days';
  if (filters.deadlineFrom && filters.deadlineTo) return 'is_between';
  if (filters.deadlineFrom) return 'is_after';
  if (filters.deadlineTo) return 'is_before';
  return 'in_next_x_days';
}

function inferPostedDateOperator(filters: FilterState): string {
  if (filters.postedFrom && filters.postedTo) return 'is_between';
  if (filters.postedFrom) return 'is_after';
  if (filters.postedTo) return 'is_before';
  return 'is_after';
}

function isFieldActive(fieldKey: string, filters: FilterState): boolean {
  switch (fieldKey) {
    case 'naics':
      return filters.naics.length > 0;
    case 'psc':
      return filters.psc.length > 0;
    case 'setAside':
      return filters.setAside.length > 0;
    case 'agency':
      return filters.agency.length > 0;
    case 'noticeType':
      return filters.noticeType.length > 0;
    case 'deadline':
      return filters.deadlinePreset !== '' || filters.deadlineFrom !== '' || filters.deadlineTo !== '';
    case 'postedDate':
      return filters.postedFrom !== '' || filters.postedTo !== '';
    case 'state':
      return filters.state !== '';
    case 'activeOnly':
      return !filters.activeOnly;
    default:
      return false;
  }
}

function deriveRowsFromFilters(filters: FilterState): BuilderRow[] {
  const rows: BuilderRow[] = [];

  for (const def of FIELD_DEFS) {
    if (!def.supported) continue;
    if (!isFieldActive(def.key, filters)) continue;

    let operator = def.defaultOperator;
    if (def.key === 'deadline') operator = inferDeadlineOperator(filters);
    else if (def.key === 'postedDate') operator = inferPostedDateOperator(filters);
    else if (def.key === 'activeOnly') operator = filters.activeOnly ? 'is_true' : 'is_false';

    rows.push({ id: genRowId(), field: def.key, operator });
  }

  if (rows.length === 0) {
    rows.push({ id: genRowId(), field: '', operator: '' });
  }

  return rows;
}

export default function AdvancedFilterBuilder({
  filters,
  setFilter,
  setFilters,
  clearFilter,
  clearAllFilters,
  facets,
  facetsLoading,
}: AdvancedFilterBuilderProps) {
  const [rows, setRows] = useState<BuilderRow[]>(() => deriveRowsFromFilters(filters));
  const [enteringIds, setEnteringIds] = useState<Set<string>>(new Set());
  const [exitingIds, setExitingIds] = useState<Set<string>>(new Set());
  const newRowFieldRef = useRef<string | null>(null);
  const fieldSelectRefs = useRef<Map<string, HTMLSelectElement>>(new Map());

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

  const usedFields = useMemo(() => new Set(rows.map((r) => r.field).filter(Boolean)), [rows]);

  const clearFieldValues = useCallback(
    (fieldKey: string) => {
      if (fieldKey === 'deadline') {
        setFilters({ deadlinePreset: '', deadlineFrom: '', deadlineTo: '' });
      } else if (fieldKey === 'postedDate') {
        setFilters({ postedFrom: '', postedTo: '' });
      } else if (fieldKey === 'activeOnly') {
        setFilter('activeOnly', true);
      } else {
        const def = FIELD_DEF_MAP.get(fieldKey);
        if (def?.supported) {
          for (const fk of def.filterKeys) {
            clearFilter(fk);
          }
        }
      }
    },
    [setFilters, setFilter, clearFilter],
  );

  const handleAddRow = useCallback(() => {
    if (rows.length >= MAX_ROWS) return;
    const newId = genRowId();
    setRows((prev) => [...prev, { id: newId, field: '', operator: '' }]);
    setEnteringIds((prev) => new Set(prev).add(newId));
    newRowFieldRef.current = newId;
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        setEnteringIds((prev) => {
          const next = new Set(prev);
          next.delete(newId);
          return next;
        });
      });
    });
  }, [rows.length]);

  // Focus the field selector of a newly added row
  useEffect(() => {
    if (newRowFieldRef.current) {
      const id = newRowFieldRef.current;
      newRowFieldRef.current = null;
      setTimeout(() => {
        fieldSelectRefs.current.get(id)?.focus();
      }, 50);
    }
  });

  const handleRemoveRow = useCallback(
    (rowId: string) => {
      const row = rows.find((r) => r.id === rowId);
      if (!row) return;

      if (row.field) {
        clearFieldValues(row.field);
      }

      setExitingIds((prev) => new Set(prev).add(rowId));
      setTimeout(() => {
        setRows((prev) => {
          const next = prev.filter((r) => r.id !== rowId);
          return next.length === 0 ? [{ id: genRowId(), field: '', operator: '' }] : next;
        });
        setExitingIds((prev) => {
          const next = new Set(prev);
          next.delete(rowId);
          return next;
        });
      }, 200);
    },
    [rows, clearFieldValues],
  );

  const handleFieldChange = useCallback(
    (rowId: string, newField: string) => {
      const row = rows.find((r) => r.id === rowId);
      if (!row) return;

      if (row.field) {
        clearFieldValues(row.field);
      }

      const def = FIELD_DEF_MAP.get(newField);
      setRows((prev) =>
        prev.map((r) =>
          r.id === rowId ? { ...r, field: newField, operator: def?.defaultOperator || '' } : r,
        ),
      );
    },
    [rows, clearFieldValues],
  );

  const handleOperatorChange = useCallback(
    (rowId: string, newOperator: string) => {
      setRows((prev) =>
        prev.map((r) => {
          if (r.id !== rowId) return r;

          if (r.field === 'deadline') {
            if (newOperator === 'in_next_x_days') {
              setFilters({ deadlineFrom: '', deadlineTo: '', deadlinePreset: '' });
            } else {
              setFilters({ deadlinePreset: '', deadlineFrom: '', deadlineTo: '' });
            }
          } else if (r.field === 'postedDate') {
            setFilters({ postedFrom: '', postedTo: '' });
          }

          return { ...r, operator: newOperator };
        }),
      );
    },
    [setFilters],
  );

  const handleReset = useCallback(() => {
    clearAllFilters();
    setRows([{ id: genRowId(), field: '', operator: '' }]);
  }, [clearAllFilters]);

  const setFieldSelectRef = useCallback((rowId: string, el: HTMLSelectElement | null) => {
    if (el) {
      fieldSelectRefs.current.set(rowId, el);
    } else {
      fieldSelectRefs.current.delete(rowId);
    }
  }, []);

  return (
    <div
      role="group"
      aria-label="Advanced filter conditions"
      className="bg-dark-850 border border-dark-700/30 rounded-xl p-4 space-y-2"
    >
      {rows.map((row, index) => {
        const entering = enteringIds.has(row.id);
        const exiting = exitingIds.has(row.id);

        return (
          <div
            key={row.id}
            className="transition-all duration-200 overflow-hidden"
            style={{
              maxHeight: entering || exiting ? 0 : 200,
              opacity: entering || exiting ? 0 : 1,
            }}
          >
            <fieldset className="flex flex-col md:flex-row items-start md:items-center gap-2 py-1">
              <legend className="sr-only">Filter condition {index + 1}</legend>

              {/* Conjunction label */}
              <span className="text-xs text-dark-500 w-12 shrink-0 md:text-right">
                {index === 0 ? 'Where' : 'And'}
              </span>

              {/* Field selector */}
              <div className="relative inline-flex">
                <select
                  ref={(el) => setFieldSelectRef(row.id, el)}
                  value={row.field}
                  onChange={(e) => handleFieldChange(row.id, e.target.value)}
                  aria-label="Select filter field"
                  className="text-xs bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 pr-7 text-dark-200
                    focus:outline-none focus:border-accent/50 min-w-[160px] appearance-none cursor-pointer"
                >
                  <option value="">Select field...</option>
                  {FIELD_DEFS.map((def) => {
                    const inUse = usedFields.has(def.key) && row.field !== def.key;
                    return (
                      <option key={def.key} value={def.key} disabled={inUse}>
                        {def.label}
                        {!def.supported ? ' (coming soon)' : ''}
                      </option>
                    );
                  })}
                </select>
                <ChevronDown
                  size={12}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-dark-400 pointer-events-none"
                />
              </div>

              {/* Operator selector */}
              {row.field && (
                <div className="relative inline-flex">
                  <select
                    value={row.operator}
                    onChange={(e) => handleOperatorChange(row.id, e.target.value)}
                    aria-label={`Select comparison operator for ${FIELD_DEF_MAP.get(row.field)?.label || row.field}`}
                    className="text-xs bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 pr-7 text-dark-200
                      focus:outline-none focus:border-accent/50 appearance-none cursor-pointer"
                  >
                    {FIELD_DEF_MAP.get(row.field)?.operators.map((op) => (
                      <option key={op.value} value={op.value}>
                        {op.label}
                      </option>
                    ))}
                  </select>
                  <ChevronDown
                    size={12}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-dark-400 pointer-events-none"
                  />
                </div>
              )}

              {/* Value input */}
              {row.field && (
                <div className="flex-1 min-w-0">
                  <RowValueInput
                    row={row}
                    filters={filters}
                    setFilter={setFilter}
                    setFilters={setFilters}
                    naicsFacets={facets?.naics}
                    pscFacets={facets?.psc}
                    setAsideOptions={setAsideOptions}
                    noticeTypeOptions={noticeTypeOptions}
                    facetsLoading={facetsLoading}
                  />
                </div>
              )}

              {/* Remove button */}
              <button
                type="button"
                onClick={() => handleRemoveRow(row.id)}
                aria-label="Remove filter condition"
                className="p-1.5 rounded-lg text-dark-500 hover:text-dark-200 hover:bg-dark-700/50
                  transition-colors shrink-0"
              >
                <X size={14} />
              </button>
            </fieldset>
          </div>
        );
      })}

      {/* Footer */}
      <div className="flex items-center justify-between pt-2">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleAddRow}
            disabled={rows.length >= MAX_ROWS}
            className="inline-flex items-center gap-1.5 text-xs text-accent hover:text-accent-hover
              disabled:text-dark-600 disabled:cursor-not-allowed transition-colors"
          >
            <Plus size={12} />
            Add filter
          </button>
          <button
            type="button"
            onClick={handleReset}
            className="inline-flex items-center gap-1.5 text-xs text-dark-400 hover:text-dark-200 transition-colors"
          >
            <RotateCcw size={12} />
            Reset
          </button>
        </div>
        <span className="text-xs text-dark-500">({rows.length}/{MAX_ROWS} filters)</span>
      </div>
    </div>
  );
}

// Sub-components for value inputs by field type

interface RowValueInputProps {
  row: BuilderRow;
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
  setFilters: UseFilterStateReturn['setFilters'];
  naicsFacets?: FacetValue[];
  pscFacets?: FacetValue[];
  setAsideOptions: Array<{ value: string; label: string; count?: number }>;
  noticeTypeOptions: Array<{ value: string; label: string; count?: number }>;
  facetsLoading: boolean;
}

function RowValueInput({
  row,
  filters,
  setFilter,
  setFilters,
  naicsFacets,
  pscFacets,
  setAsideOptions,
  noticeTypeOptions,
  facetsLoading,
}: RowValueInputProps) {
  const def = FIELD_DEF_MAP.get(row.field);
  if (!def) return null;

  if (!def.supported) {
    return <span className="text-xs text-dark-600 italic px-3 py-2">Coming soon</span>;
  }

  switch (row.field) {
    case 'naics':
      return (
        <NaicsTreeSelector
          selected={filters.naics}
          onChange={(sel) => setFilter('naics', sel)}
          facets={naicsFacets}
          label="Select NAICS"
          loading={facetsLoading}
        />
      );

    case 'psc':
      return (
        <PscTreeSelector
          selected={filters.psc}
          onChange={(sel) => setFilter('psc', sel)}
          facets={pscFacets}
          label="Select PSC"
          loading={facetsLoading}
        />
      );

    case 'setAside':
      return (
        <SearchableDropdownFilter
          label="Select set-asides"
          options={setAsideOptions}
          selected={filters.setAside}
          onSelectionChange={(sel) => setFilter('setAside', sel)}
          searchPlaceholder="Search set-asides..."
          loading={facetsLoading}
        />
      );

    case 'agency':
      return (
        <AgencyFilter
          selected={filters.agency}
          onChange={(sel) => setFilter('agency', sel)}
        />
      );

    case 'noticeType':
      return (
        <SimpleToggleFilter
          label="Select types"
          options={noticeTypeOptions}
          selected={filters.noticeType}
          onSelectionChange={(sel) => setFilter('noticeType', sel)}
          loading={facetsLoading}
        />
      );

    case 'state':
      return (
        <SearchableDropdownFilter
          label="Select state"
          options={STATE_OPTIONS}
          selected={filters.state ? [filters.state] : []}
          onSelectionChange={(sel) => setFilter('state', sel[0] || '')}
          searchPlaceholder="Search states..."
        />
      );

    case 'deadline':
      return (
        <DeadlineValueInput row={row} filters={filters} setFilter={setFilter} setFilters={setFilters} />
      );

    case 'postedDate':
      return <PostedDateValueInput row={row} filters={filters} setFilters={setFilters} />;

    case 'activeOnly':
      return <ActiveStatusInput filters={filters} setFilter={setFilter} />;

    default:
      return null;
  }
}

const dateInputClass =
  'text-xs bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 text-dark-200 focus:outline-none focus:border-accent/50';

function DeadlineValueInput({
  row,
  filters,
  setFilter,
  setFilters,
}: {
  row: BuilderRow;
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
  setFilters: UseFilterStateReturn['setFilters'];
}) {
  switch (row.operator) {
    case 'in_next_x_days':
      return (
        <div className="flex items-center gap-2">
          <input
            type="number"
            min={1}
            max={365}
            value={filters.deadlinePreset}
            onChange={(e) => setFilter('deadlinePreset', e.target.value)}
            placeholder="30"
            className={`w-20 ${dateInputClass}`}
          />
          <span className="text-xs text-dark-400">days</span>
        </div>
      );

    case 'is_after':
      return (
        <input
          type="date"
          value={filters.deadlineFrom}
          onChange={(e) =>
            setFilters({ deadlinePreset: '', deadlineFrom: e.target.value, deadlineTo: '' })
          }
          className={dateInputClass}
        />
      );

    case 'is_before':
      return (
        <input
          type="date"
          value={filters.deadlineTo}
          onChange={(e) =>
            setFilters({ deadlinePreset: '', deadlineFrom: '', deadlineTo: e.target.value })
          }
          className={dateInputClass}
        />
      );

    case 'is_between':
      return (
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={filters.deadlineFrom}
            onChange={(e) => setFilters({ deadlinePreset: '', deadlineFrom: e.target.value })}
            className={dateInputClass}
          />
          <span className="text-xs text-dark-400">and</span>
          <input
            type="date"
            value={filters.deadlineTo}
            min={filters.deadlineFrom || undefined}
            onChange={(e) => setFilters({ deadlinePreset: '', deadlineTo: e.target.value })}
            className={dateInputClass}
          />
        </div>
      );

    default:
      return null;
  }
}

function PostedDateValueInput({
  row,
  filters,
  setFilters,
}: {
  row: BuilderRow;
  filters: FilterState;
  setFilters: UseFilterStateReturn['setFilters'];
}) {
  switch (row.operator) {
    case 'is_after':
      return (
        <input
          type="date"
          value={filters.postedFrom}
          onChange={(e) => setFilters({ postedFrom: e.target.value })}
          className={dateInputClass}
        />
      );

    case 'is_before':
      return (
        <input
          type="date"
          value={filters.postedTo}
          onChange={(e) => setFilters({ postedTo: e.target.value })}
          className={dateInputClass}
        />
      );

    case 'is_between':
      return (
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={filters.postedFrom}
            onChange={(e) => setFilters({ postedFrom: e.target.value })}
            className={dateInputClass}
          />
          <span className="text-xs text-dark-400">and</span>
          <input
            type="date"
            value={filters.postedTo}
            min={filters.postedFrom || undefined}
            onChange={(e) => setFilters({ postedTo: e.target.value })}
            className={dateInputClass}
          />
        </div>
      );

    default:
      return null;
  }
}

function ActiveStatusInput({
  filters,
  setFilter,
}: {
  filters: FilterState;
  setFilter: UseFilterStateReturn['setFilter'];
}) {
  return (
    <div className="flex items-center gap-3">
      <label className="flex items-center gap-1.5 cursor-pointer text-xs">
        <input
          type="radio"
          name="activeStatus"
          checked={filters.activeOnly === true}
          onChange={() => setFilter('activeOnly', true)}
          className="accent-accent"
        />
        <span className={filters.activeOnly ? 'text-dark-200' : 'text-dark-400'}>Active only</span>
      </label>
      <label className="flex items-center gap-1.5 cursor-pointer text-xs">
        <input
          type="radio"
          name="activeStatus"
          checked={filters.activeOnly === false}
          onChange={() => setFilter('activeOnly', false)}
          className="accent-accent"
        />
        <span className={!filters.activeOnly ? 'text-dark-200' : 'text-dark-400'}>
          All (including inactive)
        </span>
      </label>
    </div>
  );
}
