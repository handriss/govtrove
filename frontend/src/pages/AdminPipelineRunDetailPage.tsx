import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ChevronLeft, CheckCircle2, XCircle, Clock, Minus } from 'lucide-react';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import {
  getAdminPipelineRunDetail,
  type PipelineRunDetailResponse,
  type IngestionRunDetail,
  type TableCounts,
  type OpportunityStats,
} from '../services/api';

function formatDateTime(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function formatDuration(ms: number | null) {
  if (ms == null) return '—';
  if (ms < 1000) return `${ms}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(1)}s`;
  return `${Math.floor(s / 60)}m ${Math.round(s % 60)}s`;
}

function statusBadge(status: string) {
  switch (status) {
    case 'completed':
      return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-emerald-500/15 border border-emerald-500/30 text-emerald-400">completed</span>;
    case 'failed':
      return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-red-500/15 border border-red-500/30 text-red-400">failed</span>;
    case 'running':
      return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-amber-500/15 border border-amber-500/30 text-amber-400">running</span>;
    default:
      return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-dark-800/60 border border-dark-700/30 text-dark-300">{status}</span>;
  }
}

function StepIcon({ status }: { status: 'completed' | 'failed' | 'running' | 'inferred' | 'none' }) {
  switch (status) {
    case 'completed': return <CheckCircle2 size={16} className="text-emerald-400" />;
    case 'failed': return <XCircle size={16} className="text-red-400" />;
    case 'running': return <Clock size={16} className="text-amber-400" />;
    case 'inferred': return <CheckCircle2 size={16} className="text-emerald-400/50" />;
    case 'none': return <Minus size={16} className="text-dark-600" />;
  }
}

function borderColor(status: string) {
  switch (status) {
    case 'completed':
    case 'inferred':
      return 'border-l-emerald-500/60';
    case 'failed':
      return 'border-l-red-500/60';
    case 'running':
      return 'border-l-amber-500/60';
    default:
      return 'border-l-dark-700/60';
  }
}

interface StepCardProps {
  name: string;
  status: 'completed' | 'failed' | 'running' | 'inferred' | 'none';
  duration: string;
  metrics?: string;
}

function StepCard({ name, status, duration, metrics }: StepCardProps) {
  const dimmed = status === 'none';
  return (
    <div className={`rounded-lg border border-dark-700/40 border-l-4 ${borderColor(status)} p-3 bg-dark-800/40 ${dimmed ? 'opacity-40' : ''}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <StepIcon status={status} />
          <span className="text-sm font-medium text-dark-200">{name}</span>
        </div>
        <span className="text-xs text-dark-400">{duration}</span>
      </div>
      {metrics && <p className="text-xs text-dark-400 mt-1.5 ml-6">{metrics}</p>}
    </div>
  );
}

function Connector() {
  return (
    <div className="flex justify-center py-0.5">
      <div className="w-px h-4 bg-dark-700/50" />
    </div>
  );
}

function StatCard({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <div className="rounded-lg border border-dark-700/40 p-3 bg-dark-800/40 text-center">
      <p className="text-xs text-dark-400 mb-1">{label}</p>
      <p className="text-lg font-semibold text-dark-200">{typeof value === 'number' ? value.toLocaleString() : value}</p>
      {sub && <p className="text-xs text-dark-500 mt-0.5">{sub}</p>}
    </div>
  );
}

function FlowSection({ data }: { data: PipelineRunDetailResponse }) {
  const { pipeline_run: pr, ingestion_runs: irs, table_counts: tc, opportunity_stats: os } = data;

  const csvRun = irs.find((ir: IngestionRunDetail) => ir.job_type === 'snapshot-csv' || ir.job_type === 'active-csv' || ir.job_type.includes('csv'));
  const apiRun = irs.find((ir: IngestionRunDetail) => ir.job_type === 'snapshot-api' || ir.job_type === 'api-enrich' || ir.job_type.includes('api'));

  const pipelineOk = pr.status === 'completed';
  const pipelineFailed = pr.status === 'failed';

  function stepStatus(ir?: IngestionRunDetail): 'completed' | 'failed' | 'running' | 'inferred' | 'none' {
    if (ir) return ir.status as 'completed' | 'failed' | 'running';
    return 'none';
  }

  function inferredStatus(): 'completed' | 'failed' | 'inferred' | 'none' {
    if (pipelineOk) return 'inferred';
    if (pipelineFailed) return 'failed';
    return 'none';
  }

  function irMetrics(ir?: IngestionRunDetail, tableName?: string, count?: number) {
    if (!ir) return undefined;
    const parts: string[] = [];
    if (ir.records_fetched != null) parts.push(`fetched: ${ir.records_fetched.toLocaleString()}`);
    if (count != null) parts.push(`${tableName}: ${count.toLocaleString()}`);
    if ((ir.records_failed ?? 0) > 0) parts.push(`failed: ${ir.records_failed!.toLocaleString()}`);
    return parts.join(' · ') || undefined;
  }

  const reconcileMetrics = [
    tc.snap_data_quality > 0 && `DQ: ${tc.snap_data_quality}`,
    tc.snap_reconcile_dq > 0 && `Reconcile DQ: ${tc.snap_reconcile_dq}`,
    os.total_affected > 0 && `Opportunities: ${os.total_affected.toLocaleString()}`,
  ].filter(Boolean).join(' · ') || undefined;

  return (
    <div className="space-y-0">
      <StepCard name="Download CSVs" status={inferredStatus()} duration="—" />
      <Connector />
      <StepCard
        name="Ingest CSV"
        status={stepStatus(csvRun)}
        duration={formatDuration(csvRun?.duration_ms ?? null)}
        metrics={irMetrics(csvRun, 'snap_csv', tc.snap_csv)}
      />
      <Connector />
      <StepCard
        name="Ingest API"
        status={stepStatus(apiRun)}
        duration={formatDuration(apiRun?.duration_ms ?? null)}
        metrics={irMetrics(apiRun, 'snap_api', tc.snap_api)}
      />
      <Connector />
      <StepCard
        name="Reconcile"
        status={inferredStatus()}
        duration="—"
        metrics={reconcileMetrics}
      />
      <Connector />
      <StepCard name="Generate Alerts" status={inferredStatus()} duration="—" />
    </div>
  );
}

function DatabaseImpact({ tc, os }: { tc: TableCounts; os: OpportunityStats }) {
  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-dark-300">Snapshot Tables</h3>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
        <StatCard label="snap_csv" value={tc.snap_csv} />
        <StatCard label="snap_api" value={tc.snap_api} />
        <StatCard label="DQ Issues" value={tc.snap_data_quality} />
        <StatCard label="Disappearances" value={tc.snap_disappearances} />
        <StatCard label="Reconcile DQ" value={tc.snap_reconcile_dq} />
      </div>

      <h3 className="text-sm font-medium text-dark-300 mt-6">Opportunities</h3>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        <StatCard label="Total Affected" value={os.total_affected} />
        <StatCard label="Inserted" value={os.inserted} sub="new" />
        <StatCard label="Updated" value={os.updated} sub="existing" />
        <StatCard label="Both Sources" value={os.from_both} />
        <StatCard label="CSV Only" value={os.from_csv_only} />
        <StatCard label="API Records" value={os.from_api} />
      </div>
    </div>
  );
}

function DetailContent() {
  const { id } = useParams<{ id: string }>();
  const { getToken } = useAdminContext();
  const [data, setData] = useState<PipelineRunDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const token = await getToken();
        const res = await getAdminPipelineRunDetail(token, id);
        if (!cancelled) setData(res);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [id, getToken]);

  if (loading) {
    return <p className="text-dark-400 text-sm py-12 text-center">Loading...</p>;
  }

  if (error || !data) {
    return (
      <div className="py-12 text-center">
        <p className="text-red-400 text-sm">{error === '404' ? 'Pipeline run not found.' : `Error: ${error}`}</p>
        <Link to="/admin?tab=pipeline" className="text-accent text-sm mt-2 inline-block hover:underline">Back to Pipeline Runs</Link>
      </div>
    );
  }

  const { pipeline_run: pr, table_counts: tc, opportunity_stats: os } = data;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <Link to="/admin?tab=pipeline" className="inline-flex items-center gap-1 text-sm text-dark-400 hover:text-accent transition-colors mb-4">
          <ChevronLeft size={14} />
          Pipeline Runs
        </Link>

        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-xl font-semibold text-dark-100">Pipeline Run</h1>
          {statusBadge(pr.status)}
        </div>

        <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-dark-400">
          <span>ID: <code className="text-dark-300">{pr.id.slice(0, 8)}</code></span>
          <span>Started: {formatDateTime(pr.started_at)}</span>
          {pr.completed_at && <span>Completed: {formatDateTime(pr.completed_at)}</span>}
          <span>Duration: {formatDuration(pr.duration_ms)}</span>
        </div>

        {pr.error_message && (
          <div className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
            {pr.error_message}
          </div>
        )}
      </div>

      {/* Step Functions Flow */}
      <div>
        <h2 className="text-sm font-medium text-dark-300 mb-3">Step Functions Flow</h2>
        <div className="max-w-md">
          <FlowSection data={data} />
        </div>
      </div>

      {/* Database Impact */}
      <div>
        <h2 className="text-sm font-medium text-dark-300 mb-3">Database Impact</h2>
        <DatabaseImpact tc={tc} os={os} />
      </div>
    </div>
  );
}

export default function AdminPipelineRunDetailPage() {
  return (
    <AdminLayout>
      <div className="p-6">
        <DetailContent />
      </div>
    </AdminLayout>
  );
}
