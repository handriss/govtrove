import { useState, useEffect, useRef, useMemo } from 'react';
import { getFacetCounts } from '../services/api';
import { useDebounce } from './useDebounce';
import type { SearchParams, FacetResult } from '../types/api';

interface UseFacetCountsReturn {
  facets: FacetResult['facets'] | null;
  total: number;
  isLoading: boolean;
  error: string | null;
}

export function useFacetCounts(params: SearchParams): UseFacetCountsReturn {
  const [facets, setFacets] = useState<FacetResult['facets'] | null>(null);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const staleRef = useRef<FacetResult['facets'] | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  // Stabilize: only facet-relevant params (exclude sort/page/order/limit)
  const paramsKey = useMemo(() => {
    const { sort: _, order: _o, page: _p, limit: _l, ...facetParams } = params;
    return JSON.stringify(facetParams);
  }, [params]);

  const debouncedKey = useDebounce(paramsKey, 300);

  useEffect(() => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const facetParams: SearchParams = JSON.parse(debouncedKey);

    setIsLoading(true);
    setError(null);

    getFacetCounts(facetParams, controller.signal)
      .then((result) => {
        staleRef.current = result.facets;
        setFacets(result.facets);
        setTotal(result.total);
        setIsLoading(false);
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === 'AbortError') return;
        setError(err instanceof Error ? err.message : 'Failed to load facets');
        setIsLoading(false);
      });

    return () => controller.abort();
  }, [debouncedKey]);

  return {
    facets: facets ?? staleRef.current,
    total,
    isLoading,
    error,
  };
}
