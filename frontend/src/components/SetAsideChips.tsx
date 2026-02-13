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

const INACTIVE = 'border-dark-700/50 bg-dark-800/30 text-dark-400 hover:text-dark-300 hover:border-dark-600/50';

export default function SetAsideChips({ selected, onChange }: SetAsideChipsProps) {
  const toggle = (code: string) => {
    if (selected.includes(code)) {
      onChange(selected.filter((s) => s !== code));
    } else {
      onChange([...selected, code]);
    }
  };

  return (
    <>
      {CHIPS.map((chip) => {
        const isActive = selected.includes(chip.code);
        return (
          <button
            key={chip.code}
            onClick={() => toggle(chip.code)}
            className={`inline-flex items-center rounded-full px-3 py-1.5 text-sm border transition-all duration-200 ${
              isActive ? chip.active : INACTIVE
            }`}
          >
            {chip.label}
          </button>
        );
      })}
    </>
  );
}
