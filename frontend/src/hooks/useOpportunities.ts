import { useQuery } from '@tanstack/react-query';
import { searchOpportunities, getOpportunity, getFilters } from '../services/api';
import type { SearchParams } from '../types/opportunity';

export function useOpportunitySearch(params: SearchParams) {
  return useQuery({
    queryKey: ['opportunities', params],
    queryFn: () => searchOpportunities(params),
    staleTime: 30000,
  });
}

export function useOpportunity(id: number | null) {
  return useQuery({
    queryKey: ['opportunity', id],
    queryFn: () => getOpportunity(id!),
    enabled: id !== null,
    staleTime: 60000,
  });
}

export function useFilters() {
  return useQuery({
    queryKey: ['filters'],
    queryFn: getFilters,
    staleTime: 300000,
  });
}
