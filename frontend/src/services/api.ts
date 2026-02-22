import type { SearchResult, Opportunity, FilterOptions, SearchParams, StatusResponse, FacetResult, SolicitationHistory, SavedSearch } from '../types/api';

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
  if (params.naics_prefix) searchParams.set('naics_prefix', params.naics_prefix);
  if (params.department) searchParams.set('department', params.department);
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

export async function getFacetCounts(params: SearchParams = {}, signal?: AbortSignal): Promise<FacetResult> {
  const searchParams = new URLSearchParams();

  if (params.q) searchParams.set('q', params.q);
  if (params.type) searchParams.set('type', params.type);
  if (params.set_aside) searchParams.set('set_aside', params.set_aside);
  if (params.naics) searchParams.set('naics', params.naics);
  if (params.naics_prefix) searchParams.set('naics_prefix', params.naics_prefix);
  if (params.state) searchParams.set('state', params.state);
  if (params.department) searchParams.set('department', params.department);
  if (params.posted_from) searchParams.set('posted_from', params.posted_from);
  if (params.posted_to) searchParams.set('posted_to', params.posted_to);
  if (params.deadline_from) searchParams.set('deadline_from', params.deadline_from);
  if (params.deadline_to) searchParams.set('deadline_to', params.deadline_to);

  const response = await fetch(`${API_BASE}/opportunities/facets?${searchParams}`, { signal });
  if (!response.ok) {
    throw new Error(`Facets failed: ${response.statusText}`);
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

export async function getSolicitationHistory(id: number): Promise<SolicitationHistory | null> {
  const response = await fetch(`${API_BASE}/opportunities/${id}/history`);
  if (!response.ok) return null;
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

export async function createAccountRequest(
  token: string,
  requestType: 'data_export' | 'account_deletion',
): Promise<void> {
  const response = await fetch(`${API_BASE}/account/requests`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ request_type: requestType }),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || response.statusText);
  }
}

// --- Saved Opportunities ---

export async function getSavedOpportunities(token: string): Promise<number[]> {
  const response = await fetch(`${API_BASE}/saved/opportunities`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`Failed to fetch saved opportunities: ${response.statusText}`);
  const data = await response.json();
  return data.opportunity_ids;
}

export async function saveOpportunity(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ opportunity_id: id }),
  });
  if (!response.ok) throw new Error(`Failed to save opportunity: ${response.statusText}`);
}

export async function unsaveOpportunity(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ opportunity_id: id }),
  });
  if (!response.ok) throw new Error(`Failed to unsave opportunity: ${response.statusText}`);
}

export async function bulkSaveOpportunities(token: string, ids: number[]): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities/bulk`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ opportunity_ids: ids }),
  });
  if (!response.ok) throw new Error(`Failed to bulk save opportunities: ${response.statusText}`);
}

// --- Saved Searches ---

export async function getSavedSearches(token: string): Promise<SavedSearch[]> {
  const response = await fetch(`${API_BASE}/saved/searches`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`Failed to fetch saved searches: ${response.statusText}`);
  return response.json();
}

export async function createSavedSearch(
  token: string,
  name: string,
  filters: object,
): Promise<SavedSearch> {
  const response = await fetch(`${API_BASE}/saved/searches`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ name, filters }),
  });
  if (!response.ok) throw new Error(`Failed to create saved search: ${response.statusText}`);
  return response.json();
}

export async function updateSavedSearch(
  token: string,
  id: number,
  data: { name?: string; filters?: object },
): Promise<SavedSearch> {
  const response = await fetch(`${API_BASE}/saved/searches/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error(`Failed to update saved search: ${response.statusText}`);
  return response.json();
}

export async function deleteSavedSearch(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/searches/${id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`Failed to delete saved search: ${response.statusText}`);
}
