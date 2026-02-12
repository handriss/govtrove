import type { SearchResult, Opportunity, FilterOptions, SearchParams, StatusResponse } from '../types/api';

const API_BASE = import.meta.env.VITE_API_URL || '/api';

export interface GovTroveUser {
  id: number;
  workos_id: string;
  email: string;
  first_name: string;
  last_name: string;
  plan: string;
  created_at: string;
  updated_at: string;
}

export async function searchOpportunities(params: SearchParams = {}): Promise<SearchResult> {
  const searchParams = new URLSearchParams();

  if (params.q) searchParams.set('q', params.q);
  if (params.type) searchParams.set('type', params.type);
  if (params.set_aside) searchParams.set('set_aside', params.set_aside);
  if (params.naics) searchParams.set('naics', params.naics);
  if (params.state) searchParams.set('state', params.state);
  if (params.posted_from) searchParams.set('posted_from', params.posted_from);
  if (params.posted_to) searchParams.set('posted_to', params.posted_to);
  if (params.deadline_from) searchParams.set('deadline_from', params.deadline_from);
  if (params.deadline_to) searchParams.set('deadline_to', params.deadline_to);
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.order) searchParams.set('order', params.order);
  if (params.page) searchParams.set('page', String(params.page));
  if (params.limit) searchParams.set('limit', String(params.limit));

  const response = await fetch(`${API_BASE}/opportunities?${searchParams}`);
  if (!response.ok) {
    throw new Error(`Search failed: ${response.statusText}`);
  }
  return response.json();
}

export async function getOpportunity(id: number): Promise<Opportunity> {
  const response = await fetch(`${API_BASE}/opportunities/${id}`);
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error('Opportunity not found');
    }
    throw new Error(`Failed to fetch opportunity: ${response.statusText}`);
  }
  return response.json();
}

export async function getFilters(): Promise<FilterOptions> {
  const response = await fetch(`${API_BASE}/filters`);
  if (!response.ok) {
    throw new Error(`Failed to fetch filters: ${response.statusText}`);
  }
  return response.json();
}

export async function getStatus(): Promise<StatusResponse> {
  const response = await fetch(`${API_BASE}/status`);
  if (!response.ok) {
    throw new Error(`Failed to fetch status: ${response.statusText}`);
  }
  return response.json();
}

function getSessionId(): string {
  let id = localStorage.getItem('session_id');
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem('session_id', id);
  }
  return id;
}

export function trackEvent(event: {
  event_type: 'search' | 'filter' | 'click' | 'page';
  query?: string;
  filters?: Record<string, unknown>;
  sort_by?: string;
  page?: number;
  total_results?: number;
  result_position?: number;
  opportunity_id?: number;
}): void {
  fetch(`${API_BASE}/events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...event, session_id: getSessionId() }),
  }).catch(() => {});
}

export async function syncUser(
  token: string,
  data: { email: string; first_name: string; last_name: string },
): Promise<GovTroveUser> {
  const response = await fetch(`${API_BASE}/auth/sync`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error(`Sync failed: ${response.statusText}`);
  }
  return response.json();
}

export async function getMe(token: string): Promise<GovTroveUser> {
  const response = await fetch(`${API_BASE}/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    throw new Error(`Get me failed: ${response.statusText}`);
  }
  return response.json();
}
