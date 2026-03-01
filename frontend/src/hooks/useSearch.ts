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
}

export function useSearch() {
  const [state, setState] = useState<UseSearchState>({
    results: [],
    total: 0,
    page: 1,
    totalPages: 0,
    loading: false,
    error: null,
    suggestion: null,
  });

  const search = useCallback(async (params: SearchParams) => {
    setState((prev) => ({ ...prev, loading: true, error: null, suggestion: null }));

    try {
      const data: SearchResult = await searchOpportunities(params);
      setState({
        results: data.opportunities || [],
        total: data.total,
        page: data.page,
        totalPages: data.total_pages,
        loading: false,
        error: null,
        suggestion: data.suggestion || null,
      });
    } catch (err) {
      setState((prev) => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'Search failed',
      }));
    }
  }, []);

  const inject = useCallback((data: SearchResult) => {
    setState({
      results: data.opportunities || [],
      total: data.total,
      page: data.page,
      totalPages: data.total_pages,
      loading: false,
      error: null,
      suggestion: data.suggestion || null,
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
    });
  }, []);

  return { ...state, search, inject, reset };
}
