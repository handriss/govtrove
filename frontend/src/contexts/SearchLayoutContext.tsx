import { createContext, useContext, useState, type ReactNode } from 'react';

interface SearchLayoutContextValue {
  heroMode: boolean;
  setHeroMode: (v: boolean) => void;
}

const SearchLayoutContext = createContext<SearchLayoutContextValue | null>(null);

export function SearchLayoutProvider({ children }: { children: ReactNode }) {
  const [heroMode, setHeroMode] = useState(true);

  return (
    <SearchLayoutContext.Provider value={{ heroMode, setHeroMode }}>
      {children}
    </SearchLayoutContext.Provider>
  );
}

export function useSearchLayout() {
  const ctx = useContext(SearchLayoutContext);
  if (!ctx) throw new Error('useSearchLayout must be used within SearchLayoutProvider');
  return ctx;
}
