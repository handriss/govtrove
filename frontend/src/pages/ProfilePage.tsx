import { useState, useEffect, useCallback } from 'react';
import { Link, Navigate } from 'react-router-dom';
import {
  ArrowLeft,
  Calendar,
  CreditCard,
  Download,
  Trash2,
  User,
  Check,
  Loader2,
  Mail,
} from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { createAccountRequest, getEmailPreferences, updateEmailPreferences } from '../services/api';

function formatMemberSince(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
}

export default function ProfilePage() {
  const { user, govtroveUser, isLoading, isAuthenticated, getAccessToken } = useAppAuth();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [exportStatus, setExportStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [deleteStatus, setDeleteStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [errorMsg, setErrorMsg] = useState('');
  const [searchAlerts, setSearchAlerts] = useState(true);
  const [opportunityAlerts, setOpportunityAlerts] = useState(true);
  const [prefsLoading, setPrefsLoading] = useState(true);

  const loadPrefs = useCallback(async () => {
    try {
      const token = await getAccessToken();
      const prefs = await getEmailPreferences(token);
      setSearchAlerts(prefs.search_alerts);
      setOpportunityAlerts(prefs.opportunity_alerts);
    } catch { /* defaults remain */ }
    finally { setPrefsLoading(false); }
  }, [getAccessToken]);

  useEffect(() => {
    if (isAuthenticated) loadPrefs();
  }, [isAuthenticated, loadPrefs]);

  async function togglePref(field: 'search_alerts' | 'opportunity_alerts', value: boolean) {
    const newSearch = field === 'search_alerts' ? value : searchAlerts;
    const newOpp = field === 'opportunity_alerts' ? value : opportunityAlerts;
    if (field === 'search_alerts') setSearchAlerts(value);
    else setOpportunityAlerts(value);
    try {
      const token = await getAccessToken();
      await updateEmailPreferences(token, { search_alerts: newSearch, opportunity_alerts: newOpp });
    } catch {
      if (field === 'search_alerts') setSearchAlerts(!value);
      else setOpportunityAlerts(!value);
    }
  }

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

      <div className="relative z-10 max-w-2xl mx-auto px-6 pt-6 pb-10">
        <Link
          to="/"
          className="inline-flex items-center gap-2 text-sm text-dark-400 hover:text-dark-200 transition-colors mb-6 md:hidden"
        >
          <ArrowLeft size={16} strokeWidth={1.5} />
          Back
        </Link>
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

        {/* Email Notifications */}
        <div className="border-t border-dark-800/50 pt-8 mb-8">
          <h2 className="text-sm font-medium text-dark-300 uppercase tracking-wider mb-4 flex items-center gap-2">
            <Mail size={14} strokeWidth={1.5} />
            Email Notifications
          </h2>
          <p className="text-xs text-dark-500 mb-4">Manage which emails you receive from GovTrove.</p>
          {prefsLoading ? (
            <div className="text-dark-500 text-sm py-4 text-center">Loading...</div>
          ) : (
            <div className="space-y-3">
              <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-dark-200">Saved Search Alerts</h3>
                  <p className="text-xs text-dark-500 mt-0.5">Get notified when new opportunities match your saved searches.</p>
                </div>
                <button
                  onClick={() => togglePref('search_alerts', !searchAlerts)}
                  className={`relative w-10 h-5 rounded-full transition-colors duration-200 ${
                    searchAlerts ? 'bg-accent' : 'bg-dark-700'
                  }`}
                >
                  <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform duration-200 ${
                    searchAlerts ? 'translate-x-5' : ''
                  }`} />
                </button>
              </div>
              <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-dark-200">Opportunity Change Alerts</h3>
                  <p className="text-xs text-dark-500 mt-0.5">Get notified when a saved opportunity is amended or changed.</p>
                </div>
                <button
                  onClick={() => togglePref('opportunity_alerts', !opportunityAlerts)}
                  className={`relative w-10 h-5 rounded-full transition-colors duration-200 ${
                    opportunityAlerts ? 'bg-accent' : 'bg-dark-700'
                  }`}
                >
                  <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform duration-200 ${
                    opportunityAlerts ? 'translate-x-5' : ''
                  }`} />
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Account Actions */}
        <div className="border-t border-dark-800/50 pt-8">
          <h2 className="text-sm font-medium text-dark-300 uppercase tracking-wider mb-4 flex items-center gap-2">
            <User size={14} strokeWidth={1.5} />
            Account Actions
          </h2>
          <div className="space-y-4">
            {/* Data Export */}
            <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-4">
              <div className="flex items-start gap-4">
                <div className="flex-1 min-w-0">
                  <h3 className="text-sm font-medium text-dark-200 mb-1">Request My Data</h3>
                  <p className="text-xs text-dark-500 leading-relaxed">
                    Get a copy of all personal data we store about you, including your profile information,
                    saved opportunities, and saved searches. We'll process your request and email you
                    within 30 days, in compliance with data protection regulations.
                  </p>
                </div>
                <div className="shrink-0 pt-0.5">
                  {exportStatus === 'success' ? (
                    <span className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-green-400 border border-green-500/20 rounded-xl bg-green-500/5">
                      <Check size={15} strokeWidth={1.5} />
                      Submitted
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
                      Request
                    </button>
                  )}
                </div>
              </div>
            </div>

            {/* Account Deletion */}
            <div className="bg-dark-900/30 border border-red-500/10 rounded-xl p-4">
              <div className="flex items-start gap-4">
                <div className="flex-1 min-w-0">
                  <h3 className="text-sm font-medium text-dark-200 mb-1">Delete Account</h3>
                  <p className="text-xs text-dark-500 leading-relaxed">
                    Permanently delete your GovTrove account and all associated data, including your profile,
                    saved opportunities, saved searches, and usage history. This action cannot be undone.
                    We'll process your request within 30 days.
                  </p>
                </div>
                <div className="shrink-0 pt-0.5">
                  {deleteStatus === 'success' ? (
                    <span className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-green-400 border border-green-500/20 rounded-xl bg-green-500/5">
                      <Check size={15} strokeWidth={1.5} />
                      Submitted
                    </span>
                  ) : !showDeleteConfirm ? (
                    <button
                      onClick={() => setShowDeleteConfirm(true)}
                      className="inline-flex items-center gap-2 px-4 py-2.5 text-sm text-red-400/70 hover:text-red-400
                                 border border-red-500/20 hover:border-red-500/30 rounded-xl
                                 bg-red-500/5 hover:bg-red-500/10 transition-all duration-200"
                    >
                      <Trash2 size={15} strokeWidth={1.5} />
                      Delete
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
                        Confirm
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
              </div>
            </div>
          </div>

          {(exportStatus === 'error' || deleteStatus === 'error') && errorMsg && (
            <p className="mt-3 text-sm text-red-400">{errorMsg}</p>
          )}
        </div>
      </div>
    </div>
  );
}
