import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Search } from 'lucide-react';
import { NAICS_CODES } from '../data/naicsCodes';
import type { NaicsCode } from '../data/naicsCodes';

interface Props {
  selected: string[];
  onChange: (codes: string[]) => void;
  placeholder?: string;
}

interface ResultItem {
  code: NaicsCode;
  role: 'parent' | 'match' | 'child';
}

function isDigits(s: string): boolean {
  return /^\d+$/.test(s);
}

function searchByCode(query: string): ResultItem[] {
  const results: ResultItem[] = [];
  for (const c of NAICS_CODES) {
    if (query.startsWith(c.code) && c.code !== query) {
      results.push({ code: c, role: 'parent' });
    } else if (c.code === query) {
      results.push({ code: c, role: 'match' });
    } else if (c.code.startsWith(query) && c.code !== query) {
      results.push({ code: c, role: 'child' });
    }
  }
  return results;
}

function searchByKeyword(query: string): ResultItem[] {
  const q = query.toLowerCase();
  const results: ResultItem[] = [];
  for (const c of NAICS_CODES) {
    if (c.title.toLowerCase().includes(q) || c.code.includes(q)) {
      results.push({ code: c, role: 'match' });
    }
  }
  return results;
}

const INDENT: Record<number, string> = {
  2: 'pl-2',
  3: 'pl-5',
  4: 'pl-8',
  5: 'pl-11',
  6: 'pl-14',
};

const ROLE_STYLE: Record<string, string> = {
  parent: 'text-dark-500',
  match: 'text-dark-100 font-medium',
  child: 'text-dark-300',
};

const TITLE_BY_CODE = new Map<string, string>();
for (const c of NAICS_CODES) TITLE_BY_CODE.set(c.code, c.title);

export default function NaicsColumnFilter({ selected, onChange, placeholder = 'All' }: Props) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [highlighted, setHighlighted] = useState(-1);
  const [pos, setPos] = useState({ top: 0, left: 0, width: 320 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const results = useMemo(() => {
    const q = search.trim();
    if (!q) return [];
    return isDigits(q) ? searchByCode(q) : searchByKeyword(q);
  }, [search]);

  const openDropdown = useCallback(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      const w = 320;
      let left = rect.left;
      if (left + w > window.innerWidth - 8) left = window.innerWidth - w - 8;
      setPos({ top: rect.bottom + 4, left, width: w });
    }
    setOpen(true);
    setSearch('');
    setHighlighted(-1);
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

  useEffect(() => {
    if (highlighted >= 0 && listRef.current) {
      const el = listRef.current.children[highlighted] as HTMLElement | undefined;
      el?.scrollIntoView({ block: 'nearest' });
    }
  }, [highlighted]);

  const toggleCode = useCallback((code: string) => {
    onChange(
      selected.includes(code)
        ? selected.filter((c) => c !== code)
        : [...selected, code],
    );
  }, [selected, onChange]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setOpen(false);
      setSearch('');
      return;
    }
    if (results.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlighted((prev) => Math.min(prev + 1, results.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlighted((prev) => Math.max(prev - 1, 0));
    } else if (e.key === 'Enter' && highlighted >= 0) {
      e.preventDefault();
      toggleCode(results[highlighted].code.code);
    }
  }, [results, highlighted, toggleCode]);

  const display =
    selected.length === 0
      ? placeholder
      : selected.length === 1
        ? selected[0]
        : `${selected.length} codes`;

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
          style={{ position: 'fixed', top: pos.top, left: pos.left, width: pos.width }}
          className="z-50 max-h-[320px] bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl overflow-hidden flex flex-col"
        >
          <div className="p-1.5 border-b border-dark-700/30">
            <div className="relative">
              <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-dark-500" />
              <input
                ref={inputRef}
                type="text"
                value={search}
                onChange={(e) => { setSearch(e.target.value); setHighlighted(-1); }}
                onKeyDown={handleKeyDown}
                placeholder="Code or keyword..."
                className="w-full text-xs bg-dark-900 border border-dark-700/50 rounded pl-6 pr-2 py-1 text-dark-200
                           placeholder:text-dark-500 focus:outline-none focus:border-accent/50"
              />
            </div>
          </div>
          <div ref={listRef} className="overflow-y-auto flex-1">
            {search.trim() === '' ? (
              <p className="text-xs text-dark-500 px-3 py-2">Type a NAICS code or keyword</p>
            ) : results.length === 0 ? (
              <p className="text-xs text-dark-500 px-3 py-2">No matches</p>
            ) : (
              results.map((item, i) => {
                const isSelected = selected.includes(item.code.code);
                return (
                  <div
                    key={item.code.code}
                    onClick={() => toggleCode(item.code.code)}
                    className={`px-2 py-1 cursor-pointer text-xs flex items-center gap-1.5
                               ${INDENT[item.code.level] || 'pl-2'}
                               ${i === highlighted ? 'bg-dark-700/80' : 'hover:bg-dark-700/50'}
                               ${ROLE_STYLE[item.role] || 'text-dark-300'}`}
                  >
                    {isSelected && (
                      <span className="text-accent text-[10px] shrink-0">&#10003;</span>
                    )}
                    <span className="font-mono text-[10px] shrink-0">{item.code.code}</span>
                    <span className="text-dark-600 text-[10px]">&mdash;</span>
                    <span className="truncate">{item.code.title}</span>
                  </div>
                );
              })
            )}
          </div>
          {selected.length > 0 && (
            <div className="p-1.5 border-t border-dark-700/30 space-y-1">
              <div className="flex flex-wrap gap-1 px-1">
                {selected.map((code) => (
                  <span
                    key={code}
                    className="inline-flex items-center gap-0.5 bg-dark-900 border border-dark-700/50 rounded px-1.5 py-0.5 text-[10px] text-dark-300"
                  >
                    <span className="font-mono">{code}</span>
                    <button
                      type="button"
                      onClick={() => toggleCode(code)}
                      className="text-dark-500 hover:text-dark-200 ml-0.5"
                    >
                      &times;
                    </button>
                  </span>
                ))}
              </div>
              <button
                type="button"
                onClick={() => onChange([])}
                className="w-full text-[11px] text-dark-400 hover:text-dark-200 py-0.5"
              >
                Clear all
              </button>
            </div>
          )}
        </div>,
        document.body,
      )}
    </div>
  );
}
