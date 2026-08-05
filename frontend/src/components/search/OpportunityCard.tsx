import { memo, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { Star, MapPin } from 'lucide-react';
import type { OpportunityListItem } from '../../types/api';

// --- Shared utilities ---

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
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatDepartment(dept: string | undefined): string {
  if (!dept) return '';
  const parts = dept.split('.');
  return parts[parts.length - 1] || parts[0];
}

// --- Urgency badge ---

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

// --- Deadline display ---

function getDeadlineColor(days: number | null): string {
  if (days === null) return 'text-dark-500';
  if (days < 0) return 'text-dark-500';
  if (days <= 2) return 'text-red-400';
  if (days <= 7) return 'text-amber-400';
  return 'text-dark-300';
}

// --- Keyword highlighting ---

// Pull the highlightable terms out of a raw query: quoted phrases stay whole,
// bare words stand alone, `OR` is syntax, and `-excluded` terms are not matches
// so highlighting them would be a lie. Longest first so a phrase wins over the
// words inside it.
export function parseQueryTerms(keyword: string | undefined): string[] {
  if (!keyword || !keyword.trim()) return [];
  const terms: string[] = [];
  const phraseRe = /"([^"]+)"/g;
  let m: RegExpExecArray | null;
  while ((m = phraseRe.exec(keyword)) !== null) {
    const phrase = m[1].trim();
    if (phrase) terms.push(phrase);
  }
  for (const word of keyword.replace(phraseRe, ' ').split(/\s+/)) {
    if (!word || word === 'OR' || word.startsWith('-') || word.length < 2) continue;
    terms.push(word);
  }
  return terms.sort((a, b) => b.length - a.length);
}

export function highlightKeywords(text: string, keyword: string | undefined): ReactNode {
  const terms = parseQueryTerms(keyword);
  if (terms.length === 0) return text;

  const escaped = terms.map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
  // Word boundaries: without them "ai" lights up inside "details" and "Air".
  const regex = new RegExp(`\\b(${escaped.join('|')})\\b`, 'gi');
  const parts = text.split(regex);
  if (parts.length === 1) return text;
  // split() with one capture group alternates [text, match, text, ...], so odd
  // indices are matches. Re-testing with a /g regex would skip every other hit.
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <mark key={i} className="bg-yellow-500/20 text-yellow-200 rounded-sm px-0.5">
        {part}
      </mark>
    ) : (
      part
    ),
  );
}

export function extractSnippet(description: string | undefined, keyword: string | undefined, maxLen = 200): string {
  if (!description) return '';
  const text = description.replace(/\s+/g, ' ').trim();
  if (!keyword || !keyword.trim()) return text.slice(0, maxLen) + (text.length > maxLen ? '...' : '');

  // Match on word boundaries for the same reason highlighting does — otherwise
  // the snippet centres on "ai" inside "details" rather than the real hit.
  let bestIdx = -1;
  for (const term of parseQueryTerms(keyword)) {
    const escaped = term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const idx = text.search(new RegExp(`\\b${escaped}\\b`, 'i'));
    if (idx !== -1 && (bestIdx === -1 || idx < bestIdx)) bestIdx = idx;
  }

  if (bestIdx === -1) return text.slice(0, maxLen) + (text.length > maxLen ? '...' : '');

  const start = Math.max(0, bestIdx - 60);
  const end = Math.min(text.length, start + maxLen);
  const snippet = text.slice(start, end);
  return (start > 0 ? '...' : '') + snippet + (end < text.length ? '...' : '');
}

// --- Component ---

interface OpportunityCardProps {
  opportunity: OpportunityListItem;
  keyword?: string;
  isSaved: boolean;
  isSelected: boolean;
  anySelected: boolean;
  onToggleSave: (id: number) => void;
  onToggleSelect: (id: number) => void;
}

export default memo(function OpportunityCard({
  opportunity: opp,
  keyword,
  isSaved,
  onToggleSave,
}: OpportunityCardProps) {
  const badge = getUrgencyBadge(opp.posted_date, opp.response_deadline);
  const days = getDaysUntilDeadline(opp.response_deadline);
  const deadlineColor = getDeadlineColor(days);
  const agency = formatDepartment(opp.department);
  const snippet = extractSnippet(opp.description, keyword);

  const deadlineDisplay = days !== null
    ? days < 0
      ? 'Closed'
      : `Due: ${formatDeadlineDate(opp.response_deadline)} (${days}d)`
    : null;

  const ariaLabel = [
    opp.title,
    agency && `${agency}`,
    deadlineDisplay && deadlineDisplay,
  ].filter(Boolean).join(', ');

  return (
    <article
      aria-label={ariaLabel}
      className="group relative rounded-xl border border-dark-800/50 bg-dark-900/30
        hover:border-dark-600/50 hover:shadow-lg hover:shadow-black/20
        transition-all duration-200"
    >
      <Link
        to={`/opportunity/${opp.id}${keyword?.trim() ? `?q=${encodeURIComponent(keyword.trim())}` : ''}`}
        className="block px-4 py-3.5"
      >
        {/* PRIMARY: Title row */}
        <div className="flex items-start gap-2">
          {/* Star */}
          <button
            type="button"
            onClick={(e) => { e.preventDefault(); e.stopPropagation(); onToggleSave(opp.id); }}
            className="shrink-0 p-1 -m-1 rounded-full hover:bg-yellow-400/10 focus-visible:ring-2 focus-visible:ring-accent/50 focus-visible:ring-offset-1 focus-visible:ring-offset-dark-950 transition-colors"
            aria-label={isSaved ? 'Remove from saved' : 'Save opportunity'}
          >
            <Star
              size={18}
              className={`transition-colors duration-150 ${
                isSaved
                  ? 'fill-yellow-400 text-yellow-400'
                  : 'text-dark-500 hover:text-yellow-400/70'
              }`}
            />
          </button>

          {/* Title + badge */}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <h3 className="text-[15px] font-medium text-dark-100 leading-snug line-clamp-2 group-hover:text-accent transition-colors">
                {highlightKeywords(opp.title, keyword)}
              </h3>
              {badge && (
                <span aria-label={badge.label} className={`shrink-0 text-[11px] font-medium rounded-full px-2 py-0.5 whitespace-nowrap ${badge.className}`}>
                  {badge.label}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* SECONDARY: Metadata row */}
        <div className="ml-[52px] mt-1.5 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-sm text-dark-400">
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

        {/* Sol number + deadline */}
        <div className="ml-[52px] mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          {opp.solicitation_number && (
            <span className="text-dark-500 font-mono text-xs tracking-tight">
              Sol#: {opp.solicitation_number}
            </span>
          )}
          {deadlineDisplay && (
            <span className={`text-xs font-medium tabular-nums ${deadlineColor}`}>
              {deadlineDisplay}
            </span>
          )}
        </div>

        {/* TERTIARY: Description + location */}
        {snippet && (
          <div className="ml-[52px] mt-2">
            <p className="text-xs text-dark-500 leading-relaxed line-clamp-3">
              {highlightKeywords(snippet, keyword)}
            </p>
          </div>
        )}
        {opp.pop_state && (
          <div className="ml-[52px] mt-1.5 flex items-center gap-1 text-xs text-dark-500">
            <MapPin size={12} className="shrink-0" />
            <span>{opp.pop_state}</span>
          </div>
        )}
      </Link>
    </article>
  );
});
