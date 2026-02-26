import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Search } from 'lucide-react';
import { useDropdownPosition } from './useDropdownPosition';
import { searchAgencies, type AgencyResult } from '../../services/api';

interface AgencyFilterProps {
  selected: string[];
  onChange: (selected: string[]) => void;
  loading?: boolean;
}

function HighlightedText({ text, query }: { text: string; query: string }) {
  if (!query) return <>{text}</>;
  const idx = text.toLowerCase().indexOf(query.toLowerCase());
  if (idx === -1) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-accent/20 text-accent">{text.slice(idx, idx + query.length)}</mark>
      {text.slice(idx + query.length)}
    </>
  );
}

export default function AgencyFilter({ selected, onChange }: AgencyFilterProps) {
  const [pending, setPending] = useState<string[]>(selected);
  const [search, setSearch] = useState('');
  const [results, setResults] = useState<AgencyResult[]>([]);
  const [fetching, setFetching] = useState(false);
  const [agencyMap, setAgencyMap] = useState<Map<string, AgencyResult>>(new Map());
  const inputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const pendingRef = useRef(pending);
  pendingRef.current = pending;
  const selectedRef = useRef(selected);
  selectedRef.current = selected;
  const initialFetchDone = useRef(false);

  const onClose = useCallback(() => {
    setSearch('');
    const p = pendingRef.current;
    const s = selectedRef.current;
    const pSorted = [...p].sort();
    const sSorted = [...s].sort();
    if (pSorted.length !== sSorted.length || pSorted.some((v, i) => v !== sSorted[i])) {
      onChange(p);
    }
  }, [onChange]);

  const { open, pos, triggerRef, dropdownRef, toggleDropdown } = useDropdownPosition({
    minWidth: 400,
    onClose,
  });

  useEffect(() => {
    setPending(selected);
  }, [selected]);

  useEffect(() => {
    if (open) {
      setPending(selected);
      setSearch('');
      setTimeout(() => inputRef.current?.focus(), 0);
      if (!initialFetchDone.current) {
        fetchAgencies('');
        initialFetchDone.current = true;
      }
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch agency info for initially selected paths that aren't in the map
  useEffect(() => {
    for (const path of selected) {
      if (!agencyMap.has(path)) {
        const name = path.split('.').pop() || '';
        if (name) {
          searchAgencies(name, 10).then((data) => {
            setAgencyMap((prev) => {
              const next = new Map(prev);
              for (const a of data.agencies) next.set(a.parent_path, a);
              return next;
            });
          }).catch(() => {});
        }
      }
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const fetchAgencies = useCallback(async (q: string) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setFetching(true);
    try {
      const data = await searchAgencies(q, 20, controller.signal);
      if (!controller.signal.aborted) {
        setResults(data.agencies);
        setAgencyMap((prev) => {
          const next = new Map(prev);
          for (const a of data.agencies) next.set(a.parent_path, a);
          return next;
        });
      }
    } catch {
      // ignore aborts and errors
    } finally {
      if (!controller.signal.aborted) setFetching(false);
    }
  }, []);

  // Debounced search
  useEffect(() => {
    if (!open) return;
    const timer = setTimeout(() => fetchAgencies(search), search ? 200 : 0);
    return () => clearTimeout(timer);
  }, [search, open, fetchAgencies]);

  const toggle = useCallback((path: string) => {
    setPending((prev) =>
      prev.includes(path) ? prev.filter((p) => p !== path) : [...prev, path],
    );
  }, []);

  // Selected agencies with metadata
  const selectedAgencies = useMemo(
    () => pending.map((path) => agencyMap.get(path)).filter(Boolean) as AgencyResult[],
    [pending, agencyMap],
  );

  // Unselected results
  const unselectedResults = useMemo(
    () => results.filter((a) => !pending.includes(a.parent_path)),
    [results, pending],
  );

  const active = selected.length > 0;
  const triggerLabel = active ? `Agency (${selected.length})` : 'Agency';

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={toggleDropdown}
        aria-expanded={open}
        className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border transition-colors cursor-pointer
          ${active
            ? 'bg-accent/10 border-accent/40 text-accent'
            : 'bg-dark-800 border-dark-700/50 text-dark-200 hover:border-dark-600'
          }`}
      >
        <span className="truncate">{triggerLabel}</span>
        <ChevronDown
          size={14}
          className={`shrink-0 text-dark-400 transition-transform ${open ? 'rotate-180' : ''}`}
        />
      </button>
      {open &&
        createPortal(
          <div
            ref={dropdownRef}
            style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: pos.width }}
            className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden flex flex-col max-h-[440px]"
          >
            {/* Search input */}
            <div className="p-1.5 border-b border-dark-700/30">
              <div className="relative">
                <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-dark-500" />
                <input
                  ref={inputRef}
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder='Search agencies (e.g. "NAVFAC", "DLA", "Army")...'
                  className="w-full text-xs bg-dark-900 border border-dark-700/50 rounded pl-6 pr-2 py-1.5 text-dark-200
                             placeholder:text-dark-500 focus:outline-none focus:border-accent/50"
                />
              </div>
            </div>

            {fetching && results.length === 0 ? (
              <p className="text-xs text-dark-500 px-3 py-3">Searching...</p>
            ) : (
              <>
                {/* Pinned selected items */}
                {selectedAgencies.length > 0 && (
                  <div className="p-1 border-b border-dark-700/30 max-h-[140px] overflow-y-auto">
                    {selectedAgencies.map((a) => (
                      <label
                        key={a.parent_path}
                        className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-dark-700/50 cursor-pointer text-xs text-accent"
                      >
                        <input
                          type="checkbox"
                          checked
                          onChange={() => toggle(a.parent_path)}
                          className="accent-accent shrink-0"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="truncate font-medium">
                            <HighlightedText text={a.short_name || a.name} query={search} />
                          </div>
                          <div className="text-[10px] text-dark-500 truncate">{a.breadcrumb}</div>
                        </div>
                        <span className="text-dark-500 tabular-nums shrink-0">
                          {a.count.toLocaleString()}
                        </span>
                      </label>
                    ))}
                  </div>
                )}

                {/* Pending selections not in map (fallback display) */}
                {pending.filter((p) => !agencyMap.has(p)).length > 0 && (
                  <div className="p-1 border-b border-dark-700/30">
                    {pending
                      .filter((p) => !agencyMap.has(p))
                      .map((path) => (
                        <label
                          key={path}
                          className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-dark-700/50 cursor-pointer text-xs text-accent"
                        >
                          <input
                            type="checkbox"
                            checked
                            onChange={() => toggle(path)}
                            className="accent-accent shrink-0"
                          />
                          <span className="truncate">{path.split('.').pop()}</span>
                        </label>
                      ))}
                  </div>
                )}

                {/* Search results */}
                {unselectedResults.length === 0 && !fetching ? (
                  <p className="text-xs text-dark-500 px-3 py-3">
                    {search ? 'No matching agencies' : 'Type to search agencies'}
                  </p>
                ) : (
                  <div className="overflow-y-auto flex-1 p-1" style={{ maxHeight: 280 }}>
                    {unselectedResults.map((a) => (
                      <label
                        key={a.parent_path}
                        className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-dark-700/50 cursor-pointer text-xs text-dark-300"
                      >
                        <input
                          type="checkbox"
                          checked={false}
                          onChange={() => toggle(a.parent_path)}
                          className="accent-accent shrink-0"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="truncate">
                            <HighlightedText text={a.short_name || a.name} query={search} />
                            {a.short_name && a.short_name !== a.name && (
                              <span className="text-dark-500 ml-1">
                                (<HighlightedText text={a.name} query={search} />)
                              </span>
                            )}
                          </div>
                          <div className="text-[10px] text-dark-500 truncate">{a.breadcrumb}</div>
                        </div>
                        <span className="text-dark-500 tabular-nums shrink-0">
                          {a.count.toLocaleString()}
                        </span>
                      </label>
                    ))}
                    {fetching && (
                      <p className="text-xs text-dark-500 px-3 py-2">Searching...</p>
                    )}
                  </div>
                )}
              </>
            )}
          </div>,
          document.body,
        )}
    </>
  );
}
