import { Link } from 'react-router-dom';
import type { SolicitationHistory, SolicitationHistoryItem } from '../types/api';

const typeLabels: Record<string, string> = {
  o: 'Solicitation',
  p: 'Presolicitation',
  k: 'Combined Synopsis/Sol',
  r: 'Sources Sought',
  g: 'Sale of Surplus',
  s: 'Special Notice',
  i: 'Intent to Bundle',
  a: 'Award Notice',
  u: 'Justification',
  j: 'Justification & Approval',
};

const typeColors: Record<string, string> = {
  r: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
  p: 'bg-violet-500/20 text-violet-400 border-violet-500/30',
  o: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  k: 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30',
  a: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
  s: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
  u: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
  j: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
  g: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
  i: 'bg-red-500/20 text-red-400 border-red-500/30',
};

const dotColors: Record<string, string> = {
  r: 'bg-amber-400',
  p: 'bg-violet-400',
  o: 'bg-blue-400',
  k: 'bg-cyan-400',
  a: 'bg-emerald-400',
  s: 'bg-gray-400',
  u: 'bg-pink-400',
  j: 'bg-pink-400',
  g: 'bg-gray-400',
  i: 'bg-red-400',
};

function formatDate(date: string | undefined) {
  if (!date) return null;
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

function detectVersionChanges(prev: SolicitationHistoryItem, curr: SolicitationHistoryItem): string[] {
  const changes: string[] = [];
  if (prev.title !== curr.title) changes.push('Title updated');
  if (prev.type !== curr.type) {
    const oldLabel = prev.type ? typeLabels[prev.type] || prev.type : null;
    const newLabel = curr.type ? typeLabels[curr.type] || curr.type : null;
    if (oldLabel && newLabel) changes.push(`${oldLabel} \u2192 ${newLabel}`);
    else changes.push('Type changed');
  }
  if (prev.response_deadline !== curr.response_deadline) {
    const formatted = formatDate(curr.response_deadline);
    changes.push(formatted ? `Deadline: ${formatted}` : 'Deadline updated');
  }
  if (prev.award_amount !== curr.award_amount && curr.award_amount) {
    changes.push(`Award: ${formatCurrency(curr.award_amount)}`);
  }
  if (prev.awardee_name !== curr.awardee_name && curr.awardee_name) {
    changes.push(`Awardee: ${curr.awardee_name}`);
  }
  if (!prev.active && curr.active) changes.push('Reactivated');
  if (prev.active && !curr.active) changes.push('Deactivated');
  return changes;
}

interface TimelineGroup {
  noticeId: string;
  items: SolicitationHistoryItem[];
}

function groupByNoticeId(notices: SolicitationHistoryItem[]): TimelineGroup[] {
  const groups: TimelineGroup[] = [];
  let current: TimelineGroup | null = null;
  for (const item of notices) {
    if (!current || current.noticeId !== item.notice_id) {
      current = { noticeId: item.notice_id, items: [item] };
      groups.push(current);
    } else {
      current.items.push(item);
    }
  }
  return groups;
}

function NoticeCard({
  item,
  isLast,
  isUpdate,
  prevItem,
}: {
  item: SolicitationHistoryItem;
  isLast: boolean;
  isUpdate?: boolean;
  prevItem?: SolicitationHistoryItem;
}) {
  const typeCode = item.type || '';
  const typeLabel = typeLabels[typeCode] || item.base_type || 'Notice';
  const badgeColor = typeColors[typeCode] || 'bg-gray-500/20 text-gray-400 border-gray-500/30';
  const dot = dotColors[typeCode] || 'bg-gray-400';
  const postedDate = formatDate(item.posted_date);

  const changes = isUpdate && prevItem
    ? detectVersionChanges(prevItem, item)
    : [];

  const content = (
    <div
      className={`rounded-lg p-3 transition-colors duration-150 ${
        item.is_current
          ? 'bg-accent/5 border border-accent/30 ring-1 ring-accent/20'
          : isUpdate
            ? 'bg-dark-800/10 border border-dashed border-dark-700/40 hover:border-dark-600/50'
            : 'bg-dark-800/20 border border-dark-800/40 hover:border-dark-700/50'
      }`}
    >
      <div className="flex flex-wrap items-center gap-2 mb-1">
        {isUpdate ? (
          <span className="inline-flex px-2 py-0.5 text-[11px] font-medium rounded-full border bg-dark-700/30 text-dark-400 border-dark-600/30">
            v{item.version} Update
          </span>
        ) : (
          <span className={`inline-flex px-2 py-0.5 text-[11px] font-medium rounded-full border ${badgeColor}`}>
            {typeLabel}
          </span>
        )}
        {item.is_current && (
          <span className="text-[11px] text-accent font-medium">Current</span>
        )}
        {!item.active && (
          <span className="text-[11px] text-dark-500">Inactive</span>
        )}
        {postedDate && (
          <span className="text-[11px] text-dark-500 ml-auto">{postedDate}</span>
        )}
      </div>

      <p className={`text-sm leading-snug ${
        item.is_current ? 'text-dark-100' : isUpdate ? 'text-dark-400' : 'text-dark-300'
      }`}>
        {item.title}
      </p>

      {item.award_amount && !isUpdate && (
        <p className="text-xs text-emerald-400 mt-1">
          Award: {formatCurrency(item.award_amount)}
          {item.awardee_name && ` to ${item.awardee_name}`}
        </p>
      )}

      {changes.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {changes.map((label, i) => (
            <span
              key={i}
              className="inline-flex px-2 py-0.5 text-[11px] rounded-full bg-dark-700/40 text-dark-400 border border-dark-700/30"
            >
              {label}
            </span>
          ))}
        </div>
      )}
    </div>
  );

  return (
    <div className="relative flex gap-4">
      {/* Timeline line + dot */}
      <div className="flex flex-col items-center w-3 flex-shrink-0">
        <div className={`rounded-full mt-4 flex-shrink-0 ${
          isUpdate
            ? 'w-1.5 h-1.5 bg-dark-600'
            : item.is_current
              ? 'w-2.5 h-2.5 ring-2 ring-accent/40 ' + dot
              : 'w-2.5 h-2.5 ' + dot
        }`} />
        {!isLast && <div className="w-px flex-1 bg-dark-700/50 mt-1" />}
      </div>

      {/* Card */}
      <div className="flex-1 pb-3">
        {item.is_current ? (
          content
        ) : (
          <Link to={`/opportunity/${item.id}`} className="block">
            {content}
          </Link>
        )}
      </div>
    </div>
  );
}

export default function SolicitationTimeline({
  history,
}: {
  history: SolicitationHistory;
  currentId: number;
}) {
  const groups = groupByNoticeId(history.notices);
  const totalEntries = history.notices.length;
  let entryIndex = 0;

  return (
    <div>
      {groups.map((group) =>
        group.items.map((item, itemIdx) => {
          const isUpdate = itemIdx > 0;
          const prevItem = itemIdx > 0 ? group.items[itemIdx - 1] : undefined;
          const isLast = ++entryIndex === totalEntries;
          return (
            <NoticeCard
              key={item.id}
              item={item}
              isLast={isLast}
              isUpdate={isUpdate}
              prevItem={prevItem}
            />
          );
        })
      )}

      {history.truncated && (
        <p className="text-xs text-dark-500 mt-3 pl-7">
          Showing 50 of {history.total_notices} notices.{' '}
          <a
            href={`https://sam.gov/search/?keywords=${encodeURIComponent(history.solicitation_number)}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:text-accent-hover transition-colors"
          >
            View all on SAM.gov
          </a>
        </p>
      )}
    </div>
  );
}
