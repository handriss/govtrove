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
  award_number?: string;
  award_amount?: number;
  awardee_name?: string;
  awardee_uei?: string;
  award_date?: string;
  active: boolean;
  ui_link?: string;
  resource_links?: string[];
  data_source?: string;
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

export interface SearchParams {
  q?: string;
  type?: string;
  set_aside?: string;
  naics?: string;
  state?: string;
  posted_from?: string;
  posted_to?: string;
  deadline_from?: string;
  deadline_to?: string;
  sort?: string;
  order?: string;
  page?: number;
  limit?: number;
}

export interface StatusResponse {
  last_synced_at: string | null;
}

export interface QueryTerm {
  id: string;
  value: string;
  type: 'include' | 'exclude' | 'phrase';
}

export interface QueryGroup {
  id: string;
  operator: 'AND' | 'OR';
  terms: QueryTerm[];
}

export interface AdvancedFilters {
  types: string[];
  setAsides: string[];
  naicsCodes: string[];
  states: string[];
  postedFrom?: string;
  postedTo?: string;
  deadlineFrom?: string;
  deadlineTo?: string;
}
