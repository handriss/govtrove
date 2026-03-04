import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft, Search, FileText, ChevronDown, ChevronRight, Bell, Zap } from 'lucide-react';
import type { OpportunityListItem } from '../types/api';
import AuthButton from '../components/AuthButton';

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

// --- Mock data types ---

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

// --- Relative dates for mock data ---

function hoursAgo(h: number): string {
  return new Date(Date.now() - h * 3600000).toISOString();
}

// --- Mock data ---

const MOCK_NOTIFICATIONS: NotificationItem[] = [
  // Today — search with 3 matches
  {
    id: 'n1',
    type: 'search_matches',
    searchName: 'Logistics',
    searchUrl: '/?q=Logistics',
    totalMatches: 3,
    isRead: false,
    createdAt: hoursAgo(2),
    matches: [
      {
        id: 4501, notice_id: 'opp-4501', title: 'Logistics Support Services — Fort Liberty',
        solicitation_number: 'W912345-26-R-0001', department: 'DEPT OF DEFENSE.DEPT OF THE ARMY',
        naics_code: '541614', set_aside_code: 'SB', set_aside_description: 'Small Business',
        posted_date: hoursAgo(36), response_deadline: '2026-04-15', active: true,
      },
      {
        id: 4502, notice_id: 'opp-4502', title: 'IT Infrastructure Modernization and Network Support',
        solicitation_number: 'FA8750-26-R-0002', department: 'DEPT OF DEFENSE.DEPT OF THE AIR FORCE',
        naics_code: '541512', posted_date: hoursAgo(6), response_deadline: '2026-05-01', active: true,
      },
      {
        id: 4503, notice_id: 'opp-4503', title: 'Supply Chain Management — DLA Distribution',
        solicitation_number: 'SP4500-26-R-0015', department: 'DEPT OF DEFENSE.DEFENSE LOGISTICS AGENCY',
        naics_code: '493110', set_aside_code: 'SDVOSBC', set_aside_description: 'SDVOSB',
        posted_date: hoursAgo(48), response_deadline: '2026-03-11', active: true,
      },
    ],
  },
  // Today — search with 8 matches
  {
    id: 'n2',
    type: 'search_matches',
    searchName: 'IT Services',
    searchUrl: '/?q=IT%20Services',
    totalMatches: 8,
    isRead: false,
    createdAt: hoursAgo(4),
    matches: [
      {
        id: 4510, notice_id: 'opp-4510', title: 'Enterprise Cloud Migration Services',
        solicitation_number: 'HC1028-26-R-0010', department: 'DEPT OF DEFENSE.DEFENSE INFORMATION SYSTEMS AGENCY',
        naics_code: '541519', set_aside_code: '8A', set_aside_description: '8(a)',
        posted_date: hoursAgo(12), response_deadline: '2026-04-20', active: true,
      },
      {
        id: 4511, notice_id: 'opp-4511', title: 'Cybersecurity Operations and Monitoring',
        solicitation_number: 'N00189-26-R-0044', department: 'DEPT OF DEFENSE.DEPT OF THE NAVY',
        naics_code: '541512', posted_date: hoursAgo(24), response_deadline: '2026-03-09', active: true,
      },
      {
        id: 4512, notice_id: 'opp-4512', title: 'Help Desk and End User Support',
        solicitation_number: 'GS-35F-0119Y', department: 'GENERAL SERVICES ADMINISTRATION',
        naics_code: '541513', set_aside_code: 'SB', set_aside_description: 'Small Business',
        posted_date: hoursAgo(30), response_deadline: '2026-05-15', active: true,
      },
    ],
  },
  // Today — 2 more searches to test "show more" cutoff
  {
    id: 'n2b',
    type: 'search_matches',
    searchName: 'Medical supplies',
    searchUrl: '/?q=Medical%20supplies',
    totalMatches: 2,
    isRead: false,
    createdAt: hoursAgo(5),
    matches: [
      {
        id: 4530, notice_id: 'opp-4530', title: 'Medical Surgical Instruments and Supplies',
        solicitation_number: 'VA248-26-R-0012', department: 'VETERANS AFFAIRS, DEPARTMENT OF',
        naics_code: '339112', set_aside_code: 'SDVOSBC', set_aside_description: 'SDVOSB',
        posted_date: hoursAgo(18), response_deadline: '2026-04-10', active: true,
      },
      {
        id: 4531, notice_id: 'opp-4531', title: 'Pharmaceutical Distribution Services',
        solicitation_number: 'VA258-26-R-0045', department: 'VETERANS AFFAIRS, DEPARTMENT OF',
        naics_code: '424210', posted_date: hoursAgo(20), response_deadline: '2026-05-20', active: true,
      },
    ],
  },
  {
    id: 'n2c',
    type: 'search_matches',
    searchName: 'Janitorial',
    searchUrl: '/?q=Janitorial',
    totalMatches: 1,
    isRead: false,
    createdAt: hoursAgo(6),
    matches: [
      {
        id: 4540, notice_id: 'opp-4540', title: 'Custodial and Janitorial Services — Pentagon',
        solicitation_number: 'HQ0034-26-R-0003', department: 'DEPT OF DEFENSE.WASHINGTON HEADQUARTERS SERVICES',
        naics_code: '561720', set_aside_code: '8A', set_aside_description: '8(a)',
        posted_date: hoursAgo(10), response_deadline: '2026-04-25', active: true,
      },
    ],
  },
  // Today — opportunity updates (4 total to test "show more")
  {
    id: 'n3',
    type: 'opportunity_update',
    opportunityId: 4400,
    title: 'Base Operations Support — Fort Eisenhower',
    solicitationNumber: 'W50S8J-26-R-0005',
    department: 'DEPT OF DEFENSE.DEPT OF THE ARMY',
    changeType: 'amendment',
    changes: [],
    isRead: false,
    createdAt: hoursAgo(3),
  },
  {
    id: 'n3b',
    type: 'opportunity_update',
    opportunityId: 4401,
    title: 'Facility Maintenance — Joint Base Andrews',
    solicitationNumber: 'FA7014-26-R-0021',
    department: 'DEPT OF DEFENSE.DEPT OF THE AIR FORCE',
    changeType: 'field_change',
    changes: [
      { field: 'Response Date', old: 'Mar 15, 2026', new: 'Apr 15, 2026' },
    ],
    isRead: false,
    createdAt: hoursAgo(4),
  },
  {
    id: 'n3c',
    type: 'opportunity_update',
    opportunityId: 4402,
    title: 'HVAC System Replacement — Camp Pendleton',
    solicitationNumber: 'N62473-26-R-0088',
    department: 'DEPT OF DEFENSE.DEPT OF THE NAVY',
    changeType: 'amendment',
    changes: [],
    isRead: false,
    createdAt: hoursAgo(5),
  },
  {
    id: 'n3d',
    type: 'opportunity_update',
    opportunityId: 4403,
    title: 'Grounds Maintenance — Fort Cavazos',
    solicitationNumber: 'W911SF-26-R-0044',
    department: 'DEPT OF DEFENSE.DEPT OF THE ARMY',
    changeType: 'field_change',
    changes: [
      { field: 'Set-Aside', old: 'Unrestricted', new: 'Small Business' },
      { field: 'Response Date', old: 'Mar 10, 2026', new: 'Mar 25, 2026' },
    ],
    isRead: false,
    createdAt: hoursAgo(6),
  },
  // Yesterday — only opportunity updates (no searches — tests empty search section)
  {
    id: 'n5',
    type: 'opportunity_update',
    opportunityId: 4350,
    title: 'Environmental Remediation Services — JBSA',
    solicitationNumber: 'FA8903-26-R-0012',
    department: 'DEPT OF DEFENSE.DEPT OF THE AIR FORCE',
    changeType: 'field_change',
    changes: [
      { field: 'Response Date', old: 'Mar 1, 2026', new: 'Apr 1, 2026' },
      { field: 'Description', new: 'Content changed' },
    ],
    isRead: false,
    createdAt: hoursAgo(28),
  },
  {
    id: 'n5b',
    type: 'opportunity_update',
    opportunityId: 4351,
    title: 'Water Treatment Plant Upgrades — NAS Pensacola',
    solicitationNumber: 'N69450-26-R-0091',
    department: 'DEPT OF DEFENSE.DEPT OF THE NAVY',
    changeType: 'amendment',
    changes: [],
    isRead: true,
    createdAt: hoursAgo(30),
  },
  // 6 days ago — only search matches (4-day quiet gap after yesterday)
  {
    id: 'n7',
    type: 'search_matches',
    searchName: 'Construction',
    searchUrl: '/?q=Construction',
    totalMatches: 1,
    isRead: true,
    createdAt: hoursAgo(144),
    matches: [
      {
        id: 4520, notice_id: 'opp-4520', title: 'Roof Replacement — Building 500, NAS Jacksonville',
        solicitation_number: 'N69450-26-R-0033', department: 'DEPT OF DEFENSE.DEPT OF THE NAVY',
        naics_code: '238160', set_aside_code: 'SB', set_aside_description: 'Small Business',
        posted_date: hoursAgo(146), response_deadline: '2026-04-30', active: true,
      },
    ],
  },
  // 8 days ago — read notification (another 1-day gap after 6 days ago)
  {
    id: 'n6',
    type: 'opportunity_update',
    opportunityId: 4300,
    title: 'Vehicle Maintenance Services — Fort Moore',
    solicitationNumber: 'W911SE-26-R-0008',
    department: 'DEPT OF DEFENSE.DEPT OF THE ARMY',
    changeType: 'field_change',
    changes: [
      { field: 'Set-Aside', old: 'Unrestricted', new: 'Small Business' },
    ],
    isRead: true,
    createdAt: hoursAgo(192),
  },
];

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

