import { useState, useCallback } from 'react';

interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
  onSearch: () => void;
}

export function SearchBar({ value, onChange, onSearch }: SearchBarProps) {
  const [inputValue, setInputValue] = useState(value);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      onChange(inputValue);
      onSearch();
    },
    [inputValue, onChange, onSearch]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        onChange(inputValue);
        onSearch();
      }
    },
    [inputValue, onChange, onSearch]
  );

  return (
    <form onSubmit={handleSubmit} className="w-full">
      <div className="relative">
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search opportunities (e.g., cybersecurity, IT services, construction)"
          className="w-full px-4 py-3 pr-24 text-gray-900 bg-white border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <button
          type="submit"
          className="absolute right-2 top-1/2 -translate-y-1/2 px-4 py-1.5 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          Search
        </button>
      </div>
    </form>
  );
}
