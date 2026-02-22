import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Star, Search, ArrowLeft, X, Loader2 } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { useSavedOpportunities } from '../hooks/useSavedOpportunities';
import { useSavedSearches } from '../hooks/useSavedSearches';
import { getOpportunity } from '../services/api';
import OpportunityCard from '../components/search/OpportunityCard';
import type { OpportunityListItem } from '../types/api';
import AuthButton from '../components/AuthButton';

function filterSummary(filters: Record<string, unknown>): string {
  const parts: string[] = [];
  if (filters.keyword) parts.push(`"${filters.keyword}"`);
  if (Array.isArray(filters.naics) && filters.naics.length) parts.push(`NAICS: ${filters.naics.join(', ')}`);
  if (Array.isArray(filters.psc) && filters.psc.length) parts.push(`PSC: ${filters.psc.join(', ')}`);
  if (Array.isArray(filters.setAside) && filters.setAside.length) parts.push(`Set-Aside: ${filters.setAside.join(', ')}`);
  if (Array.isArray(filters.noticeType) && filters.noticeType.length) parts.push(`Type: ${filters.noticeType.join(', ')}`);
  if (filters.department) parts.push(`Dept: ${filters.department}`);
  if (filters.state) parts.push(`State: ${filters.state}`);
  return parts.join(' | ') || 'All opportunities';
}

function filtersToURLParams(filters: Record<string, unknown>): string {
  const params = new URLSearchParams();
  if (filters.keyword) params.set('q', String(filters.keyword));
  if (Array.isArray(filters.naics) && filters.naics.length) params.set('naics', filters.naics.join(','));
  if (Array.isArray(filters.psc) && filters.psc.length) params.set('psc', filters.psc.join(','));
  if (Array.isArray(filters.setAside) && filters.setAside.length) params.set('set_aside', filters.setAside.join(','));
  if (Array.isArray(filters.noticeType) && filters.noticeType.length) params.set('type', filters.noticeType.join(','));
  if (filters.department) params.set('department', String(filters.department));
  if (filters.state) params.set('state', String(filters.state));
  if (filters.postedFrom) params.set('posted_from', String(filters.postedFrom));
  if (filters.postedTo) params.set('posted_to', String(filters.postedTo));
  if (filters.deadlinePreset) params.set('deadline', String(filters.deadlinePreset));
  if (filters.deadlineFrom) params.set('deadline_from', String(filters.deadlineFrom));
  if (filters.deadlineTo) params.set('deadline_to', String(filters.deadlineTo));
  return params.toString();
}

export default function SavedPage() {
  const { isAuthenticated } = useAppAuth();
  const saved = useSavedOpportunities();
  const { savedSearches, loading: searchesLoading, deleteSearch } = useSavedSearches();
  const navigate = useNavigate();
  const [opportunities, setOpportunities] = useState<OpportunityListItem[]>([]);
  const [loading, setLoading] = useState(false);

  const savedIdsList = [...saved.savedIds];

  // Fetch opportunity details for saved IDs
  useEffect(() => {
    if (savedIdsList.length === 0) {
      setOpportunities([]);
      return;
    }

    let cancelled = false;
    setLoading(true);

    (async () => {
      const results: OpportunityListItem[] = [];
      // Fetch in parallel batches of 10
      for (let i = 0; i < savedIdsList.length; i += 10) {
        if (cancelled) return;
        const batch = savedIdsList.slice(i, i + 10);
        const fetched = await Promise.allSettled(batch.map((id) => getOpportunity(id)));
        for (const result of fetched) {
          if (result.status === 'fulfilled') {
            const o = result.value;
            results.push({
              id: o.id,
              notice_id: o.notice_id,
              title: o.title,
              description: o.description,
              solicitation_number: o.solicitation_number,
              type: o.type,
              department: o.department,
              posted_date: o.posted_date,
              response_deadline: o.response_deadline,
              set_aside_code: o.set_aside_code,
              set_aside_description: o.set_aside_description,
              naics_code: o.naics_code,
              pop_state: o.pop_state,
              active: o.active,
            });
          }
        }
      }
      if (!cancelled) {
        setOpportunities(results);
        setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [saved.savedIds.size]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="min-h-screen flex flex-col relative">
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      <div className="absolute top-4 right-6 z-20">
        <AuthButton />
      </div>

      <div className="relative z-10 max-w-4xl w-full mx-auto px-6 pt-10 pb-10">
        <div className="flex items-center gap-4 mb-8">
          <Link to="/" className="text-dark-400 hover:text-dark-200 transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <h1 className="text-2xl font-semibold text-dark-100">Saved</h1>
        </div>

        {/* Saved Searches */}
        {isAuthenticated && (
          <section className="mb-10">
            <h2 className="text-lg font-medium text-dark-200 mb-4 flex items-center gap-2">
              <Search size={18} />
              Saved Searches
            </h2>
            {searchesLoading ? (
              <div className="flex items-center gap-2 text-dark-500 text-sm py-4">
                <Loader2 size={16} className="animate-spin" />
                Loading saved searches...
              </div>
            ) : savedSearches.length === 0 ? (
              <p className="text-dark-500 text-sm py-4">
                No saved searches yet. Apply filters on the search page and click "Save Search" to save them here.
              </p>
            ) : (
              <div className="space-y-2">
                {savedSearches.map((search) => (
                  <div
                    key={search.id}
                    className="group flex items-center gap-3 px-4 py-3 rounded-xl border border-dark-800/50
                               bg-dark-900/30 hover:border-dark-600/50 transition-all cursor-pointer"
                    onClick={() => navigate(`/?${filtersToURLParams(search.filters)}`)}
                  >
                    <Search size={14} className="text-dark-500 shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-dark-200 truncate">{search.name}</p>
                      <p className="text-xs text-dark-500 truncate mt-0.5">{filterSummary(search.filters)}</p>
                    </div>
                    <button
                      onClick={(e) => { e.stopPropagation(); deleteSearch(search.id); }}
                      className="opacity-0 group-hover:opacity-100 text-dark-500 hover:text-red-400 transition-all p-1"
                      aria-label={`Delete saved search "${search.name}"`}
                    >
                      <X size={14} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </section>
        )}

        {/* Saved Opportunities */}
        <section>
          <h2 className="text-lg font-medium text-dark-200 mb-4 flex items-center gap-2">
            <Star size={18} />
            Saved Opportunities
            {saved.count > 0 && (
              <span className="text-sm text-dark-500 font-normal">({saved.count})</span>
            )}
          </h2>
          {loading ? (
            <div className="flex items-center gap-2 text-dark-500 text-sm py-4">
              <Loader2 size={16} className="animate-spin" />
              Loading saved opportunities...
            </div>
          ) : savedIdsList.length === 0 ? (
            <div className="text-center py-12">
              <Star size={32} className="text-dark-700 mx-auto mb-3" />
              <p className="text-dark-400 text-sm mb-1">No saved opportunities yet</p>
              <p className="text-dark-600 text-xs">
                Star opportunities from search results to save them here.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {opportunities.map((opp) => (
                <OpportunityCard
                  key={opp.id}
                  opportunity={opp}
                  isSaved={saved.isSaved(opp.id)}
                  isSelected={false}
                  anySelected={false}
                  onToggleSave={saved.toggleSave}
                  onToggleSelect={() => {}}
                />
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
