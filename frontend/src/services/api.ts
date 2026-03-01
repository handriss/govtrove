import type { SearchResult, Opportunity, FilterOptions, SearchParams, StatusResponse, FacetResult, SolicitationHistory, SavedSearch, SavedOpportunitiesResponse, UserUpdatesResponse, UserUpdateCount, AgencyResult, AgencySearchResponse } from '../types/api';

const API_BASE = import.meta.env.VITE_API_URL || '/api';

export const AUTH_ERROR_EVENT = 'govtrove:auth-error';

function checkAuth(response: Response): void {
  if (response.status === 401) {
    window.dispatchEvent(new CustomEvent(AUTH_ERROR_EVENT));
  }
}

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
  if (params.psc) searchParams.set('psc', params.psc);
  if (params.state) searchParams.set('state', params.state);
  if (params.naics_prefix) searchParams.set('naics_prefix', params.naics_prefix);
  if (params.department) searchParams.set('department', params.department);
  if (params.agency) searchParams.set('agency', params.agency);
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
  if (params.psc) searchParams.set('psc', params.psc);
  if (params.naics_prefix) searchParams.set('naics_prefix', params.naics_prefix);
  if (params.state) searchParams.set('state', params.state);
  if (params.department) searchParams.set('department', params.department);
  if (params.agency) searchParams.set('agency', params.agency);
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

export type { AgencyResult };

export async function searchAgencies(
  q: string = '',
  limit: number = 15,
  signal?: AbortSignal,
): Promise<AgencySearchResponse> {
  const params = new URLSearchParams();
  if (q) params.set('q', q);
  if (limit) params.set('limit', String(limit));

  const response = await fetch(`${API_BASE}/agencies?${params}`, { signal });
  if (!response.ok) {
    throw new Error(`Agency search failed: ${response.statusText}`);
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
    checkAuth(response);
    throw new Error(`Sync failed: ${response.statusText}`);
  }
  return response.json();
}

export async function getMe(token: string): Promise<GovTroveUser> {
  const response = await fetch(`${API_BASE}/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    checkAuth(response);
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
    checkAuth(response);
    const text = await response.text();
    throw new Error(text || response.statusText);
  }
}

// --- Saved Opportunities ---

export async function getSavedOpportunities(token: string): Promise<number[]> {
  const response = await fetch(`${API_BASE}/saved/opportunities/ids`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to fetch saved opportunities: ${response.statusText}`); }
  const data = await response.json();
  return data.opportunity_ids;
}

export async function getSavedOpportunitiesWithDetails(
  token: string,
  params: { sort?: string; active_only?: boolean; page?: number; limit?: number } = {},
): Promise<SavedOpportunitiesResponse> {
  const searchParams = new URLSearchParams();
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.active_only) searchParams.set('active_only', 'true');
  if (params.page) searchParams.set('page', String(params.page));
  if (params.limit) searchParams.set('limit', String(params.limit));
  const response = await fetch(`${API_BASE}/saved/opportunities?${searchParams}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to fetch saved opportunities: ${response.statusText}`); }
  return response.json();
}

export async function saveOpportunity(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ opportunity_id: id }),
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to save opportunity: ${response.statusText}`); }
}

export async function unsaveOpportunity(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ opportunity_id: id }),
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to unsave opportunity: ${response.statusText}`); }
}

export async function unsaveByOpportunityId(token: string, opportunityId: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities/by-opportunity/${opportunityId}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to unsave opportunity: ${response.statusText}`); }
}

export async function bulkSaveOpportunities(token: string, ids: number[]): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities/bulk`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ opportunity_ids: ids }),
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to bulk save opportunities: ${response.statusText}`); }
}

export async function updateSavedOpportunityNotes(token: string, id: number, notes: string): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/opportunities/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ notes }),
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to update notes: ${response.statusText}`); }
}

// --- Saved Searches ---

export async function getSavedSearches(token: string): Promise<SavedSearch[]> {
  const response = await fetch(`${API_BASE}/saved/searches`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to fetch saved searches: ${response.statusText}`); }
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
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to create saved search: ${response.statusText}`); }
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
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to update saved search: ${response.statusText}`); }
  return response.json();
}

export async function deleteSavedSearch(token: string, id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/saved/searches/${id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to delete saved search: ${response.statusText}`); }
}

