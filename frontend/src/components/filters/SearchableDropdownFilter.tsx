import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Search } from 'lucide-react';
import { useDropdownPosition } from './useDropdownPosition';

interface SearchableDropdownFilterProps {
  label: string;
  options: Array<{ value: string; label: string; count?: number; searchTerms?: string[] }>;
  selected: string[];
  onSelectionChange: (selected: string[]) => void;
  searchPlaceholder?: string;
  loading?: boolean;
  disabled?: boolean;
}

const ITEM_HEIGHT = 28;
const BUFFER = 5;

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

export default function SearchableDropdownFilter({
  label,
  options,
  selected,
  onSelectionChange,
  searchPlaceholder = 'Search...',
  loading,
  disabled,
}: SearchableDropdownFilterProps) {
  const [pending, setPending] = useState<string[]>(selected);
  const [search, setSearch] = useState('');
  const [scrollTop, setScrollTop] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const pendingRef = useRef(pending);
  pendingRef.current = pending;
  const selectedRef = useRef(selected);
  selectedRef.current = selected;

  const onClose = useCallback(() => {
    setSearch('');
    const p = pendingRef.current;
    const s = selectedRef.current;
    const pSorted = [...p].sort();
    const sSorted = [...s].sort();
    if (pSorted.length !== sSorted.length || pSorted.some((v, i) => v !== sSorted[i])) {
      onSelectionChange(p);
    }
  }, [onSelectionChange]);

  const { open, pos, triggerRef, dropdownRef, toggleDropdown } = useDropdownPosition({
    minWidth: 360,
    onClose,
  });

  useEffect(() => {
    setPending(selected);
  }, [selected]);

  useEffect(() => {
    if (open) {
      setPending(selected);
      setSearch('');
      setScrollTop(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const toggle = useCallback((value: string) => {
    setPending((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value],
    );
  }, []);

  const filtered = useMemo(() => {
    if (!search) return options;
    const s = search.toLowerCase();
    return options.filter(
      (opt) =>
        opt.label.toLowerCase().includes(s) ||
        opt.value.toLowerCase().includes(s) ||
        opt.searchTerms?.some((t) => t.toLowerCase().includes(s)),
    );
  }, [options, search]);

  // Selected items pinned at top
  const selectedItems = useMemo(
    () => filtered.filter((o) => pending.includes(o.value)),
    [filtered, pending],
  );

  // Unselected sorted by count desc
  const unselectedItems = useMemo(() => {
    const items = filtered.filter((o) => !pending.includes(o.value));
    return items.sort((a, b) => (b.count ?? 0) - (a.count ?? 0));
  }, [filtered, pending]);

  // Virtual scroll for unselected items
  const totalHeight = unselectedItems.length * ITEM_HEIGHT;
  const viewportHeight = 280;
  const startIndex = Math.max(0, Math.floor(scrollTop / ITEM_HEIGHT) - BUFFER);
  const endIndex = Math.min(
    unselectedItems.length,
    Math.ceil((scrollTop + viewportHeight) / ITEM_HEIGHT) + BUFFER,
  );
  const offsetY = startIndex * ITEM_HEIGHT;
  const visibleItems = unselectedItems.slice(startIndex, endIndex);

  useEffect(() => {
    if (search) {
      setScrollTop(0);
      if (scrollRef.current) scrollRef.current.scrollTop = 0;
    }
  }, [search]);

  const handleScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop);
  }, []);

  const active = selected.length > 0;

  const triggerLabel = useMemo(() => {
    if (!active) return label;
    return `${label} (${selected.length})`;
  }, [label, active, selected.length]);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={disabled ? undefined : toggleDropdown}
        aria-expanded={open}
        disabled={disabled}
        className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border transition-colors
          ${disabled
            ? 'bg-dark-800/50 border-dark-700/30 text-dark-500 cursor-not-allowed opacity-50'
            : active
              ? 'bg-accent/10 border-accent/40 text-accent cursor-pointer'
              : 'bg-dark-800 border-dark-700/50 text-dark-200 hover:border-dark-600 cursor-pointer'
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
            className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden flex flex-col max-h-[400px]"
          >
            {/* Search */}
            <div className="p-1.5 border-b border-dark-700/30">
              <div className="relative">
                <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-dark-500" />
                <input
                  ref={inputRef}
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={searchPlaceholder}
                  className="w-full text-xs bg-dark-900 border border-dark-700/50 rounded pl-6 pr-2 py-1.5 text-dark-200
                             placeholder:text-dark-500 focus:outline-none focus:border-accent/50"
                />
              </div>
            </div>

            {loading ? (
              <p className="text-xs text-dark-500 px-3 py-3">Loading...</p>
            ) : filtered.length === 0 ? (
              <p className="text-xs text-dark-500 px-3 py-3">No matches</p>
            ) : (
              <>
                {/* Pinned selected items */}
                {selectedItems.length > 0 && (
                  <div className="p-1 border-b border-dark-700/30">
                    {selectedItems.map((opt) => (
                      <label
                        key={opt.value}
                        className="flex items-center gap-2 px-2 py-1 rounded hover:bg-dark-700/50 cursor-pointer text-xs text-accent"
                      >
                        <input
                          type="checkbox"
                          checked
                          onChange={() => toggle(opt.value)}
                          className="accent-accent shrink-0"
                        />
                        <span className="truncate flex-1">
                          <HighlightedText text={opt.label} query={search} />
                        </span>
                        {opt.count != null && (
                          <span className="text-dark-500 tabular-nums">
                            {opt.count.toLocaleString()}
                          </span>
                        )}
                      </label>
                    ))}
                  </div>
                )}

                {/* Virtual-scrolled unselected items */}
                <div
                  ref={scrollRef}
                  className="overflow-y-auto flex-1 p-1"
                  style={{ maxHeight: viewportHeight }}
                  onScroll={handleScroll}
                >
                  <div style={{ height: totalHeight, position: 'relative' }}>
                    <div style={{ position: 'absolute', top: offsetY, left: 0, right: 0 }}>
                      {visibleItems.map((opt) => (
                        <label
                          key={opt.value}
                          style={{ height: ITEM_HEIGHT }}
                          className="flex items-center gap-2 px-2 rounded hover:bg-dark-700/50 cursor-pointer text-xs text-dark-300"
                        >
                          <input
                            type="checkbox"
                            checked={false}
                            onChange={() => toggle(opt.value)}
                            className="accent-accent shrink-0"
                          />
                          <span className="truncate flex-1">
                            <HighlightedText text={opt.label} query={search} />
                          </span>
                          {opt.count != null && (
                            <span className="text-dark-500 tabular-nums">
                              {opt.count.toLocaleString()}
                            </span>
                          )}
                        </label>
                      ))}
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>,
          document.body,
        )}
    </>
  );
}
