import { useState, useEffect, useCallback, useRef } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Star, Search, X, Loader2, Bell, BellOff, StickyNote, ArrowUpDown } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { useSavedOpportunities } from '../hooks/useSavedOpportunities';
import { useSavedSearches } from '../hooks/useSavedSearches';
import { getSavedOpportunitiesWithDetails, updateSavedOpportunityNotes } from '../services/api';
import type { SavedOpportunityDetail } from '../types/api';

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

function formatDeadline(deadline?: string): string {
  if (!deadline) return '';
  const d = new Date(deadline);
  const now = new Date();
  const diff = Math.ceil((d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
  const formatted = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  if (diff < 0) return `${formatted} (expired)`;
  if (diff === 0) return `${formatted} (today)`;
  if (diff === 1) return `${formatted} (tomorrow)`;
  if (diff <= 7) return `${formatted} (${diff}d)`;
  return formatted;
}

function deadlineColor(deadline?: string): string {
  if (!deadline) return 'text-dark-500';
  const diff = Math.ceil((new Date(deadline).getTime() - Date.now()) / (1000 * 60 * 60 * 24));
  if (diff < 0) return 'text-dark-600';
  if (diff <= 2) return 'text-red-400';
  if (diff <= 7) return 'text-amber-400';
  return 'text-dark-400';
}

export default function SavedPage() {
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const saved = useSavedOpportunities();
  const { savedSearches, loading: searchesLoading, deleteSearch, toggleAlert, renameSearch } = useSavedSearches();
  const navigate = useNavigate();

  const [opportunities, setOpportunities] = useState<SavedOpportunityDetail[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [sort, setSort] = useState<'saved' | 'deadline'>('saved');
  const [activeOnly, setActiveOnly] = useState(false);
  const [editingSearchName, setEditingSearchName] = useState<number | null>(null);
  const [searchNameText, setSearchNameText] = useState('');
  const [editingNotes, setEditingNotes] = useState<number | null>(null);
  const [notesText, setNotesText] = useState('');
  const notesTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const fetchOpportunities = useCallback(async () => {
    if (!isAuthenticated) return;
    setLoading(true);
    try {
      const token = await getAccessToken();
      const data = await getSavedOpportunitiesWithDetails(token, {
        sort: sort === 'deadline' ? 'deadline' : undefined,
        active_only: activeOnly,
        limit: 100,
      });
      setOpportunities(data.opportunities);
      setTotal(data.total);
    } catch {
      // Silently fail
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, getAccessToken, sort, activeOnly]);

  useEffect(() => {
    fetchOpportunities();
  }, [fetchOpportunities]);

  const handleUnsave = useCallback(async (opportunityId: number) => {
    saved.toggleSave(opportunityId);
    setOpportunities((prev) => prev.filter((o) => o.opportunity_id !== opportunityId));
    setTotal((prev) => prev - 1);
  }, [saved]);

  const handleNotesBlur = useCallback(async (id: number, notes: string) => {
    clearTimeout(notesTimerRef.current);
    try {
      const token = await getAccessToken();
      await updateSavedOpportunityNotes(token, id, notes);
    } catch {
      // Silently fail
    }
    setEditingNotes(null);
  }, [getAccessToken]);

  const commitSearchRename = useCallback(async (id: number, newName: string, originalName: string) => {
    setEditingSearchName(null);
    const trimmed = newName.trim();
    if (!trimmed || trimmed === originalName) return;
    try {
      await renameSearch(id, trimmed);
    } catch { /* rollback handled by hook refetch */ }
  }, [renameSearch]);

  const startEditingNotes = useCallback((opp: SavedOpportunityDetail) => {
    setEditingNotes(opp.id);
    setNotesText(opp.notes || '');
  }, []);

  return (
    <div className="min-h-screen flex flex-col relative">
      <div className="fixed inset-0 bg-gradient-to-b from-dark-900/20 via-transparent to-dark-950/40 pointer-events-none" />

      <div className="relative z-10 max-w-4xl w-full mx-auto px-6 pt-10 pb-10">
        <h1 className="text-2xl font-semibold text-dark-100 mb-8">Saved</h1>

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
              <div className="text-center py-12">
                <Search size={32} className="text-dark-700 mx-auto mb-3" />
                <p className="text-dark-400 text-sm mb-1">No saved searches yet</p>
                <p className="text-dark-600 text-xs">Apply filters on the search page and click &ldquo;Save Search&rdquo; to save them here.</p>
              </div>
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
                      <div className="flex items-center gap-2">
                        {editingSearchName === search.id ? (
                          <input
                            value={searchNameText}
                            onChange={(e) => setSearchNameText(e.target.value)}
                            onBlur={() => commitSearchRename(search.id, searchNameText, search.name)}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') (e.target as HTMLInputElement).blur();
                              if (e.key === 'Escape') setEditingSearchName(null);
                            }}
                            onClick={(e) => e.stopPropagation()}
                            className="text-sm font-medium text-dark-200 bg-dark-800/50 border border-dark-700/50
                                       rounded-lg px-2 py-0.5 focus:outline-none focus:border-accent/50 min-w-0 flex-1"
                            autoFocus
                            maxLength={100}
                          />
                        ) : (
                          <p
                            className="text-sm font-medium text-dark-200 truncate cursor-text hover:text-dark-100"
                            onClick={(e) => {
                              e.stopPropagation();
                              setEditingSearchName(search.id);
                              setSearchNameText(search.name);
                            }}
                            title="Click to rename"
                          >
                            {search.name}
                          </p>
                        )}
                        {search.last_match_count > 0 && (
                          <span className="text-[10px] font-medium text-accent bg-accent/10 px-1.5 py-0.5 rounded-full shrink-0">
                            +{search.last_match_count} new
                          </span>
                        )}
                        {search.total_result_count != null && (
                          <span className="text-[10px] text-dark-600 shrink-0">
                            {search.total_result_count.toLocaleString()} total
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-dark-500 truncate mt-0.5">{filterSummary(search.filters)}</p>
                    </div>
                    <button
                      onClick={(e) => { e.stopPropagation(); toggleAlert(search.id, !search.alert_enabled); }}
                      className="text-dark-500 hover:text-dark-300 transition-colors p-1"
                      title={search.alert_enabled ? 'Disable alerts' : 'Enable alerts'}
                    >
                      {search.alert_enabled ? <Bell size={14} /> : <BellOff size={14} className="text-dark-700" />}
                    </button>
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
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-dark-200 flex items-center gap-2">
              <Star size={18} />
              Saved Opportunities
              {total > 0 && (
                <span className="text-sm text-dark-500 font-normal">({total})</span>
              )}
            </h2>
            {total > 0 && (
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setActiveOnly(!activeOnly)}
                  className={`text-xs px-2.5 py-1 rounded-lg border transition-colors ${
                    activeOnly
                      ? 'border-accent/30 text-accent bg-accent/10'
                      : 'border-dark-700/50 text-dark-500 hover:text-dark-300'
                  }`}
                >
                  Active only
                </button>
                <button
                  onClick={() => setSort(sort === 'saved' ? 'deadline' : 'saved')}
                  className="flex items-center gap-1 text-xs px-2.5 py-1 rounded-lg border border-dark-700/50
                             text-dark-500 hover:text-dark-300 transition-colors"
                >
                  <ArrowUpDown size={12} />
                  {sort === 'saved' ? 'Date saved' : 'Deadline'}
                </button>
              </div>
            )}
          </div>

          {loading ? (
            <div className="flex items-center gap-2 text-dark-500 text-sm py-4">
              <Loader2 size={16} className="animate-spin" />
              Loading saved opportunities...
            </div>
          ) : opportunities.length === 0 ? (
            <div className="text-center py-12">
              <Star size={32} className="text-dark-700 mx-auto mb-3" />
              <p className="text-dark-400 text-sm mb-1">
                {total > 0 ? 'No active opportunities match your filter' : 'No saved opportunities yet'}
              </p>
              <p className="text-dark-600 text-xs">
                {total > 0 ? 'Try toggling "Active only" off' : 'Star opportunities from search results to save them here.'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {opportunities.map((opp) => (
                <div
                  key={opp.id}
                  className="group relative px-4 py-3 rounded-xl border border-dark-800/50
                             bg-dark-900/30 hover:border-dark-600/50 transition-all"
                >
                  <div className="flex items-start gap-3">
                    <button
                      onClick={() => handleUnsave(opp.opportunity_id)}
                      className="mt-0.5 text-amber-400 hover:text-dark-500 transition-colors shrink-0"
                      title="Remove from saved"
                    >
                      <Star size={16} fill="currentColor" />
                    </button>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/opportunity/${opp.opportunity_id}`}
                          className="text-sm font-medium text-dark-200 hover:text-accent transition-colors truncate"
                        >
                          {opp.title}
                        </Link>
                        {opp.has_updates && (
                          <span className="text-[10px] font-medium text-blue-400 bg-blue-400/10 px-1.5 py-0.5 rounded-full shrink-0">
                            Updated
                          </span>
                        )}
                        {!opp.active && (
                          <span className="text-[10px] font-medium text-dark-600 bg-dark-800/50 px-1.5 py-0.5 rounded-full shrink-0">
                            Inactive
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-3 mt-1 text-xs text-dark-500">
                        {opp.department && <span className="truncate max-w-48">{opp.department}</span>}
                        {opp.solicitation_number && <span>Sol: {opp.solicitation_number}</span>}
                        {opp.set_aside_description && <span>{opp.set_aside_description}</span>}
                      </div>
                      {opp.response_deadline && (
                        <p className={`text-xs mt-1 ${deadlineColor(opp.response_deadline)}`}>
                          Due: {formatDeadline(opp.response_deadline)}
                        </p>
                      )}

                      {/* Notes */}
                      {editingNotes === opp.id ? (
                        <textarea
                          value={notesText}
                          onChange={(e) => setNotesText(e.target.value)}
                          onBlur={() => handleNotesBlur(opp.id, notesText)}
                          className="mt-2 w-full bg-dark-800/50 border border-dark-700/50 rounded-lg px-3 py-2
                                     text-xs text-dark-300 placeholder:text-dark-600 resize-none focus:outline-none focus:border-accent/30"
                          rows={2}
                          placeholder="Add a note..."
                          autoFocus
                          maxLength={5000}
                        />
                      ) : opp.notes ? (
                        <p
                          onClick={() => startEditingNotes(opp)}
                          className="mt-2 text-xs text-dark-400 bg-dark-800/30 rounded-lg px-3 py-2 cursor-pointer
                                     hover:bg-dark-800/50 transition-colors line-clamp-2"
                        >
                          {opp.notes}
                        </p>
                      ) : (
                        <button
                          onClick={() => startEditingNotes(opp)}
                          className="mt-2 flex items-center gap-1 text-xs text-dark-600 hover:text-dark-400 transition-colors
                                     opacity-0 group-hover:opacity-100"
                        >
                          <StickyNote size={12} />
                          Add note
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
