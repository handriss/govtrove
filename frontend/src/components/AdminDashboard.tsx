import { useState } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { useSearchAnalytics } from '../hooks/useAnalytics';

interface AdminDashboardProps {
  onClose: () => void;
}

export function AdminDashboard({ onClose }: AdminDashboardProps) {
  const [period, setPeriod] = useState('7d');
  const { data: analytics, isLoading, error } = useSearchAnalytics(period);

  const zeroResultRate =
    analytics && analytics.event_counts.searches > 0
      ? (analytics.zero_result_searches.reduce((sum, s) => sum + s.count, 0) /
          analytics.event_counts.searches) *
        100
      : 0;

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">Search Analytics</h1>
          <div className="flex items-center gap-4">
            <select
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
            >
              <option value="24h">Last 24 hours</option>
              <option value="7d">Last 7 days</option>
              <option value="30d">Last 30 days</option>
            </select>
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900"
            >
              Back to Search
            </button>
          </div>
        </div>
      </header>

      <main className="p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-gray-500">Loading analytics...</div>
          </div>
        ) : error ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-red-500">Failed to load analytics</div>
          </div>
        ) : analytics ? (
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <StatCard
                title="Total Searches"
                value={analytics.event_counts.searches}
              />
              <StatCard
                title="Total Clicks"
                value={analytics.event_counts.clicks}
              />
              <StatCard
                title="Click-Through Rate"
                value={`${analytics.click_stats.click_through_rate.toFixed(1)}%`}
              />
              <StatCard
                title="Zero-Result Rate"
                value={`${zeroResultRate.toFixed(1)}%`}
              />
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-white rounded-lg border border-gray-200 p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">
                  Popular Searches
                </h3>
                {analytics.popular_searches.length > 0 ? (
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart
                      data={analytics.popular_searches.slice(0, 10)}
                      layout="vertical"
                      margin={{ left: 100 }}
                    >
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis type="number" />
                      <YAxis
                        dataKey="query"
                        type="category"
                        width={100}
                        tick={{ fontSize: 12 }}
                      />
                      <Tooltip />
                      <Bar dataKey="count" fill="#3b82f6" />
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="h-64 flex items-center justify-center text-gray-500">
                    No search data available
                  </div>
                )}
              </div>

              <div className="bg-white rounded-lg border border-gray-200 p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">
                  Clicks by Position
                </h3>
                {analytics.click_stats.by_position.length > 0 ? (
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart data={analytics.click_stats.by_position.slice(0, 10)}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="position" />
                      <YAxis />
                      <Tooltip />
                      <Bar dataKey="clicks" fill="#10b981" />
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="h-64 flex items-center justify-center text-gray-500">
                    No click data available
                  </div>
                )}
              </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-white rounded-lg border border-gray-200 p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">
                  Top Search Terms
                </h3>
                <SearchTermTable terms={analytics.popular_searches} />
              </div>

              <div className="bg-white rounded-lg border border-gray-200 p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">
                  Zero-Result Searches
                </h3>
                <SearchTermTable terms={analytics.zero_result_searches} />
              </div>
            </div>

            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h3 className="text-lg font-medium text-gray-900 mb-4">
                Filter Usage
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <FilterUsageTable
                  title="Opportunity Types"
                  filters={analytics.filter_usage.types}
                />
                <FilterUsageTable
                  title="Set-Asides"
                  filters={analytics.filter_usage.set_asides}
                />
                <FilterUsageTable
                  title="States"
                  filters={analytics.filter_usage.states}
                />
              </div>
            </div>

            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h3 className="text-lg font-medium text-gray-900 mb-4">
                Event Breakdown
              </h3>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <EventStat label="Searches" value={analytics.event_counts.searches} />
                <EventStat label="Filter Changes" value={analytics.event_counts.filters} />
                <EventStat label="Clicks" value={analytics.event_counts.clicks} />
                <EventStat label="Page Views" value={analytics.event_counts.pages} />
              </div>
            </div>
          </div>
        ) : null}
      </main>
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: string | number;
}

function StatCard({ title, value }: StatCardProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <div className="text-sm text-gray-500">{title}</div>
      <div className="text-2xl font-bold text-gray-900 mt-1">{value}</div>
    </div>
  );
}

interface SearchTermTableProps {
  terms: { query: string; count: number }[];
}

function SearchTermTable({ terms }: SearchTermTableProps) {
  if (terms.length === 0) {
    return <div className="text-gray-500 text-sm">No data available</div>;
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-200">
            <th className="text-left py-2 font-medium text-gray-700">Query</th>
            <th className="text-right py-2 font-medium text-gray-700">Count</th>
          </tr>
        </thead>
        <tbody>
          {terms.slice(0, 10).map((term, idx) => (
            <tr key={idx} className="border-b border-gray-100">
              <td className="py-2 text-gray-900 truncate max-w-xs" title={term.query}>
                {term.query}
              </td>
              <td className="py-2 text-right text-gray-600">{term.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

interface FilterUsageTableProps {
  title: string;
  filters: { value: string; count: number }[];
}

function FilterUsageTable({ title, filters }: FilterUsageTableProps) {
  return (
    <div>
      <h4 className="text-sm font-medium text-gray-700 mb-2">{title}</h4>
      {filters.length === 0 ? (
        <div className="text-gray-500 text-sm">No data available</div>
      ) : (
        <div className="space-y-1">
          {filters.slice(0, 5).map((filter, idx) => (
            <div key={idx} className="flex justify-between text-sm">
              <span className="text-gray-900 truncate" title={filter.value}>
                {filter.value}
              </span>
              <span className="text-gray-500 ml-2">{filter.count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface EventStatProps {
  label: string;
  value: number;
}

function EventStat({ label, value }: EventStatProps) {
  return (
    <div className="text-center">
      <div className="text-xl font-semibold text-gray-900">{value}</div>
      <div className="text-sm text-gray-500">{label}</div>
    </div>
  );
}
