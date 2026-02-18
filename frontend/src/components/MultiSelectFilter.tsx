import { useState, useRef, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Search } from 'lucide-react';

export interface FilterOption {
  value: string;
  label: string;
  searchTerms?: string[];
}

interface Props {
  options: FilterOption[];
  selected: string[];
  onChange: (selected: string[]) => void;
  placeholder?: string;
}

export default function MultiSelectFilter({ options, selected, onChange, placeholder = 'All' }: Props) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [pos, setPos] = useState({ top: 0, left: 0, width: 260 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const openDropdown = useCallback(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      const w = Math.max(rect.width, 260);
      let left = rect.left;
      if (left + w > window.innerWidth - 8) left = window.innerWidth - w - 8;
      setPos({ top: rect.bottom + 4, left, width: w });
    }
    setOpen(true);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onMouseDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (triggerRef.current?.contains(t) || dropdownRef.current?.contains(t)) return;
      setOpen(false);
      setSearch('');
    };
    const onScroll = (e: Event) => {
      if (dropdownRef.current?.contains(e.target as Node)) return;
      setOpen(false);
      setSearch('');
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { setOpen(false); setSearch(''); }
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKey);
    window.addEventListener('scroll', onScroll, true);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKey);
      window.removeEventListener('scroll', onScroll, true);
    };
  }, [open]);

  useEffect(() => {
    if (open) setTimeout(() => inputRef.current?.focus(), 0);
  }, [open]);

  const filtered = search
    ? options.filter((opt) => {
        const s = search.toLowerCase();
        return (
          opt.label.toLowerCase().includes(s) ||
          opt.value.toLowerCase().includes(s) ||
          opt.searchTerms?.some((t) => t.toLowerCase().includes(s))
        );
      })
    : options;

  const toggle = useCallback((value: string) => {
    onChange(
      selected.includes(value)
        ? selected.filter((v) => v !== value)
        : [...selected, value],
    );
  }, [selected, onChange]);

  const display =
    selected.length === 0
      ? placeholder
      : selected.length === 1
        ? options.find((o) => o.value === selected[0])?.label ?? selected[0]
        : `${selected.length} selected`;

  return (
    <div>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => {
          if (open) { setOpen(false); setSearch(''); }
          else openDropdown();
        }}
        className={`w-full text-xs bg-dark-800 border rounded px-1.5 py-1
                   focus:outline-none cursor-pointer text-left flex items-center justify-between gap-1
                   ${selected.length > 0 ? 'border-accent/40 text-accent' : 'border-dark-700/50 text-dark-200'}`}
      >
        <span className="truncate">{display}</span>
        <ChevronDown size={10} className={`shrink-0 text-dark-500 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && createPortal(
        <div
          ref={dropdownRef}
          style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: pos.width }}
          className="z-50 max-h-[320px] bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl overflow-hidden flex flex-col"
        >
          <div className="p-1.5 border-b border-dark-700/30">
            <div className="relative">
              <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-dark-500" />
              <input
                ref={inputRef}
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search..."
                className="w-full text-xs bg-dark-900 border border-dark-700/50 rounded pl-6 pr-2 py-1 text-dark-200
                           placeholder:text-dark-500 focus:outline-none focus:border-accent/50"
              />
            </div>
          </div>
          <div className="overflow-y-auto flex-1 p-1">
            {filtered.length === 0 ? (
              <p className="text-xs text-dark-500 px-2 py-1.5">No matches</p>
            ) : (
              filtered.map((opt) => (
                <label
                  key={opt.value}
                  className="flex items-center gap-2 px-2 py-1 rounded hover:bg-dark-700/50 cursor-pointer text-xs text-dark-200"
                >
                  <input
                    type="checkbox"
                    checked={selected.includes(opt.value)}
                    onChange={() => toggle(opt.value)}
                    className="accent-accent shrink-0"
                  />
                  <span className="truncate">{opt.label}</span>
                </label>
              ))
            )}
          </div>
          {selected.length > 0 && (
            <div className="p-1.5 border-t border-dark-700/30">
              <button
                type="button"
                onClick={() => onChange([])}
                className="w-full text-[11px] text-dark-400 hover:text-dark-200 py-0.5"
              >
                Clear selection
              </button>
            </div>
          )}
        </div>,
        document.body,
      )}
    </div>
  );
}
