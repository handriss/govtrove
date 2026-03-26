import { useState, useEffect, useCallback, useRef, type RefObject } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Search,
  Calendar,
  Star,
  MapPin,
  Clock,
  ChevronLeft,
  ChevronRight,
  Inbox,
  TrendingUp,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { getSearchHistoryTimeline, getSearchHistoryDay } from '../services/api';
import type { DayCount, HistoryTimelineResponse } from '../services/api';
import type { OpportunityListItem } from '../types/api';

// --- Helpers ---

const setAsideColors: Record<string, string> = {
  SBA: 'bg-blue-500/10 text-blue-400',
  SBP: 'bg-blue-500/10 text-blue-400',
  SB: 'bg-blue-500/10 text-blue-400',
  '8A': 'bg-violet-500/10 text-violet-400',
  '8AN': 'bg-violet-500/10 text-violet-400',
  SDVOSBC: 'bg-emerald-500/10 text-emerald-400',
  SDVOSBS: 'bg-emerald-500/10 text-emerald-400',
  WOSB: 'bg-pink-500/10 text-pink-400',
  EDWOSB: 'bg-pink-500/10 text-pink-400',
  HZC: 'bg-orange-500/10 text-orange-400',
  HZS: 'bg-orange-500/10 text-orange-400',
};

function formatDepartment(dept: string | undefined): string {
  if (!dept) return '';
  const parts = dept.split('.');
  return parts[parts.length - 1] || parts[0];
}

