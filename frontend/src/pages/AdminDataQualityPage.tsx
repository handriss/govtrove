import { useState, useEffect, useCallback } from 'react';
import { ChevronLeft, ChevronRight, ChevronUp, ChevronDown, X, ExternalLink } from 'lucide-react';
import AdminLayout, { useAdminContext } from '../components/AdminLayout';
import {
  getAdminDataQualityIssues,
  getAdminDataQualityDetail,
  updateAdminDataQualityResolution,
  type AdminDataQualityRow,
  type AdminDataQualityDetail,
} from '../services/api';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

type SortKey = 'date' | 'notice_id' | 'source' | 'issue_type' | 'field' | 'resolved';

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
  detail: AdminDataQualityDetail;
  items: AdminDataQualityRow[];
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
            <h3 className="text-lg font-medium text-dark-100 mb-1">DQ Issue #{detail.id}</h3>
            <p className="text-sm text-dark-400">{detail.notice_id}</p>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-dark-500">Date</span>
              <p className="text-dark-200">{formatDate(detail.snapshot_date)}</p>
            </div>
            <div>
              <span className="text-dark-500">Source</span>
              <p className="text-dark-200">{detail.source}</p>
            </div>
            <div>
              <span className="text-dark-500">Issue Type</span>
              <p className="text-dark-200">{detail.issue_type}</p>
            </div>
            <div>
              <span className="text-dark-500">Field</span>
              <p className="text-dark-200">{detail.field_name || '—'}</p>
            </div>
          </div>

          {detail.field_value && (
            <div className="text-sm">
              <span className="text-dark-500">Value</span>
              <p className="text-dark-200 bg-dark-800/50 rounded px-3 py-2 mt-1 font-mono text-xs break-all">{detail.field_value}</p>
            </div>
          )}

          {detail.description && (
            <div className="text-sm">
              <span className="text-dark-500">Description</span>
              <p className="text-dark-200 mt-1">{detail.description}</p>
            </div>
          )}

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

export default function AdminDataQualityPage() {
  return (
    <AdminLayout>
      <AdminDataQualityContent />
    </AdminLayout>
  );
}

function AdminDataQualityContent() {
  const { getToken } = useAdminContext();
  const [items, setItems] = useState<AdminDataQualityRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [sort, setSort] = useState<SortKey>('date');
  const [order, setOrder] = useState('desc');
  const [resolvedFilter, setResolvedFilter] = useState('');
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [detail, setDetail] = useState<AdminDataQualityDetail | null>(null);
  const limit = 50;

  const fetchData = useCallback(async (p: number, s: string, o: string, rf: string) => {
    setLoading(true);
    try {
      const token = await getToken();
      const params: Record<string, string | number> = { page: p, limit, sort: s, order: o };
      if (rf) params.resolved = rf;
      const result = await getAdminDataQualityIssues(token, params as Parameters<typeof getAdminDataQualityIssues>[1]);
      setItems(result.items || []);
      setTotal(result.total);
      setPage(p);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => {
    fetchData(1, sort, order, resolvedFilter);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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
      const d = await getAdminDataQualityDetail(token, items[index].id);
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
    await updateAdminDataQualityResolution(token, detail.id, { resolved, resolution_note: note });
    const updated = [...items];
    updated[selectedIndex] = { ...updated[selectedIndex], resolved, resolution_note: note };
    setItems(updated);
    setDetail({ ...detail, resolved, resolution_note: note });
  };

  return (
    <div className="max-w-6xl mx-auto px-6 py-10">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-dark-100">
          Ingestion Data Quality <span className="text-dark-500 text-lg font-normal">({total})</span>
        </h1>
        <select
          value={resolvedFilter}
          onChange={(e) => handleFilterChange(e.target.value)}
          className="bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-1.5 text-sm text-dark-200 focus:outline-none focus:border-accent/50"
        >
          <option value="">All</option>
          <option value="false">Unresolved</option>
          <option value="true">Resolved</option>
        </select>
      </div>

      {loading && !items.length ? (
        <div className="flex items-center justify-center py-20 text-dark-500 text-sm">Loading...</div>
      ) : (
        <>
          <div className="overflow-x-auto rounded-xl border border-dark-700/50">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-dark-700/50 text-dark-400 text-left">
                  <SortHeader label="Date" sortKey="date" active={sort === 'date'} order={order} onSort={handleSort} />
                  <SortHeader label="Notice ID" sortKey="notice_id" active={sort === 'notice_id'} order={order} onSort={handleSort} />
                  <SortHeader label="Source" sortKey="source" active={sort === 'source'} order={order} onSort={handleSort} />
                  <SortHeader label="Issue Type" sortKey="issue_type" active={sort === 'issue_type'} order={order} onSort={handleSort} />
                  <SortHeader label="Field" sortKey="field" active={sort === 'field'} order={order} onSort={handleSort} />
                  <th className="px-4 py-3 font-medium">Value</th>
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
                    <td className="px-4 py-3 text-dark-300">{row.source}</td>
                    <td className="px-4 py-3 text-dark-300">{row.issue_type}</td>
                    <td className="px-4 py-3 text-dark-300">{row.field_name || '—'}</td>
                    <td className="px-4 py-3 text-dark-400 truncate max-w-[12rem]">{row.field_value || '—'}</td>
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
