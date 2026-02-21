import { useState, useRef, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown } from 'lucide-react';

const PRESETS = [
  { value: '', label: 'All' },
  { value: '7', label: '7 days' },
  { value: '14', label: '14 days' },
  { value: '30', label: '30 days' },
];

interface Props {
  value: string;
  onChange: (value: string) => void;
}

export default function DeadlineFilter({ value, onChange }: Props) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0, width: 120 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const openDropdown = useCallback(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      const w = Math.max(rect.width, 120);
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
    };
    const onScroll = (e: Event) => {
      if (dropdownRef.current?.contains(e.target as Node)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
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

  const display = PRESETS.find((p) => p.value === value)?.label ?? 'All';
  const active = value !== '';

  return (
    <div>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => (open ? setOpen(false) : openDropdown())}
        className={`w-full text-xs bg-dark-800 border rounded px-1.5 py-1
                   focus:outline-none cursor-pointer text-left flex items-center justify-between gap-1
                   ${active ? 'border-accent/40 text-accent' : 'border-dark-700/50 text-dark-200'}`}
      >
        <span className="truncate">{display}</span>
        <ChevronDown size={10} className={`shrink-0 text-dark-500 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && createPortal(
        <div
          ref={dropdownRef}
          style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: pos.width }}
          className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl overflow-hidden"
        >
          <div className="p-1">
            {PRESETS.map((preset) => (
              <button
                key={preset.value}
                type="button"
                onClick={() => { onChange(preset.value); setOpen(false); }}
                className={`w-full text-left px-2 py-1.5 rounded text-xs transition-colors
                  ${preset.value === value
                    ? 'bg-accent/15 text-accent'
                    : 'text-dark-200 hover:bg-dark-700/50'
                  }`}
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>,
        document.body,
      )}
    </div>
  );
}
