import { Link, Navigate } from 'react-router-dom';
import { ArrowLeft, Bell, Check, CheckCheck, Trash2, Loader2, Search, FileText, AlertCircle } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { useUpdates, useUpdatesCount } from '../hooks/useUpdates';
import type { UserUpdate } from '../types/api';
import AuthButton from '../components/AuthButton';

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const yesterday = new Date(today.getTime() - 86400000);
  const updateDate = new Date(d.getFullYear(), d.getMonth(), d.getDate());

  if (updateDate.getTime() === today.getTime()) return 'Today';
  if (updateDate.getTime() === yesterday.getTime()) return 'Yesterday';
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function groupByDate(updates: UserUpdate[]): Map<string, UserUpdate[]> {
  const groups = new Map<string, UserUpdate[]>();
  for (const update of updates) {
    const key = formatDate(update.created_at);
    const existing = groups.get(key) || [];
    existing.push(update);
    groups.set(key, existing);
  }
  return groups;
}

function UpdateIcon({ type }: { type: string }) {
  switch (type) {
    case 'saved_search_matches':
      return <Search size={16} className="text-accent" />;
    case 'opportunity_amended':
      return <FileText size={16} className="text-blue-400" />;
    case 'opportunity_changed':
      return <AlertCircle size={16} className="text-amber-400" />;
    default:
      return <Bell size={16} className="text-dark-500" />;
  }
}

function UpdateCard({ update, onMarkRead, onDelete }: {
  update: UserUpdate;
  onMarkRead: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const details = update.details as Record<string, unknown> | undefined;
  const changes = details?.changes as Array<{ field: string; old?: string; new?: string }> | undefined;

  return (
    <div className={`group flex gap-3 px-4 py-3 rounded-xl border transition-all ${
      update.is_read
        ? 'border-dark-800/30 bg-dark-900/20'
        : 'border-accent/20 bg-accent/5 border-l-2 border-l-accent/50'
    }`}>
      <div className="mt-0.5 shrink-0">
        <UpdateIcon type={update.update_type} />
      </div>
      <div className="flex-1 min-w-0">
        <p className={`text-sm ${update.is_read ? 'text-dark-400' : 'text-dark-200'}`}>
          {update.summary}
        </p>

        {/* Show opportunity links for search matches */}
        {update.update_type === 'saved_search_matches' && update.opportunity_ids && update.opportunity_ids.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mt-2">
            {update.opportunity_ids.slice(0, 3).map((id) => (
              <Link
                key={id}
                to={`/opportunity/${id}`}
                onClick={() => !update.is_read && onMarkRead(update.id)}
                className="text-xs text-accent/80 hover:text-accent bg-accent/5 hover:bg-accent/10
                           px-2 py-0.5 rounded transition-colors"
              >
                View #{id}
              </Link>
            ))}
            {update.opportunity_ids.length > 3 && (
              <span className="text-xs text-dark-600">+{update.opportunity_ids.length - 3} more</span>
            )}
          </div>
        )}

        {/* Show amendment link */}
        {update.update_type === 'opportunity_amended' && update.opportunity_ids?.[0] && (
          <Link
            to={`/opportunity/${update.opportunity_ids[0]}`}
            onClick={() => !update.is_read && onMarkRead(update.id)}
            className="inline-block mt-2 text-xs text-accent/80 hover:text-accent transition-colors"
          >
            View amendment
          </Link>
        )}

        {/* Show field changes diff */}
        {update.update_type === 'opportunity_changed' && changes && changes.length > 0 && (
          <div className="mt-2 space-y-1">
            {changes.map((c, i) => (
              <div key={i} className="text-xs">
                <span className="text-dark-500 capitalize">{c.field.replace(/_/g, ' ')}:</span>
                {c.old && <span className="text-red-400/60 line-through ml-1">{truncateValue(c.old)}</span>}
                {c.new && <span className="text-green-400/60 ml-1">{truncateValue(c.new)}</span>}
              </div>
            ))}
          </div>
        )}

        <p className="text-[10px] text-dark-600 mt-1.5">
          {new Date(update.created_at).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}
        </p>
      </div>
      <div className="flex items-start gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
        {!update.is_read && (
          <button
            onClick={() => onMarkRead(update.id)}
            className="text-dark-500 hover:text-accent p-1 transition-colors"
            title="Mark as read"
          >
            <Check size={14} />
          </button>
        )}
        <button
          onClick={() => onDelete(update.id)}
          className="text-dark-500 hover:text-red-400 p-1 transition-colors"
          title="Delete"
        >
          <Trash2 size={14} />
        </button>
      </div>
    </div>
  );
}

