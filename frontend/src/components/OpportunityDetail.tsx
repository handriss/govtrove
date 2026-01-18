import { useOpportunity } from '../hooks/useOpportunities';

interface OpportunityDetailProps {
  id: number;
  onClose: () => void;
}

const TYPE_LABELS: Record<string, string> = {
  o: 'Solicitation',
  p: 'Presolicitation',
  k: 'Combined Synopsis/Solicitation',
  r: 'Sources Sought',
  g: 'Sale of Surplus Property',
  s: 'Special Notice',
  i: 'Intent to Bundle',
  a: 'Award Notice',
  u: 'Justification',
  j: 'Justification and Approval',
};

function formatDate(dateStr?: string): string {
  if (!dateStr) return '-';
  try {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  } catch {
    return dateStr;
  }
}

function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '-';
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(amount);
}

export function OpportunityDetail({ id, onClose }: OpportunityDetailProps) {
  const { data: opportunity, isLoading, error } = useOpportunity(id);

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="absolute right-0 top-0 bottom-0 w-full max-w-2xl bg-white shadow-xl overflow-y-auto">
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Opportunity Details</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 text-2xl leading-none"
          >
            &times;
          </button>
        </div>

        <div className="p-6">
          {isLoading ? (
            <div className="space-y-4 animate-pulse">
              <div className="h-6 bg-gray-200 rounded w-3/4"></div>
              <div className="h-4 bg-gray-200 rounded w-1/2"></div>
              <div className="h-4 bg-gray-200 rounded w-full"></div>
              <div className="h-4 bg-gray-200 rounded w-full"></div>
            </div>
          ) : error ? (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <p className="text-red-800">Failed to load opportunity details</p>
              <p className="text-red-600 text-sm mt-1">{error.message}</p>
            </div>
          ) : opportunity ? (
            <div className="space-y-6">
              <div>
                <h1 className="text-xl font-semibold text-gray-900 mb-2">{opportunity.title}</h1>
                <div className="flex flex-wrap gap-2">
                  {opportunity.type && (
                    <span className="inline-block px-2 py-1 text-sm bg-blue-100 text-blue-800 rounded">
                      {TYPE_LABELS[opportunity.type] || opportunity.type}
                    </span>
                  )}
                  {opportunity.set_aside_code && (
                    <span className="inline-block px-2 py-1 text-sm bg-amber-100 text-amber-800 rounded">
                      {opportunity.set_aside_description || opportunity.set_aside_code}
                    </span>
                  )}
                  {!opportunity.active && (
                    <span className="inline-block px-2 py-1 text-sm bg-gray-100 text-gray-800 rounded">
                      Inactive
                    </span>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4 text-sm">
                <InfoRow label="Notice ID" value={opportunity.notice_id} />
                <InfoRow label="Solicitation #" value={opportunity.solicitation_number} />
                <InfoRow label="Posted Date" value={formatDate(opportunity.posted_date)} />
                <InfoRow label="Response Deadline" value={formatDate(opportunity.response_deadline)} />
                <InfoRow label="Department" value={opportunity.department} />
                <InfoRow label="Sub-tier" value={opportunity.sub_tier} />
                <InfoRow label="Office" value={opportunity.office} />
                <InfoRow label="NAICS Code" value={opportunity.naics_code} />
                {opportunity.naics_codes && opportunity.naics_codes.length > 1 && (
                  <InfoRow label="Additional NAICS" value={opportunity.naics_codes.slice(1).join(', ')} />
                )}
                <InfoRow label="Classification Code" value={opportunity.classification_code} />
              </div>

              {(opportunity.pop_city || opportunity.pop_state || opportunity.pop_country) && (
                <div>
                  <h3 className="text-sm font-medium text-gray-700 mb-2">Place of Performance</h3>
                  <p className="text-sm text-gray-600">
                    {[
                      opportunity.pop_street_address,
                      opportunity.pop_city,
                      opportunity.pop_state,
                      opportunity.pop_zip,
                      opportunity.pop_country,
                    ]
                      .filter(Boolean)
                      .join(', ')}
                  </p>
                </div>
              )}

              {opportunity.award_amount && (
                <div>
                  <h3 className="text-sm font-medium text-gray-700 mb-2">Award Information</h3>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <InfoRow label="Award Amount" value={formatCurrency(opportunity.award_amount)} />
                    <InfoRow label="Award Date" value={formatDate(opportunity.award_date)} />
                    <InfoRow label="Awardee" value={opportunity.awardee_name} />
                    <InfoRow label="Awardee UEI" value={opportunity.awardee_uei} />
                  </div>
                </div>
              )}

              {opportunity.description && (
                <div>
                  <h3 className="text-sm font-medium text-gray-700 mb-2">Description</h3>
                  <div className="text-sm text-gray-600 whitespace-pre-wrap bg-gray-50 p-4 rounded-lg max-h-96 overflow-y-auto">
                    {opportunity.description}
                  </div>
                </div>
              )}

              {opportunity.resource_links && opportunity.resource_links.length > 0 && (
                <div>
                  <h3 className="text-sm font-medium text-gray-700 mb-2">Resources</h3>
                  <ul className="space-y-1">
                    {opportunity.resource_links.map((link, i) => (
                      <li key={i}>
                        <a
                          href={link}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-sm text-blue-600 hover:text-blue-800 break-all"
                        >
                          {link}
                        </a>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {opportunity.ui_link && (
                <div className="pt-4 border-t border-gray-200">
                  <a
                    href={opportunity.ui_link}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    View on SAM.gov
                    <span>&rarr;</span>
                  </a>
                </div>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div>
      <span className="text-gray-500">{label}:</span>{' '}
      <span className="text-gray-900">{value}</span>
    </div>
  );
}
