import { useMemo } from 'react';
import { createPortal } from 'react-dom';
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react';
import { useDropdownPosition } from '../filters/useDropdownPosition';
import { SORT_OPTIONS } from '../filters/constants';

interface SortDropdownProps {
  sort: string;
  sortDir: 'asc' | 'desc';
  onSortChange: (sort: string, dir: 'asc' | 'desc') => void;
  hasKeyword: boolean;
}

export default function SortDropdown({ sort, sortDir, onSortChange, hasKeyword }: SortDropdownProps) {
  const { open, pos, triggerRef, dropdownRef, toggleDropdown, closeDropdown } =
    useDropdownPosition({ minWidth: 180 });

  const effectiveSort = hasKeyword && !sort ? 'relevance' : sort || 'posted_date';

  const currentLabel = useMemo(() => {
    const opt = SORT_OPTIONS.find((o) => o.value === effectiveSort);
    return opt?.label || 'Posted Date';
  }, [effectiveSort]);

  const DirIcon = sortDir === 'asc' ? ArrowUp : ArrowDown;

  return (
    <>
      <div className="inline-flex items-center gap-0.5">
        <button
          ref={triggerRef}
          type="button"
          onClick={toggleDropdown}
          aria-expanded={open}
          className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs rounded-lg border
            bg-dark-800 border-dark-700/50 text-dark-300 hover:border-dark-600 hover:text-dark-200 transition-colors"
        >
          <ArrowUpDown size={13} className="shrink-0 text-dark-400" />
          <span>{currentLabel}</span>
        </button>
        <button
          type="button"
          onClick={() => onSortChange(effectiveSort, sortDir === 'asc' ? 'desc' : 'asc')}
          className="p-1.5 rounded-lg border bg-dark-800 border-dark-700/50 text-dark-400
            hover:border-dark-600 hover:text-dark-200 transition-colors"
          aria-label={`Sort ${sortDir === 'asc' ? 'descending' : 'ascending'}`}
        >
          <DirIcon size={13} />
        </button>
      </div>
      {open &&
        createPortal(
          <div
            ref={dropdownRef}
            style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: pos.width }}
            className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden p-1"
          >
            {SORT_OPTIONS.map((opt) => {
              if (opt.value === 'relevance' && !hasKeyword) return null;
              const isActive = effectiveSort === opt.value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => {
                    onSortChange(opt.value, sortDir);
                    closeDropdown();
                  }}
                  className={`w-full text-left px-3 py-1.5 rounded text-xs transition-colors
                    ${isActive ? 'bg-accent/15 text-accent' : 'text-dark-200 hover:bg-dark-700/50'}`}
                >
                  {opt.label}
                </button>
              );
            })}
          </div>,
          document.body,
        )}
    </>
  );
}
