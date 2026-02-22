import { forwardRef } from 'react';
import { Search, X, Loader2 } from 'lucide-react';

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit?: () => void;
  placeholder?: string;
  loading?: boolean;
  size?: 'default' | 'large';
  autoFocus?: boolean;
  showSubmitButton?: boolean;
}

const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  ({ value, onChange, onSubmit, placeholder = 'Search opportunities...', loading, size = 'default', autoFocus, showSubmitButton }, ref) => {
    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        onSubmit?.();
      }
    };

    const isLarge = size === 'large';
    const hasSubmit = showSubmitButton && onSubmit;

    return (
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
        <input
          ref={ref}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          autoFocus={autoFocus}
          className={`
            w-full bg-dark-900/80 backdrop-blur-sm
            border border-dark-700/50
            text-dark-50 placeholder:text-dark-500
            transition-all duration-300 ease-out
            focus:outline-none focus:border-accent/50 focus:ring-1 focus:ring-accent/20
            focus:bg-dark-900
            hover:border-dark-600/50 hover:bg-dark-900/90
            ${isLarge
              ? `py-5 pl-14 text-lg rounded-2xl ${hasSubmit ? 'pr-20' : 'pr-14'}`
              : 'py-3 pl-12 pr-12 text-sm rounded-xl'
            }
          `}
        />
        {value && (
          <button
            onClick={() => onChange('')}
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

SearchInput.displayName = 'SearchInput';

export default SearchInput;
