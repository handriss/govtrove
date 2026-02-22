import { Link, Navigate } from 'react-router-dom';
import { ArrowLeft, Bell, Mail, Search, Star, FileText } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';

const planned = [
  { icon: Search, label: 'New matches for your saved searches' },
  { icon: Star, label: 'Updates to your saved opportunities' },
  { icon: FileText, label: 'Deadline reminders' },
  { icon: Mail, label: 'Weekly digest of new opportunities' },
];

export default function NotificationsPage() {
  const { user, isLoading, isAuthenticated } = useAppAuth();

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

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-2xl mx-auto px-6 pt-10 pb-10">
        <div className="flex items-center gap-4 mb-10">
          <Link to="/" className="text-dark-400 hover:text-dark-200 transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <h1 className="text-2xl font-semibold text-dark-100">Notifications</h1>
        </div>

        <div className="flex flex-col items-center text-center py-8">
          <div className="w-16 h-16 rounded-2xl bg-accent/10 border border-accent/20 flex items-center justify-center mb-6">
            <Bell size={28} className="text-accent/60" strokeWidth={1.5} />
          </div>

          <h2 className="text-lg font-medium text-dark-200 mb-2">Coming Soon</h2>
          <p className="text-sm text-dark-400 max-w-md mb-10">
            Email notifications and alerts are on the way. You'll be able to manage
            all your notification preferences right here.
          </p>

          <div className="w-full max-w-sm space-y-3">
            <p className="text-xs text-dark-500 uppercase tracking-wider mb-4">Planned features</p>
            {planned.map(({ icon: Icon, label }) => (
              <div
                key={label}
                className="flex items-center gap-3 px-4 py-3 rounded-xl border border-dark-800/50 bg-dark-900/30"
              >
                <Icon size={16} className="text-dark-500 shrink-0" strokeWidth={1.5} />
                <span className="text-sm text-dark-300">{label}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
