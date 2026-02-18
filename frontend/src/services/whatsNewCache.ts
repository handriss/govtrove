import { searchOpportunities } from './api';
import type { SearchResult, SearchParams } from '../types/api';

const DEFAULT_TYPES = 'Solicitation,Presolicitation,Combined Synopsis/Solicitation,Sources Sought';
const TTL_MS = 5 * 60 * 1000; // 5 minutes

let cached: SearchResult | null = null;
let cachedAt = 0;
let fetching: Promise<SearchResult> | null = null;

function isStale() {
  return !cached || Date.now() - cachedAt > TTL_MS;
}

function today() {
  return new Date().toISOString().split('T')[0];
}

export function whatsNewParams(pageNum = 1): SearchParams {
  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  return {
    type: DEFAULT_TYPES,
    posted_from: yesterday.toISOString().split('T')[0],
    deadline_from: today(),
    sort: 'posted_date',
    order: 'desc',
    page: pageNum,
    limit: 25,
  };
}

export function preloadWhatsNew(): void {
  if (!isStale() || fetching) return;
  fetching = searchOpportunities(whatsNewParams())
    .then((data) => { cached = data; cachedAt = Date.now(); return data; })
    .catch(() => null as unknown as SearchResult)
    .finally(() => { fetching = null; });
}

export async function getWhatsNew(): Promise<SearchResult> {
  if (!isStale()) return cached!;
  if (fetching) return fetching;
  const data = await searchOpportunities(whatsNewParams());
  cached = data;
  cachedAt = Date.now();
  return data;
}

export function getWhatsNewSync(): SearchResult | null {
  if (isStale()) return null;
  return cached;
}