function truncateValue(val: string): string {
  return val.length > 80 ? val.slice(0, 77) + '...' : val;
}

export default function NotificationsPage() {
  const { user, isLoading: authLoading, isAuthenticated } = useAppAuth();
  const { count, refetch: refetchCount } = useUpdatesCount();
  const {
    updates, total, loading, hasMore, filter, setFilter,
    loadMore, markRead, markAllRead, remove,
  } = useUpdates();

  if (authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 rounded-full bg-dark-800/50 animate-pulse" />
      </div>
    );
  }

  if (!isAuthenticated || !user) {
    return <Navigate to="/" replace />;
  }

  const handleMarkRead = async (id: string) => {
    await markRead(id);
    refetchCount();
  };

  const handleMarkAllRead = async () => {
    await markAllRead();
    refetchCount();
  };

  const handleDelete = async (id: string) => {
    await remove(id);
    refetchCount();
  };

  const grouped = groupByDate(updates);

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="absolute top-4 right-6 z-20">
        <AuthButton />
      </div>

      <div className="relative z-10 max-w-2xl mx-auto px-6 pt-10 pb-10">
        <div className="flex items-center gap-4 mb-6">
          <Link to="/" className="text-dark-400 hover:text-dark-200 transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <h1 className="text-2xl font-semibold text-dark-100">Notifications</h1>
          {count.unread > 0 && (
            <span className="text-xs font-medium text-accent bg-accent/10 px-2 py-0.5 rounded-full">
              {count.unread} unread
            </span>
          )}
        </div>

        {/* Filter tabs + actions */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex gap-1">
            {(['all', 'unread'] as const).map((f) => (
              <button
                key={f}
                onClick={() => setFilter(f)}
                className={`text-xs px-3 py-1.5 rounded-lg transition-colors capitalize ${
                  filter === f
                    ? 'bg-dark-800/60 text-dark-200'
                    : 'text-dark-500 hover:text-dark-300'
                }`}
              >
                {f}
              </button>
            ))}
          </div>
          {count.unread > 0 && (
            <button
              onClick={handleMarkAllRead}
              className="flex items-center gap-1.5 text-xs text-dark-500 hover:text-dark-300 transition-colors"
            >
              <CheckCheck size={14} />
              Mark all read
            </button>
          )}
        </div>

        {/* Updates feed */}
        {loading && updates.length === 0 ? (
          <div className="flex items-center justify-center gap-2 text-dark-500 text-sm py-12">
            <Loader2 size={16} className="animate-spin" />
            Loading notifications...
          </div>
        ) : updates.length === 0 ? (
          <div className="flex flex-col items-center text-center py-12">
            <div className="w-16 h-16 rounded-2xl bg-dark-800/30 border border-dark-800/50 flex items-center justify-center mb-6">
              <Bell size={28} className="text-dark-700" strokeWidth={1.5} />
            </div>
            <h2 className="text-lg font-medium text-dark-300 mb-2">
              {filter === 'unread' ? 'All caught up' : 'No notifications yet'}
            </h2>
            <p className="text-sm text-dark-500 max-w-md">
              {filter === 'unread'
                ? 'You have no unread notifications.'
                : 'Save a search or star an opportunity to start receiving notifications when new matches appear or details change.'}
            </p>
            {filter === 'all' && (
              <Link
                to="/"
                className="mt-4 text-sm text-accent hover:text-accent/80 transition-colors"
              >
                Start searching
              </Link>
            )}
          </div>
        ) : (
          <div className="space-y-6">
            {[...grouped.entries()].map(([date, dateUpdates]) => (
              <div key={date}>
                <p className="text-xs text-dark-600 uppercase tracking-wider mb-2 px-1">{date}</p>
                <div className="space-y-2">
                  {dateUpdates.map((update) => (
                    <UpdateCard
                      key={update.id}
                      update={update}
                      onMarkRead={handleMarkRead}
                      onDelete={handleDelete}
                    />
                  ))}
                </div>
              </div>
            ))}

            {hasMore && (
              <button
                onClick={loadMore}
                disabled={loading}
                className="w-full py-3 text-sm text-dark-500 hover:text-dark-300 transition-colors
                           border border-dark-800/50 rounded-xl hover:border-dark-700/50"
              >
                {loading ? (
                  <Loader2 size={14} className="animate-spin mx-auto" />
                ) : (
                  `Load more (${total - updates.length} remaining)`
                )}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
