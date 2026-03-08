export interface OpportunityListItem {
  id: number;
  notice_id: string;
  title: string;
  description?: string;
  solicitation_number?: string;
  type?: string;
  department?: string;
  posted_date?: string;
  response_deadline?: string;
  set_aside_code?: string;
  set_aside_description?: string;
  naics_code?: string;
  pop_state?: string;
  active: boolean;
}

export interface Opportunity {
  id: number;
  notice_id: string;
  title: string;
  description?: string;
  solicitation_number?: string;
  type?: string;
  base_type?: string;
  department?: string;
  posted_date?: string;
  response_deadline?: string;
  archive_date?: string;
  set_aside_code?: string;
  set_aside_description?: string;
  naics_code?: string;
  naics_codes?: string[];
  classification_code?: string;
  pop_street_address?: string;
  pop_city?: string;
  pop_state?: string;
  pop_zip?: string;
  pop_country?: string;
  pop_city_code?: string;
  pop_state_code?: string;
  pop_country_code?: string;
  award_number?: string;
  award_amount?: number;
  awardee_name?: string;
  awardee_uei?: string;
  award_date?: string;
  active: boolean;
  ui_link?: string;
  resource_links?: string[];
  data_source?: string;
  snap_csv_id?: number;
  snap_api_id?: number;
  primary_contact_title?: string;
  primary_contact_fullname?: string;
  primary_contact_email?: string;
  primary_contact_phone?: string;
  primary_contact_fax?: string;
  secondary_contact_title?: string;
  secondary_contact_fullname?: string;
  secondary_contact_email?: string;
  secondary_contact_phone?: string;
  secondary_contact_fax?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchResult {
  opportunities: OpportunityListItem[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  suggestion?: string;
}

export interface FilterOption {
  code: string;
  label?: string;
  count: number;
}

export interface FilterOptions {
  types: FilterOption[];
  set_asides: FilterOption[];
  states: FilterOption[];
}

export interface FacetValue {
  value: string;
  label?: string;
  count: number;
}

export interface FacetResult {
  total: number;
  facets: {
    set_aside: FacetValue[];
    notice_type: FacetValue[];
    agency: FacetValue[];
    naics: FacetValue[];
    psc: FacetValue[];
  };
}

export interface SearchParams {
  q?: string;
  type?: string;
  set_aside?: string;
  naics?: string;
  naics_prefix?: string;
  psc?: string;
  psc_prefix?: string;
  state?: string;
  department?: string;
  agency?: string;
  posted_from?: string;
  posted_to?: string;
  deadline_from?: string;
  deadline_to?: string;
  sol_num?: string;
  pop_city?: string;
  sort?: string;
  order?: string;
  page?: number;
  limit?: number;
}

export interface AgencyResult {
  name: string;
  short_name: string | null;
  level: string;
  parent_path: string;
  breadcrumb: string;
  count: number;
}

export interface AgencySearchResponse {
  agencies: AgencyResult[];
}

export interface StatusResponse {
  last_synced_at: string | null;
}

export interface FieldChange {
  field_name: string;
  old_value?: string;
  new_value?: string;
}

export interface SolicitationHistoryItem {
  id: number;
  notice_id: string;
  title: string;
  type?: string;
  base_type?: string;
  posted_date?: string;
  response_deadline?: string;
  archive_date?: string;
  award_date?: string;
  award_amount?: number;
  awardee_name?: string;
  set_aside_code?: string;
  active: boolean;
  is_current: boolean;
  version: number;
  contact_name?: string;
  resource_count: number;
  changes?: FieldChange[];
}

export interface SolicitationHistory {
  solicitation_number: string;
  total_notices: number;
  notices: SolicitationHistoryItem[];
  truncated: boolean;
}

export interface SavedSearch {
  id: number;
  name: string;
  filters: Record<string, unknown>;
  alert_enabled: boolean;
  last_checked_at?: string;
  last_match_count: number;
  total_result_count?: number;
  created_at: string;
  updated_at: string;
}

export interface SavedOpportunityDetail {
  id: number;
  opportunity_id: number;
  notice_id: string;
  solicitation_number?: string;
  notes?: string;
  created_at: string;
  title: string;
  description?: string;
  type?: string;
  department?: string;
  posted_date?: string;
  response_deadline?: string;
  set_aside_code?: string;
  set_aside_description?: string;
  naics_code?: string;
  pop_state?: string;
  active: boolean;
  has_updates: boolean;
}

export interface SavedOpportunitiesResponse {
  opportunities: SavedOpportunityDetail[];
  total: number;
  page: number;
  limit: number;
}

export interface Notification {
  id: string;
  user_id: number;
  update_type: string;
  source_id?: number;
  group_key?: string;
  details: Record<string, unknown>;
  is_read: boolean;
  expires_at: string;
  created_at: string;
}

export interface NotificationCount {
  unread: number;
  total: number;
}

export interface NotificationsResponse {
  notifications: Notification[];
  total: number;
  page: number;
  limit: number;
}

