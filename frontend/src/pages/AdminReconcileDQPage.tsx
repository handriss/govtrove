import { useState, useEffect, useCallback } from 'react';
import { ChevronLeft, ChevronRight, ChevronUp, ChevronDown, X, ExternalLink, Table, BarChart3 } from 'lucide-react';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import {
  getAdminReconcileDQIssues,
  getAdminReconcileDQDetail,
  getAdminReconcileDQSummary,
  updateAdminReconcileDQResolution,
  type AdminReconcileDQRow,
  type AdminReconcileDQDetail,
  type DQSummaryRow,
} from '../services/api';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

type SortKey = 'date' | 'notice_id' | 'issue_type' | 'field' | 'resolved';

function SortHeader({ label, sortKey, active, order, onSort }: {
  label: string; sortKey: SortKey; active: boolean; order: string;
  onSort: (key: SortKey) => void;
}) {
  return (
    <th
      className="px-4 py-3 font-medium cursor-pointer select-none hover:text-dark-200 transition-colors"
      onClick={() => onSort(sortKey)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {active && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
      </span>
    </th>
  );
}

function DetailPanel({ detail, items, currentIndex, onNavigate, onClose, onSave }: {
  detail: AdminReconcileDQDetail;
  items: AdminReconcileDQRow[];
  currentIndex: number;
  onNavigate: (index: number) => void;
  onClose: () => void;
  onSave: (resolved: boolean, note: string) => Promise<void>;
}) {
  const [resolved, setResolved] = useState(detail.resolved);
  const [note, setNote] = useState(detail.resolution_note || '');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setResolved(detail.resolved);
    setNote(detail.resolution_note || '');
  }, [detail]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(resolved, note);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-lg bg-dark-900 border-l border-dark-700/50 overflow-y-auto">
        <div className="sticky top-0 bg-dark-900 border-b border-dark-700/50 px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <button
              onClick={() => onNavigate(currentIndex - 1)}
              disabled={currentIndex <= 0}
              className="p-1 rounded hover:bg-dark-800 disabled:opacity-30 transition-colors"
            >
              <ChevronLeft size={18} />
            </button>
            <span className="text-sm text-dark-400">{currentIndex + 1} / {items.length}</span>
            <button
              onClick={() => onNavigate(currentIndex + 1)}
              disabled={currentIndex >= items.length - 1}
              className="p-1 rounded hover:bg-dark-800 disabled:opacity-30 transition-colors"
            >
              <ChevronRight size={18} />
            </button>
          </div>
          <button onClick={onClose} className="p-1 rounded hover:bg-dark-800 transition-colors">
            <X size={18} />
          </button>
        </div>

        <div className="px-6 py-5 space-y-5">
          <div>
            <h3 className="text-lg font-medium text-dark-100 mb-1">Reconcile DQ #{detail.id}</h3>
            <p className="text-sm text-dark-400">{detail.notice_id}</p>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-dark-500">Date</span>
              <p className="text-dark-200">{formatDate(detail.snapshot_date)}</p>
            </div>
            <div>
              <span className="text-dark-500">Issue Type</span>
              <p className="text-dark-200">{detail.issue_type}</p>
            </div>
            <div>
              <span className="text-dark-500">Field</span>
              <p className="text-dark-200">{detail.field_name}</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-dark-500">CSV Value</span>
              <p className="text-dark-200 bg-dark-800/50 rounded px-3 py-2 mt-1 font-mono text-xs break-all">{detail.csv_value || '—'}</p>
            </div>
            <div>
              <span className="text-dark-500">API Value</span>
              <p className="text-dark-200 bg-dark-800/50 rounded px-3 py-2 mt-1 font-mono text-xs break-all">{detail.api_value || '—'}</p>
            </div>
          </div>

          {detail.opp_title && (
            <div className="border-t border-dark-700/50 pt-4">
              <h4 className="text-sm font-medium text-dark-300 mb-3">Linked Opportunity</h4>
              <div className="space-y-2 text-sm">
                <p className="text-dark-200">{detail.opp_title}</p>
                {detail.opp_sol_num && <p className="text-dark-400">Sol#: {detail.opp_sol_num}</p>}
                <div className="flex gap-3">
                  {detail.opp_type && (
                    <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-dark-800/60 border border-dark-700/30 text-dark-300">
                      {detail.opp_type}
                    </span>
                  )}
                  {detail.opp_active !== null && (
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${detail.opp_active ? 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400' : 'bg-dark-800/60 border-dark-700/30 text-dark-400'}`}>
                      {detail.opp_active ? 'Active' : 'Inactive'}
                    </span>
                  )}
                </div>
                {detail.opp_ui_link && (
                  <a href={detail.opp_ui_link} target="_blank" rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-accent hover:text-accent/80 text-xs">
                    View on SAM.gov <ExternalLink size={12} />
                  </a>
                )}
              </div>
            </div>
          )}

          <div className="border-t border-dark-700/50 pt-4 space-y-3">
            <h4 className="text-sm font-medium text-dark-300">Resolution</h4>
            <label className="flex items-center gap-2 text-sm text-dark-200 cursor-pointer">
              <input type="checkbox" checked={resolved} onChange={(e) => setResolved(e.target.checked)}
                className="rounded border-dark-600 bg-dark-800 text-accent focus:ring-accent/50" />
              Resolved
            </label>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Resolution note..."
              rows={3}
              className="w-full bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 text-sm text-dark-200 placeholder:text-dark-600 focus:outline-none focus:border-accent/50"
            />
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-4 py-2 bg-accent text-white rounded-lg text-sm font-medium hover:bg-accent/90 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

interface IssueTypeGroup {
  issueType: string;
  total: number;
  unresolved: number;
  fields: DQSummaryRow[];
}

function groupByDate(data: DQSummaryRow[]) {
  const byDate: Record<string, DQSummaryRow[]> = {};
  for (const row of data) (byDate[row.snapshot_date] ||= []).push(row);

  return Object.entries(byDate).map(([date, rows]) => {
    const byType: Record<string, IssueTypeGroup> = {};
    for (const row of rows) {
      const g = (byType[row.issue_type] ||= { issueType: row.issue_type, total: 0, unresolved: 0, fields: [] });
      g.total += row.total;
      g.unresolved += row.unresolved;
      g.fields.push(row);
    }
    const total = rows.reduce((s, r) => s + r.total, 0);
    const unresolved = rows.reduce((s, r) => s + r.unresolved, 0);
    return { date, total, unresolved, issueTypes: Object.values(byType) };
  });
}

function ExpandedRows({ date, issueType, fieldName, getToken }: {
  date: string; issueType: string; fieldName: string | null; getToken: () => Promise<string>;
}) {
  const [rows, setRows] = useState<AdminReconcileDQRow[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const token = await getToken();
        const result = await getAdminReconcileDQIssues(token, {
          limit: 100,
          snapshot_date: date,
          issue_type: issueType,
          field_name: fieldName || undefined,
        });
        setRows(result.items || []);
      } catch { setRows([]); }
      finally { setLoading(false); }
    })();
  }, [date, issueType, fieldName, getToken]);

  if (loading) return (
    <tr><td colSpan={5} className="px-8 py-2 text-dark-500 text-xs">Loading...</td></tr>
  );

  return (<>
    {rows.map((row) => (
      <tr key={row.id} className="bg-dark-800/20">
        <td className="pl-8 pr-4 py-1.5 text-dark-400 font-mono text-xs">{row.notice_id}</td>
        <td className="px-4 py-1.5 text-dark-400 text-xs truncate max-w-[12rem]">{row.csv_value || '—'}</td>
        <td className="px-4 py-1.5 text-dark-400 text-xs truncate max-w-[12rem]">{row.api_value || '—'}</td>
        <td className="px-4 py-1.5 text-right">
          <span className={`inline-flex px-1.5 py-0.5 rounded text-xs ${row.resolved ? 'text-emerald-400' : 'text-amber-400'}`}>
            {row.resolved ? 'Yes' : 'No'}
          </span>
        </td>
        <td></td>
      </tr>
    ))}
  </>);
}

