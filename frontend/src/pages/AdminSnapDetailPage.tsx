import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ChevronLeft } from 'lucide-react';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import {
  getAdminSnapCSVRecord,
  getAdminSnapArchivedCSVRecord,
  getAdminSnapAPIRecord,
  type SnapCSVRecord,
  type SnapAPIRecord,
} from '../services/api';

type SnapSource = 'csv' | 'archived-csv' | 'api';

const SOURCE_LABELS: Record<SnapSource, { label: string; color: string }> = {
  csv: { label: 'CSV', color: 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400' },
  'archived-csv': { label: 'Archived CSV', color: 'bg-amber-500/15 border-amber-500/30 text-amber-400' },
  api: { label: 'API', color: 'bg-blue-500/15 border-blue-500/30 text-blue-400' },
};

function formatDateTime(dateStr: string | null | undefined) {
  if (!dateStr) return '—';
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function InfoRow({ label, value }: { label: string; value: string | number | boolean | null | undefined }) {
  const display = value == null ? '—' : typeof value === 'boolean' ? (value ? 'Yes' : 'No') : String(value);
  return (
    <div className="flex justify-between py-1.5 border-b border-dark-700/30 last:border-0">
      <span className="text-xs text-dark-400 shrink-0 mr-4">{label}</span>
      <span className="text-xs text-dark-200 text-right break-all">{display}</span>
    </div>
  );
}

function DetailContent({ source }: { source: SnapSource }) {
  const { id } = useParams<{ id: string }>();
  const { getToken } = useAdminContext();
  const [csvData, setCsvData] = useState<SnapCSVRecord | null>(null);
  const [apiData, setApiData] = useState<SnapAPIRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const token = await getToken();
        const numId = Number(id);
        if (source === 'csv') {
          const res = await getAdminSnapCSVRecord(token, numId);
          if (!cancelled) setCsvData(res);
        } else if (source === 'archived-csv') {
          const res = await getAdminSnapArchivedCSVRecord(token, numId);
          if (!cancelled) setCsvData(res);
        } else {
          const res = await getAdminSnapAPIRecord(token, numId);
          if (!cancelled) setApiData(res);
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [id, source, getToken]);

  if (loading) {
    return <p className="text-dark-400 text-sm py-12 text-center">Loading...</p>;
  }

  if (error) {
    return (
      <div className="py-12 text-center">
        <p className="text-red-400 text-sm">{error === '404' ? 'Record not found.' : `Error: ${error}`}</p>
        <Link to="/admin" className="text-accent text-sm mt-2 inline-block hover:underline">Back to Admin</Link>
      </div>
    );
  }

  const isCSV = source !== 'api';
  const record = isCSV ? csvData : apiData;
  if (!record) {
    return (
      <div className="py-12 text-center">
        <p className="text-red-400 text-sm">Record not found.</p>
        <Link to="/admin" className="text-accent text-sm mt-2 inline-block hover:underline">Back to Admin</Link>
      </div>
    );
  }

  const { label, color } = SOURCE_LABELS[source];
  const noticeId = isCSV ? (record as SnapCSVRecord).notice_id : (record as SnapAPIRecord).notice_id;
  const snapshotDate = isCSV ? (record as SnapCSVRecord).snapshot_date : (record as SnapAPIRecord).snapshot_date;
  const rawData = isCSV ? (record as SnapCSVRecord).raw_data : (record as SnapAPIRecord).raw_data;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <Link to="/admin" className="inline-flex items-center gap-1 text-sm text-dark-400 hover:text-accent transition-colors mb-4">
          <ChevronLeft size={14} />
          Admin
        </Link>

        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-xl font-semibold text-dark-100">Snap Record</h1>
          <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium border ${color}`}>{label}</span>
        </div>

        <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-dark-400">
          <span>Notice: <code className="text-dark-300">{noticeId}</code></span>
          <span>Snapshot: {formatDateTime(snapshotDate)}</span>
        </div>
      </div>

      {/* Metadata */}
      <div className="rounded-lg border border-dark-700/40 bg-dark-800/40 p-4">
        <h2 className="text-sm font-medium text-dark-300 mb-3">Metadata</h2>
        <InfoRow label="Record ID" value={record.id} />
        <InfoRow label="Run ID" value={isCSV ? (record as SnapCSVRecord).run_id : (record as SnapAPIRecord).run_id} />
        <InfoRow label="Content Hash" value={isCSV ? (record as SnapCSVRecord).content_hash : (record as SnapAPIRecord).content_hash} />
        <InfoRow label="Snapshot Date" value={formatDateTime(snapshotDate)} />
        {isCSV && <InfoRow label="Download ID" value={(record as SnapCSVRecord).download_id} />}
        <InfoRow label="Created At" value={formatDateTime(isCSV ? (record as SnapCSVRecord).created_at : (record as SnapAPIRecord).created_at)} />
      </div>

      {/* Parsed Fields (CSV only) */}
      {isCSV && (
        <div className="rounded-lg border border-dark-700/40 bg-dark-800/40 p-4">
          <h2 className="text-sm font-medium text-dark-300 mb-3">Parsed Fields</h2>
          {(() => {
            const csv = record as SnapCSVRecord;
            return (
              <>
                <InfoRow label="Solicitation Number" value={csv.solicitation_number} />
                <InfoRow label="Title" value={csv.title} />
                <InfoRow label="Type" value={csv.type} />
                <InfoRow label="Base Type" value={csv.base_type} />
                <InfoRow label="Posted Date" value={formatDateTime(csv.posted_date)} />
                <InfoRow label="Response Deadline" value={formatDateTime(csv.response_deadline)} />
                <InfoRow label="Archive Date" value={csv.archive_date} />
                <InfoRow label="Archive Type" value={csv.archive_type} />
                <InfoRow label="Set-Aside Code" value={csv.set_aside_code} />
                <InfoRow label="NAICS Code" value={csv.naics_code} />
                <InfoRow label="Classification Code" value={csv.classification_code} />
                <InfoRow label="Active" value={csv.active} />
                <InfoRow label="Department" value={csv.department} />
                <InfoRow label="Sub-Tier" value={csv.sub_tier} />
                <InfoRow label="Office" value={csv.office} />
                <InfoRow label="CGAC" value={csv.cgac} />
                <InfoRow label="FPDS Code" value={csv.fpds_code} />
                <InfoRow label="AAC Code" value={csv.aac_code} />
                <InfoRow label="Award Number" value={csv.award_number} />
                <InfoRow label="Award Date" value={csv.award_date} />
                <InfoRow label="Award Amount" value={csv.award_amount != null ? `$${csv.award_amount.toLocaleString()}` : null} />
              </>
            );
          })()}
        </div>
      )}

      {/* Raw Data */}
      <div className="rounded-lg border border-dark-700/40 bg-dark-800/40 p-4">
        <h2 className="text-sm font-medium text-dark-300 mb-3">Raw Data</h2>
        <div className="overflow-auto max-h-[600px] rounded border border-dark-700/30 bg-dark-900/60 p-3">
          <pre className="text-xs text-dark-300 whitespace-pre-wrap break-words">{JSON.stringify(rawData, null, 2)}</pre>
        </div>
      </div>
    </div>
  );
}

export default function AdminSnapDetailPage({ source }: { source: SnapSource }) {
  return (
    <AdminLayout>
      <div className="p-6">
        <DetailContent source={source} />
      </div>
    </AdminLayout>
  );
}
