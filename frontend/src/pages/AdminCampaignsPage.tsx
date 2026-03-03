import { useState, useEffect, useCallback } from 'react';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import { getAdminUTMAnalytics, type UTMAnalytics } from '../services/api';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function CampaignsContent() {
  const { getToken } = useAdminContext();
  const [period, setPeriod] = useState('30d');
  const [data, setData] = useState<UTMAnalytics | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async (p: string) => {
    setLoading(true);
    try {
      const token = await getToken();
      setData(await getAdminUTMAnalytics(token, p));
    } catch { /* ignore */ } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => {
    fetchData(period);
  }, [period]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="max-w-6xl mx-auto px-6 py-10">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-dark-100">
          Campaign Tracking{' '}
          {data && <span className="text-dark-500 text-lg font-normal">({data.total_visits} visits)</span>}
        </h1>
        <select
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          className="bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-1.5 text-sm text-dark-200 focus:outline-none focus:border-accent/50"
        >
          <option value="7d">Last 7 days</option>
          <option value="30d">Last 30 days</option>
          <option value="90d">Last 90 days</option>
          <option value="all">All time</option>
        </select>
      </div>

      {loading && !data ? (
        <div className="flex items-center justify-center py-20 text-dark-500 text-sm">Loading...</div>
      ) : !data ? (
        <div className="flex items-center justify-center py-20 text-dark-500 text-sm">Failed to load data</div>
      ) : (
        <div className="space-y-8">
          {/* Campaign Funnel */}
          <section>
            <h2 className="text-lg font-medium text-dark-200 mb-3">Campaign Funnel</h2>
            {data.campaigns && data.campaigns.length > 0 ? (
              <div className="overflow-x-auto rounded-xl border border-dark-700/50">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                      <th className="px-4 py-3 font-medium">Campaign</th>
                      <th className="px-4 py-3 font-medium text-right">Landing</th>
                      <th className="px-4 py-3 font-medium text-right">App</th>
                      <th className="px-4 py-3 font-medium text-right">Total</th>
                      <th className="px-4 py-3 font-medium text-right">Click-Through</th>
                      <th className="px-4 py-3 font-medium">First Visit</th>
                      <th className="px-4 py-3 font-medium">Last Visit</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.campaigns.map((c) => {
                      const ctr = c.landing_visits > 0 ? ((c.app_visits / c.landing_visits) * 100).toFixed(1) : '-';
                      return (
                        <tr key={c.campaign} className="border-b border-dark-700/30 hover:bg-dark-800/50 transition-colors">
                          <td className="px-4 py-3 text-dark-200 font-mono text-xs">{c.campaign}</td>
                          <td className="px-4 py-3 text-dark-300 text-right">{c.landing_visits}</td>
                          <td className="px-4 py-3 text-dark-300 text-right">{c.app_visits}</td>
                          <td className="px-4 py-3 text-dark-200 text-right font-medium">{c.total_visits}</td>
                          <td className="px-4 py-3 text-dark-300 text-right">{ctr === '-' ? '-' : `${ctr}%`}</td>
                          <td className="px-4 py-3 text-dark-400 text-xs">{formatDate(c.first_visit)}</td>
                          <td className="px-4 py-3 text-dark-400 text-xs">{formatDate(c.last_visit)}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="text-dark-500 text-sm py-8 text-center border border-dark-700/50 rounded-xl">No campaign visits yet</div>
            )}
          </section>

          {/* Sources */}
          {data.sources && data.sources.length > 0 && (
            <section>
              <h2 className="text-lg font-medium text-dark-200 mb-3">Traffic Sources</h2>
              <div className="overflow-x-auto rounded-xl border border-dark-700/50">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                      <th className="px-4 py-3 font-medium">Source</th>
                      <th className="px-4 py-3 font-medium">Medium</th>
                      <th className="px-4 py-3 font-medium text-right">Visits</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.sources.map((s, i) => (
                      <tr key={i} className="border-b border-dark-700/30 hover:bg-dark-800/50 transition-colors">
                        <td className="px-4 py-3 text-dark-200">{s.source}</td>
                        <td className="px-4 py-3 text-dark-300">{s.medium}</td>
                        <td className="px-4 py-3 text-dark-200 text-right font-medium">{s.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}

          {/* Search Activity */}
          {data.search_activity && data.search_activity.length > 0 && (
            <section>
              <h2 className="text-lg font-medium text-dark-200 mb-3">Campaign Search Activity</h2>
              <div className="space-y-4">
                {data.search_activity.map((sa) => (
                  <div key={sa.campaign} className="rounded-xl border border-dark-700/50 p-4">
                    <div className="flex items-center justify-between mb-3">
                      <span className="font-mono text-xs text-dark-200">{sa.campaign}</span>
                      <div className="flex gap-4 text-xs text-dark-400">
                        <span>{sa.total_searches} searches</span>
                        <span>{sa.total_views} views</span>
                      </div>
                    </div>
                    {sa.top_queries && sa.top_queries.length > 0 && (
                      <div className="space-y-1">
                        {sa.top_queries.map((q, i) => (
                          <div key={i} className="flex items-center justify-between text-sm">
                            <span className="text-dark-300 truncate mr-4">{q.query}</span>
                            <span className="text-dark-500 text-xs shrink-0">{q.count}x</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* Daily Visits */}
          {data.daily_visits && data.daily_visits.length > 0 && (
            <section>
              <h2 className="text-lg font-medium text-dark-200 mb-3">Visits by Day</h2>
              <div className="rounded-xl border border-dark-700/50 p-4">
                <div className="space-y-1 text-sm">
                  {data.daily_visits.map((d) => (
                    <div key={d.date} className="flex items-center justify-between">
                      <span className="text-dark-400">{formatDate(d.date)}</span>
                      <span className="text-dark-200 font-medium">{d.count}</span>
                    </div>
                  ))}
                </div>
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}

export default function AdminCampaignsPage() {
  return (
    <AdminLayout>
      <CampaignsContent />
    </AdminLayout>
  );
}
