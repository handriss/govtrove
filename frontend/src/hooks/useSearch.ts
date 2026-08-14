import { useState, useCallback } from 'react';
import { searchOpportunities } from '../services/api';
import type { SearchParams, SearchResult, OpportunityListItem } from '../types/api';

interface UseSearchState {
  results: OpportunityListItem[];
  total: number;
  page: number;
  totalPages: number;
  loading: boolean;
  error: string | null;
  suggestion: string | null;
  relaxedQuery: string | null;
  relaxedTotal: number;
}

interface UseSearchOptions {
  getAccessToken?: () => Promise<string>;
}

export function useSearch(options?: UseSearchOptions) {
  const [state, setState] = useState<UseSearchState>({
    results: [],
    total: 0,
    page: 1,
    totalPages: 0,
    loading: false,
    error: null,
    suggestion: null,
    relaxedQuery: null,
    relaxedTotal: 0,
  });

  const search = useCallback(async (params: SearchParams) => {
    setState((prev) => ({ ...prev, loading: true, error: null, suggestion: null, relaxedQuery: null, relaxedTotal: 0 }));

    try {
      let token: string | undefined;
      if (options?.getAccessToken) {
        try { token = await options.getAccessToken(); } catch { /* not authenticated */ }
      }
      const data: SearchResult = await searchOpportunities(params, token);
      setState({
        results: data.opportunities || [],
        total: data.total,
        page: data.page,
        totalPages: data.total_pages,
        loading: false,
        error: null,
        suggestion: data.suggestion || null,
        relaxedQuery: data.relaxed_query || null,
        relaxedTotal: data.relaxed_total || 0,
      });
    } catch (err) {
      setState((prev) => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'Search failed',
      }));
    }
  }, [options?.getAccessToken]);

  const inject = useCallback((data: SearchResult) => {
    setState({
      results: data.opportunities || [],
      total: data.total,
      page: data.page,
      totalPages: data.total_pages,
      loading: false,
      error: null,
      suggestion: data.suggestion || null,
      relaxedQuery: data.relaxed_query || null,
      relaxedTotal: data.relaxed_total || 0,
    });
  }, []);

  const reset = useCallback(() => {
    setState({
      results: [],
      total: 0,
      page: 1,
      totalPages: 0,
      loading: false,
      error: null,
      suggestion: null,
      relaxedQuery: null,
      relaxedTotal: 0,
    });
  }, []);

  return { ...state, search, inject, reset };
}
