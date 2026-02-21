import { Link } from 'react-router-dom';
import { ExternalLink } from 'lucide-react';
import type { OpportunityListItem } from '../types/api';
import { trackEvent } from '../services/api';

const setAsideColors: Record<string, string> = {
  'SBA': 'bg-blue-500/10 text-blue-400 border-blue-500/20',
  'SB': 'bg-blue-500/10 text-blue-400 border-blue-500/20',
  '8A': 'bg-violet-500/10 text-violet-400 border-violet-500/20',
  '8(a)': 'bg-violet-500/10 text-violet-400 border-violet-500/20',
  'SDVOSB': 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
  'WOSB': 'bg-pink-500/10 text-pink-400 border-pink-500/20',
  'HUBZone': 'bg-orange-500/10 text-orange-400 border-orange-500/20',
  'NONE': 'bg-dark-700/50 text-dark-400 border-dark-600/30',
};

function getDaysUntilDeadline(deadline: string | undefined) {
  if (!deadline) return null;
  const now = new Date();
  const todayStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  const today = new Date(todayStr + 'T00:00:00');
  const deadlineDate = new Date(deadline.slice(0, 10) + 'T00:00:00');
  return Math.round((deadlineDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
}

function getDeadlineStyle(days: number | null) {
  if (days === null) return { className: 'text-dark-500', text: '\u2014' };
  if (days < 0) return { className: 'text-dark-500', text: 'Closed' };
  if (days === 0) return { className: 'text-red-400', text: 'Today' };
  if (days <= 7) return { className: 'text-red-400', text: `${days}d` };
  if (days <= 14) return { className: 'text-amber-400', text: `${days}d` };
  return { className: 'text-emerald-400', text: `${days}d` };
}

function formatPostedDate(date: string | undefined) {
  if (!date) return '\u2014';
  const d = new Date(date);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatDepartment(dept: string | undefined) {
  if (!dept) return '\u2014';
  const parts = dept.split('.');
  return parts[parts.length - 1] || parts[0];
}

interface ResultRowProps {
  opportunity: OpportunityListItem;
  index: number;
}

export default function ResultRow({ opportunity, index }: ResultRowProps) {
  const daysUntil = getDaysUntilDeadline(opportunity.response_deadline);
  const deadline = getDeadlineStyle(daysUntil);

  return (
    <tr className="group hover:bg-dark-800/30 transition-colors duration-150">
      <td className="px-4 py-3.5 w-10">
        {opportunity.active && (
          <span className="flex h-2 w-2">
            <span className="animate-pulse absolute inline-flex h-2 w-2 rounded-full bg-emerald-400/40" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
          </span>
        )}
      </td>

      <td className="px-4 py-3.5">
        <Link
          to={`/opportunity/${opportunity.id}`}
          className="block"
          onClick={() => trackEvent({
            event_type: 'click',
            opportunity_id: opportunity.id,
            result_position: index + 1,
          })}
        >
          <p className="text-dark-100 font-medium group-hover:text-accent transition-colors duration-150 line-clamp-1">
            {opportunity.title}
          </p>
          <p className="text-xs text-dark-500 font-mono mt-0.5 tracking-tight">
            {opportunity.solicitation_number || opportunity.notice_id}
          </p>
          {opportunity.description && (
            <p className="text-xs text-dark-400 mt-1.5 line-clamp-2 leading-relaxed">
              {opportunity.description}
            </p>
          )}
        </Link>
      </td>

      <td className="px-4 py-3.5 text-sm text-dark-400 max-w-[180px]">
        <p className="truncate" title={opportunity.department || ''}>
          {formatDepartment(opportunity.department)}
        </p>
      </td>

      <td className="px-4 py-3.5 text-sm text-dark-400 tabular-nums whitespace-nowrap">
        {formatPostedDate(opportunity.posted_date)}
      </td>

      <td className="px-4 py-3.5">
        {opportunity.set_aside_code && (
          <span className={`inline-flex px-2 py-0.5 text-[10px] font-medium rounded border ${setAsideColors[opportunity.set_aside_code] || setAsideColors['NONE']}`}>
            {opportunity.set_aside_code}
          </span>
        )}
      </td>

      <td className="px-4 py-3.5">
        <span className={`text-sm font-medium tabular-nums ${deadline.className}`}>
          {deadline.text}
        </span>
      </td>

      <td className="px-4 py-3.5 text-sm text-dark-500 font-mono tracking-tight">
        {opportunity.naics_code || '\u2014'}
      </td>

      <td className="px-4 py-3.5 text-sm text-dark-400">
        {opportunity.pop_state || '\u2014'}
      </td>

      <td className="px-4 py-3.5 text-right">
        <a
          href={`https://sam.gov/opp/${opportunity.notice_id}/view`}
          target="_blank"
          rel="noopener noreferrer"
          className="p-1.5 text-dark-600 hover:text-accent rounded-lg transition-all duration-150
                     opacity-0 group-hover:opacity-100 hover:bg-dark-700/30"
          title="Open in SAM.gov"
          onClick={(e) => e.stopPropagation()}
        >
          <ExternalLink size={14} strokeWidth={1.5} />
        </a>
      </td>
    </tr>
  );
}