export async function runSavedSearch(
  token: string,
  id: number,
): Promise<{ search: SavedSearch; results: SearchResult }> {
  const response = await fetch(`${API_BASE}/saved/searches/${id}/run`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to run saved search: ${response.statusText}`); }
  return response.json();
}

// --- User Updates ---

export async function getUpdates(
  token: string,
  params: { unread_only?: boolean; page?: number; limit?: number } = {},
): Promise<UserUpdatesResponse> {
  const searchParams = new URLSearchParams();
  if (params.unread_only) searchParams.set('unread_only', 'true');
  if (params.page) searchParams.set('page', String(params.page));
  if (params.limit) searchParams.set('limit', String(params.limit));
  const response = await fetch(`${API_BASE}/updates?${searchParams}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to fetch updates: ${response.statusText}`); }
  return response.json();
}

export async function getUpdatesCount(token: string): Promise<UserUpdateCount> {
  const response = await fetch(`${API_BASE}/updates/count`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to fetch updates count: ${response.statusText}`); }
  return response.json();
}

export async function markUpdateRead(token: string, id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/updates/${id}/read`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to mark update read: ${response.statusText}`); }
}

export async function markAllUpdatesRead(token: string): Promise<void> {
  const response = await fetch(`${API_BASE}/updates/read-all`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to mark all updates read: ${response.statusText}`); }
}

export async function deleteUpdate(token: string, id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/updates/${id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) { checkAuth(response); throw new Error(`Failed to delete update: ${response.statusText}`); }
}

// Admin

export interface AdminUser {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  plan: string;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
}

export async function getAdminUsers(token: string): Promise<AdminUser[]> {
  const response = await fetch(`${API_BASE}/admin/users`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function getAdminNotifications(
  token: string,
  userId: number,
  page = 1,
): Promise<UserUpdatesResponse> {
  const response = await fetch(`${API_BASE}/admin/notifications?user_id=${userId}&page=${page}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export interface AdminApiKey {
  key_hash: string;
  email: string;
  daily_limit: number;
  created_at: string;
  expires_at: string;
}

export async function getAdminApiKeys(token: string): Promise<AdminApiKey[]> {
  const response = await fetch(`${API_BASE}/admin/api-keys`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export interface AdminSamgovRequest {
  id: number;
  request_timestamp: string;
  endpoint: string;
  method: string;
  http_status_code: number | null;
  response_time_ms: number | null;
  request_params: string | null;
  response_size_bytes: number | null;
  error_message: string | null;
  success: boolean;
  api_key_hash: string | null;
}

export interface AdminSamgovRequestsResponse {
  requests: AdminSamgovRequest[];
  total: number;
  page: number;
  limit: number;
}

export async function getAdminSamgovRequests(
  token: string,
  page = 1,
): Promise<AdminSamgovRequestsResponse> {
  const response = await fetch(`${API_BASE}/admin/samgov-requests?page=${page}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export interface UsageBucket {
  timestamp: string;
  success: number;
  failed: number;
}

// --- Pipeline Runs ---

export interface PipelineStep {
  id: string;
  step_name: string;
  status: string;
  started_at: string;
  completed_at: string | null;
  duration_ms: number | null;
  stats: Record<string, unknown> | null;
  error_message: string | null;
}

export interface PipelineExecution {
  execution_id: string;
  status: 'completed' | 'failed' | 'running';
  started_at: string;
  completed_at: string | null;
  duration_ms: number | null;
  step_count: number;
  steps: PipelineStep[];
}

export interface PipelineExecutionsResponse {
  runs: PipelineExecution[];
  total: number;
  page: number;
  limit: number;
}

// --- Pipeline Run Detail ---

export interface TableCounts {
  snap_csv: number;
  snap_api: number;
  snap_data_quality: number;
  snap_disappearances: number;
  snap_reconcile_dq: number;
}

export interface OpportunityStats {
  total_affected: number;
  inserted: number;
  updated: number;
  from_csv_only: number;
  from_api: number;
  from_both: number;
}

export interface PipelineRunDetailResponse {
  execution_id: string;
  status: string;
  started_at: string;
  completed_at: string | null;
  duration_ms: number | null;
  steps: PipelineStep[];
  table_counts: TableCounts;
  opportunity_stats: OpportunityStats;
}

export async function getAdminPipelineRunDetail(
  token: string,
  id: string,
): Promise<PipelineRunDetailResponse> {
  const response = await fetch(`${API_BASE}/admin/pipeline-runs/${id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function getAdminPipelineRuns(
  token: string,
  page = 1,
): Promise<PipelineExecutionsResponse> {
  const params = new URLSearchParams({ page: String(page) });
  const response = await fetch(`${API_BASE}/admin/pipeline-runs?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

// --- Search Events ---

export interface AdminSearchEvent {
  id: number;
  query: string | null;
  filters: string | null;
  sort_by: string | null;
  page: number | null;
  total_results: number | null;
  user_id: string | null;
  created_at: string;
}

export interface AdminSearchEventsResponse {
  events: AdminSearchEvent[];
  total: number;
  page: number;
  limit: number;
}

export async function getAdminSearchEvents(
  token: string,
  page = 1,
  emptyOnly = false,
): Promise<AdminSearchEventsResponse> {
  const params = new URLSearchParams({ page: String(page) });
  if (emptyOnly) params.set('empty_only', 'true');
  const response = await fetch(`${API_BASE}/admin/search-events?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

// --- Analytics ---

export interface SearchTermStat {
  query: string;
  count: number;
}

export interface FilterStat {
  value: string;
  count: number;
}

export interface SearchAnalytics {
  popular_searches: SearchTermStat[];
  zero_result_searches: SearchTermStat[];
  filter_usage: {
    types: FilterStat[];
    set_asides: FilterStat[];
    states: FilterStat[];
  };
  click_stats: {
    total_views: number;
    total_searches: number;
    click_through_rate: number;
  };
  event_counts: {
    searches: number;
    views: number;
    saves: number;
    search_saves: number;
  };
}

export async function getAdminAnalytics(
  token: string,
  period = '7d',
): Promise<SearchAnalytics> {
  const response = await fetch(`${API_BASE}/admin/analytics?period=${period}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function getAdminApiKeyUsage(
  token: string,
  keyHash: string,
  days: number,
): Promise<{ buckets: UsageBucket[] }> {
  const response = await fetch(
    `${API_BASE}/admin/api-key-usage?key_hash=${encodeURIComponent(keyHash)}&days=${days}`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

// --- Data Quality ---

export interface AdminDataQualityRow {
  id: number;
  notice_id: string;
  snapshot_date: string;
  source: string;
  issue_type: string;
  field_name: string | null;
  field_value: string | null;
  description: string | null;
  resolved: boolean;
  resolved_at: string | null;
  resolution_note: string | null;
  created_at: string;
}

export interface AdminDataQualityDetail extends AdminDataQualityRow {
  opp_title: string | null;
  opp_sol_num: string | null;
  opp_type: string | null;
  opp_active: boolean | null;
  opp_ui_link: string | null;
}

export interface AdminReconcileDQRow {
  id: number;
  notice_id: string;
  snapshot_date: string;
  issue_type: string;
  field_name: string;
  csv_value: string | null;
  api_value: string | null;
  resolved: boolean;
  resolved_at: string | null;
  resolution_note: string | null;
  created_at: string;
}

export interface AdminReconcileDQDetail extends AdminReconcileDQRow {
  opp_title: string | null;
  opp_sol_num: string | null;
  opp_type: string | null;
  opp_active: boolean | null;
  opp_ui_link: string | null;
}

export interface AdminDQListResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

export async function getAdminDataQualityIssues(
  token: string,
  params: { page?: number; limit?: number; sort?: string; order?: string; resolved?: string } = {},
): Promise<AdminDQListResponse<AdminDataQualityRow>> {
  const sp = new URLSearchParams();
  if (params.page) sp.set('page', String(params.page));
  if (params.limit) sp.set('limit', String(params.limit));
  if (params.sort) sp.set('sort', params.sort);
  if (params.order) sp.set('order', params.order);
  if (params.resolved) sp.set('resolved', params.resolved);
  const response = await fetch(`${API_BASE}/admin/data-quality?${sp}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function getAdminDataQualityDetail(
  token: string,
  id: number,
): Promise<AdminDataQualityDetail> {
  const response = await fetch(`${API_BASE}/admin/data-quality/${id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function updateAdminDataQualityResolution(
  token: string,
  id: number,
  data: { resolved: boolean; resolution_note: string },
): Promise<void> {
  const response = await fetch(`${API_BASE}/admin/data-quality/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error(`${response.status}`);
}

export async function getAdminReconcileDQIssues(
  token: string,
  params: { page?: number; limit?: number; sort?: string; order?: string; resolved?: string } = {},
): Promise<AdminDQListResponse<AdminReconcileDQRow>> {
  const sp = new URLSearchParams();
  if (params.page) sp.set('page', String(params.page));
  if (params.limit) sp.set('limit', String(params.limit));
  if (params.sort) sp.set('sort', params.sort);
  if (params.order) sp.set('order', params.order);
  if (params.resolved) sp.set('resolved', params.resolved);
  const response = await fetch(`${API_BASE}/admin/reconcile-dq?${sp}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function getAdminReconcileDQDetail(
  token: string,
  id: number,
): Promise<AdminReconcileDQDetail> {
  const response = await fetch(`${API_BASE}/admin/reconcile-dq/${id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) throw new Error(`${response.status}`);
  return response.json();
}

export async function updateAdminReconcileDQResolution(
  token: string,
  id: number,
  data: { resolved: boolean; resolution_note: string },
): Promise<void> {
  const response = await fetch(`${API_BASE}/admin/reconcile-dq/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error(`${response.status}`);
}