function SummaryView({ data, getToken }: { data: DQSummaryRow[]; getToken: () => Promise<string> }) {
  const days = groupByDate(data);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  if (days.length === 0) {
    return <div className="text-center text-dark-500 text-sm py-20">No data</div>;
  }

  const toggleKey = (key: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });
  };

  return (
    <div className="space-y-6">
      {days.map(({ date, total: dayTotal, unresolved: dayUnresolved, issueTypes }) => (
        <div key={date} className="rounded-xl border border-dark-700/50 overflow-hidden">
          <div className="px-4 py-3 bg-dark-800/30 border-b border-dark-700/50 flex items-center justify-between">
            <span className="text-sm font-medium text-dark-200">{formatDate(date)}</span>
            <div className="flex gap-4 text-xs text-dark-400">
              <span>{dayTotal} total</span>
              {dayUnresolved > 0 && (
                <span className="text-amber-400">{dayUnresolved} unresolved</span>
              )}
            </div>
          </div>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                <th className="px-4 py-2 font-medium">Issue Type / Field</th>
                <th className="px-4 py-2 font-medium text-right">Total</th>
                <th className="px-4 py-2 font-medium text-right">Unresolved</th>
                <th className="px-4 py-2 font-medium text-right">Resolved</th>
                <th className="w-8"></th>
              </tr>
            </thead>
            <tbody>
              {issueTypes.map((group) => {
                const singleField = group.fields.length === 1;
                const rowKey = `${date}::${group.issueType}::${singleField ? (group.fields[0].field_name || '') : ''}`;

                if (singleField) {
                  const row = group.fields[0];
                  const isOpen = expanded.has(rowKey);
                  return (<>
                    <tr
                      key={rowKey}
                      onClick={() => toggleKey(rowKey)}
                      className="border-b border-dark-700/30 last:border-0 cursor-pointer hover:bg-dark-800/30"
                    >
                      <td className="px-4 py-2 text-dark-200">
                        {group.issueType}
                        {row.field_name && <span className="text-dark-500 ml-2">{row.field_name}</span>}
                      </td>
                      <td className="px-4 py-2 text-dark-300 text-right">{group.total}</td>
                      <td className="px-4 py-2 text-right">
                        <span className={group.unresolved > 0 ? 'text-amber-400' : 'text-dark-500'}>{group.unresolved}</span>
                      </td>
                      <td className="px-4 py-2 text-right">
                        <span className={group.total - group.unresolved > 0 ? 'text-emerald-400' : 'text-dark-500'}>{group.total - group.unresolved}</span>
                      </td>
                      <td className="px-2 py-2 text-dark-500">
                        <ChevronRight size={14} className={`transition-transform ${isOpen ? 'rotate-90' : ''}`} />
                      </td>
                    </tr>
                    {isOpen && <ExpandedRows date={date} issueType={group.issueType} fieldName={row.field_name} getToken={getToken} />}
                  </>);
                }

                return (<>
                  <tr key={`${date}::${group.issueType}::header`} className="border-b border-dark-700/30">
                    <td className="px-4 py-2 text-dark-200 font-medium">{group.issueType}</td>
                    <td className="px-4 py-2 text-dark-300 text-right">{group.total}</td>
                    <td className="px-4 py-2 text-right">
                      <span className={group.unresolved > 0 ? 'text-amber-400' : 'text-dark-500'}>{group.unresolved}</span>
                    </td>
                    <td className="px-4 py-2 text-right">
                      <span className={group.total - group.unresolved > 0 ? 'text-emerald-400' : 'text-dark-500'}>{group.total - group.unresolved}</span>
                    </td>
                    <td></td>
                  </tr>
                  {group.fields.map((row) => {
                    const fieldKey = `${date}::${group.issueType}::${row.field_name || ''}`;
                    const isOpen = expanded.has(fieldKey);
                    return (<>
                      <tr
                        key={fieldKey}
                        onClick={() => toggleKey(fieldKey)}
                        className="border-b border-dark-700/30 last:border-0 cursor-pointer hover:bg-dark-800/30"
                      >
                        <td className="pl-8 pr-4 py-2 text-dark-300">{row.field_name || '—'}</td>
                        <td className="px-4 py-2 text-dark-300 text-right">{row.total}</td>
                        <td className="px-4 py-2 text-right">
                          <span className={row.unresolved > 0 ? 'text-amber-400' : 'text-dark-500'}>{row.unresolved}</span>
                        </td>
                        <td className="px-4 py-2 text-right">
                          <span className={row.total - row.unresolved > 0 ? 'text-emerald-400' : 'text-dark-500'}>{row.total - row.unresolved}</span>
                        </td>
                        <td className="px-2 py-2 text-dark-500">
                          <ChevronRight size={14} className={`transition-transform ${isOpen ? 'rotate-90' : ''}`} />
                        </td>
                      </tr>
                      {isOpen && <ExpandedRows date={date} issueType={group.issueType} fieldName={row.field_name} getToken={getToken} />}
                    </>);
                  })}
                </>);
              })}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  );
}

export default function AdminReconcileDQPage() {
  return (
    <AdminLayout>
      <AdminReconcileDQContent />
    </AdminLayout>
  );
}

function AdminReconcileDQContent() {
  const { getToken } = useAdminContext();
  const [view, setView] = useState<'table' | 'summary'>('summary');
  const [items, setItems] = useState<AdminReconcileDQRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [sort, setSort] = useState<SortKey>('date');
  const [order, setOrder] = useState('desc');
  const [resolvedFilter, setResolvedFilter] = useState('');
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [detail, setDetail] = useState<AdminReconcileDQDetail | null>(null);
  const [summaryData, setSummaryData] = useState<DQSummaryRow[]>([]);
  const limit = 50;

  const fetchData = useCallback(async (p: number, s: string, o: string, rf: string) => {
    setLoading(true);
    try {
      const token = await getToken();
      const params: Record<string, string | number> = { page: p, limit, sort: s, order: o };
      if (rf) params.resolved = rf;
      const result = await getAdminReconcileDQIssues(token, params as Parameters<typeof getAdminReconcileDQIssues>[1]);
      setItems(result.items || []);
      setTotal(result.total);
      setPage(p);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  const fetchSummary = useCallback(async () => {
    setLoading(true);
    try {
      const token = await getToken();
      const data = await getAdminReconcileDQSummary(token);
      setSummaryData(data || []);
    } catch {
      setSummaryData([]);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => {
    if (view === 'table') {
      fetchData(1, sort, order, resolvedFilter);
    } else {
      fetchSummary();
    }
  }, [view]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSort = (key: SortKey) => {
    const newOrder = sort === key && order === 'desc' ? 'asc' : 'desc';
    setSort(key);
    setOrder(newOrder);
    fetchData(1, key, newOrder, resolvedFilter);
  };

  const handleFilterChange = (value: string) => {
    setResolvedFilter(value);
    fetchData(1, sort, order, value);
  };

  const openDetail = async (index: number) => {
    try {
      const token = await getToken();
      const d = await getAdminReconcileDQDetail(token, items[index].id);
      setDetail(d);
      setSelectedIndex(index);
    } catch { /* ignore */ }
  };

  const handleNavigate = (index: number) => {
    if (index >= 0 && index < items.length) openDetail(index);
  };

  const handleSave = async (resolved: boolean, note: string) => {
    if (selectedIndex === null || !detail) return;
    const token = await getToken();
    await updateAdminReconcileDQResolution(token, detail.id, { resolved, resolution_note: note });
    const updated = [...items];
    updated[selectedIndex] = { ...updated[selectedIndex], resolved, resolution_note: note };
    setItems(updated);
    setDetail({ ...detail, resolved, resolution_note: note });
  };

  const summaryTotal = summaryData.reduce((s, r) => s + r.total, 0);

  return (
    <div className="max-w-6xl mx-auto px-6 py-10">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-dark-100">
          Reconcile Data Quality{' '}
          <span className="text-dark-500 text-lg font-normal">
            ({view === 'table' ? total : summaryTotal})
          </span>
        </h1>
        <div className="flex items-center gap-3">
          <div className="flex rounded-lg border border-dark-700/50 overflow-hidden">
            <button
              onClick={() => setView('summary')}
              className={`px-3 py-1.5 text-sm flex items-center gap-1.5 transition-colors ${
                view === 'summary'
                  ? 'bg-dark-700/50 text-dark-100'
                  : 'text-dark-400 hover:text-dark-200'
              }`}
            >
              <BarChart3 size={14} /> Summary
            </button>
            <button
              onClick={() => setView('table')}
              className={`px-3 py-1.5 text-sm flex items-center gap-1.5 transition-colors ${
                view === 'table'
                  ? 'bg-dark-700/50 text-dark-100'
                  : 'text-dark-400 hover:text-dark-200'
              }`}
            >
              <Table size={14} /> Table
            </button>
          </div>
          {view === 'table' && (
            <select
              value={resolvedFilter}
              onChange={(e) => handleFilterChange(e.target.value)}
              className="bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-1.5 text-sm text-dark-200 focus:outline-none focus:border-accent/50"
            >
              <option value="">All</option>
              <option value="false">Unresolved</option>
              <option value="true">Resolved</option>
            </select>
          )}
        </div>
      </div>

      {loading && !items.length && !summaryData.length ? (
        <div className="flex items-center justify-center py-20 text-dark-500 text-sm">Loading...</div>
      ) : view === 'summary' ? (
        <SummaryView data={summaryData} getToken={getToken} />
      ) : (
        <>
          <div className="overflow-x-auto rounded-xl border border-dark-700/50">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                  <SortHeader label="Date" sortKey="date" active={sort === 'date'} order={order} onSort={handleSort} />
                  <SortHeader label="Notice ID" sortKey="notice_id" active={sort === 'notice_id'} order={order} onSort={handleSort} />
                  <SortHeader label="Issue Type" sortKey="issue_type" active={sort === 'issue_type'} order={order} onSort={handleSort} />
                  <SortHeader label="Field" sortKey="field" active={sort === 'field'} order={order} onSort={handleSort} />
                  <th className="px-4 py-3 font-medium">CSV Value</th>
                  <th className="px-4 py-3 font-medium">API Value</th>
                  <SortHeader label="Resolved" sortKey="resolved" active={sort === 'resolved'} order={order} onSort={handleSort} />
                </tr>
              </thead>
              <tbody>
                {items.map((row, i) => (
                  <tr
                    key={row.id}
                    onClick={() => openDetail(i)}
                    className="border-b border-dark-700/30 last:border-0 hover:bg-dark-800/30 cursor-pointer"
                  >
                    <td className="px-4 py-3 text-dark-400 whitespace-nowrap">{formatDate(row.snapshot_date)}</td>
                    <td className="px-4 py-3 text-dark-200 font-mono text-xs">{row.notice_id}</td>
                    <td className="px-4 py-3 text-dark-300">{row.issue_type}</td>
                    <td className="px-4 py-3 text-dark-300">{row.field_name}</td>
                    <td className="px-4 py-3 text-dark-400 truncate max-w-[10rem]">{row.csv_value || '—'}</td>
                    <td className="px-4 py-3 text-dark-400 truncate max-w-[10rem]">{row.api_value || '—'}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
                        row.resolved
                          ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400'
                          : 'bg-amber-500/15 border border-amber-500/30 text-amber-400'
                      }`}>
                        {row.resolved ? 'Yes' : 'No'}
                      </span>
                    </td>
                  </tr>
                ))}
                {items.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-dark-500">No issues found</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between mt-4">
            <span className="text-sm text-dark-500">
              Page {page} of {Math.max(1, Math.ceil(total / limit))}
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => fetchData(page - 1, sort, order, resolvedFilter)}
                disabled={page <= 1}
                className="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-dark-300 bg-dark-800 rounded-lg border border-dark-700/50 hover:bg-dark-700/50 disabled:opacity-30 transition-colors"
              >
                <ChevronLeft size={16} /> Previous
              </button>
              <button
                onClick={() => fetchData(page + 1, sort, order, resolvedFilter)}
                disabled={page >= Math.ceil(total / limit)}
                className="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-dark-300 bg-dark-800 rounded-lg border border-dark-700/50 hover:bg-dark-700/50 disabled:opacity-30 transition-colors"
              >
                Next <ChevronRight size={16} />
              </button>
            </div>
          </div>
        </>
      )}

      {selectedIndex !== null && detail && (
        <DetailPanel
          detail={detail}
          items={items}
          currentIndex={selectedIndex}
          onNavigate={handleNavigate}
          onClose={() => { setSelectedIndex(null); setDetail(null); }}
          onSave={handleSave}
        />
      )}
    </div>
  );
}
