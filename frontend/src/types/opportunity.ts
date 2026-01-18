export interface Opportunity {
  id: number;
  notice_id: string;
  title: string;
  solicitation_number?: string;
  type?: string;
  base_type?: string;
  department?: string;
  sub_tier?: string;
  office?: string;
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
  description?: string;
  award_number?: string;
  award_amount?: number;
  awardee_name?: string;
  awardee_uei?: string;
  award_date?: string;
  active: boolean;
  ui_link?: string;
  resource_links?: string[];
  data_source?: string;
  created_at: string;
  updated_at: string;
}

export interface OpportunityListItem {
  id: number;
  notice_id: string;
  title: string;
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

export interface SearchParams {
  q?: string;
  type?: string[];
  set_aside?: string[];
  naics?: string[];
  state?: string[];
  posted_from?: string;
  posted_to?: string;
  deadline_from?: string;
  deadline_to?: string;
  sort?: string;
  order?: string;
  page?: number;
  limit?: number;
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
