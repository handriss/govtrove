import { useState, useEffect, useCallback } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Shield, Bell, Search, ChevronLeft, ChevronRight, RotateCw, Plus, Download, Trash2, Loader2, ExternalLink, Copy, Send as SendIcon, Edit3, X } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import {
  getAdminNotifications, getAdminApiKeys, getAdminSamgovRequests,
  getAdminApiKeyUsage, getAdminPipelineRuns, getAdminSearchEvents, getAdminAnalytics,
  getAdminEmailPreferences, updateAdminEmailPreference, getAdminSentEmails, adminResendEmail,
  adminSendNewEmail, adminExportUserData, adminDeleteUser,
  getAdminPromoCodes, adminCreatePromoCode, adminSendPromoInvite, adminRevokePromoCode,
  getAdminInviteLinks, adminCreateInviteLink, getAdminInviteLinkDetail, adminUpdateInviteLink, adminDeactivateInviteLink,
  getAdminGiftCodes, adminCreateGiftCode, getAdminGiftCodeDetail, adminUpdateGiftCode, adminDeactivateGiftCode,
  getAdminMcpUsage, adminSetFreeForever,
  type AdminSendNewEmailInput,
  type AdminUser, type AdminApiKey, type AdminSamgovRequest,
  type UsageBucket, type PipelineExecution, type AdminSearchEvent,
  type SearchAnalytics, type AdminEmailPreference, type AdminSentEmail,
  type AdminPromoCode, type McpUsageEvent,
  type AdminInviteLink, type AdminInviteLinkRedemption,
  type AdminGiftCode, type AdminGiftCodeRedemption,
} from '../services/api';
import type { Notification } from '../types/api';
import { useAppAuth } from '../contexts/AuthContext';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatDateTime(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

const NOTIFICATION_TYPE_LABELS: Record<string, string> = {
  search_matches: 'Search Match',
  opportunity_amended: 'Amendment',
  opportunity_updated: 'Change',
};

function NotificationTypeIcon({ type }: { type: string }) {
  switch (type) {
    case 'search_matches': return <Search size={14} className="text-accent" />;
    case 'opportunity_amended': return <Bell size={14} className="text-amber-400" />;
    case 'opportunity_updated': return <Bell size={14} className="text-blue-400" />;
    default: return <Bell size={14} className="text-dark-400" />;
  }
}

function UsersTab({ users, getToken, onUserDeleted, onUserUpdated }: { users: AdminUser[]; getToken: () => Promise<string>; onUserDeleted: (userId: number) => void; onUserUpdated: (userId: number, updates: Partial<AdminUser>) => void }) {
  const [exportingId, setExportingId] = useState<number | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [togglingFFId, setTogglingFFId] = useState<number | null>(null);
  const [confirmDeleteUser, setConfirmDeleteUser] = useState<AdminUser | null>(null);
  const [confirmText, setConfirmText] = useState('');

  const handleToggleFreeForever = async (user: AdminUser) => {
    setTogglingFFId(user.id);
    try {
      const token = await getToken();
      await adminSetFreeForever(token, user.id, !user.free_forever);
      onUserUpdated(user.id, { free_forever: !user.free_forever });
    } catch (e) {
      alert(`Failed: ${e instanceof Error ? e.message : 'Unknown error'}`);
    } finally {
      setTogglingFFId(null);
    }
  };

  const handleExport = async (user: AdminUser) => {
    setExportingId(user.id);
    try {
      const token = await getToken();
      const data = await adminExportUserData(token, user.id);
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `dsar-export-${user.email}-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      alert(`Export failed: ${e instanceof Error ? e.message : 'Unknown error'}`);
    } finally {
      setExportingId(null);
    }
  };

  const handleDelete = async () => {
    if (!confirmDeleteUser || confirmText !== 'DELETE') return;
    setDeletingId(confirmDeleteUser.id);
    try {
      const token = await getToken();
      await adminDeleteUser(token, confirmDeleteUser.id);
      onUserDeleted(confirmDeleteUser.id);
      setConfirmDeleteUser(null);
      setConfirmText('');
    } catch (e) {
      alert(`Delete failed: ${e instanceof Error ? e.message : 'Unknown error'}`);
    } finally {
      setDeletingId(null);
    }
  };

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
              <th className="px-4 py-3 font-medium">Free Forever</th>
              <th className="px-4 py-3 font-medium">Joined</th>
              <th className="px-4 py-3 font-medium">Requests</th>
              <th className="px-4 py-3 font-medium">Export</th>
              <th className="px-4 py-3 font-medium">Delete</th>
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
                <td className="px-4 py-3">
                  <button
                    onClick={() => handleToggleFreeForever(u)}
                    disabled={togglingFFId === u.id}
                    className={`relative w-8 h-4 rounded-full transition-colors duration-200 ${
                      u.free_forever ? 'bg-accent' : 'bg-dark-700'
                    } disabled:opacity-50`}
                  >
                    <span className={`absolute top-0.5 left-0.5 w-3 h-3 rounded-full bg-white transition-transform duration-200 ${
                      u.free_forever ? 'translate-x-4' : ''
                    }`} />
                  </button>
                </td>
                <td className="px-4 py-3 text-dark-400">{formatDate(u.created_at)}</td>
                <td className="px-4 py-3">
                  <div className="flex flex-col gap-1">
                    {u.pending_export && (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-500/15 border border-blue-500/30 text-blue-400">
                        <Download size={10} />
                        Export
                      </span>
                    )}
                    {u.pending_deletion && (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-red-500/15 border border-red-500/30 text-red-400">
                        <Trash2 size={10} />
                        Deletion
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => handleExport(u)}
                    disabled={exportingId === u.id}
                    className="p-1.5 rounded hover:bg-dark-700/50 text-dark-400 hover:text-dark-200 disabled:opacity-50 transition-colors"
                    title="Export user data (DSAR)"
                  >
                    {exportingId === u.id ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
                  </button>
                </td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => { setConfirmDeleteUser(u); setConfirmText(''); }}
                    disabled={u.is_admin || deletingId === u.id}
                    className="p-1.5 rounded hover:bg-red-500/15 text-dark-500 hover:text-red-400 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                    title={u.is_admin ? 'Cannot delete admin users' : 'Delete user'}
                  >
                    {deletingId === u.id ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {confirmDeleteUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-dark-800 border border-dark-700 rounded-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-medium text-dark-200 mb-2">Delete user</h3>
            <p className="text-dark-400 text-sm mb-4">
              Delete user <strong className="text-dark-200">{confirmDeleteUser.email}</strong>? This will permanently remove all their data from GovTrove and WorkOS. Type <strong className="text-red-400">DELETE</strong> to confirm.
            </p>
            <input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder="Type DELETE to confirm"
              className="w-full px-3 py-2 rounded-lg bg-dark-900 border border-dark-600 text-dark-200 text-sm mb-4 focus:outline-none focus:border-red-500"
              autoFocus
            />
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => { setConfirmDeleteUser(null); setConfirmText(''); }}
                className="px-4 py-2 text-sm rounded-lg border border-dark-600 text-dark-300 hover:bg-dark-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={confirmText !== 'DELETE' || deletingId !== null}
                className="px-4 py-2 text-sm rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {deletingId ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

function NotificationsTab({ users, getToken }: { users: AdminUser[]; getToken: () => Promise<string> }) {
  const [selectedUserId, setSelectedUserId] = useState<number | ''>('');
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const limit = 50;

  const fetchNotifications = useCallback(async (userId: number, p: number) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminNotifications(token, userId, p);
      setNotifications(res.notifications || []);
      setTotal(res.total);
      setPage(p);
    } catch {
      setNotifications([]);
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

          {!loading && notifications.length === 0 && (
            <p className="text-dark-500 text-sm py-8 text-center">No notifications for this user.</p>
          )}

          {notifications.length > 0 && (
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
                  {notifications.map((n) => (
                    <tr key={n.id} className={`border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 ${!n.is_read ? 'bg-accent/[0.03]' : ''}`}>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center gap-1.5 text-dark-300">
                          <NotificationTypeIcon type={n.update_type} />
                          <span className="text-xs">{NOTIFICATION_TYPE_LABELS[n.update_type] || n.update_type}</span>
                        </span>
                      </td>
                      <td className="px-4 py-3 text-dark-200 max-w-md truncate">{(n.details as Record<string, unknown>)?.summary as string || n.update_type}</td>
                      <td className="px-4 py-3">
                        {n.is_read
                          ? <span className="text-dark-600 text-xs">Read</span>
                          : <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-accent/15 border border-accent/30 text-accent">Unread</span>
                        }
                      </td>
                      <td className="px-4 py-3 text-dark-400 text-xs">{formatDateTime(n.created_at)}</td>
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

  function buildSearchParams(ev: AdminSearchEvent): URLSearchParams {
    const params = new URLSearchParams();
    if (ev.query) params.set('q', ev.query);
    if (ev.filters) {
      try {
        const f = JSON.parse(ev.filters);
        for (const [k, v] of Object.entries(f)) {
          if (v == null || v === '') continue;
          params.set(k, Array.isArray(v) ? v.join(',') : String(v));
        }
      } catch { /* ignore */ }
    }
    if (ev.sort_by) params.set('sort', ev.sort_by);
    return params;
  }

  function buildApiUrl(ev: AdminSearchEvent): string {
    return `https://api.govtrove.com/api/opportunities?${buildSearchParams(ev)}`;
  }

  function buildAppUrl(ev: AdminSearchEvent): string {
    return `/?${buildSearchParams(ev)}`;
  }

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
                  <th className="px-4 py-3 font-medium w-20">Duration</th>
                  <th className="px-4 py-3 font-medium w-32">User ID</th>
                  <th className="px-4 py-3 font-medium w-12"></th>
                  <th className="px-4 py-3 font-medium w-12"></th>
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
                      <td className="px-4 py-3 text-xs text-dark-400 tabular-nums">
                        {ev.duration_ms != null ? `${ev.duration_ms}ms` : '—'}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-dark-400 truncate max-w-[8rem]">{ev.user_id ?? '—'}</td>
                      <td className="px-4 py-3">
                        <a href={buildApiUrl(ev)} target="_blank" rel="noopener noreferrer" className="text-dark-500 hover:text-accent transition-colors" title="Raw API response">
                          <ExternalLink size={14} />
                        </a>
                      </td>
                      <td className="px-4 py-3">
                        <a href={buildAppUrl(ev)} target="_blank" rel="noopener noreferrer" className="text-dark-500 hover:text-accent transition-colors" title="Open in app">
                          <Search size={14} />
                        </a>
                      </td>
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

function PromoCodesTab({ users, getToken }: { users: AdminUser[]; getToken: () => Promise<string> }) {
  const [codes, setCodes] = useState<AdminPromoCode[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedUserId, setSelectedUserId] = useState<number | ''>('');
  const [expiresInDays, setExpiresInDays] = useState(365);
  const [generating, setGenerating] = useState(false);
  const [sendingId, setSendingId] = useState<number | null>(null);
  const [sendStatus, setSendStatus] = useState<Record<number, 'success' | 'error'>>({});
  const [revokingId, setRevokingId] = useState<number | null>(null);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState<number | null>(null);

  const freeUsers = users.filter(u => u.plan !== 'pro' && !u.free_forever);

  const loadCodes = useCallback(async () => {
    try {
      const token = await getToken();
      setCodes(await getAdminPromoCodes(token));
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [getToken]);

  useEffect(() => { loadCodes(); }, [loadCodes]);

  async function handleGenerate() {
    if (!selectedUserId) return;
    setGenerating(true);
    setError('');
    try {
      const token = await getToken();
      await adminCreatePromoCode(token, Number(selectedUserId), expiresInDays || undefined);
      setSelectedUserId('');
      await loadCodes();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to generate code');
    } finally {
      setGenerating(false);
    }
  }

  async function handleSend(promoId: number) {
    setSendingId(promoId);
    try {
      const token = await getToken();
      await adminSendPromoInvite(token, promoId);
      setSendStatus(prev => ({ ...prev, [promoId]: 'success' }));
    } catch {
      setSendStatus(prev => ({ ...prev, [promoId]: 'error' }));
    } finally {
      setSendingId(null);
    }
  }

  function handleCopy(code: AdminPromoCode) {
    const url = `${window.location.origin}/profile?promo=${code.code}`;
    navigator.clipboard.writeText(url);
    setCopied(code.id);
    setTimeout(() => setCopied(null), 2000);
  }

  async function handleRevoke(promoId: number) {
    if (!confirm('Revoke this promo code? If redeemed, the user\'s Pro access will be cancelled.')) return;
    setRevokingId(promoId);
    try {
      const token = await getToken();
      await adminRevokePromoCode(token, promoId);
      await loadCodes();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to revoke');
    } finally {
      setRevokingId(null);
    }
  }

  function statusBadge(code: AdminPromoCode) {
    if (code.revoked_at) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">Revoked</span>;
    }
    if (code.redeemed_at) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20">Redeemed</span>;
    }
    if (code.expires_at && new Date(code.expires_at) < new Date()) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">Expired</span>;
    }
    return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">Pending</span>;
  }

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">Promo Codes</h2>

      {/* Generate form */}
      <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4 mb-6">
        <h3 className="text-sm font-medium text-dark-300 mb-3">Generate New Code</h3>
        <div className="flex items-end gap-3 flex-wrap">
          <div>
            <label className="block text-xs text-dark-400 mb-1">User</label>
            <select
              value={selectedUserId}
              onChange={(e) => setSelectedUserId(e.target.value ? Number(e.target.value) : '')}
              className="px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50 min-w-[200px]"
            >
              <option value="">Select user...</option>
              {freeUsers.map(u => (
                <option key={u.id} value={u.id}>{u.email} ({u.first_name})</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Expires in (days)</label>
            <input
              type="number"
              value={expiresInDays}
              onChange={(e) => setExpiresInDays(Number(e.target.value))}
              min={0}
              className="w-24 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50"
            />
          </div>
          <button
            onClick={handleGenerate}
            disabled={!selectedUserId || generating}
            className="px-4 py-2 text-sm font-medium rounded-lg bg-accent text-dark-950 hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {generating ? 'Generating...' : 'Generate Code'}
          </button>
        </div>
        {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
      </div>

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && codes.length === 0 && (
        <p className="text-dark-500 text-sm py-8 text-center">No promo codes yet.</p>
      )}

      {!loading && codes.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-dark-700/50">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-3 font-medium">User</th>
                <th className="px-4 py-3 font-medium">Code</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium">Redeemed</th>
                <th className="px-4 py-3 font-medium">Expires</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {codes.map((c) => (
                <tr key={c.id} className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-dark-200 text-xs">{c.for_user_email || '—'}</td>
                  <td className="px-4 py-3 text-dark-100 text-xs font-mono">{c.code}</td>
                  <td className="px-4 py-3">{statusBadge(c)}</td>
                  <td className="px-4 py-3 text-dark-400 text-xs">{formatDate(c.created_at)}</td>
                  <td className="px-4 py-3 text-dark-400 text-xs">
                    {c.redeemed_at ? formatDateTime(c.redeemed_at) : '—'}
                    {c.redeemed_email && <span className="text-dark-500 ml-1">({c.redeemed_email})</span>}
                  </td>
                  <td className="px-4 py-3 text-dark-400 text-xs">{c.expires_at ? formatDate(c.expires_at) : '—'}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {!c.revoked_at && (
                        <>
                          <button
                            onClick={() => handleCopy(c)}
                            title="Copy invite link"
                            className="p-1 text-dark-400 hover:text-accent transition-colors"
                          >
                            {copied === c.id ? <span className="text-xs text-green-400">Copied</span> : <Copy size={14} />}
                          </button>
                          {!c.redeemed_at && c.for_user_email && (
                            sendStatus[c.id] === 'success' ? (
                              <span className="text-xs text-green-400">Sent</span>
                            ) : (
                              <button
                                onClick={() => handleSend(c.id)}
                                disabled={sendingId === c.id}
                                title="Send invite email"
                                className="p-1 text-dark-400 hover:text-accent transition-colors disabled:opacity-50"
                              >
                                {sendingId === c.id ? <Loader2 size={14} className="animate-spin" /> : <SendIcon size={14} />}
                              </button>
                            )
                          )}
                          <button
                            onClick={() => handleRevoke(c.id)}
                            disabled={revokingId === c.id}
                            title="Revoke code"
                            className="p-1 text-dark-400 hover:text-red-400 transition-colors disabled:opacity-50"
                          >
                            {revokingId === c.id ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
                          </button>
                        </>
                      )}
                    </div>
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

function GiftCodesTab({ getToken }: { getToken: () => Promise<string> }) {
  const [codes, setCodes] = useState<AdminGiftCode[]>([]);
  const [loading, setLoading] = useState(true);
  const [campaignName, setCampaignName] = useState('');
  const [customCode, setCustomCode] = useState('');
  const [durationDays, setDurationDays] = useState(365);
  const [maxRedemptions, setMaxRedemptions] = useState(0);
  const [expiresInDays, setExpiresInDays] = useState(0);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState<number | null>(null);
  const [deactivatingId, setDeactivatingId] = useState<number | null>(null);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [redemptions, setRedemptions] = useState<AdminGiftCodeRedemption[]>([]);
  const [redemptionsLoading, setRedemptionsLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editMax, setEditMax] = useState(0);

  const loadCodes = useCallback(async () => {
    try {
      const token = await getToken();
      setCodes(await getAdminGiftCodes(token));
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [getToken]);

  useEffect(() => { loadCodes(); }, [loadCodes]);

  async function handleCreate() {
    if (!campaignName) return;
    setGenerating(true);
    setError('');
    try {
      const token = await getToken();
      await adminCreateGiftCode(token, {
        campaign_name: campaignName,
        code: customCode || undefined,
        duration_days: durationDays,
        max_redemptions: maxRedemptions,
        expires_in_days: expiresInDays || undefined,
      });
      setCampaignName('');
      setCustomCode('');
      await loadCodes();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create gift code');
    } finally {
      setGenerating(false);
    }
  }

  function handleCopy(gc: AdminGiftCode) {
    const url = `${window.location.origin}/redeem/${gc.code}`;
    navigator.clipboard.writeText(url);
    setCopied(gc.id);
    setTimeout(() => setCopied(null), 2000);
  }

  async function handleExpand(id: number) {
    if (expandedId === id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(id);
    setRedemptionsLoading(true);
    try {
      const token = await getToken();
      const detail = await getAdminGiftCodeDetail(token, id);
      setRedemptions(detail.redemptions);
    } catch {
      setRedemptions([]);
    } finally {
      setRedemptionsLoading(false);
    }
  }

  async function handleDeactivate(id: number) {
    if (!confirm('Deactivate this gift code? New users will no longer be able to redeem it.')) return;
    setDeactivatingId(id);
    try {
      const token = await getToken();
      await adminDeactivateGiftCode(token, id);
      await loadCodes();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to deactivate');
    } finally {
      setDeactivatingId(null);
    }
  }

  async function handleSaveMax(id: number) {
    try {
      const token = await getToken();
      await adminUpdateGiftCode(token, id, { max_redemptions: editMax });
      setEditingId(null);
      await loadCodes();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update');
    }
  }

  function statusBadge(gc: AdminGiftCode) {
    if (gc.deactivated_at) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">Deactivated</span>;
    }
    if (gc.expires_at && new Date(gc.expires_at) < new Date()) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">Expired</span>;
    }
    if (gc.max_redemptions > 0 && gc.redemption_count >= gc.max_redemptions) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">Full</span>;
    }
    return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20">Active</span>;
  }

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">Gift Codes</h2>

      <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4 mb-6">
        <h3 className="text-sm font-medium text-dark-300 mb-3">Create New Gift Code</h3>
        <div className="flex items-end gap-3 flex-wrap">
          <div>
            <label className="block text-xs text-dark-400 mb-1">Campaign Name</label>
            <input
              type="text"
              value={campaignName}
              onChange={(e) => setCampaignName(e.target.value)}
              placeholder="e.g. Beta Testers"
              className="px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50 min-w-[200px]"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Custom Code (optional)</label>
            <input
              type="text"
              value={customCode}
              onChange={(e) => setCustomCode(e.target.value.toUpperCase())}
              placeholder="GIFT-BETA-2026"
              className="px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50 min-w-[200px]"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Duration (days)</label>
            <input
              type="number"
              value={durationDays}
              onChange={(e) => setDurationDays(Number(e.target.value))}
              min={1}
              className="w-24 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Max Redemptions</label>
            <input
              type="number"
              value={maxRedemptions}
              onChange={(e) => setMaxRedemptions(Number(e.target.value))}
              min={0}
              placeholder="0 = unlimited"
              className="w-24 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Code expires in (days)</label>
            <input
              type="number"
              value={expiresInDays}
              onChange={(e) => setExpiresInDays(Number(e.target.value))}
              min={0}
              placeholder="0 = never"
              className="w-24 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50"
            />
          </div>
          <button
            onClick={handleCreate}
            disabled={!campaignName || generating}
            className="px-4 py-2 text-sm font-medium rounded-lg bg-accent text-dark-950 hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {generating ? 'Creating...' : 'Create Gift Code'}
          </button>
        </div>
        {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
      </div>

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && codes.length === 0 && (
        <p className="text-dark-500 text-sm py-8 text-center">No gift codes yet.</p>
      )}

      {!loading && codes.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-dark-700/50">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-3 font-medium">Campaign</th>
                <th className="px-4 py-3 font-medium">Code</th>
                <th className="px-4 py-3 font-medium">Duration</th>
                <th className="px-4 py-3 font-medium">Redemptions</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium">Expires</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {codes.map((gc) => (
                <>
                  <tr
                    key={gc.id}
                    className={`border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 cursor-pointer ${expandedId === gc.id ? 'bg-dark-800/20' : ''}`}
                    onClick={() => handleExpand(gc.id)}
                  >
                    <td className="px-4 py-3 text-dark-200">{gc.campaign_name}</td>
                    <td className="px-4 py-3 text-dark-100 text-xs font-mono">{gc.code}</td>
                    <td className="px-4 py-3 text-dark-300">{gc.duration_days}d</td>
                    <td className="px-4 py-3 text-dark-300">
                      {editingId === gc.id ? (
                        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                          <span className="text-dark-400">{gc.redemption_count} /</span>
                          <input
                            type="number"
                            value={editMax}
                            onChange={(e) => setEditMax(Number(e.target.value))}
                            min={0}
                            className="w-16 px-1 py-0.5 text-xs bg-dark-800 border border-dark-600 rounded text-dark-100"
                            autoFocus
                          />
                          <button onClick={() => handleSaveMax(gc.id)} className="text-green-400 hover:text-green-300 text-xs">Save</button>
                          <button onClick={() => setEditingId(null)} className="text-dark-500 hover:text-dark-300"><X size={12} /></button>
                        </div>
                      ) : (
                        <span>
                          {gc.redemption_count} / {gc.max_redemptions || '\u221e'}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">{statusBadge(gc)}</td>
                    <td className="px-4 py-3 text-dark-400 text-xs">{formatDate(gc.created_at)}</td>
                    <td className="px-4 py-3 text-dark-400 text-xs">{gc.expires_at ? formatDate(gc.expires_at) : '\u2014'}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                        {!gc.deactivated_at && (
                          <>
                            <button
                              onClick={() => handleCopy(gc)}
                              title="Copy redeem URL"
                              className="p-1 text-dark-400 hover:text-accent transition-colors"
                            >
                              {copied === gc.id ? <span className="text-xs text-green-400">Copied</span> : <Copy size={14} />}
                            </button>
                            <button
                              onClick={() => { setEditingId(gc.id); setEditMax(gc.max_redemptions); }}
                              title="Edit max redemptions"
                              className="p-1 text-dark-400 hover:text-accent transition-colors"
                            >
                              <Edit3 size={14} />
                            </button>
                            <button
                              onClick={() => handleDeactivate(gc.id)}
                              disabled={deactivatingId === gc.id}
                              title="Deactivate"
                              className="p-1 text-dark-400 hover:text-red-400 transition-colors disabled:opacity-50"
                            >
                              {deactivatingId === gc.id ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                  {expandedId === gc.id && (
                    <tr key={`${gc.id}-detail`} className="border-b border-dark-700/30">
                      <td colSpan={8} className="px-8 py-4 bg-dark-900/30">
                        <h4 className="text-sm font-medium text-dark-300 mb-2">Redemptions</h4>
                        {redemptionsLoading ? (
                          <p className="text-dark-400 text-xs">Loading...</p>
                        ) : redemptions.length === 0 ? (
                          <p className="text-dark-500 text-xs">No redemptions yet.</p>
                        ) : (
                          <div className="space-y-1">
                            {redemptions.map((r) => (
                              <div key={r.user_id} className="flex items-center gap-4 text-xs text-dark-300">
                                <span className="text-dark-200">{r.email}</span>
                                <span className="text-dark-500">{r.name}</span>
                                <span className="text-dark-400">until {formatDate(r.granted_until)}</span>
                                <span className="text-dark-500">{formatDateTime(r.redeemed_at)}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function InviteLinksTab({ getToken }: { getToken: () => Promise<string> }) {
  const [links, setLinks] = useState<AdminInviteLink[]>([]);
  const [loading, setLoading] = useState(true);
  const [campaignName, setCampaignName] = useState('');
  const [customCode, setCustomCode] = useState('');
  const [maxRedemptions, setMaxRedemptions] = useState(50);
  const [expiresInDays, setExpiresInDays] = useState(0);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState<number | null>(null);
  const [deactivatingId, setDeactivatingId] = useState<number | null>(null);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [redemptions, setRedemptions] = useState<AdminInviteLinkRedemption[]>([]);
  const [redemptionsLoading, setRedemptionsLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editMax, setEditMax] = useState(0);

  const loadLinks = useCallback(async () => {
    try {
      const token = await getToken();
      setLinks(await getAdminInviteLinks(token));
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [getToken]);

  useEffect(() => { loadLinks(); }, [loadLinks]);

  async function handleCreate() {
    if (!campaignName) return;
    setGenerating(true);
    setError('');
    try {
      const token = await getToken();
      await adminCreateInviteLink(token, {
        campaign_name: campaignName,
        code: customCode || undefined,
        max_redemptions: maxRedemptions,
        expires_in_days: expiresInDays || undefined,
      });
      setCampaignName('');
      setCustomCode('');
      await loadLinks();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create invite link');
    } finally {
      setGenerating(false);
    }
  }

  function handleCopy(link: AdminInviteLink) {
    const url = `${window.location.origin}/profile?promo=${link.code}&utm_campaign=${link.code}&utm_source=invite&utm_medium=link`;
    navigator.clipboard.writeText(url);
    setCopied(link.id);
    setTimeout(() => setCopied(null), 2000);
  }

  async function handleExpand(id: number) {
    if (expandedId === id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(id);
    setRedemptionsLoading(true);
    try {
      const token = await getToken();
      const detail = await getAdminInviteLinkDetail(token, id);
      setRedemptions(detail.redemptions);
    } catch {
      setRedemptions([]);
    } finally {
      setRedemptionsLoading(false);
    }
  }

  async function handleDeactivate(id: number) {
    if (!confirm('Deactivate this invite link? New users will no longer be able to use it.')) return;
    setDeactivatingId(id);
    try {
      const token = await getToken();
      await adminDeactivateInviteLink(token, id);
      await loadLinks();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to deactivate');
    } finally {
      setDeactivatingId(null);
    }
  }

  async function handleSaveMax(id: number) {
    try {
      const token = await getToken();
      await adminUpdateInviteLink(token, id, { max_redemptions: editMax });
      setEditingId(null);
      await loadLinks();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update');
    }
  }

  function statusBadge(link: AdminInviteLink) {
    if (link.deactivated_at) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">Deactivated</span>;
    }
    if (link.expires_at && new Date(link.expires_at) < new Date()) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">Expired</span>;
    }
    if (link.max_redemptions > 0 && link.redemption_count >= link.max_redemptions) {
      return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">Full</span>;
    }
    return <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20">Active</span>;
  }

  return (
    <section>
      <h2 className="text-lg font-medium text-dark-200 mb-4">Invite Links</h2>

      <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4 mb-6">
        <h3 className="text-sm font-medium text-dark-300 mb-3">Create New Invite Link</h3>
        <div className="flex items-end gap-3 flex-wrap">
          <div>
            <label className="block text-xs text-dark-400 mb-1">Campaign Name</label>
            <input
              type="text"
              value={campaignName}
              onChange={(e) => setCampaignName(e.target.value)}
              placeholder="e.g. APEX Conference"
              className="px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50 min-w-[200px]"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Custom Code (optional)</label>
            <input
              type="text"
              value={customCode}
              onChange={(e) => setCustomCode(e.target.value.toUpperCase())}
              placeholder="GOVTROVE-APEX-2026"
              className="px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50 min-w-[200px]"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Max Redemptions</label>
            <input
              type="number"
              value={maxRedemptions}
              onChange={(e) => setMaxRedemptions(Number(e.target.value))}
              min={0}
              className="w-24 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50"
            />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Expires in (days)</label>
            <input
              type="number"
              value={expiresInDays}
              onChange={(e) => setExpiresInDays(Number(e.target.value))}
              min={0}
              placeholder="0 = never"
              className="w-24 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg text-dark-100 focus:outline-none focus:border-accent/50"
            />
          </div>
          <button
            onClick={handleCreate}
            disabled={!campaignName || generating}
            className="px-4 py-2 text-sm font-medium rounded-lg bg-accent text-dark-950 hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {generating ? 'Creating...' : 'Create Link'}
          </button>
        </div>
        {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
      </div>

      {loading && <p className="text-dark-400 text-sm py-8 text-center">Loading...</p>}

      {!loading && links.length === 0 && (
        <p className="text-dark-500 text-sm py-8 text-center">No invite links yet.</p>
      )}

      {!loading && links.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-dark-700/50">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-3 font-medium">Campaign</th>
                <th className="px-4 py-3 font-medium">Code</th>
                <th className="px-4 py-3 font-medium">Redemptions</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium">Expires</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {links.map((link) => (
                <>
                  <tr
                    key={link.id}
                    className={`border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 cursor-pointer ${expandedId === link.id ? 'bg-dark-800/20' : ''}`}
                    onClick={() => handleExpand(link.id)}
                  >
                    <td className="px-4 py-3 text-dark-200">{link.campaign_name}</td>
                    <td className="px-4 py-3 text-dark-100 text-xs font-mono">{link.code}</td>
                    <td className="px-4 py-3 text-dark-300">
                      {editingId === link.id ? (
                        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                          <span className="text-dark-400">{link.redemption_count} /</span>
                          <input
                            type="number"
                            value={editMax}
                            onChange={(e) => setEditMax(Number(e.target.value))}
                            min={0}
                            className="w-16 px-1 py-0.5 text-xs bg-dark-800 border border-dark-600 rounded text-dark-100"
                            autoFocus
                          />
                          <button onClick={() => handleSaveMax(link.id)} className="text-green-400 hover:text-green-300 text-xs">Save</button>
                          <button onClick={() => setEditingId(null)} className="text-dark-500 hover:text-dark-300"><X size={12} /></button>
                        </div>
                      ) : (
                        <span>
                          {link.redemption_count} / {link.max_redemptions || '∞'}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">{statusBadge(link)}</td>
                    <td className="px-4 py-3 text-dark-400 text-xs">{formatDate(link.created_at)}</td>
                    <td className="px-4 py-3 text-dark-400 text-xs">{link.expires_at ? formatDate(link.expires_at) : '—'}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                        {!link.deactivated_at && (
                          <>
                            <button
                              onClick={() => handleCopy(link)}
                              title="Copy invite URL"
                              className="p-1 text-dark-400 hover:text-accent transition-colors"
                            >
                              {copied === link.id ? <span className="text-xs text-green-400">Copied</span> : <Copy size={14} />}
                            </button>
                            <button
                              onClick={() => { setEditingId(link.id); setEditMax(link.max_redemptions); }}
                              title="Edit max redemptions"
                              className="p-1 text-dark-400 hover:text-accent transition-colors"
                            >
                              <Edit3 size={14} />
                            </button>
                            <button
                              onClick={() => handleDeactivate(link.id)}
                              disabled={deactivatingId === link.id}
                              title="Deactivate link"
                              className="p-1 text-dark-400 hover:text-red-400 transition-colors disabled:opacity-50"
                            >
                              {deactivatingId === link.id ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                  {expandedId === link.id && (
                    <tr key={`${link.id}-detail`} className="border-b border-dark-700/30">
                      <td colSpan={7} className="px-8 py-4 bg-dark-900/30">
                        <h4 className="text-sm font-medium text-dark-300 mb-2">Redemptions</h4>
                        {redemptionsLoading ? (
                          <p className="text-dark-400 text-xs">Loading...</p>
                        ) : redemptions.length === 0 ? (
                          <p className="text-dark-500 text-xs">No redemptions yet.</p>
                        ) : (
                          <div className="space-y-1">
                            {redemptions.map((r) => (
                              <div key={r.user_id} className="flex items-center gap-4 text-xs text-dark-300">
                                <span className="text-dark-200">{r.email}</span>
                                <span className="text-dark-500">{r.name}</span>
                                <span className="text-dark-400">{formatDateTime(r.redeemed_at)}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function McpUsageTab({ getToken }: { getToken: () => Promise<string> }) {
  const [events, setEvents] = useState<McpUsageEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const limit = 50;

  const fetchData = useCallback(async (p: number) => {
    setLoading(true);
    try {
      const token = await getToken();
      const res = await getAdminMcpUsage(token, p);
      setEvents(res.events || []);
      setTotal(res.total);
      setPage(p);
    } catch {
      setEvents([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => { fetchData(1); }, [fetchData]);

  const totalPages = Math.ceil(total / limit);

  function formatParams(params: string | null): string {
    if (!params) return '—';
    try {
      const parsed = JSON.parse(params);
      const parts: string[] = [];
      for (const [k, v] of Object.entries(parsed)) {
        if (v == null || v === '') continue;
        parts.push(`${k}: ${typeof v === 'string' ? v : JSON.stringify(v)}`);
      }
      return parts.length > 0 ? parts.join(', ') : '—';
    } catch {
      return params.slice(0, 80);
    }
  }

  return (
    <section className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-medium text-dark-200">MCP Tool Usage</h2>
        <span className="text-sm text-dark-400">{total} total calls</span>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="animate-spin text-dark-400" size={24} /></div>
      ) : events.length === 0 ? (
        <p className="text-dark-400 text-center py-12">No MCP usage yet.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="pb-3 pr-4 font-medium">User</th>
                <th className="pb-3 pr-4 font-medium">Tool</th>
                <th className="pb-3 pr-4 font-medium">Parameters</th>
                <th className="pb-3 pr-4 font-medium text-right">Results</th>
                <th className="pb-3 pr-4 font-medium text-right">Latency</th>
                <th className="pb-3 font-medium">Time</th>
              </tr>
            </thead>
            <tbody>
              {events.map((ev) => (
                <tr key={ev.id} className="border-b border-dark-700/30 hover:bg-dark-800/30">
                  <td className="py-2.5 pr-4 text-dark-300">{ev.user_email || '—'}</td>
                  <td className="py-2.5 pr-4">
                    <span className="px-2 py-0.5 rounded bg-accent/10 text-accent text-xs font-medium">{ev.tool_name}</span>
                  </td>
                  <td className="py-2.5 pr-4 text-dark-400 max-w-xs truncate" title={ev.request_params || undefined}>
                    {formatParams(ev.request_params)}
                  </td>
                  <td className="py-2.5 pr-4 text-right text-dark-300">{ev.result_count ?? '—'}</td>
                  <td className="py-2.5 pr-4 text-right text-dark-400">{ev.latency_ms != null ? `${ev.latency_ms}ms` : '—'}</td>
                  <td className="py-2.5 text-dark-400">{formatDateTime(ev.called_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-between pt-2">
          <span className="text-sm text-dark-400">Page {page} of {totalPages}</span>
          <div className="flex gap-2">
            <button onClick={() => fetchData(page - 1)} disabled={page <= 1}
              className="p-2 rounded-lg border border-dark-700/50 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed">
              <ChevronLeft size={16} />
            </button>
            <button onClick={() => fetchData(page + 1)} disabled={page >= totalPages}
              className="p-2 rounded-lg border border-dark-700/50 text-dark-400 hover:text-dark-200 disabled:opacity-30 disabled:cursor-not-allowed">
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      )}
    </section>
  );
}

type Tab = 'users' | 'notifications' | 'api-keys' | 'samgov-requests' | 'usage' | 'pipeline' | 'searches' | 'email-prefs' | 'sent-emails' | 'promo-codes' | 'invite-links' | 'gift-codes' | 'mcp-usage';

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
  const { users, setUsers, getToken } = useAdminContext();

  return (
    <div className="max-w-6xl mx-auto px-6 py-10">
      {activeTab === 'users' && <UsersTab users={users} getToken={getToken} onUserDeleted={(id) => setUsers(prev => prev.filter(u => u.id !== id))} onUserUpdated={(id, updates) => setUsers(prev => prev.map(u => u.id === id ? { ...u, ...updates } : u))} />}
      {activeTab === 'notifications' && <NotificationsTab users={users} getToken={getToken} />}
      {activeTab === 'api-keys' && <ApiKeysTab getToken={getToken} />}
      {activeTab === 'samgov-requests' && <SamgovRequestsTab getToken={getToken} />}
      {activeTab === 'usage' && <UsageChartTab getToken={getToken} />}
      {activeTab === 'pipeline' && <PipelineRunsTab getToken={getToken} />}
      {activeTab === 'searches' && <SearchAnalyticsTab getToken={getToken} />}
      {activeTab === 'email-prefs' && <EmailPrefsTab getToken={getToken} />}
      {activeTab === 'sent-emails' && <SentEmailsTab getToken={getToken} />}
      {activeTab === 'promo-codes' && <PromoCodesTab users={users} getToken={getToken} />}
      {activeTab === 'invite-links' && <InviteLinksTab getToken={getToken} />}
      {activeTab === 'gift-codes' && <GiftCodesTab getToken={getToken} />}
      {activeTab === 'mcp-usage' && <McpUsageTab getToken={getToken} />}
    </div>
  );
}
