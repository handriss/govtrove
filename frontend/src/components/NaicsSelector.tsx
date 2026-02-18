import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { Search, X } from 'lucide-react';
import { NAICS_CODES } from '../data/naicsCodes';
import type { NaicsCode } from '../data/naicsCodes';

interface NaicsSelectorProps {
  selected: string[];
  onChange: (codes: string[]) => void;
  placeholder?: string;
  maxHeight?: string;
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

// Pre-build a lookup for titles by code
const TITLE_BY_CODE = new Map<string, string>();
for (const c of NAICS_CODES) TITLE_BY_CODE.set(c.code, c.title);

export default function NaicsSelector({
  selected,
  onChange,
  placeholder = 'Search NAICS codes or keywords...',
  maxHeight = '320px',
}: NaicsSelectorProps) {
  const [search, setSearch] = useState('');
  const [open, setOpen] = useState(false);
  const [highlighted, setHighlighted] = useState(-1);
  const [pos, setPos] = useState({ top: 0, left: 0, width: 400 });
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const results = useMemo(() => {
    const q = search.trim();
    if (!q) return [];
    return isDigits(q) ? searchByCode(q) : searchByKeyword(q);
  }, [search]);

  const updatePosition = useCallback(() => {
    if (containerRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      const w = Math.max(rect.width, 400);
      let left = rect.left;
      if (left + w > window.innerWidth - 8) left = window.innerWidth - w - 8;
      setPos({ top: rect.bottom + 4, left, width: w });
    }
  }, []);

  useEffect(() => {
    if (search.trim() && results.length > 0) {
      updatePosition();
      setOpen(true);
      setHighlighted(-1);
    } else {
      setOpen(false);
    }
  }, [search, results, updatePosition]);

  useEffect(() => {
    if (!open) return;
    const onMouseDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (containerRef.current?.contains(t) || dropdownRef.current?.contains(t)) return;
      setOpen(false);
    };
    const onScroll = (e: Event) => {
      if (dropdownRef.current?.contains(e.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener('mousedown', onMouseDown);
    window.addEventListener('scroll', onScroll, true);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      window.removeEventListener('scroll', onScroll, true);
    };
  }, [open]);

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
      return;
    }
    if (!open || results.length === 0) return;
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
  }, [open, results, highlighted, toggleCode]);

  useEffect(() => {
    if (highlighted >= 0 && listRef.current) {
      const el = listRef.current.children[highlighted] as HTMLElement | undefined;
      el?.scrollIntoView({ block: 'nearest' });
    }
  }, [highlighted]);

  return (
    <div ref={containerRef}>
      <div className="relative">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-500" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => { if (search.trim() && results.length > 0) { updatePosition(); setOpen(true); } }}
          placeholder={placeholder}
          className="w-full text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg pl-9 pr-3 py-2 text-dark-100
                     placeholder:text-dark-500 focus:outline-none focus:border-accent/50"
        />
      </div>

      {selected.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {selected.map((code) => (
            <span
              key={code}
              className="inline-flex items-center gap-1 bg-dark-800 border border-dark-700/50 rounded-md px-2 py-0.5 text-xs text-dark-300"
            >
              <span className="font-mono">{code}</span>
              <span className="text-dark-500">&mdash;</span>
              <span className="truncate max-w-[180px]">{TITLE_BY_CODE.get(code) ?? code}</span>
              <button
                type="button"
                onClick={() => toggleCode(code)}
                className="ml-0.5 text-dark-500 hover:text-dark-200"
              >
                <X size={10} />
              </button>
            </span>
          ))}
        </div>
      )}

      {open && createPortal(
        <div
          ref={dropdownRef}
          style={{ position: 'fixed', top: pos.top, left: pos.left, width: pos.width, maxHeight }}
          className="z-50 bg-dark-900 border border-dark-700/50 rounded-lg shadow-xl overflow-hidden flex flex-col"
        >
          <div ref={listRef} className="overflow-y-auto flex-1">
            {results.map((item, i) => {
              const isSelected = selected.includes(item.code.code);
              return (
                <div
                  key={item.code.code}
                  onClick={() => toggleCode(item.code.code)}
                  className={`px-3 py-1.5 cursor-pointer text-sm flex items-center gap-2
                             ${INDENT[item.code.level] || 'pl-2'}
                             ${i === highlighted ? 'bg-dark-800/80' : 'hover:bg-dark-800/50'}
                             ${ROLE_STYLE[item.role] || 'text-dark-300'}`}
                >
                  {isSelected && (
                    <span className="text-accent text-xs shrink-0">&#10003;</span>
                  )}
                  <span className="font-mono text-xs shrink-0">{item.code.code}</span>
                  <span className="text-dark-600 text-xs">&mdash;</span>
                  <span className="truncate">{item.code.title}</span>
                </div>
              );
            })}
          </div>
        </div>,
        document.body,
      )}
    </div>
  );
}
