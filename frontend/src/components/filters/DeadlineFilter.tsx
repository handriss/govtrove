import { useState, useCallback, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Calendar } from 'lucide-react';
import { useDropdownPosition } from './useDropdownPosition';
import { DEADLINE_PRESET_LABELS } from './constants';

const PRESETS = [
  { value: '', label: 'All' },
  { value: '7', label: 'Next 7 days' },
  { value: '14', label: 'Next 14 days' },
  { value: '30', label: 'Next 30 days' },
  { value: '60', label: 'Next 60 days' },
  { value: 'quarter', label: 'This quarter' },
];

interface DeadlineFilterProps {
  deadlinePreset: string;
  deadlineFrom: string;
  deadlineTo: string;
  onChange: (values: { deadlinePreset: string; deadlineFrom: string; deadlineTo: string }) => void;
  facetCounts?: Array<{ preset: string; count: number }>;
}

function today(): string {
  return new Date().toISOString().split('T')[0];
}

function formatDateShort(iso: string): string {
  const d = new Date(iso + 'T00:00:00');
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export default function EnhancedDeadlineFilter({
  deadlinePreset,
  deadlineFrom,
  deadlineTo,
  onChange,
  facetCounts,
}: DeadlineFilterProps) {
  const [showCustom, setShowCustom] = useState(false);
  const [customFrom, setCustomFrom] = useState(deadlineFrom);
  const [customTo, setCustomTo] = useState(deadlineTo);

  const onClose = useCallback(() => {
    setShowCustom(false);
  }, []);

  const { open, pos, triggerRef, dropdownRef, toggleDropdown, closeDropdown } =
    useDropdownPosition({ minWidth: 220, onClose });

  const selectPreset = useCallback(
    (value: string) => {
      onChange({ deadlinePreset: value, deadlineFrom: '', deadlineTo: '' });
      closeDropdown();
    },
    [onChange, closeDropdown],
  );

  const applyCustom = useCallback(() => {
    onChange({ deadlinePreset: '', deadlineFrom: customFrom, deadlineTo: customTo });
    closeDropdown();
  }, [onChange, customFrom, customTo, closeDropdown]);

  const countMap = useMemo(() => {
    if (!facetCounts) return null;
    return new Map(facetCounts.map((f) => [f.preset, f.count]));
  }, [facetCounts]);

  const active = deadlinePreset !== '' || deadlineFrom !== '' || deadlineTo !== '';

  const triggerLabel = useMemo(() => {
    if (deadlinePreset) {
      return `Due: ${DEADLINE_PRESET_LABELS[deadlinePreset] || deadlinePreset}`;
    }
    if (deadlineFrom || deadlineTo) {
      const from = deadlineFrom ? formatDateShort(deadlineFrom) : '';
      const to = deadlineTo ? formatDateShort(deadlineTo) : '';
      if (from && to) return `Due: ${from} \u2013 ${to}`;
      if (from) return `Due: from ${from}`;
      return `Due: until ${to}`;
    }
    return 'Deadline';
  }, [deadlinePreset, deadlineFrom, deadlineTo]);

  const todayStr = today();

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
        <Calendar size={14} className="shrink-0" />
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
            className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden"
          >
            {/* Presets */}
            <div className="p-1">
              {PRESETS.map((preset) => {
                const count = countMap?.get(preset.value);
                const isCurrent =
                  preset.value === deadlinePreset ||
                  (preset.value === '' && !deadlinePreset && !deadlineFrom && !deadlineTo);
                return (
                  <button
                    key={preset.value}
                    type="button"
                    onClick={() => selectPreset(preset.value)}
                    className={`w-full text-left px-3 py-1.5 rounded text-xs transition-colors flex items-center justify-between
                      ${isCurrent
                        ? 'bg-accent/15 text-accent'
                        : 'text-dark-200 hover:bg-dark-700/50'
                      }`}
                  >
                    <span>{preset.label}</span>
                    {count != null && (
                      <span className="text-dark-500 tabular-nums text-[11px]">
                        {count.toLocaleString()}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>

            {/* Custom range */}
            <div className="border-t border-dark-700/30">
              <button
                type="button"
                onClick={() => setShowCustom(!showCustom)}
                className="w-full text-left px-3 py-2 text-xs text-dark-400 hover:text-dark-200 transition-colors"
              >
                Custom range...
              </button>
              {showCustom && (
                <div className="px-3 pb-3 space-y-2">
                  <div className="flex gap-2 items-center">
                    <label className="flex flex-col gap-1 flex-1">
                      <span className="text-[11px] text-dark-500">From</span>
                      <input
                        type="date"
                        value={customFrom}
                        min={todayStr}
                        onChange={(e) => setCustomFrom(e.target.value)}
                        className="text-xs bg-dark-900 border border-dark-700/50 rounded px-2 py-1 text-dark-200
                                   focus:outline-none focus:border-accent/50 w-full"
                      />
                    </label>
                    <label className="flex flex-col gap-1 flex-1">
                      <span className="text-[11px] text-dark-500">To</span>
                      <input
                        type="date"
                        value={customTo}
                        min={customFrom || todayStr}
                        onChange={(e) => setCustomTo(e.target.value)}
                        className="text-xs bg-dark-900 border border-dark-700/50 rounded px-2 py-1 text-dark-200
                                   focus:outline-none focus:border-accent/50 w-full"
                      />
                    </label>
                  </div>
                  <button
                    type="button"
                    onClick={applyCustom}
                    disabled={!customFrom && !customTo}
                    className="w-full text-xs bg-accent hover:bg-accent-hover disabled:opacity-40 disabled:cursor-not-allowed
                               text-white rounded px-3 py-1.5 transition-colors"
                  >
                    Apply
                  </button>
                </div>
              )}
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}
