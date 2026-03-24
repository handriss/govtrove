import { useState, useEffect, useCallback } from 'react';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import { getAdminAccountRequests, adminExecuteExport, adminExecuteDeletion, type AccountRequest } from '../services/api';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function StatusBadge({ status }: { status: string }) {
  const colors = status === 'pending'
    ? 'bg-amber-500/10 text-amber-400 border-amber-500/20'
    : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${colors}`}>
      {status}
    </span>
  );
}

function TypeBadge({ type }: { type: string }) {
  const colors = type === 'data_export'
    ? 'bg-blue-500/10 text-blue-400 border-blue-500/20'
    : 'bg-red-500/10 text-red-400 border-red-500/20';
  const label = type === 'data_export' ? 'Export' : 'Deletion';
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${colors}`}>
      {label}
    </span>
  );
}

function DSARContent() {
  const { getToken } = useAdminContext();
  const [requests, setRequests] = useState<AccountRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [executing, setExecuting] = useState<number | null>(null);
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<AccountRequest | null>(null);
  const [deleteEmail, setDeleteEmail] = useState('');
  const [error, setError] = useState('');

  const fetchRequests = useCallback(async () => {
    try {
      const token = await getToken();
      setRequests(await getAdminAccountRequests(token, filter));
    } catch { /* ignore */ } finally {
      setLoading(false);
    }
  }, [getToken, filter]);

  useEffect(() => {
    fetchRequests();
  }, [fetchRequests]);

  async function handleExport(req: AccountRequest) {
    if (!confirm(`Execute data export for ${req.user_email}? This will upload their data to S3 and email them a download link.`)) return;
    setExecuting(req.id);
    setError('');
    setDownloadUrl(null);
    try {
      const token = await getToken();
      const result = await adminExecuteExport(token, req.id);
      setDownloadUrl(result.download_url);
      fetchRequests();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Export failed');
    } finally {
      setExecuting(null);
    }
  }

  function openDeleteConfirm(req: AccountRequest) {
    setDeleteConfirm(req);
    setDeleteEmail('');
    setError('');
  }

  async function handleDeletion() {
    if (!deleteConfirm) return;
    if (deleteEmail !== deleteConfirm.user_email) return;
    setExecuting(deleteConfirm.id);
    setError('');
    try {
      const token = await getToken();
      await adminExecuteDeletion(token, deleteConfirm.id);
      setDeleteConfirm(null);
      fetchRequests();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Deletion failed');
    } finally {
      setExecuting(null);
    }
  }

  return (
    <div className="max-w-6xl mx-auto px-6 py-10">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-dark-100">DSAR Requests</h1>
        <select
          value={filter}
          onChange={(e) => { setFilter(e.target.value); setLoading(true); }}
          className="bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-1.5 text-sm text-dark-200 focus:outline-none focus:border-accent/50"
        >
          <option value="">All</option>
          <option value="pending">Pending</option>
          <option value="completed">Completed</option>
        </select>
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          {error}
        </div>
      )}

      {downloadUrl && (
        <div className="mb-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm">
          Export complete.{' '}
          <a href={downloadUrl} target="_blank" rel="noopener noreferrer" className="underline font-medium">
            Download link
          </a>{' '}
          (also emailed to user, expires in 48h)
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-20">
          <div className="h-6 w-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : requests.length === 0 ? (
        <p className="text-dark-500 text-center py-20">No requests found.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-dark-500 border-b border-dark-700/50">
                <th className="pb-3 pr-4 font-medium">User</th>
                <th className="pb-3 pr-4 font-medium">Type</th>
                <th className="pb-3 pr-4 font-medium">Status</th>
                <th className="pb-3 pr-4 font-medium">Requested</th>
                <th className="pb-3 pr-4 font-medium">Completed</th>
                <th className="pb-3 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              {requests.map((req) => (
                <tr key={req.id} className="border-b border-dark-800/50 hover:bg-dark-800/30">
                  <td className="py-3 pr-4">
                    <div className="text-dark-200">{req.user_email}</div>
                    {req.user_name.trim() && <div className="text-dark-500 text-xs">{req.user_name}</div>}
                  </td>
                  <td className="py-3 pr-4"><TypeBadge type={req.request_type} /></td>
                  <td className="py-3 pr-4"><StatusBadge status={req.status} /></td>
                  <td className="py-3 pr-4 text-dark-400">{formatDate(req.created_at)}</td>
                  <td className="py-3 pr-4 text-dark-400">
                    {req.completed_at ? (
                      <div>
                        <div>{formatDate(req.completed_at)}</div>
                        {req.completed_by && <div className="text-dark-500 text-xs">by {req.completed_by}</div>}
                      </div>
                    ) : (
                      <span className="text-dark-600">—</span>
                    )}
                  </td>
                  <td className="py-3">
                    {req.status === 'pending' && req.request_type === 'data_export' && (
                      <button
                        onClick={() => handleExport(req)}
                        disabled={executing !== null}
                        className="px-3 py-1 rounded-lg text-xs font-medium bg-blue-600 hover:bg-blue-700 text-white disabled:opacity-50 transition-colors"
                      >
                        {executing === req.id ? 'Exporting...' : 'Execute Export'}
                      </button>
                    )}
                    {req.status === 'pending' && req.request_type === 'account_deletion' && (
                      <button
                        onClick={() => openDeleteConfirm(req)}
                        disabled={executing !== null}
                        className="px-3 py-1 rounded-lg text-xs font-medium bg-red-600 hover:bg-red-700 text-white disabled:opacity-50 transition-colors"
                      >
                        Execute Deletion
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Deletion confirmation modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-dark-900 border border-dark-700/50 rounded-xl p-6 max-w-md w-full mx-4">
            <h2 className="text-lg font-semibold text-dark-100 mb-2">Confirm Account Deletion</h2>
            <p className="text-dark-400 text-sm mb-4">
              This will permanently delete the account for <strong className="text-dark-200">{deleteConfirm.user_email}</strong>.
              This action cannot be undone.
            </p>
            <label className="block text-sm text-dark-400 mb-2">
              Type the user's email to confirm:
            </label>
            <input
              type="text"
              value={deleteEmail}
              onChange={(e) => setDeleteEmail(e.target.value)}
              placeholder={deleteConfirm.user_email}
              className="w-full bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 text-sm text-dark-200 mb-4 focus:outline-none focus:border-red-500/50"
            />
            {error && <p className="text-red-400 text-sm mb-3">{error}</p>}
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 rounded-lg text-sm text-dark-400 hover:text-dark-200 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDeletion}
                disabled={deleteEmail !== deleteConfirm.user_email || executing !== null}
                className="px-4 py-2 rounded-lg text-sm font-medium bg-red-600 hover:bg-red-700 text-white disabled:opacity-50 transition-colors"
              >
                {executing ? 'Deleting...' : 'Delete Account'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function AdminDSARPage() {
  return (
    <AdminLayout>
      <DSARContent />
    </AdminLayout>
  );
}
