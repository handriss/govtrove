import { forwardRef, useState, useEffect, useRef, useCallback } from 'react';
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
  'Search contracts & awards…',
  'Try: cybersecurity training',
  'Use "quotes" for an exact phrase',
  'Try: cloud migration -support',
];

// A query is "exact" when the whole thing is one quoted phrase.
function isExactPhrase(s: string) {
  const t = s.trim();
  return t.length > 2 && t.startsWith('"') && t.endsWith('"') && t.indexOf('"', 1) === t.length - 1;
}

function wordCount(s: string) {
  return s.trim().split(/\s+/).filter(Boolean).length;
}

const KeywordChipInput = forwardRef<HTMLInputElement, KeywordChipInputProps>(
  ({ value, onChange, onSubmit, loading, size = 'default', autoFocus, showSubmitButton }, ref) => {
    // The box holds the raw query. It is passed through untouched — the old
    // version quoted every term and joined them with OR, which silently turned
    // "ai assessment" into an exact-phrase search and made extra words widen
    // rather than narrow the results.
    const [text, setText] = useState(value);
    const [hintIndex, setHintIndex] = useState(0);
    const [hintFading, setHintFading] = useState(false);
    const innerRef = useRef<HTMLInputElement>(null);

    // Keep in sync when the query changes elsewhere (URL, filter chip removal, clear).
    useEffect(() => { setText(value); }, [value]);

    const setRefs = useCallback((el: HTMLInputElement | null) => {
      (innerRef as React.MutableRefObject<HTMLInputElement | null>).current = el;
      if (typeof ref === 'function') ref(el);
      else if (ref) (ref as React.MutableRefObject<HTMLInputElement | null>).current = el;
    }, [ref]);

    useEffect(() => {
      if (text) return;
      const interval = setInterval(() => {
        setHintFading(true);
        setTimeout(() => {
          setHintIndex(i => (i + 1) % PLACEHOLDER_HINTS.length);
          setHintFading(false);
        }, 300);
      }, 4000);
      return () => clearInterval(interval);
    }, [text]);

    const commit = (next: string) => {
      onChange(next);
      onSubmit?.();
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        commit(text.trim());
      }
    };

    const toggleExact = () => {
      const t = text.trim();
      const next = isExactPhrase(t) ? t.slice(1, -1).trim() : `"${t.replace(/"/g, '')}"`;
      setText(next);
      commit(next);
    };

    const clear = () => { setText(''); onChange(''); };

    const isLarge = size === 'large';
    const hasSubmit = showSubmitButton && onSubmit;
    const exact = isExactPhrase(text);
    // Only worth offering when the distinction changes the result set.
    const showModeRow = wordCount(text) > 1;

    return (
      <div className="w-full">
        <div className="relative w-full group">
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
            className={`
              w-full bg-dark-800 border border-dark-600/50
              text-dark-50
              transition-all duration-300 ease-out
              focus-within:border-accent/50 focus-within:ring-2 focus-within:ring-accent/20
              focus-within:shadow-[0_0_24px_rgba(59,130,246,0.15)]
              hover:border-dark-500/60
              shadow-lg shadow-black/30
              flex items-center cursor-text
              ${isLarge
                ? `py-3 pl-14 rounded-2xl ${hasSubmit ? 'pr-20' : 'pr-14'} min-h-[62px]`
                : 'py-2 pl-12 pr-12 rounded-xl min-h-[46px]'
              }
            `}
            onClick={() => innerRef.current?.focus()}
          >
            <div className="relative flex-1 min-w-[120px]">
              <input
                ref={setRefs}
                type="text"
                value={text}
                onChange={(e) => setText(e.target.value)}
                onKeyDown={handleKeyDown}
                autoFocus={autoFocus}
                aria-label="Search keywords"
                className={`w-full bg-transparent outline-none text-dark-50 placeholder:text-dark-400
                  ${isLarge ? 'text-lg py-1' : 'text-sm py-0.5'}`}
                placeholder=""
              />
              {!text && (
                <span
                  className={`absolute inset-x-0 top-1/2 -translate-y-1/2 truncate pr-8 text-dark-400 pointer-events-none
                    transition-opacity duration-300 ${hintFading ? 'opacity-0' : 'opacity-100'}
                    ${isLarge ? 'text-lg' : 'text-sm'}`}
                >
                  {PLACEHOLDER_HINTS[hintIndex]}
                </span>
              )}
            </div>
          </div>
          {text && (
            <button
              onClick={clear}
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
              onClick={() => commit(text.trim())}
              className="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 rounded-xl
                text-dark-500 hover:text-dark-300 transition-colors"
              aria-label="Search"
            >
              <Search size={18} strokeWidth={1.5} />
            </button>
          )}
        </div>

        {/* Says which mode is active and lets you flip it without knowing the
            syntax — quotes are discoverable for people who never read help. */}
        {showModeRow && (
          <div className={`flex items-center gap-2 text-xs text-dark-400 ${isLarge ? 'mt-2.5 px-5' : 'mt-2 px-1'}`}>
            <span>
              {exact ? 'Exact phrase' : 'All words, any order'}
            </span>
            <span className="text-dark-600">·</span>
            <button
              type="button"
              onClick={toggleExact}
              className="text-accent hover:underline"
            >
              {exact ? 'match all words instead' : 'match exact phrase instead'}
            </button>
          </div>
        )}
      </div>
    );
  }
);

KeywordChipInput.displayName = 'KeywordChipInput';

export default KeywordChipInput;
