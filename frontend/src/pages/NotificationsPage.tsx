import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Search, FileText, ChevronDown, ChevronRight, Bell, Zap, CheckCheck, Loader2, Trash2 } from 'lucide-react';
import type { OpportunityListItem, Notification } from '../types/api';
import { useNotifications } from '../hooks/useNotifications';

// --- Local utils (duplicated from OpportunityCard to avoid coupling) ---

const setAsideColors: Record<string, string> = {
  'SBA': 'bg-blue-500/10 text-blue-400',
  'SBP': 'bg-blue-500/10 text-blue-400',
  'SB': 'bg-blue-500/10 text-blue-400',
  '8A': 'bg-violet-500/10 text-violet-400',
  '8AN': 'bg-violet-500/10 text-violet-400',
  'SDVOSBC': 'bg-emerald-500/10 text-emerald-400',
  'SDVOSBS': 'bg-emerald-500/10 text-emerald-400',
  'WOSB': 'bg-pink-500/10 text-pink-400',
  'WOSBSS': 'bg-pink-500/10 text-pink-400',
  'EDWOSB': 'bg-pink-500/10 text-pink-400',
  'EDWOSBSS': 'bg-pink-500/10 text-pink-400',
  'HZC': 'bg-orange-500/10 text-orange-400',
  'HZS': 'bg-orange-500/10 text-orange-400',
  'VSA': 'bg-teal-500/10 text-teal-400',
  'VSS': 'bg-teal-500/10 text-teal-400',
};

