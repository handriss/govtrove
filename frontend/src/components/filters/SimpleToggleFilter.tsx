import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown } from 'lucide-react';
import { useDropdownPosition } from './useDropdownPosition';

interface SimpleToggleFilterProps {
  label: string;
  options: Array<{ value: string; label: string; count?: number }>;
  selected: string[];
  onSelectionChange: (selected: string[]) => void;
  loading?: boolean;
}

export default function SimpleToggleFilter({
  label,
  options,
  selected,
  onSelectionChange,
  loading,
}: SimpleToggleFilterProps) {
  const [pending, setPending] = useState<string[]>(selected);
  const pendingRef = useRef(pending);
  pendingRef.current = pending;
  const selectedRef = useRef(selected);
  selectedRef.current = selected;

  const onClose = useCallback(() => {
    const p = pendingRef.current;
    const s = selectedRef.current;
    if (p.length !== s.length || p.some((v, i) => v !== s[i])) {
      onSelectionChange(p);
    }
  }, [onSelectionChange]);

  const { open, pos, triggerRef, dropdownRef, toggleDropdown } = useDropdownPosition({
    minWidth: 240,
    onClose,
  });

  // Sync pending when selected changes externally
  useEffect(() => {
    setPending(selected);
  }, [selected]);

  // Reset pending when opening
  useEffect(() => {
    if (open) setPending(selected);
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const toggle = useCallback((value: string) => {
    setPending((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value],
    );
  }, []);

  const selectAll = useCallback(() => {
    setPending(options.map((o) => o.value));
  }, [options]);

  const clearAll = useCallback(() => {
    setPending([]);
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
        onClick={toggleDropdown}
        aria-expanded={open}
        className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border cursor-pointer transition-colors
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
            className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden flex flex-col"
          >
            <div className="flex items-center justify-between px-3 py-2 border-b border-dark-700/30">
              <span className="text-xs font-medium text-dark-300">{label}</span>
              <div className="flex gap-2 text-[11px]">
                <button
                  type="button"
                  onClick={selectAll}
                  className="text-accent hover:text-accent-hover"
                >
                  Select all
                </button>
                <button
                  type="button"
                  onClick={clearAll}
                  className="text-dark-400 hover:text-dark-200"
                >
                  Clear all
                </button>
              </div>
            </div>
            <div className="overflow-y-auto max-h-[320px] p-1">
              {loading ? (
                <p className="text-xs text-dark-500 px-2 py-1.5">Loading...</p>
              ) : (
                options.map((opt) => {
                  const disabled = opt.count === 0;
                  return (
                    <label
                      key={opt.value}
                      className={`flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer text-xs
                        ${disabled ? 'opacity-40 cursor-not-allowed' : 'hover:bg-dark-700/50'}
                        ${pending.includes(opt.value) ? 'text-dark-100' : 'text-dark-300'}`}
                    >
                      <input
                        type="checkbox"
                        checked={pending.includes(opt.value)}
                        onChange={() => toggle(opt.value)}
                        disabled={disabled}
                        className="accent-accent shrink-0"
                      />
                      <span className="truncate flex-1">{opt.label}</span>
                      {opt.count != null && (
                        <span className="text-dark-500 tabular-nums">
                          {opt.count.toLocaleString()}
                        </span>
                      )}
                    </label>
                  );
                })
              )}
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}
