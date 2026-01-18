import type { OpportunityListItem } from '../types/opportunity';

interface OpportunityCardProps {
  opportunity: OpportunityListItem;
  onClick: () => void;
}

const TYPE_LABELS: Record<string, string> = {
  o: 'Solicitation',
  p: 'Presolicitation',
  k: 'Combined Synopsis',
  r: 'Sources Sought',
  g: 'Surplus Sale',
  s: 'Special Notice',
  i: 'Intent to Bundle',
  a: 'Award Notice',
  u: 'Justification',
  j: 'J&A',
};

function formatDate(dateStr?: string): string {
  if (!dateStr) return '-';
  try {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return dateStr;
  }
}

function getDeadlineStatus(deadline?: string): { label: string; className: string } {
  if (!deadline) return { label: 'No deadline', className: 'text-gray-500' };

  const deadlineDate = new Date(deadline);
  const now = new Date();
  const daysUntil = Math.ceil((deadlineDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

  if (daysUntil < 0) {
    return { label: 'Closed', className: 'text-gray-500' };
  } else if (daysUntil <= 3) {
    return { label: `${daysUntil}d left`, className: 'text-red-600 font-medium' };
  } else if (daysUntil <= 7) {
    return { label: `${daysUntil}d left`, className: 'text-orange-600' };
  } else {
    return { label: formatDate(deadline), className: 'text-gray-600' };
  }
}

export function OpportunityCard({ opportunity, onClick }: OpportunityCardProps) {
  const deadlineStatus = getDeadlineStatus(opportunity.response_deadline);
  const typeLabel = opportunity.type ? TYPE_LABELS[opportunity.type] || opportunity.type : 'Unknown';

  return (
    <div
      onClick={onClick}
      className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md hover:border-blue-300 transition-all cursor-pointer"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-gray-900 line-clamp-2 mb-1">{opportunity.title}</h3>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-gray-600">
            {opportunity.department && (
              <span className="truncate max-w-[200px]" title={opportunity.department}>
                {opportunity.department}
              </span>
            )}
            {opportunity.solicitation_number && (
              <span className="font-mono text-xs bg-gray-100 px-1.5 py-0.5 rounded">
                {opportunity.solicitation_number}
              </span>
            )}
          </div>
        </div>
        <div className="flex-shrink-0 text-right">
          <span
            className={`inline-block px-2 py-1 text-xs rounded-full ${
              opportunity.type === 'o'
                ? 'bg-blue-100 text-blue-800'
                : opportunity.type === 'p'
                  ? 'bg-purple-100 text-purple-800'
                  : opportunity.type === 'k'
                    ? 'bg-green-100 text-green-800'
                    : 'bg-gray-100 text-gray-800'
            }`}
          >
            {typeLabel}
          </span>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
        <div>
          <span className="text-gray-500">Posted:</span>{' '}
          <span className="text-gray-700">{formatDate(opportunity.posted_date)}</span>
        </div>
        <div>
          <span className="text-gray-500">Deadline:</span>{' '}
          <span className={deadlineStatus.className}>{deadlineStatus.label}</span>
        </div>
        {opportunity.set_aside_code && (
          <span className="bg-amber-100 text-amber-800 text-xs px-2 py-0.5 rounded">
            {opportunity.set_aside_description || opportunity.set_aside_code}
          </span>
        )}
        {opportunity.naics_code && (
          <span className="text-gray-500 text-xs">NAICS: {opportunity.naics_code}</span>
        )}
        {opportunity.pop_state && (
          <span className="text-gray-500 text-xs">Location: {opportunity.pop_state}</span>
        )}
      </div>
    </div>
  );
}
