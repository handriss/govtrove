import { Link } from 'react-router-dom';
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
  if (days <= 0) return { className: 'text-dark-500', text: 'Closed' };
  if (days <= 7) return { className: 'text-red-400', text: `${days}d left` };
  if (days <= 14) return { className: 'text-amber-400', text: `${days}d left` };
  return { className: 'text-emerald-400', text: `${days}d left` };
}

function formatDepartment(dept: string | undefined) {
  if (!dept) return null;
  const parts = dept.split('.');
  return parts[parts.length - 1] || parts[0];
}

interface ResultCardProps {
  opportunity: OpportunityListItem;
  index: number;
}

export default function ResultCard({ opportunity, index }: ResultCardProps) {
  const daysUntil = getDaysUntilDeadline(opportunity.response_deadline);
  const deadline = getDeadlineStyle(daysUntil);
  const agency = formatDepartment(opportunity.department);

  return (
    <Link
      to={`/opportunity/${opportunity.id}`}
      className="block p-4 border border-dark-800/50 rounded-xl bg-dark-900/30
                 active:bg-dark-800/40 transition-colors duration-150"
      onClick={() => trackEvent({
        event_type: 'click',
        opportunity_id: opportunity.id,
        result_position: index + 1,
      })}
    >
      <div className="flex items-start justify-between gap-3 mb-2">
        <p className="text-dark-100 font-medium text-sm leading-snug line-clamp-2 flex-1">
          {opportunity.title}
        </p>
        <span className={`text-xs font-medium tabular-nums whitespace-nowrap ${deadline.className}`}>
          {deadline.text}
        </span>
      </div>

      <p className="text-[11px] text-dark-500 font-mono tracking-tight mb-2">
        {opportunity.solicitation_number || opportunity.notice_id}
      </p>

      <div className="flex flex-wrap items-center gap-2">
        {opportunity.set_aside_code && (
          <span className={`inline-flex px-2 py-0.5 text-[10px] font-medium rounded border ${setAsideColors[opportunity.set_aside_code] || setAsideColors['NONE']}`}>
            {opportunity.set_aside_code}
          </span>
        )}
        {agency && (
          <span className="text-xs text-dark-400 truncate">
            {agency}
          </span>
        )}
      </div>
    </Link>
  );
}
