import type { PostHog } from 'posthog-js';

export function trackSearch(
  ph: PostHog | undefined,
  query: string,
  filters: Record<string, unknown>,
  resultCount: number,
) {
  ph?.capture('search_performed', {
    query,
    naics: filters.naics || null,
    set_aside: filters.setAside || null,
    type: filters.noticeType || null,
    state: filters.state || null,
    sort: filters.sort || null,
    result_count: resultCount,
  });
}

export function trackOpportunityViewed(
  ph: PostHog | undefined,
  opp: { notice_id: string; naics_code?: string; set_aside_code?: string; type?: string; department?: string },
) {
  ph?.capture('opportunity_viewed', {
    notice_id: opp.notice_id,
    naics: opp.naics_code || null,
    set_aside: opp.set_aside_code || null,
    type: opp.type || null,
    agency: opp.department || null,
  });
}

export function trackOpportunitySaved(ph: PostHog | undefined, noticeId: number) {
  ph?.capture('opportunity_saved', { notice_id: noticeId });
}

export function trackOpportunityUnsaved(ph: PostHog | undefined, noticeId: number) {
  ph?.capture('opportunity_unsaved', { notice_id: noticeId });
}

export function trackSavedSearchCreated(ph: PostHog | undefined, filters: Record<string, unknown>) {
  ph?.capture('saved_search_created', {
    has_naics: Array.isArray(filters.naics) ? filters.naics.length > 0 : !!filters.naics,
    has_set_aside: Array.isArray(filters.setAside) ? filters.setAside.length > 0 : !!filters.setAside,
    has_type: Array.isArray(filters.noticeType) ? filters.noticeType.length > 0 : !!filters.noticeType,
  });
}

export function trackUpgradeClicked(ph: PostHog | undefined, source: string) {
  ph?.capture('upgrade_clicked', { source });
}

export function trackSignIn(ph: PostHog | undefined) {
  ph?.capture('sign_in');
}

export function trackSignUp(ph: PostHog | undefined) {
  ph?.capture('sign_up');
}

export function trackFounderCtaClicked(ph: PostHog | undefined, variant: string) {
  ph?.capture('founder_cta_clicked', { variant });
}
