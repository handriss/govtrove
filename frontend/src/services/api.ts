import type { Opportunity, SearchParams, SearchResult, FilterOptions } from '../types/opportunity';

const API_BASE = '/api';

function buildSearchQuery(params: SearchParams): string {
  const searchParams = new URLSearchParams();

  if (params.q) searchParams.set('q', params.q);
  if (params.type?.length) searchParams.set('type', params.type.join(','));
  if (params.set_aside?.length) searchParams.set('set_aside', params.set_aside.join(','));
  if (params.naics?.length) searchParams.set('naics', params.naics.join(','));
  if (params.state?.length) searchParams.set('state', params.state.join(','));
  if (params.posted_from) searchParams.set('posted_from', params.posted_from);
  if (params.posted_to) searchParams.set('posted_to', params.posted_to);
  if (params.deadline_from) searchParams.set('deadline_from', params.deadline_from);
  if (params.deadline_to) searchParams.set('deadline_to', params.deadline_to);
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.order) searchParams.set('order', params.order);
  if (params.page) searchParams.set('page', String(params.page));
  if (params.limit) searchParams.set('limit', String(params.limit));

  return searchParams.toString();
}

export async function searchOpportunities(params: SearchParams): Promise<SearchResult> {
  const query = buildSearchQuery(params);
  const url = `${API_BASE}/opportunities${query ? `?${query}` : ''}`;

  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Search failed: ${response.statusText}`);
  }
  return response.json();
}

export async function getOpportunity(id: number): Promise<Opportunity> {
  const response = await fetch(`${API_BASE}/opportunities/${id}`);
  if (!response.ok) {
    throw new Error(`Failed to get opportunity: ${response.statusText}`);
  }
  return response.json();
}

export async function getFilters(): Promise<FilterOptions> {
  const response = await fetch(`${API_BASE}/filters`);
  if (!response.ok) {
    throw new Error(`Failed to get filters: ${response.statusText}`);
  }
  return response.json();
}

export interface TrackEventData {
  event_type: 'search' | 'filter' | 'click' | 'page';
  session_id: string;
  query?: string;
  filters?: Record<string, unknown>;
  sort_by?: string;
  page?: number;
  total_results?: number;
  result_position?: number;
  opportunity_id?: number;
}

export async function trackEvent(data: TrackEventData): Promise<void> {
  try {
    await fetch(`${API_BASE}/events`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  } catch {
    // Fire and forget - don't let analytics errors affect user experience
  }
}

export interface SearchTermStat {
  query: string;
  count: number;
}

export interface FilterStat {
  value: string;
  count: number;
}

export interface FilterUsageStats {
  types: FilterStat[];
  set_asides: FilterStat[];
  states: FilterStat[];
}

export interface PositionStat {
  position: number;
  clicks: number;
}

export interface ClickStats {
  total_clicks: number;
  total_searches: number;
  click_through_rate: number;
  by_position: PositionStat[];
}

export interface EventCounts {
  searches: number;
  filters: number;
  clicks: number;
  pages: number;
}

export interface SearchAnalytics {
  popular_searches: SearchTermStat[];
  zero_result_searches: SearchTermStat[];
  filter_usage: FilterUsageStats;
  click_stats: ClickStats;
  event_counts: EventCounts;
}

export async function getAnalytics(period: string): Promise<SearchAnalytics> {
  const response = await fetch(`${API_BASE}/admin/analytics?period=${period}`);
  if (!response.ok) {
    throw new Error(`Failed to get analytics: ${response.statusText}`);
  }
  return response.json();
}