function getDaysUntilDeadline(deadline: string | undefined): number | null {
  if (!deadline) return null;
  const now = new Date();
  const todayStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  const today = new Date(todayStr + 'T00:00:00');
  const deadlineDate = new Date(deadline.slice(0, 10) + 'T00:00:00');
  return Math.round((deadlineDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
}

function getHoursAgo(date: string | undefined): number | null {
  if (!date) return null;
  return (Date.now() - new Date(date).getTime()) / (1000 * 60 * 60);
}

function formatDeadlineDate(deadline: string | undefined): string {
  if (!deadline) return '';
  const d = new Date(deadline.slice(0, 10) + 'T00:00:00');
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatDepartment(dept: string | undefined): string {
  if (!dept) return '';
  const parts = dept.split('.');
  return parts[parts.length - 1] || parts[0];
}

type UrgencyBadge = { label: string; className: string } | null;

function getUrgencyBadge(postedDate: string | undefined, deadline: string | undefined): UrgencyBadge {
  const days = getDaysUntilDeadline(deadline);
  if (days !== null && days >= 0 && days <= 2) {
    return { label: 'Closing Today', className: 'bg-red-500/15 text-red-400' };
  }
  if (days !== null && days >= 0 && days <= 7) {
    return { label: 'Closing Soon', className: 'bg-amber-500/15 text-amber-400' };
  }
  const hoursAgo = getHoursAgo(postedDate);
  if (hoursAgo !== null && hoursAgo <= 48) {
    return { label: 'New', className: 'bg-emerald-500/15 text-emerald-400' };
  }
  return null;
}

function getDeadlineColor(days: number | null): string {
  if (days === null) return 'text-dark-500';
  if (days < 0) return 'text-dark-500';
  if (days <= 2) return 'text-red-400';
  if (days <= 7) return 'text-amber-400';
  return 'text-dark-300';
}

// --- Date helpers ---

function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const yesterday = new Date(today.getTime() - 86400000);
  const updateDate = new Date(d.getFullYear(), d.getMonth(), d.getDate());

  if (updateDate.getTime() === today.getTime()) return 'Today';
  if (updateDate.getTime() === yesterday.getTime()) return 'Yesterday';
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatTimeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

// --- Notification types ---

interface FieldDiff {
  field: string;
  old?: string;
  new: string;
}

interface SearchMatchNotification {
  id: string;
  type: 'search_matches';
  searchName: string;
  searchUrl: string;
  matches: OpportunityListItem[];
  totalMatches: number;
  isRead: boolean;
  createdAt: string;
}

interface OpportunityUpdateNotification {
  id: string;
  type: 'opportunity_update';
  opportunityId: number;
  title: string;
  solicitationNumber: string;
  department: string;
  changeType: 'amendment' | 'field_change';
  changes: FieldDiff[];
  isRead: boolean;
  createdAt: string;
}

type NotificationItem = SearchMatchNotification | OpportunityUpdateNotification;

// --- Map API notification to local types ---

function mapNotification(n: Notification): NotificationItem | null {
  const d = n.details;
  if (n.update_type === 'search_matches') {
    const matches = (d.matches as OpportunityListItem[] | undefined) || [];
    return {
      id: n.id,
      type: 'search_matches',
      searchName: (d.search_name as string) || 'Search',
      searchUrl: (d.search_url as string) || '/',
      matches,
      totalMatches: (d.total_matches as number) || matches.length,
      isRead: n.is_read,
      createdAt: n.created_at,
    };
  }
  if (n.update_type === 'opportunity_update') {
    const changes = ((d.changes as FieldDiff[]) || []).map((c) => ({
      field: c.field,
      old: c.old,
      new: c.new,
    }));
    return {
      id: n.id,
      type: 'opportunity_update',
      opportunityId: n.source_id || 0,
      title: (d.title as string) || '',
      solicitationNumber: (d.solicitation_number as string) || '',
      department: (d.department as string) || '',
      changeType: (d.change_type as 'amendment' | 'field_change') || 'field_change',
      changes,
      isRead: n.is_read,
      createdAt: n.created_at,
    };
  }
  return null;
}

// --- Mini opportunity card ---

function MiniOpportunityCard({ opp }: { opp: OpportunityListItem }) {
  const badge = getUrgencyBadge(opp.posted_date, opp.response_deadline);
  const days = getDaysUntilDeadline(opp.response_deadline);
  const deadlineColor = getDeadlineColor(days);
  const agency = formatDepartment(opp.department);

  const deadlineDisplay = days !== null
    ? days < 0
      ? 'Closed'
      : `Due: ${formatDeadlineDate(opp.response_deadline)} (${days}d)`
    : null;

  return (
    <Link
      to={`/opportunity/${opp.id}`}
      className="block px-3 py-2.5 rounded-lg border border-dark-800/40 bg-dark-900/40
        hover:border-dark-600/50 hover:bg-dark-800/30 transition-all"
    >
      <div className="flex items-start justify-between gap-2">
        <h4 className="text-sm font-medium text-dark-200 leading-snug line-clamp-1">
          {opp.title}
        </h4>
        {badge && (
          <span className={`shrink-0 text-[10px] font-medium rounded-full px-1.5 py-0.5 whitespace-nowrap ${badge.className}`}>
            {badge.label}
          </span>
        )}
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-dark-400">
        {agency && <span className="truncate max-w-[180px]">{agency}</span>}
        {agency && opp.naics_code && <span className="text-dark-600">&middot;</span>}
        {opp.naics_code && (
          <span className="bg-dark-700/50 text-dark-300 rounded-full px-1.5 py-0.5 font-mono text-[11px]">
            {opp.naics_code}
          </span>
        )}
        {opp.set_aside_code && (
          <span className={`rounded-full px-1.5 py-0.5 text-[11px] font-medium ${setAsideColors[opp.set_aside_code] || 'bg-dark-700/50 text-dark-400'}`}>
            {opp.set_aside_code}
          </span>
        )}
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-x-3 text-xs">
        {opp.solicitation_number && (
          <span className="text-dark-500 font-mono text-[11px] tracking-tight">
            Sol#: {opp.solicitation_number}
          </span>
        )}
        {deadlineDisplay && (
          <span className={`text-[11px] font-medium tabular-nums ${deadlineColor}`}>
            {deadlineDisplay}
          </span>
        )}
      </div>
    </Link>
  );
}

// --- Search match group card ---

function SearchMatchCard({ notification, onMarkRead, onDelete }: {
  notification: SearchMatchNotification;
  onMarkRead: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(true);
  const displayedMatches = notification.matches.slice(0, 3);
  const remaining = notification.totalMatches - 3;

  return (
    <div
      className={`rounded-xl border transition-all ${
        notification.isRead
          ? 'border-dark-800/30 bg-dark-900/20'
          : 'border-accent/20 bg-accent/5 border-l-2 border-l-accent/50'
      }`}
    >
      {/* Header */}
      <div className="px-4 py-3 flex items-start gap-3">
        <div className="mt-0.5 shrink-0 w-7 h-7 rounded-lg bg-accent/10 flex items-center justify-center">
          <Search size={14} className="text-accent" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <button
              onClick={() => setExpanded(!expanded)}
              className="flex items-center gap-1.5 text-sm text-dark-200 hover:text-dark-100 transition-colors"
            >
              {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              <span className="font-medium">"{notification.searchName}"</span>
              <span className="text-dark-400">
                — {notification.totalMatches} new match{notification.totalMatches !== 1 ? 'es' : ''}
              </span>
            </button>
            <div className="flex items-center gap-1.5 shrink-0">
              <span className="text-[11px] text-dark-600">{formatTimeAgo(notification.createdAt)}</span>
              {!notification.isRead && (
                <button
                  onClick={() => onMarkRead(notification.id)}
                  className="p-1 text-dark-600 hover:text-dark-300 transition-colors"
                  title="Mark as read"
                >
                  <CheckCheck size={13} />
                </button>
              )}
              <button
                onClick={() => onDelete(notification.id)}
                className="p-1 text-dark-600 hover:text-red-400 transition-colors"
                title="Dismiss"
              >
                <Trash2 size={13} />
              </button>
            </div>
          </div>
          <div className="mt-1">
            <Link
              to={notification.searchUrl}
              className="text-xs text-accent/70 hover:text-accent transition-colors"
            >
              View all results &rarr;
            </Link>
          </div>
        </div>
      </div>

      {/* Expanded opportunity list */}
      {expanded && (
        <div className="px-4 pb-3 space-y-1.5">
          {displayedMatches.map((opp) => (
            <MiniOpportunityCard key={opp.id} opp={opp} />
          ))}
          {remaining > 0 && (
            <Link
              to={notification.searchUrl}
              className="block text-center text-xs text-dark-500 hover:text-dark-300
                py-1.5 transition-colors"
            >
              + {remaining} more match{remaining !== 1 ? 'es' : ''}
            </Link>
          )}
        </div>
      )}
    </div>
  );
}

// --- Opportunity update card ---

function OpportunityUpdateCard({ notification, onMarkRead, onDelete }: {
  notification: OpportunityUpdateNotification;
  onMarkRead: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const agency = formatDepartment(notification.department);

  return (
    <div
      className={`rounded-xl border transition-all ${
        notification.isRead
          ? 'border-dark-800/30 bg-dark-900/20'
          : 'border-blue-400/20 bg-blue-400/5 border-l-2 border-l-blue-400/50'
      }`}
    >
      <div className="px-4 py-3">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 shrink-0 w-7 h-7 rounded-lg bg-blue-400/10 flex items-center justify-center">
            <FileText size={14} className="text-blue-400" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <h4 className="text-sm font-medium text-dark-200 leading-snug line-clamp-1">
                {notification.title}
              </h4>
              <div className="flex items-center gap-1.5 shrink-0">
                <span className="text-[11px] text-dark-600">{formatTimeAgo(notification.createdAt)}</span>
                {!notification.isRead && (
                  <button
                    onClick={() => onMarkRead(notification.id)}
                    className="p-1 text-dark-600 hover:text-dark-300 transition-colors"
                    title="Mark as read"
                  >
                    <CheckCheck size={13} />
                  </button>
                )}
                <button
                  onClick={() => onDelete(notification.id)}
                  className="p-1 text-dark-600 hover:text-red-400 transition-colors"
                  title="Dismiss"
                >
                  <Trash2 size={13} />
                </button>
              </div>
            </div>
            <p className="text-xs text-dark-500 mt-0.5">
              Sol#: {notification.solicitationNumber} &middot; {agency}
            </p>

            {/* Change type label */}
            <div className="flex items-center gap-1.5 mt-2 text-xs font-medium text-amber-400">
              <Zap size={12} />
              {notification.changeType === 'amendment' ? 'Amendment posted' : 'Fields updated'}
            </div>

            {/* Inline diff */}
            {notification.changes.length > 0 && (
              <div className="mt-2 rounded-lg border border-dark-800/40 bg-dark-950/50 overflow-hidden">
                {notification.changes.map((change, i) => (
                  <div
                    key={i}
                    className={`px-3 py-1.5 flex items-baseline gap-2 text-xs ${
                      i > 0 ? 'border-t border-dark-800/30' : ''
                    }`}
                  >
                    <span className="text-dark-500 shrink-0 w-28 font-medium">{change.field}:</span>
                    <div>
                      {change.old && (
                        <span className="text-red-400/70 line-through mr-2">{change.old}</span>
                      )}
                      {change.old && <span className="text-dark-600 mr-2">&rarr;</span>}
                      <span className="text-green-400/80">{change.new}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Link */}
            <div className="mt-2">
              <Link
                to={`/opportunity/${notification.opportunityId}`}
                className="text-xs text-accent/70 hover:text-accent transition-colors"
              >
                View opportunity &rarr;
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// --- Timeline with quiet-day gaps ---

type TimelineEntry =
  | { kind: 'date'; label: string; items: NotificationItem[] }
  | { kind: 'quiet'; label: string };

function toDateKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function buildTimeline(notifications: NotificationItem[]): TimelineEntry[] {
  if (notifications.length === 0) return [];

  const byDate = new Map<string, NotificationItem[]>();
  for (const n of notifications) {
    const d = new Date(n.createdAt);
    const key = toDateKey(d);
    const existing = byDate.get(key) || [];
    existing.push(n);
    byDate.set(key, existing);
  }

  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const dates = [...byDate.keys()].sort().reverse();
  const oldest = new Date(dates[dates.length - 1] + 'T00:00:00');

  const timeline: TimelineEntry[] = [];
  let quietStart: Date | null = null;
  let quietEnd: Date | null = null;

  const cursor = new Date(today);
  while (cursor >= oldest) {
    const key = toDateKey(cursor);
    const items = byDate.get(key);

    if (items) {
      if (quietStart && quietEnd) {
        timeline.push({ kind: 'quiet', label: formatQuietRange(quietStart, quietEnd) });
        quietStart = null;
        quietEnd = null;
      }
      timeline.push({ kind: 'date', label: formatDateLabel(items[0].createdAt), items });
    } else {
      if (!quietStart) {
        quietStart = new Date(cursor);
        quietEnd = new Date(cursor);
      } else {
        quietEnd = new Date(cursor);
      }
    }

    cursor.setDate(cursor.getDate() - 1);
  }

  if (quietStart && quietEnd) {
    timeline.push({ kind: 'quiet', label: formatQuietRange(quietStart, quietEnd) });
  }

  return timeline;
}

function formatQuietRange(start: Date, end: Date): string {
  const fmt = (d: Date) => formatDateLabel(d.toISOString());
  if (start.getTime() === end.getTime()) return fmt(start);
  return `${fmt(end)} – ${fmt(start)}`;
}

// --- Empty state ---

function EmptyState() {
  return (
    <div className="flex flex-col items-center text-center py-16">
      <div className="w-16 h-16 rounded-2xl bg-dark-800/30 border border-dark-800/50 flex items-center justify-center mb-6">
        <Bell size={28} className="text-dark-700" strokeWidth={1.5} />
      </div>
      <h2 className="text-lg font-medium text-dark-300 mb-2">You're all caught up</h2>
      <p className="text-sm text-dark-500 max-w-md">
        Save searches or star opportunities to get updates here.
      </p>
      <Link
        to="/"
        className="mt-4 text-sm text-accent hover:text-accent/80 transition-colors"
      >
        Start searching
      </Link>
    </div>
  );
}

// --- Collapsible section with show-more ---

const INITIAL_VISIBLE = 3;

function CollapsibleSection<T extends NotificationItem>({
  items,
  icon,
  label,
  iconColor,
  labelColor,
  renderItem,
}: {
  items: T[];
  icon: React.ReactNode;
  label: string;
  iconColor: string;
  labelColor: string;
  renderItem: (item: T) => React.ReactNode;
}) {
  const [expanded, setExpanded] = useState(false);
  const visible = expanded ? items : items.slice(0, INITIAL_VISIBLE);
  const hiddenCount = items.length - INITIAL_VISIBLE;

  return (
    <div className="mb-4">
      <div className="flex items-center gap-2 mb-2 px-1">
        <span className={iconColor}>{icon}</span>
        <p className={`text-[11px] font-medium uppercase tracking-wider ${labelColor}`}>
          {label}
        </p>
        <span className={`text-[11px] ${labelColor}`}>({items.length})</span>
      </div>
      <div className="space-y-3">
        {visible.map(renderItem)}
      </div>
      {hiddenCount > 0 && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full mt-2 py-2 text-xs text-dark-500 hover:text-dark-300
            border border-dark-800/40 rounded-lg hover:border-dark-700/50 transition-colors"
        >
          {expanded ? 'Show less' : `+ ${hiddenCount} more`}
        </button>
      )}
    </div>
  );
}

// --- Page ---

export default function NotificationsPage() {
  const { notifications: rawNotifications, loading, hasMore, markRead, markAllRead, remove, loadMore } = useNotifications();

  const items: NotificationItem[] = rawNotifications
    .map(mapNotification)
    .filter((n): n is NotificationItem => n !== null);

  const unreadCount = items.filter((n) => !n.isRead).length;
  const timeline = buildTimeline(items);

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-3xl mx-auto px-6 pt-10 pb-10">
        <div className="flex items-center gap-4 mb-8">
          <h1 className="text-2xl font-semibold text-dark-100">Updates</h1>
          {unreadCount > 0 && (
            <span className="text-xs font-medium text-accent bg-accent/10 px-2 py-0.5 rounded-full">
              {unreadCount} unread
            </span>
          )}
          <div className="flex-1" />
          {unreadCount > 0 && (
            <button
              onClick={() => { markAllRead(); }}
              className="text-xs text-dark-500 hover:text-dark-300 transition-colors flex items-center gap-1"
            >
              <CheckCheck size={14} />
              Mark all read
            </button>
          )}
        </div>

        {loading && items.length === 0 ? (
          <div className="flex justify-center py-16">
            <Loader2 size={24} className="text-dark-500 animate-spin" />
          </div>
        ) : items.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="space-y-8">
            {timeline.map((entry) => {
              if (entry.kind === 'quiet') {
                return (
                  <div key={`quiet-${entry.label}`} className="flex items-center gap-3 px-1">
                    <div className="flex-1 border-t border-dark-800/30" />
                    <p className="text-[11px] text-dark-600 italic whitespace-nowrap">
                      No updates &middot; {entry.label}
                    </p>
                    <div className="flex-1 border-t border-dark-800/30" />
                  </div>
                );
              }

              const searches = entry.items.filter((n): n is SearchMatchNotification => n.type === 'search_matches');
              const updates = entry.items.filter((n): n is OpportunityUpdateNotification => n.type === 'opportunity_update');

              return (
                <div key={entry.label}>
                  <p className="text-xs text-dark-600 uppercase tracking-wider mb-4 px-1">{entry.label}</p>

                  {searches.length > 0 && (
                    <CollapsibleSection
                      items={searches}
                      icon={<Search size={12} />}
                      label="New search matches"
                      iconColor="text-accent/60"
                      labelColor="text-accent/60"
                      renderItem={(n) => <SearchMatchCard key={n.id} notification={n} onMarkRead={markRead} onDelete={remove} />}
                    />
                  )}

                  {updates.length > 0 && (
                    <CollapsibleSection
                      items={updates}
                      icon={<FileText size={12} />}
                      label="Opportunity changes"
                      iconColor="text-blue-400/60"
                      labelColor="text-blue-400/60"
                      renderItem={(n) => <OpportunityUpdateCard key={n.id} notification={n} onMarkRead={markRead} onDelete={remove} />}
                    />
                  )}
                </div>
              );
            })}

            {hasMore && (
              <div className="flex justify-center">
                <button
                  onClick={loadMore}
                  disabled={loading}
                  className="text-sm text-dark-400 hover:text-dark-200 transition-colors
                    border border-dark-800/40 rounded-lg px-4 py-2 hover:border-dark-700/50
                    disabled:opacity-50"
                >
                  {loading ? <Loader2 size={16} className="animate-spin" /> : 'Load more'}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
