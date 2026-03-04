import { useState, useEffect, useCallback } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Shield, Bell, Search, FileText, AlertCircle, ChevronLeft, ChevronRight, RotateCw, Plus } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import {
  getAdminNotifications, getAdminApiKeys, getAdminSamgovRequests,
  getAdminApiKeyUsage, getAdminPipelineRuns, getAdminSearchEvents, getAdminAnalytics,
  getAdminEmailPreferences, updateAdminEmailPreference, getAdminSentEmails, adminResendEmail,
  adminSendNewEmail, type AdminSendNewEmailInput,
  type AdminUser, type AdminApiKey, type AdminSamgovRequest,
  type UsageBucket, type PipelineExecution, type AdminSearchEvent,
  type SearchAnalytics, type AdminEmailPreference, type AdminSentEmail,
} from '../services/api';
import type { UserUpdate } from '../types/api';
import { useAppAuth } from '../contexts/AuthContext';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatDateTime(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

const UPDATE_TYPE_LABELS: Record<string, string> = {
  saved_search_matches: 'Search Match',
  opportunity_amended: 'Amendment',
  opportunity_changed: 'Change',
};

function UpdateTypeIcon({ type }: { type: string }) {
  switch (type) {
    case 'saved_search_matches': return <Search size={14} className="text-accent" />;
    case 'opportunity_amended': return <FileText size={14} className="text-amber-400" />;
    case 'opportunity_changed': return <AlertCircle size={14} className="text-blue-400" />;
    default: return <Bell size={14} className="text-dark-400" />;
  }
}

function UsersTab({ users }: { users: AdminUser[] }) {
  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">
        Users <span className="text-dark-500 text-sm font-normal">({users.length})</span>
      </h2>
      <div className="overflow-x-auto rounded-xl border border-dark-700/50">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-dark-700/50 text-dark-400 text-left">
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Plan</th>
              <th className="px-4 py-3 font-medium">Admin</th>
              <th className="px-4 py-3 font-medium">Joined</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30">
                <td className="px-4 py-3 text-dark-200">{u.email}</td>
                <td className="px-4 py-3 text-dark-300">
                  {u.first_name || u.last_name
                    ? `${u.first_name} ${u.last_name}`.trim()
                    : <span className="text-dark-600">&mdash;</span>}
                </td>
                <td className="px-4 py-3">
                  <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-dark-800/60 border border-dark-700/30 text-dark-300">
                    {u.plan}
                  </span>
                </td>
                <td className="px-4 py-3">
                  {u.is_admin && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-emerald-500/15 border border-emerald-500/30 text-emerald-400">
                      <Shield size={12} />
                      Admin
                    </span>
                  )}
                </td>
                <td className="px-4 py-3 text-dark-400">{formatDate(u.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function NotificationsTab({ users, getToken }: { users: AdminUser[]; getToken: () => Promise<string> }) {
  const [selectedUserId, setSelectedUserId] = useState<number | ''>('');
  const [updates, setUpdates] = useState<UserUpdate[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const limit = 50;

  const fetchNotifications = useCallback(async (userId: number, p: number) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminNotifications(token, userId, p);
      setUpdates(res.updates || []);
      setTotal(res.total);
      setPage(p);
    } catch {
      setUpdates([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => {
    if (selectedUserId === '') return;
    fetchNotifications(selectedUserId, 1);
  }, [selectedUserId, fetchNotifications]);

  const totalPages = Math.ceil(total / limit);
  const selectedUser = users.find(u => u.id === selectedUserId);

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">Notifications</h2>

      <div className="mb-6">
        <label className="block text-sm text-dark-400 mb-2">Select user</label>
        <select
          value={selectedUserId}
          onChange={(e) => setSelectedUserId(e.target.value ? Number(e.target.value) : '')}
          className="w-full max-w-sm px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg
                     text-dark-100 focus:outline-none focus:border-accent/50"
        >
          <option value="">Choose a user...</option>
          {users.map((u) => (
            <option key={u.id} value={u.id}>
              {u.email}{u.first_name ? ` (${u.first_name} ${u.last_name})`.trimEnd() : ''}
            </option>
          ))}
        </select>
      </div>

      {selectedUserId !== '' && (
        <>
          <div className="flex items-center justify-between mb-3">
            <p className="text-sm text-dark-400">
              {loading ? 'Loading...' : `${total} notification${total !== 1 ? 's' : ''} for ${selectedUser?.email}`}
            </p>
            {totalPages > 1 && (
              <div className="flex items-center gap-2 text-sm">
                <button
                  onClick={() => fetchNotifications(selectedUserId as number, page - 1)}
                  disabled={page <= 1 || loading}
                  className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  <ChevronLeft size={16} />
                </button>
                <span className="text-dark-400">{page} / {totalPages}</span>
                <button
                  onClick={() => fetchNotifications(selectedUserId as number, page + 1)}
                  disabled={page >= totalPages || loading}
                  className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  <ChevronRight size={16} />
                </button>
              </div>
            )}
          </div>

          {!loading && updates.length === 0 && (
            <p className="text-dark-500 text-sm py-8 text-center">No notifications for this user.</p>
          )}

          {updates.length > 0 && (
            <div className="overflow-x-auto rounded-xl border border-dark-700/50">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                    <th className="px-4 py-3 font-medium w-32">Type</th>
                    <th className="px-4 py-3 font-medium">Summary</th>
                    <th className="px-4 py-3 font-medium w-24">Read</th>
                    <th className="px-4 py-3 font-medium w-44">Date</th>
                  </tr>
                </thead>
                <tbody>
                  {updates.map((u) => (
                    <tr key={u.id} className={`border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 ${!u.is_read ? 'bg-accent/[0.03]' : ''}`}>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center gap-1.5 text-dark-300">
                          <UpdateTypeIcon type={u.update_type} />
                          <span className="text-xs">{UPDATE_TYPE_LABELS[u.update_type] || u.update_type}</span>
                        </span>
                      </td>
                      <td className="px-4 py-3 text-dark-200 max-w-md truncate">{u.summary}</td>
                      <td className="px-4 py-3">
                        {u.is_read
                          ? <span className="text-dark-600 text-xs">Read</span>
                          : <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-accent/15 border border-accent/30 text-accent">Unread</span>
                        }
                      </td>
                      <td className="px-4 py-3 text-dark-400 text-xs">{formatDateTime(u.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function ApiKeysTab({ getToken }: { getToken: () => Promise<string> }) {
  const [keys, setKeys] = useState<AdminApiKey[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const token = await getToken();
        setKeys(await getAdminApiKeys(token));
      } catch { /* empty */ }
      finally { setLoading(false); }
    })();
  }, [getToken]);

  if (loading) return <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>;

  const now = new Date();

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">
        API Keys <span className="text-dark-500 text-sm font-normal">({keys.length})</span>
      </h2>
      <div className="overflow-x-auto rounded-xl border border-dark-700/50">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-dark-700/50 text-dark-400 text-left">
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Key Hash</th>
              <th className="px-4 py-3 font-medium">Daily Limit</th>
              <th className="px-4 py-3 font-medium">Created</th>
              <th className="px-4 py-3 font-medium">Expires</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((k) => {
              const expires = new Date(k.expires_at);
              const isExpired = expires < now;
              const expiresSoon = !isExpired && expires.getTime() - now.getTime() < 30 * 86400000;
              return (
                <tr key={k.key_hash} className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-dark-200">{k.email}</td>
                  <td className="px-4 py-3 font-mono text-xs text-dark-300">{k.key_hash}</td>
                  <td className="px-4 py-3 text-dark-300">{k.daily_limit.toLocaleString()}</td>
                  <td className="px-4 py-3 text-dark-400">{formatDate(k.created_at)}</td>
                  <td className="px-4 py-3">
                    <span className="text-dark-400">{formatDate(k.expires_at)}</span>
                    {isExpired && (
                      <span className="ml-2 inline-flex px-2 py-0.5 rounded text-xs font-medium bg-red-500/15 border border-red-500/30 text-red-400">
                        Expired
                      </span>
                    )}
                    {expiresSoon && (
                      <span className="ml-2 inline-flex px-2 py-0.5 rounded text-xs font-medium bg-amber-500/15 border border-amber-500/30 text-amber-400">
                        Expiring soon
                      </span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function SamgovRequestsTab({ getToken }: { getToken: () => Promise<string> }) {
  const [requests, setRequests] = useState<AdminSamgovRequest[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const limit = 50;

  const fetchPage = useCallback(async (p: number) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminSamgovRequests(token, p);
      setRequests(res.requests || []);
      setTotal(res.total);
      setPage(p);
    } catch {
      setRequests([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => { fetchPage(1); }, [fetchPage]);

  const totalPages = Math.ceil(total / limit);

  function statusColor(code: number | null) {
    if (!code) return 'text-dark-500';
    if (code < 300) return 'text-emerald-400';
    if (code < 400) return 'text-amber-400';
    return 'text-red-400';
  }

  return (
    <section>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-medium text-dark-200">
          SAM.gov Requests <span className="text-dark-500 text-sm font-normal">({total.toLocaleString()})</span>
        </h2>
        {totalPages > 1 && (
          <div className="flex items-center gap-2 text-sm">
            <button
              onClick={() => fetchPage(page - 1)}
              disabled={page <= 1 || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="text-dark-400">{page} / {totalPages}</span>
            <button
              onClick={() => fetchPage(page + 1)}
              disabled={page >= totalPages || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && requests.length === 0 && (
        <p className="text-dark-500 text-sm py-8 text-center">No requests found.</p>
      )}

      {!loading && requests.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-dark-700/50">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-3 font-medium w-44">Timestamp</th>
                <th className="px-4 py-3 font-medium">Endpoint</th>
                <th className="px-4 py-3 font-medium w-20">Method</th>
                <th className="px-4 py-3 font-medium w-20">Status</th>
                <th className="px-4 py-3 font-medium w-24">Time (ms)</th>
                <th className="px-4 py-3 font-medium w-20">OK</th>
                <th className="px-4 py-3 font-medium w-32">Key Hash</th>
                <th className="px-4 py-3 font-medium">Error</th>
              </tr>
            </thead>
            <tbody>
              {requests.map((req) => (
                <tr key={req.id} className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-dark-400 text-xs">{formatDateTime(req.request_timestamp)}</td>
                  <td className="px-4 py-3 text-dark-200 max-w-xs truncate font-mono text-xs">{req.endpoint}</td>
                  <td className="px-4 py-3 text-dark-300 text-xs">{req.method}</td>
                  <td className={`px-4 py-3 font-medium text-xs ${statusColor(req.http_status_code)}`}>
                    {req.http_status_code ?? '—'}
                  </td>
                  <td className="px-4 py-3 text-dark-300 text-xs">{req.response_time_ms?.toLocaleString() ?? '—'}</td>
                  <td className="px-4 py-3">
                    {req.success
                      ? <span className="text-emerald-400 text-xs">Yes</span>
                      : <span className="text-red-400 text-xs">No</span>
                    }
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-dark-400">{req.api_key_hash ?? '—'}</td>
                  <td className="px-4 py-3 text-red-400/80 text-xs max-w-xs truncate">{req.error_message ?? ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function UsageChartTab({ getToken }: { getToken: () => Promise<string> }) {
  const [keys, setKeys] = useState<AdminApiKey[]>([]);
  const [selectedKey, setSelectedKey] = useState('');
  const [days, setDays] = useState(7);
  const [buckets, setBuckets] = useState<UsageBucket[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const token = await getToken();
        const k = await getAdminApiKeys(token);
        setKeys(k);
        if (k.length > 0) setSelectedKey(k[0].key_hash);
      } catch { /* empty */ }
    })();
  }, [getToken]);

  useEffect(() => {
    if (!selectedKey) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const token = await getToken();
        const res = await getAdminApiKeyUsage(token, selectedKey, days);
        if (!cancelled) setBuckets(res.buckets || []);
      } catch {
        if (!cancelled) setBuckets([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [selectedKey, days, getToken]);

  const selectedKeyData = keys.find(k => k.key_hash === selectedKey);

  const chartData = buckets.map(b => ({
    ts: b.timestamp,
    success: b.success,
    failed: b.failed,
  }));

  const periodOptions = [
    { value: 7, label: '7d' },
    { value: 14, label: '14d' },
    { value: 30, label: '30d' },
  ];

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">API Usage (Rolling 24h)</h2>

      <div className="flex flex-wrap items-end gap-4 mb-6">
        <div>
          <label className="block text-sm text-dark-400 mb-2">API Key</label>
          <select
            value={selectedKey}
            onChange={(e) => setSelectedKey(e.target.value)}
            className="w-full max-w-sm px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg
                       text-dark-100 focus:outline-none focus:border-accent/50"
          >
            {keys.map((k) => (
              <option key={k.key_hash} value={k.key_hash}>
                {k.email} ({k.key_hash})
              </option>
            ))}
          </select>
        </div>
        <div className="flex gap-1">
          {periodOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setDays(opt.value)}
              className={`px-3 py-2 text-sm rounded-lg border transition-colors ${
                days === opt.value
                  ? 'border-accent/50 bg-accent/10 text-accent'
                  : 'border-dark-700/50 text-dark-400 hover:text-dark-200'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && chartData.length > 0 && (
        <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
          <ResponsiveContainer width="100%" height={350}>
            <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 20 }} stackOffset="none">
              <defs>
                <linearGradient id="successGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="failedGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#ef4444" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
              <XAxis
                dataKey="ts"
                tickLine={false}
                interval="preserveStartEnd"
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                tick={(props: any) => {
                  const { x, y, payload, index } = props;
                  const d = new Date(payload.value);
                  const time = d.toLocaleString('en-US', { hour: 'numeric', minute: '2-digit' });
                  const dateStr = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                  const prevDate = index > 0 ? new Date(chartData[index - 1].ts).toDateString() : '';
                  const showDate = index === 0 || d.toDateString() !== prevDate;
                  return (
                    <g transform={`translate(${x},${y})`}>
                      <text x={0} y={12} textAnchor="middle" fill="#6b7280" fontSize={11}>{time}</text>
                      {showDate && <text x={0} y={26} textAnchor="middle" fill="#9ca3af" fontSize={10}>{dateStr}</text>}
                    </g>
                  );
                }}
              />
              <YAxis
                tick={{ fill: '#6b7280', fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                allowDecimals={false}
              />
              <Tooltip
                labelFormatter={(val) => new Date(String(val)).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                contentStyle={{
                  backgroundColor: '#1f2937',
                  border: '1px solid rgba(255,255,255,0.1)',
                  borderRadius: '8px',
                  fontSize: '12px',
                  color: '#e5e7eb',
                }}
              />
              {selectedKeyData && (
                <ReferenceLine
                  y={selectedKeyData.daily_limit}
                  stroke="#ef4444"
                  strokeDasharray="6 3"
                  label={{ value: `Daily limit: ${selectedKeyData.daily_limit}`, fill: '#ef4444', fontSize: 11, position: 'right' }}
                />
              )}
              <Area
                type="monotone"
                dataKey="success"
                stackId="1"
                stroke="#10b981"
                fill="url(#successGradient)"
                strokeWidth={2}
                name="Success"
              />
              <Area
                type="monotone"
                dataKey="failed"
                stackId="1"
                stroke="#ef4444"
                fill="url(#failedGradient)"
                strokeWidth={2}
                name="Failed"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}

      {!loading && chartData.length === 0 && selectedKey && (
        <p className="text-dark-500 text-sm py-8 text-center">No usage data for this key.</p>
      )}
    </section>
  );
}

function PipelineRunsTab({ getToken }: { getToken: () => Promise<string> }) {
  const navigate = useNavigate();
  const [runs, setRuns] = useState<PipelineExecution[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const limit = 20;

  const fetchPage = useCallback(async (p: number) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminPipelineRuns(token, p);
      setRuns(res.runs || []);
      setTotal(res.total);
      setPage(p);
    } catch {
      setRuns([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => { fetchPage(1); }, [fetchPage]);

  const totalPages = Math.ceil(total / limit);

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

  function formatDuration(ms: number | null) {
    if (ms == null) return '—';
    if (ms < 1000) return `${ms}ms`;
    const s = ms / 1000;
    if (s < 60) return `${s.toFixed(1)}s`;
    return `${Math.floor(s / 60)}m ${Math.round(s % 60)}s`;
  }

  function stepSummary(exec: PipelineExecution) {
    const failed = exec.steps.filter(s => s.status === 'failed' && s.is_latest);
    if (failed.length > 0) return failed.map(s => s.step_name).join(', ');
    const uniqueSteps = new Set(exec.steps.map(s => s.step_name)).size;
    return `${uniqueSteps} steps`;
  }

  return (
    <section>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-medium text-dark-200">
          Pipeline Runs <span className="text-dark-500 text-sm font-normal">({total.toLocaleString()})</span>
        </h2>
        {totalPages > 1 && (
          <div className="flex items-center gap-2 text-sm">
            <button
              onClick={() => fetchPage(page - 1)}
              disabled={page <= 1 || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="text-dark-400">{page} / {totalPages}</span>
            <button
              onClick={() => fetchPage(page + 1)}
              disabled={page >= totalPages || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && runs.length === 0 && (
        <p className="text-dark-500 text-sm py-8 text-center">No pipeline runs found.</p>
      )}

      {!loading && runs.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-dark-700/50">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Started</th>
                <th className="px-4 py-3 font-medium">Duration</th>
                <th className="px-4 py-3 font-medium">Steps</th>
                <th className="px-4 py-3 font-medium">Details</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((exec) => (
                    <tr
                      key={exec.execution_id}
                      onClick={() => navigate(`/admin/pipeline/${exec.execution_id}`)}
                      className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 cursor-pointer"
                    >
                      <td className="px-4 py-3">{statusBadge(exec.status)}</td>
                      <td className="px-4 py-3 text-dark-300 text-xs">{formatDateTime(exec.started_at)}</td>
                      <td className="px-4 py-3 text-dark-300 text-xs">{formatDuration(exec.duration_ms)}</td>
                      <td className="px-4 py-3 text-dark-400 text-xs">{exec.step_count}</td>
                      <td className="px-4 py-3 text-dark-400 text-xs max-w-xs truncate">{stepSummary(exec)}</td>
                    </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function SearchAnalyticsTab({ getToken }: { getToken: () => Promise<string> }) {
  const [analytics, setAnalytics] = useState<SearchAnalytics | null>(null);
  const [period, setPeriod] = useState('7d');
  const [analyticsLoading, setAnalyticsLoading] = useState(true);

  const [events, setEvents] = useState<AdminSearchEvent[]>([]);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventsPage, setEventsPage] = useState(1);
  const [emptyOnly, setEmptyOnly] = useState(false);
  const [eventsLoading, setEventsLoading] = useState(true);
  const eventsLimit = 50;

  useEffect(() => {
    let cancelled = false;
    setAnalyticsLoading(true);
    (async () => {
      try {
        const token = await getToken();
        const data = await getAdminAnalytics(token, period);
        if (!cancelled) setAnalytics(data);
      } catch {
        if (!cancelled) setAnalytics(null);
      } finally {
        if (!cancelled) setAnalyticsLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [getToken, period]);

  const fetchEvents = useCallback(async (p: number, empty: boolean) => {
    setEventsLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminSearchEvents(token, p, empty);
      setEvents(res.events || []);
      setEventsTotal(res.total);
      setEventsPage(p);
    } catch {
      setEvents([]);
      setEventsTotal(0);
    } finally {
      setEventsLoading(false);
    }
  }, [getToken]);

  useEffect(() => { fetchEvents(1, emptyOnly); }, [fetchEvents, emptyOnly]);

  const eventsTotalPages = Math.ceil(eventsTotal / eventsLimit);
  const periodOptions = [
    { value: '24h', label: '24h' },
    { value: '7d', label: '7d' },
    { value: '30d', label: '30d' },
  ];

  function truncateFilters(filters: string | null) {
    if (!filters) return '—';
    try {
      const parsed = JSON.parse(filters);
      const keys = Object.keys(parsed).filter(k => parsed[k] != null && parsed[k] !== '' && !(Array.isArray(parsed[k]) && parsed[k].length === 0));
      if (keys.length === 0) return '—';
      return keys.join(', ');
    } catch {
      return filters.slice(0, 50);
    }
  }

  return (
    <section className="space-y-8">
      {/* Analytics Summary */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-medium text-dark-200">Search Analytics</h2>
          <div className="flex gap-1">
            {periodOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => setPeriod(opt.value)}
                className={`px-3 py-1.5 text-sm rounded-lg border transition-colors ${
                  period === opt.value
                    ? 'border-accent/50 bg-accent/10 text-accent'
                    : 'border-dark-700/50 text-dark-400 hover:text-dark-200'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        {analyticsLoading && <p className="text-dark-400 text-sm py-4 text-center">Loading...</p>}

        {!analyticsLoading && analytics && (
          <>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
              <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
                <p className="text-xs text-dark-400 mb-1">Searches</p>
                <p className="text-2xl font-semibold text-dark-100">{analytics.event_counts.searches.toLocaleString()}</p>
              </div>
              <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
                <p className="text-xs text-dark-400 mb-1">Views</p>
                <p className="text-2xl font-semibold text-dark-100">{analytics.event_counts.views.toLocaleString()}</p>
              </div>
              <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
                <p className="text-xs text-dark-400 mb-1">CTR</p>
                <p className="text-2xl font-semibold text-dark-100">{analytics.click_stats.click_through_rate.toFixed(1)}%</p>
              </div>
              <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
                <p className="text-xs text-dark-400 mb-1">Saves</p>
                <p className="text-2xl font-semibold text-dark-100">{analytics.event_counts.saves.toLocaleString()}</p>
              </div>
            </div>

            <div className="grid sm:grid-cols-2 gap-4 mb-2">
              <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
                <h3 className="text-sm font-medium text-dark-300 mb-3">Popular Searches</h3>
                {analytics.popular_searches.length === 0 ? (
                  <p className="text-dark-500 text-xs">No data</p>
                ) : (
                  <ul className="space-y-1.5">
                    {analytics.popular_searches.slice(0, 5).map((s) => (
                      <li key={s.query} className="flex items-center justify-between text-xs">
                        <span className="text-dark-200 truncate mr-2">{s.query || '(empty)'}</span>
                        <span className="text-dark-400 tabular-nums">{s.count}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <div className="rounded-xl border border-dark-700/50 p-4 bg-dark-800/20">
                <h3 className="text-sm font-medium text-dark-300 mb-3">Empty Searches</h3>
                {analytics.zero_result_searches.length === 0 ? (
                  <p className="text-dark-500 text-xs">No data</p>
                ) : (
                  <ul className="space-y-1.5">
                    {analytics.zero_result_searches.slice(0, 5).map((s) => (
                      <li key={s.query} className="flex items-center justify-between text-xs">
                        <span className="text-dark-200 truncate mr-2">{s.query || '(empty)'}</span>
                        <span className="text-red-400 tabular-nums">{s.count}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Search Event Log */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-4">
            <h2 className="text-lg font-medium text-dark-200">
              Search Log <span className="text-dark-500 text-sm font-normal">({eventsTotal.toLocaleString()})</span>
            </h2>
            <label className="flex items-center gap-2 text-sm text-dark-400 cursor-pointer">
              <input
                type="checkbox"
                checked={emptyOnly}
                onChange={(e) => { setEmptyOnly(e.target.checked); setEventsPage(1); }}
                className="rounded border-dark-600 bg-dark-800/50 text-accent focus:ring-accent/50"
              />
              Empty searches only
            </label>
          </div>
          {eventsTotalPages > 1 && (
            <div className="flex items-center gap-2 text-sm">
              <button
                onClick={() => fetchEvents(eventsPage - 1, emptyOnly)}
                disabled={eventsPage <= 1 || eventsLoading}
                className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronLeft size={16} />
              </button>
              <span className="text-dark-400">{eventsPage} / {eventsTotalPages}</span>
              <button
                onClick={() => fetchEvents(eventsPage + 1, emptyOnly)}
                disabled={eventsPage >= eventsTotalPages || eventsLoading}
                className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronRight size={16} />
              </button>
            </div>
          )}
        </div>

        {eventsLoading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

        {!eventsLoading && events.length === 0 && (
          <p className="text-dark-500 text-sm py-8 text-center">No search events found.</p>
        )}

        {!eventsLoading && events.length > 0 && (
          <div className="overflow-x-auto rounded-xl border border-dark-700/50">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                  <th className="px-4 py-3 font-medium w-44">Time</th>
                  <th className="px-4 py-3 font-medium">Query</th>
                  <th className="px-4 py-3 font-medium">Filters</th>
                  <th className="px-4 py-3 font-medium w-24">Results</th>
                  <th className="px-4 py-3 font-medium w-32">User ID</th>
                </tr>
              </thead>
              <tbody>
                {events.map((ev) => {
                  const isZero = ev.total_results === 0 || ev.total_results == null;
                  return (
                    <tr key={ev.id} className={`border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 ${isZero ? 'bg-red-500/[0.04]' : ''}`}>
                      <td className="px-4 py-3 text-dark-400 text-xs">{formatDateTime(ev.created_at)}</td>
                      <td className="px-4 py-3 text-dark-200 text-xs max-w-xs truncate">{ev.query || <span className="text-dark-600">(empty)</span>}</td>
                      <td className="px-4 py-3 text-dark-400 text-xs max-w-xs truncate">{truncateFilters(ev.filters)}</td>
                      <td className={`px-4 py-3 text-xs font-medium ${isZero ? 'text-red-400' : 'text-dark-200'}`}>
                        {ev.total_results?.toLocaleString() ?? '—'}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-dark-400 truncate max-w-[8rem]">{ev.user_id ?? '—'}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </section>
  );
}

function EmailPrefsTab({ getToken }: { getToken: () => Promise<string> }) {
  const [prefs, setPrefs] = useState<AdminEmailPreference[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const token = await getToken();
        setPrefs(await getAdminEmailPreferences(token));
      } catch { /* empty */ }
      finally { setLoading(false); }
    })();
  }, [getToken]);

  async function togglePref(userId: number, field: 'search_alerts' | 'opportunity_alerts') {
    const user = prefs.find(p => p.user_id === userId);
    if (!user) return;
    const newValue = !user[field];
    const updated = { search_alerts: user.search_alerts, opportunity_alerts: user.opportunity_alerts, [field]: newValue };
    setPrefs(prev => prev.map(p => p.user_id === userId ? { ...p, [field]: newValue } : p));
    try {
      const token = await getToken();
      await updateAdminEmailPreference(token, userId, updated);
    } catch {
      setPrefs(prev => prev.map(p => p.user_id === userId ? { ...p, [field]: !newValue } : p));
    }
  }

  if (loading) return <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>;

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">
        Email Preferences <span className="text-dark-500 text-sm font-normal">({prefs.length})</span>
      </h2>
      <div className="overflow-x-auto rounded-xl border border-dark-700/50">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-dark-700/50 text-dark-400 text-left">
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium w-28">Search</th>
              <th className="px-4 py-3 font-medium w-28">Opportunity</th>
              <th className="px-4 py-3 font-medium w-32">Unsubscribed</th>
              <th className="px-4 py-3 font-medium w-24">Reason</th>
            </tr>
          </thead>
          <tbody>
            {prefs.map((p) => (
              <tr key={p.user_id} className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30">
                <td className="px-4 py-3 text-dark-200">{p.email}</td>
                <td className="px-4 py-3 text-dark-300">
                  {p.first_name || p.last_name
                    ? `${p.first_name} ${p.last_name}`.trim()
                    : <span className="text-dark-600">&mdash;</span>}
                </td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => togglePref(p.user_id, 'search_alerts')}
                    className={`relative w-9 h-5 rounded-full transition-colors duration-200 ${
                      p.search_alerts ? 'bg-emerald-500' : 'bg-dark-700'
                    }`}
                  >
                    <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform duration-200 ${
                      p.search_alerts ? 'translate-x-4' : ''
                    }`} />
                  </button>
                </td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => togglePref(p.user_id, 'opportunity_alerts')}
                    className={`relative w-9 h-5 rounded-full transition-colors duration-200 ${
                      p.opportunity_alerts ? 'bg-emerald-500' : 'bg-dark-700'
                    }`}
                  >
                    <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform duration-200 ${
                      p.opportunity_alerts ? 'translate-x-4' : ''
                    }`} />
                  </button>
                </td>
                <td className="px-4 py-3 text-dark-400 text-xs">
                  {p.unsubscribed_at ? formatDate(p.unsubscribed_at) : <span className="text-dark-600">&mdash;</span>}
                </td>
                <td className="px-4 py-3 text-dark-400 text-xs">{p.unsubscribe_reason ?? <span className="text-dark-600">&mdash;</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

interface TemplateField {
  name: string;
  label: string;
  type: 'text' | 'number' | 'select' | 'json';
  options?: string[];
  default?: string;
}

const UNSUB_DEFAULT = 'https://api.govtrove.com/api/unsubscribe?token=test&uid=1';

const TEMPLATE_FIELDS: Record<string, TemplateField[]> = {
  'welcome.html': [
    { name: 'Greeting', label: 'Greeting', type: 'text', default: 'Hi there,' },
  ],
  'opportunity_update.html': [
    { name: 'ChangeType', label: 'ChangeType', type: 'select', options: ['amendment', 'field_changes'], default: 'amendment' },
    { name: 'OpportunityTitle', label: 'OpportunityTitle', type: 'text', default: 'Sample Opportunity Title' },
    { name: 'SolicitationNumber', label: 'SolicitationNumber', type: 'text', default: 'W912345-26-R-0001' },
    { name: 'OpportunityURL', label: 'OpportunityURL', type: 'text', default: 'https://app.govtrove.com/opportunities/1' },
    { name: 'Changes', label: 'Changes', type: 'json', default: JSON.stringify([{ Field: "Response Date", Old: "2026-03-01", New: "2026-04-01" }], null, 2) },
    { name: 'UnsubscribeURL', label: 'UnsubscribeURL', type: 'text', default: UNSUB_DEFAULT },
  ],
  'search_results.html': [
    { name: 'MatchCount', label: 'MatchCount', type: 'number', default: '3' },
    { name: 'SearchName', label: 'SearchName', type: 'text', default: 'My Saved Search' },
    { name: 'SearchURL', label: 'SearchURL', type: 'text', default: 'https://app.govtrove.com/?q=logistics' },
    { name: 'Opportunities', label: 'Opportunities', type: 'json', default: JSON.stringify([{ Title: "Logistics Support Services", URL: "https://app.govtrove.com/opportunities/1", Department: "Department of the Army", Deadline: "Apr 15, 2026" }, { Title: "IT Infrastructure Modernization", URL: "https://app.govtrove.com/opportunities/2", Department: "Department of the Air Force", Deadline: "May 1, 2026" }], null, 2) },
    { name: 'HasMore', label: 'HasMore', type: 'select', options: ['true', 'false'], default: 'false' },
    { name: 'RemainingCount', label: 'RemainingCount', type: 'number', default: '0' },
    { name: 'UnsubscribeURL', label: 'UnsubscribeURL', type: 'text', default: UNSUB_DEFAULT },
  ],
  'digest.html': [
    { name: 'Greeting', label: 'Greeting', type: 'text', default: 'Hi there,' },
    { name: 'SearchAlerts', label: 'SearchAlerts', type: 'json', default: JSON.stringify([{ SearchName: "Logistics", MatchCount: 2, SearchURL: "https://app.govtrove.com/?q=logistics", TopOpportunities: [{ Title: "Logistics Support", URL: "https://app.govtrove.com/opportunities/1" }], HasMore: true, RemainingCount: 1 }], null, 2) },
    { name: 'OpportunityAlerts', label: 'OpportunityAlerts', type: 'json', default: JSON.stringify([{ OpportunityTitle: "Sample Opportunity", SolicitationNumber: "FA8750-26-R-0002", Summary: "Response deadline extended to April 2026.", OpportunityURL: "https://app.govtrove.com/opportunities/2" }], null, 2) },
    { name: 'UnsubscribeURL', label: 'UnsubscribeURL', type: 'text', default: UNSUB_DEFAULT },
  ],
};

const TEMPLATE_NAMES = Object.keys(TEMPLATE_FIELDS);

function getTemplateDefaults(templateName: string): Record<string, string> {
  const fields = TEMPLATE_FIELDS[templateName] || [];
  const defaults: Record<string, string> = {};
  for (const f of fields) {
    if (f.default !== undefined) defaults[f.name] = f.default;
  }
  return defaults;
}

function SentEmailsTab({ getToken }: { getToken: () => Promise<string> }) {
  const [emails, setEmails] = useState<AdminSentEmail[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [resendingId, setResendingId] = useState<string | null>(null);
  const [resendEmail, setResendEmail] = useState('');
  const [resendStatus, setResendStatus] = useState<Record<string, 'idle' | 'loading' | 'success' | 'error'>>({});
  const limit = 50;

  const { user } = useAppAuth();

  const [showCompose, setShowCompose] = useState(false);
  const [composeTemplate, setComposeTemplate] = useState(TEMPLATE_NAMES[0]);
  const [composeToEmail, setComposeToEmail] = useState(user?.email || '');
  const [composeSubject, setComposeSubject] = useState('');
  const [composeFields, setComposeFields] = useState<Record<string, string>>(() => getTemplateDefaults(TEMPLATE_NAMES[0]));
  const [composeSending, setComposeSending] = useState(false);
  const [composeStatus, setComposeStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [composeError, setComposeError] = useState('');

  const fetchPage = useCallback(async (p: number) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminSentEmails(token, p);
      setEmails(res.emails || []);
      setTotal(res.total);
      setPage(p);
    } catch {
      setEmails([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => { fetchPage(1); }, [fetchPage]);

  function handleTemplateChange(name: string) {
    setComposeTemplate(name);
    setComposeFields(getTemplateDefaults(name));
    setComposeStatus('idle');
    setComposeError('');
  }

  async function handleComposeSend() {
    setComposeError('');
    const fields = TEMPLATE_FIELDS[composeTemplate] || [];
    const templateData: Record<string, unknown> = {};
    for (const f of fields) {
      const raw = composeFields[f.name] || '';
      if (f.type === 'json') {
        if (raw.trim()) {
          try { templateData[f.name] = JSON.parse(raw); } catch {
            setComposeError(`Invalid JSON in ${f.label}`);
            return;
          }
        }
      } else if (f.type === 'number') {
        templateData[f.name] = raw ? parseInt(raw, 10) : 0;
      } else if (f.type === 'select' && (raw === 'true' || raw === 'false')) {
        templateData[f.name] = raw === 'true';
      } else {
        templateData[f.name] = raw;
      }
    }

    setComposeSending(true);
    try {
      const token = await getToken();
      const input: AdminSendNewEmailInput = {
        template_name: composeTemplate,
        to_email: composeToEmail,
        subject: composeSubject,
        template_data: templateData,
      };
      await adminSendNewEmail(token, input);
      setComposeStatus('success');
      fetchPage(1);
      setTimeout(() => {
        setShowCompose(false);
        setComposeToEmail(user?.email || '');
        setComposeSubject('');
        setComposeTemplate(TEMPLATE_NAMES[0]);
        setComposeFields(getTemplateDefaults(TEMPLATE_NAMES[0]));
        setComposeStatus('idle');
      }, 1500);
    } catch {
      setComposeStatus('error');
      setComposeError('Failed to send email');
    } finally {
      setComposeSending(false);
    }
  }

  const totalPages = Math.ceil(total / limit);

  function statusBadge(status: string) {
    switch (status) {
      case 'delivered':
        return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-emerald-500/15 border border-emerald-500/30 text-emerald-400">delivered</span>;
      case 'bounced':
        return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-red-500/15 border border-red-500/30 text-red-400">bounced</span>;
      case 'complained':
        return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-amber-500/15 border border-amber-500/30 text-amber-400">complained</span>;
      default:
        return <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-dark-800/60 border border-dark-700/30 text-dark-300">{status}</span>;
    }
  }

  async function handleResend(id: string) {
    if (!resendEmail) return;
    setResendStatus(prev => ({ ...prev, [id]: 'loading' }));
    try {
      const token = await getToken();
      await adminResendEmail(token, id, resendEmail);
      setResendStatus(prev => ({ ...prev, [id]: 'success' }));
      setResendingId(null);
      setResendEmail('');
    } catch {
      setResendStatus(prev => ({ ...prev, [id]: 'error' }));
    }
  }

  const composeFieldDefs = TEMPLATE_FIELDS[composeTemplate] || [];
  const inputClass = 'w-full px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50 placeholder:text-dark-600';

  return (
    <section>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-medium text-dark-200">
            Sent Emails <span className="text-dark-500 text-sm font-normal">({total.toLocaleString()})</span>
          </h2>
          <button
            onClick={() => setShowCompose(!showCompose)}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent/10 text-accent hover:bg-accent/20 border border-accent/20 transition-colors"
          >
            <Plus size={14} />
            Send New Email
          </button>
        </div>
        {totalPages > 1 && (
          <div className="flex items-center gap-2 text-sm">
            <button
              onClick={() => fetchPage(page - 1)}
              disabled={page <= 1 || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="text-dark-400">{page} / {totalPages}</span>
            <button
              onClick={() => fetchPage(page + 1)}
              disabled={page >= totalPages || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>

      {showCompose && (
        <div className="bg-dark-800/30 border border-dark-700/50 rounded-xl p-4 mb-4">
          <div className="grid grid-cols-3 gap-3 mb-3">
            <div>
              <label className="block text-xs text-dark-400 mb-1">Template</label>
              <select
                value={composeTemplate}
                onChange={(e) => handleTemplateChange(e.target.value)}
                className={inputClass}
              >
                {TEMPLATE_NAMES.map((t) => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">Recipient</label>
              <input
                type="email"
                value={composeToEmail}
                onChange={(e) => setComposeToEmail(e.target.value)}
                placeholder="to@example.com"
                className={inputClass}
              />
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">Subject</label>
              <input
                type="text"
                value={composeSubject}
                onChange={(e) => setComposeSubject(e.target.value)}
                placeholder="Email subject"
                className={inputClass}
              />
            </div>
          </div>

          {composeFieldDefs.length > 0 && (
            <div className="grid grid-cols-2 gap-3 mb-3">
              {composeFieldDefs.map((f) => (
                <div key={f.name} className={f.type === 'json' ? 'col-span-2' : ''}>
                  <label className="block text-xs text-dark-400 mb-1">{f.label}</label>
                  {f.type === 'select' ? (
                    <select
                      value={composeFields[f.name] || ''}
                      onChange={(e) => setComposeFields(prev => ({ ...prev, [f.name]: e.target.value }))}
                      className={inputClass}
                    >
                      <option value="">Select...</option>
                      {f.options?.map((o) => <option key={o} value={o}>{o}</option>)}
                    </select>
                  ) : f.type === 'json' ? (
                    <textarea
                      rows={3}
                      value={composeFields[f.name] || ''}
                      onChange={(e) => setComposeFields(prev => ({ ...prev, [f.name]: e.target.value }))}
                      placeholder="JSON value"
                      className={inputClass + ' font-mono text-xs'}
                    />
                  ) : (
                    <input
                      type={f.type === 'number' ? 'number' : 'text'}
                      value={composeFields[f.name] || ''}
                      onChange={(e) => setComposeFields(prev => ({ ...prev, [f.name]: e.target.value }))}
                      className={inputClass}
                    />
                  )}
                </div>
              ))}
            </div>
          )}

          <div className="flex items-center gap-3">
            <button
              onClick={handleComposeSend}
              disabled={composeSending || !composeToEmail || !composeSubject}
              className="px-4 py-2 text-sm font-medium rounded-lg bg-accent text-dark-950 hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {composeSending ? 'Sending...' : 'Send'}
            </button>
            <button
              onClick={() => { setShowCompose(false); setComposeStatus('idle'); setComposeError(''); }}
              className="px-4 py-2 text-sm text-dark-400 hover:text-dark-200 transition-colors"
            >
              Cancel
            </button>
            {composeStatus === 'success' && <span className="text-emerald-400 text-sm">Sent</span>}
            {composeError && <span className="text-red-400 text-sm">{composeError}</span>}
          </div>
        </div>
      )}

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && emails.length === 0 && (
        <p className="text-dark-500 text-sm py-8 text-center">No emails sent yet.</p>
      )}

      {!loading && emails.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-dark-700/50">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-3 font-medium w-44">Date</th>
                <th className="px-4 py-3 font-medium">To</th>
                <th className="px-4 py-3 font-medium w-28">Type</th>
                <th className="px-4 py-3 font-medium">Subject</th>
                <th className="px-4 py-3 font-medium w-24">Status</th>
                <th className="px-4 py-3 font-medium w-36">Opened</th>
                <th className="px-4 py-3 font-medium w-36">Clicked</th>
                <th className="px-4 py-3 font-medium w-24">Action</th>
              </tr>
            </thead>
            <tbody>
              {emails.map((e) => (
                <tr key={e.id} className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-dark-400 text-xs">{formatDateTime(e.created_at)}</td>
                  <td className="px-4 py-3 text-dark-200 text-xs">{e.to_email}</td>
                  <td className="px-4 py-3 text-dark-300 text-xs">{e.email_type}</td>
                  <td className="px-4 py-3 text-dark-200 text-xs max-w-xs truncate">{e.subject}</td>
                  <td className="px-4 py-3">{statusBadge(e.status)}</td>
                  <td className="px-4 py-3 text-dark-400 text-xs">{e.opened_at ? formatDateTime(e.opened_at) : '—'}</td>
                  <td className="px-4 py-3 text-dark-400 text-xs">{e.clicked_at ? formatDateTime(e.clicked_at) : '—'}</td>
                  <td className="px-4 py-3">
                    {resendingId === e.id ? (
                      <div className="flex items-center gap-1">
                        <input
                          type="email"
                          value={resendEmail}
                          onChange={(ev) => setResendEmail(ev.target.value)}
                          placeholder="email"
                          className="w-32 px-2 py-1 text-xs bg-dark-800/50 border border-dark-700/50 rounded text-dark-100 focus:outline-none focus:border-accent/50"
                        />
                        <button
                          onClick={() => handleResend(e.id)}
                          disabled={resendStatus[e.id] === 'loading'}
                          className="p-1 text-accent hover:text-accent/80 disabled:opacity-50 transition-colors"
                          title="Send"
                        >
                          <RotateCw size={14} className={resendStatus[e.id] === 'loading' ? 'animate-spin' : ''} />
                        </button>
                        <button
                          onClick={() => { setResendingId(null); setResendEmail(''); }}
                          className="text-xs text-dark-500 hover:text-dark-300"
                        >
                          &times;
                        </button>
                      </div>
                    ) : resendStatus[e.id] === 'success' ? (
                      <span className="text-emerald-400 text-xs">Sent</span>
                    ) : (
                      <button
                        onClick={() => setResendingId(e.id)}
                        className="text-xs text-dark-400 hover:text-accent transition-colors"
                      >
                        Resend
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

type Tab = 'users' | 'notifications' | 'api-keys' | 'samgov-requests' | 'usage' | 'pipeline' | 'searches' | 'email-prefs' | 'sent-emails';

export default function AdminPage() {
  const [searchParams] = useSearchParams();
  const activeTab = (searchParams.get('tab') as Tab) || 'users';

  return (
    <AdminLayout>
      <AdminPageContent activeTab={activeTab} />
    </AdminLayout>
  );
}

function AdminPageContent({ activeTab }: { activeTab: Tab }) {
  const { users, getToken } = useAdminContext();

  return (
    <div className="max-w-6xl mx-auto px-6 py-10">
      {activeTab === 'users' && <UsersTab users={users} />}
      {activeTab === 'notifications' && <NotificationsTab users={users} getToken={getToken} />}
      {activeTab === 'api-keys' && <ApiKeysTab getToken={getToken} />}
      {activeTab === 'samgov-requests' && <SamgovRequestsTab getToken={getToken} />}
      {activeTab === 'usage' && <UsageChartTab getToken={getToken} />}
      {activeTab === 'pipeline' && <PipelineRunsTab getToken={getToken} />}
      {activeTab === 'searches' && <SearchAnalyticsTab getToken={getToken} />}
      {activeTab === 'email-prefs' && <EmailPrefsTab getToken={getToken} />}
      {activeTab === 'sent-emails' && <SentEmailsTab getToken={getToken} />}
    </div>
  );
}
