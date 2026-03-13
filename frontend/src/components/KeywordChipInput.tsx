import { forwardRef, useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Search, X, Loader2 } from 'lucide-react';

interface KeywordChipInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit?: () => void;
  placeholder?: string;
  loading?: boolean;
  size?: 'default' | 'large';
  autoFocus?: boolean;
  showSubmitButton?: boolean;
}

const PLACEHOLDER_HINTS = [
  'Search contracts, solicitations, awards...',
  'Try: cybersecurity, cloud migration',
  'Tip: press Enter to add multiple keywords',
  'Try: IT services, construction',
];

// Strip surrounding quotes for display
function unquote(s: string) { return s.replace(/^"|"$/g, ''); }
// Ensure term is quoted for phrase matching
function ensureQuoted(s: string) {
  const t = s.trim();
  if (t.startsWith('"') && t.endsWith('"')) return t;
  return `"${t}"`;
}

const KeywordChipInput = forwardRef<HTMLInputElement, KeywordChipInputProps>(
  ({ value, onChange, onSubmit, loading, size = 'default', autoFocus, showSubmitButton }, ref) => {
    const [inputText, setInputText] = useState('');
    const [hintIndex, setHintIndex] = useState(0);
    const [hintFading, setHintFading] = useState(false);
    const innerRef = useRef<HTMLInputElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);

    // Expose the inner input ref for "/" shortcut focus
    const setRefs = useCallback((el: HTMLInputElement | null) => {
      (innerRef as React.MutableRefObject<HTMLInputElement | null>).current = el;
      if (typeof ref === 'function') ref(el);
      else if (ref) (ref as React.MutableRefObject<HTMLInputElement | null>).current = el;
    }, [ref]);

    // chips = display values (without quotes)
    const chips = useMemo(() => {
      if (!value) return [];
      return value.split(' OR ').filter(Boolean).map(unquote);
    }, [value]);

    const updateChips = useCallback((newChips: string[]) => {
      onChange(newChips.map(ensureQuoted).join(' OR '));
    }, [onChange]);

    const addChip = useCallback((text: string) => {
      const trimmed = text.trim().replace(/^"|"$/g, '');
      if (!trimmed) return false;
      if (chips.some(c => c.toLowerCase() === trimmed.toLowerCase())) return false;
      updateChips([...chips, trimmed]);
      return true;
    }, [chips, updateChips]);

    const removeChip = useCallback((index: number) => {
      const newChips = chips.filter((_, i) => i !== index);
      updateChips(newChips);
    }, [chips, updateChips]);

    // Rotate placeholder hints when no chips and no input text
    useEffect(() => {
      if (chips.length > 0 || inputText) return;
      const interval = setInterval(() => {
        setHintFading(true);
        setTimeout(() => {
          setHintIndex(i => (i + 1) % PLACEHOLDER_HINTS.length);
          setHintFading(false);
        }, 300);
      }, 4000);
      return () => clearInterval(interval);
    }, [chips.length, inputText]);

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (inputText.trim()) {
          addChip(inputText);
          setInputText('');
        }
        onSubmit?.();
      } else if (e.key === 'Backspace' && !inputText && chips.length > 0) {
        removeChip(chips.length - 1);
      }
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      // Comma creates a chip
      if (val.includes(',')) {
        const parts = val.split(',');
        for (let i = 0; i < parts.length - 1; i++) {
          addChip(parts[i]);
        }
        setInputText(parts[parts.length - 1]);
        return;
      }
      setInputText(val);
    };

    const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
      const text = e.clipboardData.getData('text');
      if (text.includes(',') || text.includes(' OR ')) {
        e.preventDefault();
        const parts = text.split(/,| OR /).map(s => s.trim()).filter(Boolean);
        for (const part of parts) {
          addChip(part);
        }
        setInputText('');
      }
    };

    const handleContainerClick = () => {
      innerRef.current?.focus();
    };

    const isLarge = size === 'large';
    const hasSubmit = showSubmitButton && onSubmit;

    return (
      <div className="relative w-full group" ref={containerRef}>
        <div
          className={`
            absolute top-1/2 -translate-y-1/2 text-dark-400
            transition-colors duration-200 group-focus-within:text-accent
            ${isLarge ? 'left-5' : 'left-4'}
          `}
        >
          {loading ? (
            <Loader2 size={isLarge ? 22 : 18} className="animate-spin" />
          ) : (
            <Search size={isLarge ? 22 : 18} strokeWidth={1.5} />
          )}
        </div>
        <div
          onClick={handleContainerClick}
          className={`
            w-full bg-dark-800 border border-dark-600/50
            text-dark-50
            transition-all duration-300 ease-out
            focus-within:border-accent/50 focus-within:ring-2 focus-within:ring-accent/20
            focus-within:shadow-[0_0_24px_rgba(59,130,246,0.15)]
            hover:border-dark-500/60
            shadow-lg shadow-black/30
            flex flex-wrap items-center gap-1.5 cursor-text
            ${isLarge
              ? `py-3 pl-14 rounded-2xl ${hasSubmit ? 'pr-20' : 'pr-14'} min-h-[62px]`
              : 'py-2 pl-12 pr-12 rounded-xl min-h-[46px]'
            }
          `}
        >
          {chips.map((chip, i) => (
            <span
              key={`${chip}-${i}`}
              className={`inline-flex items-center gap-1 bg-accent/15 border border-accent/25 text-accent
                rounded-full ${isLarge ? 'text-sm px-3 py-1' : 'text-xs px-2.5 py-0.5'}`}
            >
              {chip}
              <button
                type="button"
                onClick={(e) => { e.stopPropagation(); removeChip(i); }}
                className="hover:text-white transition-colors"
              >
                <X size={isLarge ? 14 : 12} strokeWidth={2} />
              </button>
            </span>
          ))}
          <div className="relative flex-1 min-w-[120px]">
            <input
              ref={setRefs}
              type="text"
              value={inputText}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              onPaste={handlePaste}
              autoFocus={autoFocus}
              className={`w-full bg-transparent outline-none text-dark-50 placeholder:text-dark-400
                ${isLarge ? 'text-lg py-1' : 'text-sm py-0.5'}`}
              placeholder={chips.length === 0 ? '' : 'Add keyword...'}
            />
            {chips.length === 0 && !inputText && (
              <span
                className={`absolute left-0 top-1/2 -translate-y-1/2 text-dark-400 pointer-events-none
                  transition-opacity duration-300 ${hintFading ? 'opacity-0' : 'opacity-100'}
                  ${isLarge ? 'text-lg' : 'text-sm'}`}
              >
                {PLACEHOLDER_HINTS[hintIndex]}
              </span>
            )}
          </div>
        </div>
        {(value || inputText) && (
          <button
            onClick={() => { onChange(''); setInputText(''); }}
            className={`
              absolute top-1/2 -translate-y-1/2
              text-dark-500 hover:text-dark-200
              transition-all duration-200
              hover:scale-110 active:scale-95
              ${isLarge ? (hasSubmit ? 'right-12' : 'right-5') : 'right-4'}
            `}
            title="Clear"
          >
            <X size={isLarge ? 20 : 16} strokeWidth={1.5} />
          </button>
        )}
        {hasSubmit && (
          <button
            onClick={onSubmit}
            className="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 rounded-xl
              text-dark-500 hover:text-dark-300 transition-colors"
            aria-label="Search"
          >
            <Search size={18} strokeWidth={1.5} />
          </button>
        )}
      </div>
    );
  }
);

KeywordChipInput.displayName = 'KeywordChipInput';

export default KeywordChipInput;
