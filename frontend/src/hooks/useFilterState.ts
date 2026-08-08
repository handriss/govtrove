import { useReducer, useCallback, useEffect, useRef, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { expandToLeafCodes, naicsDataReady, NODE_BY_CODE as NAICS_NODES } from '../components/filters/naicsTree';
import { expandPscToLeafCodes, pscDataReady } from '../components/filters/pscTree';
import { DEFAULT_NOTICE_TYPES } from '../components/filters/constants';
import type { SearchParams } from '../types/api';

const treeDataReady = Promise.all([naicsDataReady, pscDataReady]);

export interface FilterState {
  keyword: string;
  naics: string[];
  psc: string[];
  setAside: string[];
  agency: string[];
  state: string;
  noticeType: string[];
  deadlinePreset: string;
  postedFrom: string;
  postedTo: string;
  deadlineFrom: string;
  deadlineTo: string;
  solicitationNumber: string;
  popCity: string;
  activeOnly: boolean;
  sort: string;
  sortDir: 'asc' | 'desc';
  page: number;
}

const DEFAULTS: FilterState = {
  keyword: '',
  naics: [],
  psc: [],
  setAside: [],
  agency: [],
  state: '',
  noticeType: [...DEFAULT_NOTICE_TYPES],
  deadlinePreset: '',
  postedFrom: '',
  postedTo: '',
  deadlineFrom: '',
  deadlineTo: '',
  solicitationNumber: '',
  popCity: '',
  activeOnly: true,
  sort: 'posted_date',
  sortDir: 'desc',
  page: 1,
};

const SORT_STORAGE_KEY = 'govtrove_sort';

type Action =
  | { type: 'SET'; key: keyof FilterState; value: FilterState[keyof FilterState] }
  | { type: 'SET_MANY'; partial: Partial<FilterState> }
  | { type: 'ADD'; key: 'naics' | 'psc' | 'setAside' | 'noticeType' | 'agency'; value: string }
  | { type: 'REMOVE'; key: 'naics' | 'psc' | 'setAside' | 'noticeType' | 'agency'; value: string }
  | { type: 'CLEAR'; key: keyof FilterState }
  | { type: 'CLEAR_ALL' }
  | { type: 'INIT'; state: FilterState };

function reducer(state: FilterState, action: Action): FilterState {
  switch (action.type) {
    case 'INIT':
      return action.state;

    case 'SET': {
      const resetPage = action.key !== 'page' && action.key !== 'sort' && action.key !== 'sortDir';
      return { ...state, [action.key]: action.value, ...(resetPage ? { page: 1 } : {}) };
    }

    case 'SET_MANY': {
      const hasNonPaginationChange = Object.keys(action.partial).some(
        (k) => k !== 'page' && k !== 'sort' && k !== 'sortDir',
      );
      return {
        ...state,
        ...action.partial,
        ...(hasNonPaginationChange && !('page' in action.partial) ? { page: 1 } : {}),
      };
    }

    case 'ADD': {
      const arr = state[action.key];
      if (arr.includes(action.value)) return state;
      return { ...state, [action.key]: [...arr, action.value], page: 1 };
    }

    case 'REMOVE': {
      const arr = state[action.key];
      const filtered = arr.filter((v) => v !== action.value);
      if (filtered.length === arr.length) return state;
      return { ...state, [action.key]: filtered, page: 1 };
    }

    case 'CLEAR': {
      const def = DEFAULTS[action.key];
      if (JSON.stringify(state[action.key]) === JSON.stringify(def)) return state;
      return { ...state, [action.key]: def, page: 1 };
    }

    case 'CLEAR_ALL':
      return { ...DEFAULTS };
  }
}

// URL param <-> FilterState field mapping
const URL_MAP: [keyof FilterState, string][] = [
  ['keyword', 'q'],
  ['naics', 'naics'],
  ['psc', 'psc'],
  ['setAside', 'set_aside'],
  ['agency', 'agency'],
  ['state', 'state'],
  ['noticeType', 'type'],
  ['deadlinePreset', 'deadline'],
  ['postedFrom', 'posted_from'],
  ['postedTo', 'posted_to'],
  ['deadlineFrom', 'deadline_from'],
  ['deadlineTo', 'deadline_to'],
  ['solicitationNumber', 'sol_num'],
  ['popCity', 'pop_city'],
  ['activeOnly', 'active'],
  ['sort', 'sort'],
  ['sortDir', 'order'],
  ['page', 'page'],
];

const ARRAY_FIELDS = new Set<keyof FilterState>(['naics', 'psc', 'setAside', 'noticeType', 'agency']);

export function parseStateFromURL(urlParams: URLSearchParams): FilterState {
  const state = { ...DEFAULTS };

  // Sort from localStorage if URL doesn't specify
  const savedSort = localStorage.getItem(SORT_STORAGE_KEY);
  if (savedSort) {
    try {
      const parsed = JSON.parse(savedSort);
      if (parsed.sort) state.sort = parsed.sort;
      if (parsed.order) state.sortDir = parsed.order;
    } catch {
      // ignore corrupted localStorage
    }
  }

  for (const [field, param] of URL_MAP) {
    const raw = urlParams.get(param);
    if (raw === null) continue;

    if (ARRAY_FIELDS.has(field)) {
      (state as Record<string, unknown>)[field] = raw.split(',').filter(Boolean);
    } else if (field === 'activeOnly') {
      state.activeOnly = raw !== 'false';
    } else if (field === 'page') {
      const n = parseInt(raw, 10);
      if (n > 0) state.page = n;
    } else if (field === 'sortDir') {
      state.sortDir = raw === 'asc' ? 'asc' : 'desc';
    } else {
      (state as Record<string, unknown>)[field] = raw;
    }
  }

  return state;
}

const SECONDARY_FIELDS = new Set<keyof FilterState>(['sort', 'sortDir', 'page']);

function stateToURL(state: FilterState): URLSearchParams {
  const params = new URLSearchParams();

  // First pass: collect non-secondary params
  for (const [field, param] of URL_MAP) {
    if (SECONDARY_FIELDS.has(field)) continue;
    const value = state[field];
    const def = DEFAULTS[field];

    if (Array.isArray(value)) {
      const defArr = def as string[];
      const same = value.length === defArr.length && value.slice().sort().join(',') === defArr.slice().sort().join(',');
      if (value.length > 0 && !same) params.set(param, value.join(','));
    } else if (typeof value === 'boolean') {
      if (value !== def) params.set(param, String(value));
    } else if (typeof value === 'number') {
      if (value !== def) params.set(param, String(value));
    } else if (value && value !== def) {
      params.set(param, value);
    }
  }

  // Only include sort/order/page when there are actual search params
  if (params.size > 0) {
    for (const [field, param] of URL_MAP) {
      if (!SECONDARY_FIELDS.has(field)) continue;
      const value = state[field];
      const def = DEFAULTS[field];

      if (typeof value === 'number') {
        if (value !== def) params.set(param, String(value));
      } else if (value && value !== def) {
        params.set(param, value as string);
      }
    }
  }

  return params;
}

function today(): string {
  return new Date().toISOString().split('T')[0];
}

function deadlinePresetToDate(preset: string): string | undefined {
  if (preset === 'quarter') {
    const now = new Date();
    const endMonth = (Math.floor(now.getMonth() / 3) + 1) * 3;
    return new Date(now.getFullYear(), endMonth, 0).toISOString().split('T')[0];
  }
  const days = parseInt(preset, 10);
  if (!days) return undefined;
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().split('T')[0];
}

export interface UseFilterStateReturn {
  filters: FilterState;
  setFilter: <K extends keyof FilterState>(key: K, value: FilterState[K]) => void;
  addFilter: (key: 'naics' | 'psc' | 'setAside' | 'noticeType' | 'agency', value: string) => void;
  removeFilter: (key: 'naics' | 'psc' | 'setAside' | 'noticeType' | 'agency', value: string) => void;
  clearFilter: (key: keyof FilterState) => void;
  clearAllFilters: () => void;
  setFilters: (partial: Partial<FilterState>) => void;
  filterCount: number;
  dataReady: boolean;
  toSearchParams: () => SearchParams;
  toFacetParams: () => SearchParams;
}

export function useFilterState(): UseFilterStateReturn {
  const [searchParams, setSearchParams] = useSearchParams();
  const [filters, dispatch] = useReducer(reducer, searchParams, parseStateFromURL);
  const skipURLSync = useRef(false);
  const initialized = useRef(false);
  const [dataReady, setDataReady] = useState(false);

  useEffect(() => {
    treeDataReady.then(() => setDataReady(true));
  }, []);

  // On mount, mark initialized after first render
  useEffect(() => {
    initialized.current = true;
  }, []);

  // Sync state -> URL (skip when URL triggered the change)
  useEffect(() => {
    if (skipURLSync.current) {
      skipURLSync.current = false;
      return;
    }
    if (!initialized.current) return;

    const newParams = stateToURL(filters);
    const currentStr = searchParams.toString();
    const newStr = newParams.toString();
    if (currentStr === newStr) return;

    // Push a new history entry when going from empty to having params (first search),
    // so the landing page stays in browser history. Replace for subsequent changes.
    const replace = currentStr !== '' || newStr === '';
    setSearchParams(newParams, { replace });
  }, [filters, searchParams, setSearchParams]);

  // Persist sort preference
  useEffect(() => {
    localStorage.setItem(SORT_STORAGE_KEY, JSON.stringify({ sort: filters.sort, order: filters.sortDir }));
  }, [filters.sort, filters.sortDir]);

  // Sync URL -> state (browser back/forward)
  useEffect(() => {
    const fromURL = parseStateFromURL(searchParams);
    const currentURL = stateToURL(filters).toString();
    const incomingURL = stateToURL(fromURL).toString();
    if (currentURL !== incomingURL) {
      skipURLSync.current = true;
      dispatch({ type: 'INIT', state: fromURL });
    }
  }, [searchParams]); // eslint-disable-line react-hooks/exhaustive-deps

  const setFilter = useCallback(<K extends keyof FilterState>(key: K, value: FilterState[K]) => {
    dispatch({ type: 'SET', key, value: value as FilterState[keyof FilterState] });
  }, []);

  const addFilter = useCallback((key: 'naics' | 'psc' | 'setAside' | 'noticeType' | 'agency', value: string) => {
    dispatch({ type: 'ADD', key, value });
  }, []);

  const removeFilter = useCallback((key: 'naics' | 'psc' | 'setAside' | 'noticeType' | 'agency', value: string) => {
    dispatch({ type: 'REMOVE', key, value });
  }, []);

  const clearFilter = useCallback((key: keyof FilterState) => {
    dispatch({ type: 'CLEAR', key });
  }, []);

  const clearAllFilters = useCallback(() => {
    dispatch({ type: 'CLEAR_ALL' });
  }, []);

  const setFilters = useCallback((partial: Partial<FilterState>) => {
    dispatch({ type: 'SET_MANY', partial });
  }, []);

  const filterCount = useMemo(() => {
    let count = 0;
    if (filters.keyword) count++;
    if (filters.naics.length) count++;
    if (filters.psc.length) count++;
    if (filters.setAside.length) count++;
    if (filters.agency.length) count++;
    if (filters.state) count++;
    if (JSON.stringify(filters.noticeType.slice().sort()) !== JSON.stringify(DEFAULTS.noticeType.slice().sort())) count++;
    if (filters.deadlinePreset) count++;
    if (filters.solicitationNumber) count++;
    if (filters.popCity) count++;
    if (filters.postedFrom || filters.postedTo) count++;
    if (filters.deadlineFrom || filters.deadlineTo) count++;
    return count;
  }, [filters]);

  const toSearchParams = useCallback((): SearchParams => {
    const p: SearchParams = {};
    if (filters.keyword) p.q = filters.keyword;
    if (filters.noticeType.length) p.type = filters.noticeType.join(',');
    if (filters.setAside.length) p.set_aside = filters.setAside.join(',');
    if (filters.naics.length) {
      const prefixes: string[] = [];
      const leafCodes: string[] = [];
      for (const code of filters.naics) {
        const node = NAICS_NODES.get(code);
        if (node && node.leafCodes.length > 50) {
          // Sector-level: send as prefix query (e.g. "31-33" → prefixes "31","32","33")
          if (code.includes('-')) {
            const [start, end] = code.split('-').map(Number);
            for (let i = start; i <= end; i++) prefixes.push(String(i));
          } else {
            prefixes.push(code);
          }
        } else {
          leafCodes.push(...expandToLeafCodes([code]));
        }
      }
      if (leafCodes.length) p.naics = leafCodes.join(',');
      if (prefixes.length) p.naics_prefixes = prefixes.join(',');
    }
    if (filters.psc.length) p.psc = expandPscToLeafCodes(filters.psc).join(',');
    if (filters.state) p.state = filters.state;
    if (filters.agency.length) p.agency = filters.agency.join(',');
    if (filters.postedFrom) p.posted_from = filters.postedFrom;
    if (filters.postedTo) p.posted_to = filters.postedTo;

    if (filters.deadlinePreset) {
      p.deadline_from = today();
      const to = deadlinePresetToDate(filters.deadlinePreset);
      if (to) p.deadline_to = to;
    } else if (filters.deadlineFrom || filters.deadlineTo) {
      if (filters.deadlineFrom) p.deadline_from = filters.deadlineFrom;
      if (filters.deadlineTo) p.deadline_to = filters.deadlineTo;
    } else if (filters.activeOnly) {
      p.deadline_from = today();
    }

    if (filters.solicitationNumber) p.sol_num = filters.solicitationNumber;
    if (filters.popCity) p.pop_city = filters.popCity;
    if (filters.sort) p.sort = filters.sort;
    if (filters.sortDir) p.order = filters.sortDir;
    if (filters.page > 1) p.page = filters.page;
    return p;
  }, [filters]);

  const toFacetParams = useCallback((): SearchParams => {
    const p = toSearchParams();
    delete p.sort;
    delete p.order;
    delete p.page;
    delete p.limit;
    return p;
  }, [toSearchParams]);

  return {
    filters,
    setFilter,
    addFilter,
    removeFilter,
    clearFilter,
    clearAllFilters,
    setFilters,
    filterCount,
    dataReady,
    toSearchParams,
    toFacetParams,
  };
}
