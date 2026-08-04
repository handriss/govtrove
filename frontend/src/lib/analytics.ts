import type { PostHog } from 'posthog-js';

export function trackSearch(
  ph: PostHog | undefined,
  query: string,
  filters: Record<string, unknown>,
  resultCount: number,
) {
  const hasFilters = !!(filters.naics || filters.set_aside || filters.type ||
    filters.state || filters.agency);
  ph?.capture('search_performed', {
    query,
    naics_code: filters.naics || null,
    set_aside: filters.set_aside || null,
    agency: filters.agency || null,
    type: filters.type || null,
    state: filters.state || null,
    sort: filters.sort || null,
    result_count: resultCount,
    has_filters: hasFilters,
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
  const referrer = document.referrer || '';
  let referringDomain = '';
  try { if (referrer) referringDomain = new URL(referrer).hostname; } catch {}

  const params = new URLSearchParams(window.location.search);
  ph?.capture('sign_up', {
    utm_source: ph?.get_property?.('$initial_utm_source') ?? params.get('utm_source') ?? null,
    utm_medium: ph?.get_property?.('$initial_utm_medium') ?? params.get('utm_medium') ?? null,
    utm_campaign: ph?.get_property?.('$initial_utm_campaign') ?? params.get('utm_campaign') ?? null,
    referrer: referrer || null,
    referring_domain: referringDomain || null,
  });
}

export function trackFounderCtaClicked(ph: PostHog | undefined, variant: string) {
  ph?.capture('founder_cta_clicked', { variant });
}

// --- Code-finder flow (cookieless: flow_id arrives via the ?fid= URL param) ---

export function registerFlowId(ph: PostHog | undefined, flowId: string) {
  // Super-property: every subsequent event in this page load carries flow_id,
  // so the finder→app funnel can be aggregated by it without cookies.
  ph?.register({ flow_id: flowId });
}

export function trackCodeFinderLanding(
  ph: PostHog | undefined,
  flowId: string,
  codeType: string | null,
  code: string | null,
) {
  ph?.capture('code_finder_app_landing', { flow_id: flowId, code_type: codeType, code });
}

export function trackCodeFinderZeroOpportunities(
  ph: PostHog | undefined,
  codeType: string | null,
  code: string | null,
) {
  ph?.capture('code_finder_zero_opportunities', { code_type: codeType, code });
}

export function trackAlertCreated(
  ph: PostHog | undefined,
  source: string,
  codeType?: string | null,
  code?: string | null,
) {
  ph?.capture('alert_created', { source, code_type: codeType ?? null, code: code ?? null });
}
