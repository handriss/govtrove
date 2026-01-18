import { useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { trackEvent, getAnalytics } from '../services/api';
import type { SearchParams } from '../types/opportunity';

const SESSION_KEY = 'opscout_session_id';

function getSessionId(): string {
  let sessionId = localStorage.getItem(SESSION_KEY);
  if (!sessionId) {
    sessionId = crypto.randomUUID();
    localStorage.setItem(SESSION_KEY, sessionId);
  }
  return sessionId;
}

function buildFilters(params: SearchParams): Record<string, unknown> | undefined {
  const filters: Record<string, unknown> = {};
  if (params.type?.length) filters.type = params.type;
  if (params.set_aside?.length) filters.set_aside = params.set_aside;
  if (params.naics?.length) filters.naics = params.naics;
  if (params.state?.length) filters.state = params.state;
  if (params.posted_from) filters.posted_from = params.posted_from;
  if (params.posted_to) filters.posted_to = params.posted_to;
  if (params.deadline_from) filters.deadline_from = params.deadline_from;
  if (params.deadline_to) filters.deadline_to = params.deadline_to;
  return Object.keys(filters).length > 0 ? filters : undefined;
}

export function useAnalytics() {
  const sessionId = useMemo(() => getSessionId(), []);

  const trackSearch = useCallback(
    (params: SearchParams, totalResults: number) => {
      trackEvent({
        event_type: 'search',
        session_id: sessionId,
        query: params.q || undefined,
        filters: buildFilters(params),
        sort_by: params.sort,
        page: params.page,
        total_results: totalResults,
      });
    },
    [sessionId]
  );

  const trackFilter = useCallback(
    (params: SearchParams, totalResults: number) => {
      trackEvent({
        event_type: 'filter',
        session_id: sessionId,
        query: params.q || undefined,
        filters: buildFilters(params),
        sort_by: params.sort,
        total_results: totalResults,
      });
    },
    [sessionId]
  );

  const trackClick = useCallback(
    (opportunityId: number, resultPosition: number, params: SearchParams) => {
      trackEvent({
        event_type: 'click',
        session_id: sessionId,
        query: params.q || undefined,
        filters: buildFilters(params),
        opportunity_id: opportunityId,
        result_position: resultPosition,
      });
    },
    [sessionId]
  );

  const trackPage = useCallback(
    (page: number, params: SearchParams, totalResults: number) => {
      trackEvent({
        event_type: 'page',
        session_id: sessionId,
        query: params.q || undefined,
        filters: buildFilters(params),
        page,
        total_results: totalResults,
      });
    },
    [sessionId]
  );

  return { trackSearch, trackFilter, trackClick, trackPage };
}

export function useSearchAnalytics(period: string) {
  return useQuery({
    queryKey: ['analytics', period],
    queryFn: () => getAnalytics(period),
    staleTime: 60000,
  });
}
