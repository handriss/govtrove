export const NOTICE_TYPE_OPTIONS = [
  { value: 'Solicitation', label: 'Solicitation' },
  { value: 'Presolicitation', label: 'Presolicitation' },
  { value: 'Combined Synopsis/Solicitation', label: 'Combined Synopsis' },
  { value: 'Sources Sought', label: 'Sources Sought' },
  { value: 'Special Notice', label: 'Special Notice' },
  { value: 'Award Notice', label: 'Award Notice' },
  { value: 'Modification/Amendment/Cancel', label: 'Modification' },
  { value: 'Intent to Bundle', label: 'Intent to Bundle' },
  { value: 'Sale of Surplus Property', label: 'Surplus Property' },
  { value: 'Fair Opportunity / Limited Sources Justification', label: 'Fair Opportunity' },
];

export const DEFAULT_NOTICE_TYPES = [
  'Solicitation',
  'Presolicitation',
  'Combined Synopsis/Solicitation',
  'Sources Sought',
  'Special Notice',
];

export const NOTICE_TYPE_LABELS: Record<string, string> = Object.fromEntries(
  NOTICE_TYPE_OPTIONS.map((o) => [o.value, o.label]),
);

export const DEADLINE_PRESET_LABELS: Record<string, string> = {
  '7': 'next 7 days',
  '14': 'next 14 days',
  '30': 'next 30 days',
  '60': 'next 60 days',
  'quarter': 'this quarter',
};

export const SET_ASIDE_LABELS: Record<string, string> = {
  'SBA': 'SBA',
  'SBP': 'Small Business',
  '8A': '8(a)',
  '8AN': '8(a) Sole Source',
  'SDVOSBC': 'SDVOSB',
  'SDVOSBS': 'SDVOSB Sole Source',
  'WOSB': 'WOSB',
  'WOSBSS': 'WOSB Sole Source',
  'EDWOSB': 'EDWOSB',
  'EDWOSBSS': 'EDWOSB Sole Source',
  'HZC': 'HUBZone',
  'HZS': 'HUBZone Sole Source',
  'VSA': 'VOSB',
  'VSS': 'VOSB Sole Source',
};

export const SORT_OPTIONS = [
  { value: 'relevance', label: 'Relevance' },
  { value: 'deadline', label: 'Deadline' },
  { value: 'posted_date', label: 'Posted Date' },
  { value: 'department', label: 'Agency' },
];

export const STATE_NAMES: Record<string, string> = {
  'AL': 'Alabama', 'AK': 'Alaska', 'AZ': 'Arizona', 'AR': 'Arkansas',
  'CA': 'California', 'CO': 'Colorado', 'CT': 'Connecticut', 'DE': 'Delaware',
  'DC': 'District of Columbia', 'FL': 'Florida', 'GA': 'Georgia', 'HI': 'Hawaii',
  'ID': 'Idaho', 'IL': 'Illinois', 'IN': 'Indiana', 'IA': 'Iowa',
  'KS': 'Kansas', 'KY': 'Kentucky', 'LA': 'Louisiana', 'ME': 'Maine',
  'MD': 'Maryland', 'MA': 'Massachusetts', 'MI': 'Michigan', 'MN': 'Minnesota',
  'MS': 'Mississippi', 'MO': 'Missouri', 'MT': 'Montana', 'NE': 'Nebraska',
  'NV': 'Nevada', 'NH': 'New Hampshire', 'NJ': 'New Jersey', 'NM': 'New Mexico',
  'NY': 'New York', 'NC': 'North Carolina', 'ND': 'North Dakota', 'OH': 'Ohio',
  'OK': 'Oklahoma', 'OR': 'Oregon', 'PA': 'Pennsylvania', 'RI': 'Rhode Island',
  'SC': 'South Carolina', 'SD': 'South Dakota', 'TN': 'Tennessee', 'TX': 'Texas',
  'UT': 'Utah', 'VT': 'Vermont', 'VA': 'Virginia', 'WA': 'Washington',
  'WV': 'West Virginia', 'WI': 'Wisconsin', 'WY': 'Wyoming',
  'AS': 'American Samoa', 'GU': 'Guam', 'MP': 'Northern Mariana Islands',
  'PR': 'Puerto Rico', 'VI': 'U.S. Virgin Islands',
};
