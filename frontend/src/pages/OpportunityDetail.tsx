import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  ArrowLeft,
  ExternalLink,
  Copy,
  Loader2,
  Check,
  Calendar,
  Clock,
  Building2,
  MapPin,
  Award,
  FileText,
  AlertCircle,
  Users,
  Mail,
  Phone,
  AlignLeft,
  Pilcrow,
  GitBranch,
} from 'lucide-react';
import { getOpportunity, getSolicitationHistory } from '../services/api';
import { formatDescription } from '../utils/formatDescription';
import SolicitationTimeline from '../components/SolicitationTimeline';
import type { Opportunity, SolicitationHistory } from '../types/api';

const typeLabels: Record<string, string> = {
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

const setAsideColors: Record<string, string> = {
  SBA: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
  SB: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
  '8A': 'bg-violet-500/10 text-violet-400 border-violet-500/20',
  '8(a)': 'bg-violet-500/10 text-violet-400 border-violet-500/20',
  SDVOSB: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
  WOSB: 'bg-pink-500/10 text-pink-400 border-pink-500/20',
  HUBZone: 'bg-orange-500/10 text-orange-400 border-orange-500/20',
};

function formatCurrency(value: number | undefined) {
  if (!value) return '—';
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

function getDaysUntilDeadline(deadline: string | undefined) {
  if (!deadline) return null;
  const now = new Date();
  const deadlineDate = new Date(deadline);
  return Math.ceil((deadlineDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
}

function formatDate(date: string | undefined) {
  if (!date) return '—';
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function formatDateTime(date: string | undefined) {
  if (!date) return '—';
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    timeZoneName: 'short',
  });
}

function parseDepartmentHierarchy(dept: string | undefined) {
  if (!dept) return { department: '—', subTier: null, office: null };
  const parts = dept.split('.');
  return {
    department: parts[0] || '—',
    subTier: parts[1] || null,
    office: parts[2] || null,
  };
}

function InfoRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex justify-between items-start py-2.5 border-b border-dark-800/30 last:border-0">
      <dt className="text-dark-400 text-sm">{label}</dt>
      <dd className={`text-dark-200 text-sm text-right max-w-[60%] ${mono ? 'font-mono tracking-tight' : ''}`}>
        {value}
      </dd>
    </div>
  );
}

function Section({
  icon: Icon,
  title,
  children,
  defaultOpen = true,
}: {
  icon: React.ElementType;
  title: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
}) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border border-dark-800/50 rounded-xl bg-dark-900/20 overflow-hidden">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full px-5 py-4 flex items-center gap-3 hover:bg-dark-800/20 transition-colors duration-150"
      >
        <Icon size={18} className="text-dark-400" strokeWidth={1.5} />
        <span className="text-sm font-medium text-dark-200 flex-1 text-left">{title}</span>
        <svg
          className={`w-4 h-4 text-dark-500 transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {isOpen && <div className="px-5 pb-5 pt-1">{children}</div>}
    </div>
  );
}

function ContactCard({
  label,
  name,
  title,
  email,
  phone,
  fax,
}: {
  label: string;
  name?: string;
  title?: string;
  email?: string;
  phone?: string;
  fax?: string;
}) {
  return (
    <div className="p-4 rounded-lg bg-dark-800/30 border border-dark-700/30">
      <p className="text-xs text-dark-500 uppercase tracking-wider mb-2">{label}</p>
      {name && <p className="text-sm text-dark-100 font-medium">{name}</p>}
      {title && <p className="text-sm text-dark-400">{title}</p>}
      <div className="mt-2 space-y-1">
        {email && (
          <a
            href={`mailto:${email}`}
            className="flex items-center gap-2 text-sm text-accent hover:text-accent-hover transition-colors"
          >
            <Mail size={14} strokeWidth={1.5} className="flex-shrink-0" />
            {email}
          </a>
        )}
        {phone && (
          <a
            href={`tel:${phone}`}
            className="flex items-center gap-2 text-sm text-dark-300 hover:text-dark-100 transition-colors"
          >
            <Phone size={14} strokeWidth={1.5} className="flex-shrink-0" />
            {phone}
          </a>
        )}
        {fax && (
          <p className="flex items-center gap-2 text-sm text-dark-400">
            <Phone size={14} strokeWidth={1.5} className="flex-shrink-0" />
            {fax} (fax)
          </p>
        )}
      </div>
    </div>
  );
}

export default function OpportunityDetail() {
  const { id } = useParams();
  const [opportunity, setOpportunity] = useState<Opportunity | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [formatted, setFormatted] = useState(() => localStorage.getItem('govtrove_format_desc') !== 'false');
  const [history, setHistory] = useState<SolicitationHistory | null>(null);

  useEffect(() => {
    if (!id) return;

    setLoading(true);
    setError(null);

    getOpportunity(parseInt(id))
      .then(setOpportunity)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (opportunity?.solicitation_number) {
      getSolicitationHistory(opportunity.id).then(setHistory);
    }
  }, [opportunity?.id, opportunity?.solicitation_number]);

  const handleCopyLink = () => {
    navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-accent/70" strokeWidth={1.5} />
          <span className="text-sm text-dark-400">Loading opportunity...</span>
        </div>
      </div>
    );
  }

  if (error || !opportunity) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 rounded-2xl bg-dark-800/50 border border-dark-700/30 flex items-center justify-center mx-auto mb-6">
            <AlertCircle size={28} className="text-dark-500" strokeWidth={1.5} />
          </div>
          <h1 className="text-lg font-medium text-dark-200 mb-2">{error || 'Opportunity not found'}</h1>
          <p className="text-dark-400 text-sm mb-6">The opportunity may have been removed or the link is invalid.</p>
          <Link
            to="/"
            className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent hover:bg-accent-hover text-white font-medium rounded-xl transition-colors duration-200"
          >
            Back to Search
          </Link>
        </div>
      </div>
    );
  }

  const daysUntil = getDaysUntilDeadline(opportunity.response_deadline);
  const isExpired = daysUntil !== null && daysUntil <= 0;
  const isUrgent = daysUntil !== null && daysUntil > 0 && daysUntil <= 7;
  const deptHierarchy = parseDepartmentHierarchy(opportunity.department);
  const typeLabel = opportunity.type ? typeLabels[opportunity.type] || opportunity.type : '—';
  const hasAward = opportunity.award_number || opportunity.award_amount || opportunity.awardee_name;
  const hasPrimaryContact = opportunity.primary_contact_fullname || opportunity.primary_contact_email || opportunity.primary_contact_phone;
  const hasSecondaryContact = opportunity.secondary_contact_fullname || opportunity.secondary_contact_email || opportunity.secondary_contact_phone;
  const hasContacts = hasPrimaryContact || hasSecondaryContact;

  const placeOfPerformance = [
    opportunity.pop_street_address,
    opportunity.pop_city,
    opportunity.pop_state,
    opportunity.pop_zip,
    opportunity.pop_country,
  ]
    .filter(Boolean)
    .join(', ');

  return (
    <div className="min-h-screen relative">
      {/* Background */}
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-6xl mx-auto px-6 py-8">
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 text-sm text-dark-400 hover:text-dark-200 transition-colors mb-6"
        >
          <ArrowLeft size={14} strokeWidth={1.5} />
          Back to results
        </Link>
        {/* Title Banner */}
        <div className="mb-8">
          {/* Status & Type Badges */}
          <div className="flex flex-wrap items-center gap-2 mb-4">
            {isExpired ? (
              <span className="inline-flex px-3 py-1 text-xs font-medium rounded-full bg-dark-700/50 text-dark-400 border border-dark-600/30">
                Closed
              </span>
            ) : (
              <span className="inline-flex px-3 py-1 text-xs font-medium rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Active
              </span>
            )}
            <span className="inline-flex px-3 py-1 text-xs font-medium rounded-full bg-dark-700/50 text-dark-300 border border-dark-600/30">
              {typeLabel}
            </span>
            {opportunity.set_aside_code && (
              <span
                className={`inline-flex px-3 py-1 text-xs font-medium rounded-full border ${
                  setAsideColors[opportunity.set_aside_code] || 'bg-dark-700/50 text-dark-400 border-dark-600/30'
                }`}
              >
                {opportunity.set_aside_description || opportunity.set_aside_code}
              </span>
            )}
          </div>

          {/* Title */}
          <h1 className="text-2xl font-semibold text-dark-50 leading-relaxed mb-3">{opportunity.title}</h1>

          {/* Solicitation Number */}
          <p className="font-mono text-sm text-dark-400 tracking-tight mb-6">
            {opportunity.solicitation_number || opportunity.notice_id}
          </p>

          {/* Action Buttons */}
          <div className="flex flex-wrap items-center gap-3">
            <a
              href={opportunity.ui_link || `https://sam.gov/opp/${opportunity.notice_id}/view`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent hover:bg-accent-hover
                         text-white font-medium rounded-xl transition-all duration-200 text-sm
                         hover:shadow-lg hover:shadow-accent/20"
            >
              <ExternalLink size={16} strokeWidth={1.5} />
              View on SAM.gov
            </a>
            <button
              onClick={handleCopyLink}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl border border-dark-700/50 bg-dark-800/30
                         text-dark-300 hover:text-dark-100 hover:border-dark-600/50 hover:bg-dark-800/50
                         transition-all duration-200 text-sm font-medium"
            >
              {copied ? <Check size={16} strokeWidth={1.5} /> : <Copy size={16} strokeWidth={1.5} />}
              {copied ? 'Copied' : 'Copy Link'}
            </button>
          </div>
        </div>

        {/* Key Dates Bar */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
          <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4">
            <div className="flex items-center gap-2 text-dark-500 text-xs uppercase tracking-wider mb-2">
              <Calendar size={14} strokeWidth={1.5} />
              Posted Date
            </div>
            <p className="text-dark-100 font-medium">{formatDate(opportunity.posted_date)}</p>
          </div>

          <div
            className={`rounded-xl p-4 border ${
              isExpired
                ? 'bg-dark-800/30 border-dark-700/50'
                : isUrgent
                  ? 'bg-red-500/5 border-red-500/20'
                  : 'bg-dark-900/30 border-dark-800/50'
            }`}
          >
            <div
              className={`flex items-center gap-2 text-xs uppercase tracking-wider mb-2 ${
                isUrgent ? 'text-red-400' : 'text-dark-500'
              }`}
            >
              <Clock size={14} strokeWidth={1.5} />
              Response Deadline
            </div>
            <p className={`font-medium ${isExpired ? 'text-dark-500' : isUrgent ? 'text-red-400' : 'text-dark-100'}`}>
              {formatDateTime(opportunity.response_deadline)}
            </p>
            {daysUntil !== null && !isExpired && (
              <p className={`text-xs mt-1 ${isUrgent ? 'text-red-400/70' : 'text-dark-500'}`}>
                {daysUntil} day{daysUntil !== 1 ? 's' : ''} remaining
              </p>
            )}
          </div>

          <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4">
            <div className="flex items-center gap-2 text-dark-500 text-xs uppercase tracking-wider mb-2">
              <Calendar size={14} strokeWidth={1.5} />
              Archive Date
            </div>
            <p className="text-dark-100 font-medium">{formatDate(opportunity.archive_date)}</p>
          </div>
        </div>

        {/* Content Sections */}
        <div className="space-y-4">
          {/* Description */}
          {opportunity.description && (
            <Section icon={FileText} title="Description" defaultOpen={true}>
              <div className="flex items-center justify-end mb-2">
                <button
                  onClick={() => {
                    setFormatted(!formatted);
                    localStorage.setItem('govtrove_format_desc', String(!formatted));
                  }}
                  className="text-xs text-dark-500 hover:text-dark-300 transition-colors flex items-center gap-1"
                >
                  {formatted ? <AlignLeft size={12} /> : <Pilcrow size={12} />}
                  {formatted ? 'Raw text' : 'Format text'}
                </button>
              </div>
              <div className="text-sm text-dark-300 leading-relaxed whitespace-pre-wrap">
                {formatted ? formatDescription(opportunity.description) : opportunity.description}
              </div>
            </Section>
          )}

          {/* General Information */}
          <Section icon={FileText} title="General Information" defaultOpen={true}>
            <dl>
              <InfoRow label="Notice Type" value={typeLabel} />
              {opportunity.base_type && <InfoRow label="Base Type" value={opportunity.base_type} />}
              <InfoRow label="Solicitation Number" value={opportunity.solicitation_number || '—'} mono />
              <InfoRow label="Notice ID" value={opportunity.notice_id} mono />
              {opportunity.set_aside_code && (
                <InfoRow
                  label="Set-Aside"
                  value={opportunity.set_aside_description || opportunity.set_aside_code}
                />
              )}
              <InfoRow label="NAICS Code" value={opportunity.naics_code || '—'} mono />
              {opportunity.naics_codes && opportunity.naics_codes.length > 1 && (
                <InfoRow label="Additional NAICS" value={opportunity.naics_codes.slice(1).join(', ')} mono />
              )}
              <InfoRow label="Classification Code" value={opportunity.classification_code || '—'} mono />
            </dl>
          </Section>

          {/* Solicitation History */}
          {history && history.total_notices > 1 && (
            <Section
              icon={GitBranch}
              title={`Solicitation History (${history.total_notices})`}
              defaultOpen={history.total_notices <= 10}
            >
              <SolicitationTimeline history={history} currentId={opportunity.id} />
            </Section>
          )}

          {/* Contracting Office */}
          <Section icon={Building2} title="Contracting Office" defaultOpen={true}>
            <dl>
              <InfoRow label="Department" value={deptHierarchy.department} />
              {deptHierarchy.subTier && <InfoRow label="Sub-Tier" value={deptHierarchy.subTier} />}
              {deptHierarchy.office && <InfoRow label="Office" value={deptHierarchy.office} />}
            </dl>
            {opportunity.department && (
              <p className="mt-3 text-xs text-dark-500 font-mono break-all">{opportunity.department}</p>
            )}
          </Section>

          {/* Contacts */}
          {hasContacts && (
            <Section icon={Users} title="Contacts" defaultOpen={true}>
              <div className="space-y-4">
                {hasPrimaryContact && (
                  <ContactCard
                    label="Primary Contact"
                    name={opportunity.primary_contact_fullname}
                    title={opportunity.primary_contact_title}
                    email={opportunity.primary_contact_email}
                    phone={opportunity.primary_contact_phone}
                    fax={opportunity.primary_contact_fax}
                  />
                )}
                {hasSecondaryContact && (
                  <ContactCard
                    label="Secondary Contact"
                    name={opportunity.secondary_contact_fullname}
                    title={opportunity.secondary_contact_title}
                    email={opportunity.secondary_contact_email}
                    phone={opportunity.secondary_contact_phone}
                    fax={opportunity.secondary_contact_fax}
                  />
                )}
              </div>
            </Section>
          )}

          {/* Place of Performance */}
          {placeOfPerformance && (
            <Section icon={MapPin} title="Place of Performance" defaultOpen={true}>
              <dl>
                {opportunity.pop_street_address && (
                  <InfoRow label="Address" value={opportunity.pop_street_address} />
                )}
                {opportunity.pop_city && <InfoRow label="City" value={opportunity.pop_city} />}
                {opportunity.pop_state && <InfoRow label="State" value={opportunity.pop_state} />}
                {opportunity.pop_zip && <InfoRow label="ZIP Code" value={opportunity.pop_zip} mono />}
                {opportunity.pop_country && <InfoRow label="Country" value={opportunity.pop_country} />}
              </dl>
            </Section>
          )}

          {/* Award Information */}
          {hasAward && (
            <Section icon={Award} title="Award Information" defaultOpen={true}>
              <dl>
                {opportunity.award_number && <InfoRow label="Award Number" value={opportunity.award_number} mono />}
                {opportunity.award_amount && (
                  <InfoRow label="Award Amount" value={formatCurrency(opportunity.award_amount)} />
                )}
                {opportunity.award_date && <InfoRow label="Award Date" value={formatDate(opportunity.award_date)} />}
                {opportunity.awardee_name && <InfoRow label="Awardee" value={opportunity.awardee_name} />}
                {opportunity.awardee_uei && <InfoRow label="Awardee UEI" value={opportunity.awardee_uei} mono />}
              </dl>
            </Section>
          )}

          {/* Attachments / Resources */}
          {opportunity.resource_links && opportunity.resource_links.length > 0 && (
            <Section icon={FileText} title="Attachments & Links" defaultOpen={true}>
              <div className="space-y-2">
                {opportunity.resource_links.map((link, idx) => (
                  <a
                    key={idx}
                    href={link}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-3 p-3 rounded-lg bg-dark-800/30 border border-dark-700/30
                               hover:bg-dark-800/50 hover:border-dark-600/50 transition-all duration-200 group"
                  >
                    <ExternalLink
                      size={14}
                      strokeWidth={1.5}
                      className="flex-shrink-0 text-dark-500 group-hover:text-accent"
                    />
                    <span className="text-sm text-dark-300 group-hover:text-accent truncate">{link}</span>
                  </a>
                ))}
              </div>
            </Section>
          )}
        </div>

        {/* Footer Meta */}
        <div className="mt-8 pt-6 border-t border-dark-800/50">
          <div className="flex flex-wrap items-center justify-between gap-4 text-xs text-dark-500">
            <div className="flex items-center gap-4">
              <span>
                Source: <span className="text-dark-400">{opportunity.data_source || 'SAM.gov'}</span>
              </span>
              <span>
                Last Updated: <span className="text-dark-400">{formatDate(opportunity.updated_at)}</span>
              </span>
            </div>
            <span className="font-mono">{opportunity.notice_id}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
