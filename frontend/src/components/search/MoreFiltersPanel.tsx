import { useState, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { X, SlidersHorizontal } from 'lucide-react';

interface MoreFiltersPanelProps {
  postedFrom: string;
  postedTo: string;
  onChange: (values: { postedFrom: string; postedTo: string }) => void;
}

export default function MoreFiltersPanel({ postedFrom, postedTo, onChange }: MoreFiltersPanelProps) {
  const [open, setOpen] = useState(false);
  const [localFrom, setLocalFrom] = useState(postedFrom);
  const [localTo, setLocalTo] = useState(postedTo);

  useEffect(() => {
    if (open) {
      setLocalFrom(postedFrom);
      setLocalTo(postedTo);
    }
  }, [open, postedFrom, postedTo]);

  // Lock body scroll when open
  useEffect(() => {
    if (!open) return;
    document.body.style.overflow = 'hidden';
    return () => { document.body.style.overflow = ''; };
  }, [open]);

  const handleApply = useCallback(() => {
    onChange({ postedFrom: localFrom, postedTo: localTo });
    setOpen(false);
  }, [localFrom, localTo, onChange]);

  const handleReset = useCallback(() => {
    setLocalFrom('');
    setLocalTo('');
    onChange({ postedFrom: '', postedTo: '' });
    setOpen(false);
  }, [onChange]);

  const activeCount = (postedFrom || postedTo) ? 1 : 0;

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-expanded={open}
        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border cursor-pointer transition-colors
          bg-dark-800 border-dark-700/50 text-dark-200 hover:border-dark-600"
      >
        <SlidersHorizontal size={14} className="shrink-0" />
        <span>More</span>
        {activeCount > 0 && (
          <span className="inline-flex items-center justify-center w-4 h-4 rounded-full bg-accent text-[10px] font-medium text-white">
            {activeCount}
          </span>
        )}
      </button>

      {open &&
        createPortal(
          <>
            {/* Backdrop */}
            <div
              className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm"
              onClick={() => setOpen(false)}
            />

            {/* Panel */}
            <div className="fixed inset-y-0 right-0 z-50 w-full max-w-sm bg-dark-900 border-l border-dark-700/50
              shadow-2xl shadow-black/50 flex flex-col animate-in slide-in-from-right duration-200">
              {/* Header */}
              <div className="flex items-center justify-between px-5 py-4 border-b border-dark-800/50">
                <h2 className="text-sm font-medium text-dark-100">More Filters</h2>
                <button
                  type="button"
                  onClick={() => setOpen(false)}
                  className="p-1 rounded-lg text-dark-400 hover:text-dark-200 transition-colors"
                >
                  <X size={18} />
                </button>
              </div>

              {/* Body */}
              <div className="flex-1 overflow-y-auto px-5 py-4 space-y-6">
                {/* Posted Date Range */}
                <div className="space-y-2">
                  <label className="text-xs font-medium text-dark-300">Posted Date</label>
                  <div className="flex gap-2 items-center">
                    <div className="flex-1">
                      <span className="text-[11px] text-dark-500 block mb-1">From</span>
                      <input
                        type="date"
                        value={localFrom}
                        onChange={(e) => setLocalFrom(e.target.value)}
                        className="w-full text-xs bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 text-dark-200
                          focus:outline-none focus:border-accent/50"
                      />
                    </div>
                    <div className="flex-1">
                      <span className="text-[11px] text-dark-500 block mb-1">To</span>
                      <input
                        type="date"
                        value={localTo}
                        min={localFrom || undefined}
                        onChange={(e) => setLocalTo(e.target.value)}
                        className="w-full text-xs bg-dark-800 border border-dark-700/50 rounded-lg px-3 py-2 text-dark-200
                          focus:outline-none focus:border-accent/50"
                      />
                    </div>
                  </div>
                </div>

                {/* Stub: Solicitation Number */}
                <div className="space-y-2 opacity-50">
                  <label className="text-xs font-medium text-dark-500">
                    Solicitation Number <span className="text-dark-600">(coming soon)</span>
                  </label>
                  <input
                    type="text"
                    disabled
                    placeholder="e.g. W911NF-24-R-0001"
                    className="w-full text-xs bg-dark-800/50 border border-dark-700/30 rounded-lg px-3 py-2 text-dark-500
                      placeholder:text-dark-600 cursor-not-allowed"
                  />
                </div>

                {/* Stub: Place of Performance */}
                <div className="space-y-2 opacity-50">
                  <label className="text-xs font-medium text-dark-500">
                    Place of Performance <span className="text-dark-600">(coming soon)</span>
                  </label>
                  <input
                    type="text"
                    disabled
                    placeholder="City or zip code"
                    className="w-full text-xs bg-dark-800/50 border border-dark-700/30 rounded-lg px-3 py-2 text-dark-500
                      placeholder:text-dark-600 cursor-not-allowed"
                  />
                </div>

                {/* Stub: Award Value Range */}
                <div className="space-y-2 opacity-50">
                  <label className="text-xs font-medium text-dark-500">
                    Award Value Range <span className="text-dark-600">(coming soon)</span>
                  </label>
                  <div className="flex gap-2 items-center">
                    <input
                      type="text"
                      disabled
                      placeholder="Min"
                      className="w-full text-xs bg-dark-800/50 border border-dark-700/30 rounded-lg px-3 py-2 text-dark-500
                        placeholder:text-dark-600 cursor-not-allowed"
                    />
                    <span className="text-dark-600 text-xs">&ndash;</span>
                    <input
                      type="text"
                      disabled
                      placeholder="Max"
                      className="w-full text-xs bg-dark-800/50 border border-dark-700/30 rounded-lg px-3 py-2 text-dark-500
                        placeholder:text-dark-600 cursor-not-allowed"
                    />
                  </div>
                </div>
              </div>

              {/* Footer */}
              <div className="flex items-center gap-3 px-5 py-4 border-t border-dark-800/50">
                <button
                  type="button"
                  onClick={handleReset}
                  className="flex-1 text-xs py-2 rounded-lg border border-dark-700/50 text-dark-400
                    hover:text-dark-200 hover:border-dark-600 transition-colors"
                >
                  Reset
                </button>
                <button
                  type="button"
                  onClick={handleApply}
                  className="flex-1 text-xs py-2 rounded-lg bg-accent hover:bg-accent-hover text-white transition-colors"
                >
                  Apply
                </button>
              </div>
            </div>
          </>,
          document.body,
        )}
    </>
  );
}
