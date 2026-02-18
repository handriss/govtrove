interface SetAsideChipsProps {
  selected: string[];
  onChange: (selected: string[]) => void;
}

const CHIPS = [
  { code: 'SBA', label: 'Small Business', active: 'border-blue-500/30 bg-blue-500/10 text-blue-400' },
  { code: '8A', label: '8(a)', active: 'border-violet-500/30 bg-violet-500/10 text-violet-400' },
  { code: 'SDVOSB', label: 'SDVOSB', active: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400' },
  { code: 'WOSB', label: 'WOSB', active: 'border-pink-500/30 bg-pink-500/10 text-pink-400' },
  { code: 'HUBZone', label: 'HUBZone', active: 'border-orange-500/30 bg-orange-500/10 text-orange-400' },
] as const;

export default function SetAsideChips({ selected: _selected, onChange: _onChange }: SetAsideChipsProps) {
  return (
    <div className="relative opacity-50 pointer-events-none">
      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-fuchsia-500 bg-fuchsia-500/10 px-3 py-2">
        {CHIPS.map((chip) => (
          <span
            key={chip.code}
            className="inline-flex items-center rounded-full px-3 py-1.5 text-sm border border-dark-700/50 bg-dark-800/30 text-dark-400"
          >
            {chip.label}
          </span>
        ))}
        <span className="text-xs text-fuchsia-400 ml-1">Coming soon</span>
      </div>
    </div>
  );
}
