import { useState, useCallback, useEffect } from 'react';

const STORAGE_KEY = 'govtrove_saved_opportunities';
const MAX_SAVED = 500;

function loadSaved(): Set<number> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return new Set();
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr)) return new Set();
    return new Set(arr as number[]);
  } catch {
    return new Set();
  }
}

function persist(ids: Set<number>) {
  const arr = [...ids];
  // FIFO eviction: keep the most recently added (end of array)
  const trimmed = arr.length > MAX_SAVED ? arr.slice(arr.length - MAX_SAVED) : arr;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(trimmed));
}

export interface UseSavedOpportunitiesReturn {
  savedIds: Set<number>;
  isSaved: (id: number) => boolean;
  toggleSave: (id: number) => void;
  saveAll: (ids: number[]) => void;
  count: number;
}

export function useSavedOpportunities(): UseSavedOpportunitiesReturn {
  const [savedIds, setSavedIds] = useState(loadSaved);

  // Sync across tabs
  useEffect(() => {
    function onStorage(e: StorageEvent) {
      if (e.key === STORAGE_KEY) setSavedIds(loadSaved());
    }
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  const isSaved = useCallback((id: number) => savedIds.has(id), [savedIds]);

  const toggleSave = useCallback((id: number) => {
    setSavedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      persist(next);
      return next;
    });
  }, []);

  const saveAll = useCallback((ids: number[]) => {
    setSavedIds((prev) => {
      const next = new Set(prev);
      for (const id of ids) next.add(id);
      persist(next);
      return next;
    });
  }, []);

  return { savedIds, isSaved, toggleSave, saveAll, count: savedIds.size };
}
