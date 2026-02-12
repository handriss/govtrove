import { useState } from 'react';
import { Link, Navigate } from 'react-router-dom';
import {
  ArrowLeft,
  LogOut,
  Calendar,
  CreditCard,
  Download,
  Trash2,
  User,
  Check,
  Loader2,
} from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { createAccountRequest } from '../services/api';

function formatMemberSince(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
}

export default function ProfilePage() {
  const { user, govtroveUser, isLoading, isAuthenticated, signOut, getAccessToken } = useAppAuth();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [exportStatus, setExportStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [deleteStatus, setDeleteStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [errorMsg, setErrorMsg] = useState('');

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 rounded-full bg-dark-800/50 animate-pulse" />
      </div>
    );
  }

  if (!isAuthenticated || !user) {
    return <Navigate to="/" replace />;
  }

  const initial = (user.firstName?.[0] || user.email[0] || '?').toUpperCase();
  const fullName = [user.firstName, user.lastName].filter(Boolean).join(' ') || 'User';
  const memberSince = govtroveUser?.created_at ? formatMemberSince(govtroveUser.created_at) : null;
  const plan = govtroveUser?.plan || 'free';

  async function handleRequest(type: 'data_export' | 'account_deletion') {
    const setStatus = type === 'data_export' ? setExportStatus : setDeleteStatus;
    setStatus('loading');
    setErrorMsg('');
    try {
      const token = await getAccessToken();
      await createAccountRequest(token, type);
      setStatus('success');
      if (type === 'account_deletion') setShowDeleteConfirm(false);
    } catch (e) {
      setStatus('error');
      setErrorMsg(e instanceof Error ? e.message : 'Something went wrong');
    }
  }

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <header className="relative z-20 border-b border-dark-800/50 bg-dark-950/80 backdrop-blur-md sticky top-0">
        <div className="max-w-2xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              to="/"
              className="p-2 -ml-2 text-dark-400 hover:text-dark-100 rounded-lg hover:bg-dark-800/50 transition-all duration-200"
            >
              <ArrowLeft size={20} strokeWidth={1.5} />
            </Link>
            <span className="text-sm text-dark-400">Back to Search</span>
          </div>
          <button
            onClick={() => signOut()}
            className="flex items-center gap-2 px-3 py-1.5 text-sm text-dark-300 hover:text-dark-50
                       border border-dark-700/50 hover:border-dark-600/50 rounded-lg
                       bg-dark-800/30 hover:bg-dark-800/50 transition-all duration-200"
          >
            <LogOut size={14} strokeWidth={1.5} />
            Sign Out
          </button>
        </div>
      </header>

      <div className="relative z-10 max-w-2xl mx-auto px-6 py-10">
        {/* User Info */}
        <div className="flex flex-col items-center text-center mb-8">
          <div className="w-16 h-16 rounded-full bg-accent/20 border border-accent/30 text-accent text-2xl font-semibold flex items-center justify-center mb-4">
            {initial}
          </div>
          <h1 className="text-xl font-semibold text-dark-50 mb-1">{fullName}</h1>
          <p className="text-sm text-dark-400">{user.email}</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-4 mb-8">
          {memberSince && (
            <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4">
              <div className="flex items-center gap-2 text-dark-500 text-xs uppercase tracking-wider mb-2">
                <Calendar size={14} strokeWidth={1.5} />
                Member since
              </div>
              <p className="text-dark-100 font-medium">{memberSince}</p>
            </div>
          )}
          <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4">
            <div className="flex items-center gap-2 text-dark-500 text-xs uppercase tracking-wider mb-2">
              <CreditCard size={14} strokeWidth={1.5} />
              Plan
            </div>
            <p className="text-dark-100 font-medium capitalize">{plan}</p>
          </div>
        </div>

        {/* Account Actions */}
        <div className="border-t border-dark-800/50 pt-8">
          <h2 className="text-sm font-medium text-dark-300 uppercase tracking-wider mb-4 flex items-center gap-2">
            <User size={14} strokeWidth={1.5} />
            Account Actions
          </h2>
          <div className="flex flex-wrap gap-3">
            {exportStatus === 'success' ? (
              <span className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-green-400 border border-green-500/20 rounded-xl bg-green-500/5">
                <Check size={15} strokeWidth={1.5} />
                Request submitted
              </span>
            ) : (
              <button
                onClick={() => handleRequest('data_export')}
                disabled={exportStatus === 'loading'}
                className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-dark-300 hover:text-dark-100
                           border border-dark-700/50 hover:border-dark-600/50 rounded-xl
                           bg-dark-800/30 hover:bg-dark-800/50 transition-all duration-200
                           disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {exportStatus === 'loading' ? (
                  <Loader2 size={15} strokeWidth={1.5} className="animate-spin" />
                ) : (
                  <Download size={15} strokeWidth={1.5} />
                )}
                Request My Data
              </button>
            )}

            {deleteStatus === 'success' ? (
              <span className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-green-400 border border-green-500/20 rounded-xl bg-green-500/5">
                <Check size={15} strokeWidth={1.5} />
                Request submitted
              </span>
            ) : !showDeleteConfirm ? (
              <button
                onClick={() => setShowDeleteConfirm(true)}
                className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-red-400/70 hover:text-red-400
                           border border-red-500/20 hover:border-red-500/30 rounded-xl
                           bg-red-500/5 hover:bg-red-500/10 transition-all duration-200"
              >
                <Trash2 size={15} strokeWidth={1.5} />
                Delete Account
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleRequest('account_deletion')}
                  disabled={deleteStatus === 'loading'}
                  className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-red-400
                             border border-red-500/30 rounded-xl bg-red-500/10 hover:bg-red-500/15
                             transition-all duration-200 font-medium
                             disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {deleteStatus === 'loading' ? (
                    <Loader2 size={15} strokeWidth={1.5} className="animate-spin" />
                  ) : (
                    <Trash2 size={15} strokeWidth={1.5} />
                  )}
                  Confirm — Delete Account
                </button>
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="px-3 py-2.5 text-sm text-dark-400 hover:text-dark-200 transition-colors"
                >
                  Cancel
                </button>
              </div>
            )}
          </div>

          {(exportStatus === 'error' || deleteStatus === 'error') && errorMsg && (
            <p className="mt-3 text-sm text-red-400">{errorMsg}</p>
          )}
        </div>
      </div>
    </div>
  );
}
