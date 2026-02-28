import { useState, useEffect, useCallback, Fragment } from 'react';
import { Link, Navigate, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Shield, Users, Bell, Search, FileText, AlertCircle, ChevronLeft, ChevronRight, Key, Globe, BarChart3, Activity, ChevronDown, ChevronUp } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import { useAppAuth } from '../contexts/AuthContext';
import {
  getAdminUsers, getAdminNotifications, getAdminApiKeys, getAdminSamgovRequests,
  getAdminApiKeyUsage, getAdminPipelineRuns, getAdminSearchEvents, getAdminAnalytics,
  type AdminUser, type AdminApiKey, type AdminSamgovRequest,
  type UsageBucket, type AdminPipelineRun, type AdminIngestionRun, type AdminSearchEvent,
  type SearchAnalytics,
} from '../services/api';
import type { UserUpdate } from '../types/api';

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
    time: new Date(b.timestamp).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }),
    count: b.count,
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
            <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="usageGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="rgb(var(--color-accent))" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="rgb(var(--color-accent))" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
              <XAxis
                dataKey="time"
                tick={{ fill: '#6b7280', fontSize: 11 }}
                tickLine={false}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={{ fill: '#6b7280', fontSize: 11 }}
                tickLine={false}
                axisLine={false}
              />
              <Tooltip
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
                  label={{ value: `Limit: ${selectedKeyData.daily_limit}`, fill: '#ef4444', fontSize: 11, position: 'right' }}
                />
              )}
              <Area
                type="monotone"
                dataKey="count"
                stroke="rgb(var(--color-accent))"
                fill="url(#usageGradient)"
                strokeWidth={2}
                name="Requests (24h)"
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
  const [runs, setRuns] = useState<AdminPipelineRun[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [fullOnly, setFullOnly] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const limit = 20;

  const fetchPage = useCallback(async (p: number, full: boolean) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminPipelineRuns(token, p, full);
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

  useEffect(() => { fetchPage(1, fullOnly); }, [fetchPage, fullOnly]);

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

  return (
    <section>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4">
          <h2 className="text-lg font-medium text-dark-200">
            Pipeline Runs <span className="text-dark-500 text-sm font-normal">({total.toLocaleString()})</span>
          </h2>
          <label className="flex items-center gap-2 text-sm text-dark-400 cursor-pointer">
            <input
              type="checkbox"
              checked={fullOnly}
              onChange={(e) => { setFullOnly(e.target.checked); setPage(1); }}
              className="rounded border-dark-600 bg-dark-800/50 text-accent focus:ring-accent/50"
            />
            Full runs only
          </label>
        </div>
        {totalPages > 1 && (
          <div className="flex items-center gap-2 text-sm">
            <button
              onClick={() => fetchPage(page - 1, fullOnly)}
              disabled={page <= 1 || loading}
              className="p-1 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="text-dark-400">{page} / {totalPages}</span>
            <button
              onClick={() => fetchPage(page + 1, fullOnly)}
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
                <th className="px-4 py-3 font-medium w-8"></th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Started</th>
                <th className="px-4 py-3 font-medium">Duration</th>
                <th className="px-4 py-3 font-medium">Ingestion Jobs</th>
                <th className="px-4 py-3 font-medium">Error</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => {
                const isExpanded = expandedId === run.id;
                const ingestionRuns = run.ingestion_runs || [];
                return (
                  <Fragment key={run.id}>
                    <tr
                      onClick={() => setExpandedId(isExpanded ? null : run.id)}
                      className={`border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 cursor-pointer ${isExpanded ? 'bg-dark-800/20' : ''}`}
                    >
                      <td className="px-4 py-3 text-dark-500">
                        {ingestionRuns.length > 0 && (isExpanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                      </td>
                      <td className="px-4 py-3">{statusBadge(run.status)}</td>
                      <td className="px-4 py-3 text-dark-300 text-xs">{formatDateTime(run.started_at)}</td>
                      <td className="px-4 py-3 text-dark-300 text-xs">{formatDuration(run.duration_ms)}</td>
                      <td className="px-4 py-3 text-dark-400 text-xs">{ingestionRuns.length || '—'}</td>
                      <td className="px-4 py-3 text-red-400/80 text-xs max-w-xs truncate">{run.error_message ?? ''}</td>
                    </tr>
                    {isExpanded && ingestionRuns.length > 0 && (
                      <tr>
                        <td colSpan={6} className="px-4 py-3 bg-dark-800/30">
                          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                            {ingestionRuns.map((ir: AdminIngestionRun) => (
                              <div key={ir.run_id} className="rounded-lg border border-dark-700/40 p-3 bg-dark-800/40">
                                <div className="flex items-center justify-between mb-2">
                                  <span className="text-xs font-medium text-dark-200">{ir.job_type}</span>
                                  {statusBadge(ir.status)}
                                </div>
                                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                                  <span className="text-dark-400">Fetched</span>
                                  <span className="text-dark-200 text-right">{ir.records_fetched?.toLocaleString() ?? '—'}</span>
                                  <span className="text-dark-400">Inserted</span>
                                  <span className="text-dark-200 text-right">{ir.records_inserted?.toLocaleString() ?? '—'}</span>
                                  <span className="text-dark-400">Updated</span>
                                  <span className="text-dark-200 text-right">{ir.records_updated?.toLocaleString() ?? '—'}</span>
                                  <span className="text-dark-400">Failed</span>
                                  <span className={`text-right ${(ir.records_failed ?? 0) > 0 ? 'text-red-400' : 'text-dark-200'}`}>
                                    {ir.records_failed?.toLocaleString() ?? '—'}
                                  </span>
                                  <span className="text-dark-400">Skipped</span>
                                  <span className="text-dark-200 text-right">{ir.records_skipped?.toLocaleString() ?? '—'}</span>
                                  <span className="text-dark-400">Duration</span>
                                  <span className="text-dark-200 text-right">{formatDuration(ir.duration_ms)}</span>
                                </div>
                                {ir.error_message && (
                                  <p className="mt-2 text-xs text-red-400/80 truncate">{ir.error_message}</p>
                                )}
                              </div>
                            ))}
                          </div>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
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

type Tab = 'users' | 'notifications' | 'api-keys' | 'samgov-requests' | 'usage' | 'pipeline' | 'searches';

export default function AdminPage() {
  const { isAuthenticated, isLoading, getAccessToken } = useAppAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [denied, setDenied] = useState(false);

  const activeTab = (searchParams.get('tab') as Tab) || 'users';
  const setActiveTab = (tab: Tab) => {
    setSearchParams(tab === 'users' ? {} : { tab }, { replace: true });
  };

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      setDenied(true);
      setLoading(false);
      return;
    }

    (async () => {
      try {
        const token = await getAccessToken();
        setUsers(await getAdminUsers(token));
      } catch {
        setDenied(true);
      } finally {
        setLoading(false);
      }
    })();
  }, [isAuthenticated, isLoading, getAccessToken]);

  if (loading || isLoading) {
    return <div className="min-h-screen flex items-center justify-center" />;
  }

  if (denied) {
    return <Navigate to="/" replace />;
  }

  const tabs: { key: Tab; label: string; icon: typeof Users }[] = [
    { key: 'users', label: 'Users', icon: Users },
    { key: 'notifications', label: 'Notifications', icon: Bell },
    { key: 'api-keys', label: 'API Keys', icon: Key },
    { key: 'samgov-requests', label: 'SAM.gov Requests', icon: Globe },
    { key: 'usage', label: 'Usage Chart', icon: BarChart3 },
    { key: 'pipeline', label: 'Pipeline', icon: Activity },
    { key: 'searches', label: 'Searches', icon: Search },
  ];

  return (
    <div className="min-h-screen">
      <div className="max-w-6xl mx-auto px-6 py-10">
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 text-sm text-dark-400 hover:text-dark-200 transition-colors mb-6"
        >
          <ArrowLeft size={16} />
          Back to search
        </Link>

        <h1 className="text-2xl font-semibold text-dark-100 mb-6">Admin</h1>

        <nav className="flex gap-1 mb-8 border-b border-dark-700/50">
          {tabs.map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key)}
              className={`inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
                activeTab === key
                  ? 'border-accent text-accent'
                  : 'border-transparent text-dark-400 hover:text-dark-200'
              }`}
            >
              <Icon size={16} />
              {label}
            </button>
          ))}
        </nav>

        {activeTab === 'users' && <UsersTab users={users} />}
        {activeTab === 'notifications' && <NotificationsTab users={users} getToken={getAccessToken} />}
        {activeTab === 'api-keys' && <ApiKeysTab getToken={getAccessToken} />}
        {activeTab === 'samgov-requests' && <SamgovRequestsTab getToken={getAccessToken} />}
        {activeTab === 'usage' && <UsageChartTab getToken={getAccessToken} />}
        {activeTab === 'pipeline' && <PipelineRunsTab getToken={getAccessToken} />}
        {activeTab === 'searches' && <SearchAnalyticsTab getToken={getAccessToken} />}
      </div>
    </div>
  );
}