function getDaysUntil(deadline: string | undefined): number | null {
  if (!deadline) return null;
  const now = new Date();
  const todayStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  const today = new Date(todayStr + 'T00:00:00');
  const d = new Date(deadline.slice(0, 10) + 'T00:00:00');
  return Math.round((d.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
}

function formatDeadlineDate(deadline: string): string {
  const d = new Date(deadline.slice(0, 10) + 'T00:00:00');
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatDayLabel(dateStr: string): string {
  const today = new Date();
  const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);
  const yesterdayStr = `${yesterday.getFullYear()}-${String(yesterday.getMonth() + 1).padStart(2, '0')}-${String(yesterday.getDate()).padStart(2, '0')}`;

  if (dateStr === todayStr) return 'Today';
  if (dateStr === yesterdayStr) return 'Yesterday';
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
}

function formatFullDate(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' });
}

function filterSummary(filters: Record<string, unknown>): string {
  const parts: string[] = [];
  if (filters.keyword) parts.push(`"${filters.keyword}"`);
  if (Array.isArray(filters.naics) && filters.naics.length) parts.push(`NAICS: ${filters.naics.join(', ')}`);
  if (Array.isArray(filters.psc) && filters.psc.length) parts.push(`PSC: ${filters.psc.join(', ')}`);
  if (Array.isArray(filters.setAside) && filters.setAside.length) parts.push(`Set-Aside: ${filters.setAside.join(', ')}`);
  if (Array.isArray(filters.noticeType) && filters.noticeType.length) parts.push(`Type: ${filters.noticeType.join(', ')}`);
  if (filters.state) parts.push(`State: ${filters.state}`);
  return parts.join(' | ') || 'All opportunities';
}

// --- Components ---

const PAGE_SIZE = 25;

function TimelineDot({ count, isSelected, isEmpty }: { count: number; isSelected: boolean; isEmpty: boolean }) {
  if (isEmpty) {
    return (
      <div className={`w-3 h-3 rounded-full border-2 transition-colors duration-200 ${
        isSelected ? 'border-dark-400 bg-dark-900' : 'border-dark-700 bg-dark-900/50'
      }`} />
    );
  }
  const isLarge = count >= 10;
  return (
    <div className={`relative flex items-center justify-center rounded-full transition-all duration-200 ${
      isLarge ? 'w-8 h-7 rounded-lg' : 'w-7 h-7'
    } ${
      isSelected
        ? 'bg-accent text-white shadow-lg shadow-accent/30 scale-110'
        : 'bg-accent/15 text-accent hover:bg-accent/25'
    }`}>
      <span className={`font-bold ${isLarge ? 'text-[10px]' : 'text-[11px]'}`}>{count}</span>
    </div>
  );
}

function OpportunityRow({ opp }: { opp: OpportunityListItem }) {
  const agency = formatDepartment(opp.department);
  const days = getDaysUntil(opp.response_deadline);
  const deadlineColor = days === null ? 'text-dark-500'
    : days <= 2 ? 'text-red-400'
    : days <= 7 ? 'text-amber-400'
    : 'text-dark-300';

  return (
    <Link
      to={`/opportunity/${opp.id}`}
      className="group block rounded-xl border border-dark-800/50 bg-dark-900/30
        hover:border-dark-600/50 hover:shadow-lg hover:shadow-black/20
        transition-all duration-200"
    >
      <div className="px-4 py-3.5">
        <div className="flex items-start gap-2">
          <button
            type="button"
            onClick={(e) => { e.preventDefault(); e.stopPropagation(); }}
            className="shrink-0 p-1 -m-1 rounded-full hover:bg-yellow-400/10 transition-colors"
          >
            <Star size={18} className="text-dark-500 hover:text-yellow-400/70 transition-colors duration-150" />
          </button>
          <div className="flex-1 min-w-0">
            <h3 className="text-[15px] font-medium text-dark-100 leading-snug line-clamp-2 group-hover:text-accent transition-colors">
              {opp.title}
            </h3>
          </div>
        </div>

        <div className="ml-[34px] mt-1.5 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-sm text-dark-400">
          {agency && <span className="truncate max-w-[200px]">{agency}</span>}
          {agency && opp.naics_code && <span className="text-dark-600">&middot;</span>}
          {opp.naics_code && (
            <span className="bg-dark-700/50 text-dark-300 rounded-full px-2 py-0.5 text-xs font-mono">
              {opp.naics_code}
            </span>
          )}
          {opp.set_aside_code && (
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${setAsideColors[opp.set_aside_code] || 'bg-dark-700/50 text-dark-400'}`}>
              {opp.set_aside_description || opp.set_aside_code}
            </span>
          )}
        </div>

        <div className="ml-[34px] mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          {opp.solicitation_number && (
            <span className="text-dark-500 font-mono text-xs tracking-tight">
              Sol#: {opp.solicitation_number}
            </span>
          )}
          {opp.response_deadline && days !== null && (
            <span className={`text-xs font-medium tabular-nums ${deadlineColor}`}>
              {days < 0 ? 'Closed' : `Due: ${formatDeadlineDate(opp.response_deadline)} (${days}d)`}
            </span>
          )}
        </div>

        {opp.pop_state && (
          <div className="ml-[34px] mt-1.5 flex items-center gap-1 text-xs text-dark-500">
            <MapPin size={12} className="shrink-0" />
            <span>{opp.pop_state}</span>
          </div>
        )}
      </div>
    </Link>
  );
}

function EmptyDayCard() {
  return (
    <div className="rounded-xl border border-dashed border-dark-800/40 bg-dark-900/10 px-6 py-8 text-center">
      <Inbox size={24} className="mx-auto text-dark-600 mb-2" />
      <p className="text-sm text-dark-500">No new matches on this day</p>
    </div>
  );
}

function StatCard({ icon: Icon, label, value }: { icon: React.ElementType; label: string; value: string }) {
  return (
    <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl px-3 sm:px-4 py-2.5 sm:py-3">
      <div className="flex items-center gap-1.5 sm:gap-2 text-dark-500 text-[10px] sm:text-xs mb-0.5 sm:mb-1">
        <Icon size={12} strokeWidth={1.5} className="shrink-0 hidden sm:block" />
        {label}
      </div>
      <p className="text-base sm:text-lg font-semibold text-dark-100">{value}</p>
    </div>
  );
}

// --- Mobile date strip ---

function MobileDateStrip({ days, selectedIdx, onSelect }: {
  days: DayCount[];
  selectedIdx: number;
  onSelect: (idx: number) => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const selectedRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (selectedRef.current && scrollRef.current) {
      const container = scrollRef.current;
      const el = selectedRef.current;
      const left = el.offsetLeft - container.offsetWidth / 2 + el.offsetWidth / 2;
      container.scrollTo({ left, behavior: 'smooth' });
    }
  }, [selectedIdx]);

  return (
    <div className="md:hidden mb-4 -mx-4 sm:-mx-6">
      <div
        ref={scrollRef}
        className="flex gap-1.5 overflow-x-auto px-4 sm:px-6 pb-2 scrollbar-none"
        style={{ scrollbarWidth: 'none', WebkitOverflowScrolling: 'touch' }}
      >
        {days.map((day, idx) => {
          const isEmpty = day.count === 0;
          const isSelected = idx === selectedIdx;
          const d = new Date(day.date + 'T00:00:00');
          const weekday = d.toLocaleDateString('en-US', { weekday: 'short' });
          const dayNum = d.getDate();

          return (
            <button
              key={day.date}
              ref={isSelected ? selectedRef as RefObject<HTMLButtonElement> : undefined}
              onClick={() => onSelect(idx)}
              className={`flex flex-col items-center shrink-0 w-14 py-2 rounded-xl transition-all duration-200 ${
                isSelected
                  ? 'bg-accent text-white'
                  : isEmpty
                    ? 'bg-dark-900/30 text-dark-600'
                    : 'bg-dark-800/40 text-dark-300'
              }`}
            >
              <span className={`text-[10px] font-medium uppercase ${isSelected ? 'text-white/70' : ''}`}>
                {weekday}
              </span>
              <span className={`text-lg font-semibold leading-tight ${isSelected ? 'text-white' : ''}`}>
                {dayNum}
              </span>
              {!isEmpty && (
                <span className={`text-[10px] font-medium mt-0.5 ${
                  isSelected ? 'text-white/80' : 'text-accent'
                }`}>
                  {day.count}
                </span>
              )}
              {isEmpty && (
                <span className="text-[10px] mt-0.5">&mdash;</span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

// --- Shared day content ---

function DayContent({ selectedDay, dayOpps, dayTotal, dayLoading, dayLoadingMore, hasMore, showMore, selectedIdx, totalDays, onNavigate, showFullDate }: {
  selectedDay: DayCount | null;
  dayOpps: OpportunityListItem[];
  dayTotal: number;
  dayLoading: boolean;
  dayLoadingMore: boolean;
  hasMore: boolean;
  showMore: () => void;
  selectedIdx: number;
  totalDays: number;
  onNavigate: (idx: number) => void;
  showFullDate: boolean;
}) {
  if (!selectedDay) return null;

  return (
    <div className="flex-1 min-w-0">
      {/* Day header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2 text-dark-200 min-w-0">
          <Calendar size={16} strokeWidth={1.5} className="text-dark-400 shrink-0" />
          <h2 className="text-base sm:text-lg font-medium truncate">{formatDayLabel(selectedDay.date)}</h2>
          {showFullDate && (
            <span className="text-sm text-dark-500 hidden lg:inline">{formatFullDate(selectedDay.date)}</span>
          )}
        </div>
        {selectedDay.count > 0 && (
          <span className="bg-accent/10 text-accent text-xs font-medium rounded-full px-2.5 py-1 shrink-0">
            {selectedDay.count} new
          </span>
        )}
      </div>

      {/* Results */}
      {selectedDay.count === 0 ? (
        <EmptyDayCard />
      ) : dayLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 size={24} className="animate-spin text-accent/70" strokeWidth={1.5} />
        </div>
      ) : (
        <>
          {dayTotal > PAGE_SIZE && (
            <div className="mb-3 flex items-center gap-3">
              <div className="flex-1 h-1 rounded-full bg-dark-800/50 overflow-hidden">
                <div
                  className="h-full rounded-full bg-accent/40 transition-all duration-300"
                  style={{ width: `${Math.min(100, (dayOpps.length / dayTotal) * 100)}%` }}
                />
              </div>
              <span className="text-xs text-dark-500 tabular-nums shrink-0">
                {dayOpps.length} of {dayTotal}
              </span>
            </div>
          )}

          <div className="space-y-2">
            {dayOpps.map((opp) => (
              <OpportunityRow key={opp.id} opp={opp} />
            ))}
          </div>

          {hasMore && (
            <button
              onClick={showMore}
              disabled={dayLoadingMore}
              className="w-full mt-4 py-3 rounded-xl border border-dark-700/50 bg-dark-800/30
                text-sm font-medium text-dark-300 hover:text-dark-100 hover:border-dark-600/50
                hover:bg-dark-800/50 disabled:opacity-50 transition-all duration-200"
            >
              {dayLoadingMore ? (
                <Loader2 size={16} className="animate-spin mx-auto" />
              ) : (
                `Show more (${dayTotal - dayOpps.length} remaining)`
              )}
            </button>
          )}
        </>
      )}

      {/* Navigation hint */}
      <div className="mt-6 flex items-center justify-between text-xs text-dark-500">
        <button
          onClick={() => onNavigate(Math.min(selectedIdx + 1, totalDays - 1))}
          disabled={selectedIdx >= totalDays - 1}
          className="flex items-center gap-1 hover:text-dark-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronLeft size={12} />
          Older
        </button>
        <span>{selectedIdx + 1} of {totalDays} days</span>
        <button
          onClick={() => onNavigate(Math.max(selectedIdx - 1, 0))}
          disabled={selectedIdx <= 0}
          className="flex items-center gap-1 hover:text-dark-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          Newer
          <ChevronRight size={12} />
        </button>
      </div>
    </div>
  );
}

// --- Page ---

export default function SearchHistoryPage() {
  const { searchId } = useParams();
  const { isAuthenticated, getAccessToken } = useAppAuth();

  const [timeline, setTimeline] = useState<HistoryTimelineResponse | null>(null);
  const [timelineLoading, setTimelineLoading] = useState(true);
  const [timelineError, setTimelineError] = useState<string | null>(null);

  const [selectedIdx, setSelectedIdx] = useState(0);
  const [dayOpps, setDayOpps] = useState<OpportunityListItem[]>([]);
  const [dayTotal, setDayTotal] = useState(0);
  const [dayPage, setDayPage] = useState(1);
  const [dayLoading, setDayLoading] = useState(false);
  const [dayLoadingMore, setDayLoadingMore] = useState(false);

  const dayCache = useRef<Map<string, { opps: OpportunityListItem[]; total: number }>>(new Map());

  const selectedDay: DayCount | null = timeline?.days[selectedIdx] ?? null;

  // Fetch timeline on mount
  useEffect(() => {
    if (!isAuthenticated || !searchId) return;
    let cancelled = false;

    (async () => {
      setTimelineLoading(true);
      setTimelineError(null);
      try {
        const token = await getAccessToken();
        const data = await getSearchHistoryTimeline(token, parseInt(searchId), 7);
        if (!cancelled) {
          setTimeline(data);
          setSelectedIdx(0);
        }
      } catch (err) {
        if (!cancelled) setTimelineError(err instanceof Error ? err.message : 'Failed to load');
      } finally {
        if (!cancelled) setTimelineLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [isAuthenticated, searchId, getAccessToken]);

  // Fetch day detail when selected day changes
  const fetchDay = useCallback(async (date: string, page: number, append: boolean) => {
    if (!searchId) return;

    if (page === 1 && !append) {
      const cached = dayCache.current.get(date);
      if (cached) {
        setDayOpps(cached.opps);
        setDayTotal(cached.total);
        setDayPage(Math.ceil(cached.opps.length / PAGE_SIZE));
        return;
      }
    }

    if (append) {
      setDayLoadingMore(true);
    } else {
      setDayLoading(true);
      setDayOpps([]);
    }

    try {
      const token = await getAccessToken();
      const result = await getSearchHistoryDay(token, parseInt(searchId), date, page, PAGE_SIZE);
      if (append) {
        setDayOpps(prev => {
          const combined = [...prev, ...(result.opportunities || [])];
          dayCache.current.set(date, { opps: combined, total: result.total });
          return combined;
        });
      } else {
        const opps = result.opportunities || [];
        setDayOpps(opps);
        dayCache.current.set(date, { opps, total: result.total });
      }
      setDayTotal(result.total);
      setDayPage(page);
    } finally {
      setDayLoading(false);
      setDayLoadingMore(false);
    }
  }, [searchId, getAccessToken]);

  useEffect(() => {
    if (!selectedDay || selectedDay.count === 0) {
      setDayOpps([]);
      setDayTotal(0);
      return;
    }
    fetchDay(selectedDay.date, 1, false);
  }, [selectedDay, fetchDay]);

  const hasMore = dayOpps.length < dayTotal;

  function showMore() {
    if (!selectedDay || dayLoadingMore) return;
    fetchDay(selectedDay.date, dayPage + 1, true);
  }

  // --- Loading / Error states ---

  if (timelineLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-accent/70" strokeWidth={1.5} />
          <span className="text-sm text-dark-400">Loading search history...</span>
        </div>
      </div>
    );
  }

  if (timelineError || !timeline) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 rounded-2xl bg-dark-800/50 border border-dark-700/30 flex items-center justify-center mx-auto mb-6">
            <AlertCircle size={28} className="text-dark-500" strokeWidth={1.5} />
          </div>
          <h1 className="text-lg font-medium text-dark-200 mb-2">{timelineError || 'Search not found'}</h1>
          <Link
            to="/saved"
            className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent hover:bg-accent-hover text-white font-medium rounded-xl transition-colors duration-200"
          >
            Back to Saved Searches
          </Link>
        </div>
      </div>
    );
  }

  const days = timeline.days;
  const searchFilters = typeof timeline.search.filters === 'string'
    ? JSON.parse(timeline.search.filters)
    : timeline.search.filters;

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-5xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
        {/* Back link */}
        <Link
          to="/saved"
          className="inline-flex items-center gap-1.5 text-sm text-dark-400 hover:text-dark-200 transition-colors mb-4 sm:mb-6"
        >
          <ArrowLeft size={14} strokeWidth={1.5} />
          Back to saved searches
        </Link>

        {/* Header */}
        <div className="mb-6 sm:mb-8">
          <div className="flex items-start gap-3 mb-2">
            <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-xl bg-accent/10 border border-accent/20 flex items-center justify-center shrink-0">
              <Search size={18} className="text-accent sm:hidden" strokeWidth={1.5} />
              <Search size={20} className="text-accent hidden sm:block" strokeWidth={1.5} />
            </div>
            <div className="min-w-0">
              <h1 className="text-lg sm:text-xl font-semibold text-dark-50 leading-snug">{timeline.search.name}</h1>
              <p className="text-xs sm:text-sm text-dark-400 mt-0.5 line-clamp-2">{filterSummary(searchFilters)}</p>
            </div>
          </div>
        </div>

        {/* Stats bar */}
        <div className="grid grid-cols-3 gap-2 sm:gap-3 mb-6 sm:mb-8">
          <StatCard icon={TrendingUp} label="New this week" value={String(timeline.total_new)} />
          <StatCard icon={Calendar} label="Days with matches" value={`${timeline.days_with_matches} of ${days.length}`} />
          <StatCard icon={Clock} label="Avg per day" value={days.length > 0 ? (timeline.total_new / days.length).toFixed(1) : '0'} />
        </div>

        {/* Mobile: Horizontal date strip */}
        <MobileDateStrip
          days={days}
          selectedIdx={selectedIdx}
          onSelect={setSelectedIdx}
        />

        {/* Desktop: Timeline sidebar + Content */}
        <div className="hidden md:flex gap-6">
          {/* Timeline sidebar */}
          <div className="w-48 shrink-0">
            <div className="sticky top-8">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xs font-semibold text-dark-400 uppercase tracking-wider">Timeline</h2>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setSelectedIdx(Math.min(selectedIdx + 1, days.length - 1))}
                    disabled={selectedIdx >= days.length - 1}
                    className="p-1 rounded text-dark-500 hover:text-dark-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                  >
                    <ChevronLeft size={14} />
                  </button>
                  <button
                    onClick={() => setSelectedIdx(Math.max(selectedIdx - 1, 0))}
                    disabled={selectedIdx <= 0}
                    className="p-1 rounded text-dark-500 hover:text-dark-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                  >
                    <ChevronRight size={14} />
                  </button>
                </div>
              </div>

              <div className="relative">
                <div className="absolute left-[13px] top-3 bottom-3 w-px bg-dark-800/60" />

                <div className="space-y-1">
                  {days.map((day, idx) => {
                    const isEmpty = day.count === 0;
                    const isSelected = idx === selectedIdx;
                    return (
                      <button
                        key={day.date}
                        onClick={() => setSelectedIdx(idx)}
                        className={`w-full flex items-center gap-3 px-2 py-2 rounded-lg text-left transition-all duration-200 ${
                          isSelected ? 'bg-dark-800/50' : 'hover:bg-dark-800/30'
                        }`}
                      >
                        <TimelineDot count={day.count} isSelected={isSelected} isEmpty={isEmpty} />
                        <div className="flex-1 min-w-0">
                          <p className={`text-sm font-medium truncate ${
                            isSelected ? 'text-dark-100' : isEmpty ? 'text-dark-500' : 'text-dark-300'
                          }`}>
                            {formatDayLabel(day.date)}
                          </p>
                          <p className={`text-xs ${isEmpty ? 'text-dark-600' : 'text-dark-400'}`}>
                            {isEmpty
                              ? 'No matches'
                              : `${day.count} new match${day.count !== 1 ? 'es' : ''}`}
                          </p>
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>

          {/* Desktop main content */}
          <DayContent
            selectedDay={selectedDay}
            dayOpps={dayOpps}
            dayTotal={dayTotal}
            dayLoading={dayLoading}
            dayLoadingMore={dayLoadingMore}
            hasMore={hasMore}
            showMore={showMore}
            selectedIdx={selectedIdx}
            totalDays={days.length}
            onNavigate={setSelectedIdx}
            showFullDate
          />
        </div>

        {/* Mobile: Content only (no sidebar) */}
        <div className="md:hidden">
          <DayContent
            selectedDay={selectedDay}
            dayOpps={dayOpps}
            dayTotal={dayTotal}
            dayLoading={dayLoading}
            dayLoadingMore={dayLoadingMore}
            hasMore={hasMore}
            showMore={showMore}
            selectedIdx={selectedIdx}
            totalDays={days.length}
            onNavigate={setSelectedIdx}
            showFullDate={false}
          />
        </div>
      </div>
    </div>
  );
}