function SearchMatchCard({ notification }: { notification: SearchMatchNotification }) {
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
            <span className="text-[11px] text-dark-600 shrink-0">{formatTimeAgo(notification.createdAt)}</span>
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

function OpportunityUpdateCard({ notification }: { notification: OpportunityUpdateNotification }) {
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
              <span className="text-[11px] text-dark-600 shrink-0">{formatTimeAgo(notification.createdAt)}</span>
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

const SHOW_EMPTY = false;

export default function NotificationsV2Page() {
  const notifications = SHOW_EMPTY ? [] : MOCK_NOTIFICATIONS;
  const unreadCount = notifications.filter((n) => !n.isRead).length;
  const timeline = buildTimeline(notifications);

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="absolute top-4 right-6 z-20">
        <AuthButton />
      </div>

      <div className="relative z-10 max-w-3xl mx-auto px-6 pt-10 pb-10">
        <div className="flex items-center gap-4 mb-8">
          <Link to="/" className="text-dark-400 hover:text-dark-200 transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <h1 className="text-2xl font-semibold text-dark-100">Updates</h1>
          {unreadCount > 0 && (
            <span className="text-xs font-medium text-accent bg-accent/10 px-2 py-0.5 rounded-full">
              {unreadCount} unread
            </span>
          )}
        </div>

        {notifications.length === 0 ? (
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
                      renderItem={(n) => <SearchMatchCard key={n.id} notification={n} />}
                    />
                  )}

                  {updates.length > 0 && (
                    <CollapsibleSection
                      items={updates}
                      icon={<FileText size={12} />}
                      label="Opportunity changes"
                      iconColor="text-blue-400/60"
                      labelColor="text-blue-400/60"
                      renderItem={(n) => <OpportunityUpdateCard key={n.id} notification={n} />}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
