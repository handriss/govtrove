import { useState, useEffect } from 'react';
import { Link, Navigate } from 'react-router-dom';
import { ArrowLeft, Shield } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { getAdminUsers, type AdminUser } from '../services/api';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export default function AdminPage() {
  const { isAuthenticated, isLoading, getAccessToken } = useAppAuth();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [denied, setDenied] = useState(false);

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

        <h1 className="text-2xl font-semibold text-dark-100 mb-8">Admin</h1>

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
      </div>
    </div>
  );
}
