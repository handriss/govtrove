import { useState, useEffect, useCallback } from 'react';
import { Link, Navigate, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Shield, Users, Bell, Search, FileText, AlertCircle, ChevronLeft, ChevronRight } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { getAdminUsers, getAdminNotifications, type AdminUser } from '../services/api';
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

type Tab = 'users' | 'notifications';

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
  ];

  return (
    <div className="min-h-screen">
      <div className="max-w-5xl mx-auto px-6 py-10">
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
      </div>
    </div>
  );
}
