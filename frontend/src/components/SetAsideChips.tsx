interface SetAsideChipsProps {
  selected: string[];
  onChange: (selected: string[]) => void;
}

const ENABLED_CHIPS = [
  { code: 'SBA', label: 'Small Business', active: 'border-blue-500/30 bg-blue-500/10 text-blue-400' },
] as const;

const COMING_SOON_CHIPS = [
  { code: '8A', label: '8(a)' },
  { code: 'SDVOSB', label: 'SDVOSB' },
  { code: 'WOSB', label: 'WOSB' },
  { code: 'HUBZone', label: 'HUBZone' },
] as const;

export default function SetAsideChips({ selected, onChange }: SetAsideChipsProps) {
  const toggle = (code: string) => {
    onChange(
      selected.includes(code)
        ? selected.filter((c) => c !== code)
        : [...selected, code],
    );
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      {ENABLED_CHIPS.map((chip) => {
        const isSelected = selected.includes(chip.code);
        return (
          <button
            key={chip.code}
            type="button"
            onClick={() => toggle(chip.code)}
            className={`inline-flex items-center rounded-full px-3 py-1.5 text-sm border transition-all duration-150 cursor-pointer
                       ${isSelected ? chip.active : 'border-dark-700/50 bg-dark-800/30 text-dark-400 hover:border-dark-600 hover:text-dark-300'}`}
          >
            {chip.label}
          </button>
        );
      })}
      <div className="flex items-center gap-2 opacity-50">
        {COMING_SOON_CHIPS.map((chip) => (
          <span
            key={chip.code}
            className="inline-flex items-center rounded-full px-3 py-1.5 text-sm border border-dark-700/50 bg-dark-800/30 text-dark-500"
          >
            {chip.label}
          </span>
        ))}
        <span className="text-xs text-dark-500 ml-1">Coming soon</span>
      </div>
    </div>
  );
}
