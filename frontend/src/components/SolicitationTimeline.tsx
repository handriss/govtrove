import { Link } from 'react-router-dom';
import type { SolicitationHistory, SolicitationHistoryItem, FieldChange } from '../types/api';

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

function formatChangeLabel(change: FieldChange): string {
  const { field_name, old_value, new_value } = change;

  // Status changes are self-describing
  if (field_name === 'Reactivated' || field_name === 'Deactivated' || field_name === 'New notice' || field_name === 'Description') {
    return field_name === 'Description' ? 'Description updated' : field_name;
  }

  if (!old_value && new_value) return `${field_name}: ${new_value}`;
  if (old_value && !new_value) return `${field_name} removed`;
  if (old_value && new_value) return `${field_name}: ${old_value} → ${new_value}`;
  return field_name;
}

function NoticeCard({
  item,
  isLast,
  isUpdate,
}: {
  item: SolicitationHistoryItem;
  isLast: boolean;
  isUpdate: boolean;
}) {
  const typeCode = item.type || '';
  const typeLabel = typeLabels[typeCode] || item.base_type || 'Notice';
  const badgeColor = typeColors[typeCode] || 'bg-gray-500/20 text-gray-400 border-gray-500/30';
  const dot = dotColors[typeCode] || 'bg-gray-400';
  const postedDate = formatDate(item.posted_date);
  const changes = item.changes || [];

  const content = (
    <div
      className={`rounded-lg p-3 transition-colors duration-150 ${
        item.is_current
          ? 'bg-accent/5 border border-accent/30 ring-1 ring-accent/20'
          : !item.active
            ? 'bg-dark-800/10 border border-dark-800/30 opacity-50'
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
          {changes.map((change, i) => (
            <span
              key={i}
              className="inline-flex px-2 py-0.5 text-[11px] rounded-full bg-dark-700/40 text-dark-400 border border-dark-700/30"
            >
              {formatChangeLabel(change)}
            </span>
          ))}
        </div>
      )}
    </div>
  );

  return (
    <div className="relative flex gap-4">
      <div className="flex flex-col items-center w-3 flex-shrink-0">
        <div className={`rounded-full mt-4 flex-shrink-0 ${
          isUpdate
            ? 'w-1.5 h-1.5 bg-dark-600'
            : !item.active
              ? 'w-2.5 h-2.5 bg-dark-600'
              : item.is_current
                ? 'w-2.5 h-2.5 ring-2 ring-accent/40 ' + dot
                : 'w-2.5 h-2.5 ' + dot
        }`} />
        {!isLast && <div className="w-px flex-1 bg-dark-700/50 mt-1" />}
      </div>

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
  const notices = history.notices;

  return (
    <div>
      {notices.map((item, idx) => {
        const isUpdate = idx > 0 && item.notice_id === notices[idx - 1].notice_id;
        return (
          <NoticeCard
            key={item.id}
            item={item}
            isLast={idx === notices.length - 1}
            isUpdate={isUpdate}
          />
        );
      })}

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
