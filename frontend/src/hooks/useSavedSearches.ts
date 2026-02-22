import { useState, useCallback, useEffect, useRef } from 'react';
import { useAppAuth } from '../contexts/AuthContext';
import type { SavedSearch } from '../types/api';
import {
  getSavedSearches,
  createSavedSearch,
  deleteSavedSearch as deleteSavedSearchAPI,
  updateSavedSearch,
} from '../services/api';

export interface UseSavedSearchesReturn {
  savedSearches: SavedSearch[];
  loading: boolean;
  saveCurrentSearch: (name: string, filters: Record<string, unknown>) => Promise<void>;
  deleteSearch: (id: number) => Promise<void>;
  renameSearch: (id: number, name: string) => Promise<void>;
}

export function useSavedSearches(): UseSavedSearchesReturn {
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const [savedSearches, setSavedSearches] = useState<SavedSearch[]>([]);
  const [loading, setLoading] = useState(false);
  const fetchedRef = useRef(false);

  useEffect(() => {
    if (!isAuthenticated) {
      fetchedRef.current = false;
      setSavedSearches([]);
      return;
    }

    if (fetchedRef.current) return;
    fetchedRef.current = true;

    let cancelled = false;
    setLoading(true);

    (async () => {
      try {
        const token = await getAccessToken();
        const searches = await getSavedSearches(token);
        if (!cancelled) setSavedSearches(searches);
      } catch {
        // Silently fail
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [isAuthenticated, getAccessToken]);

  const saveCurrentSearch = useCallback(async (name: string, filters: Record<string, unknown>) => {
    const token = await getAccessToken();
    const search = await createSavedSearch(token, name, filters);
    setSavedSearches((prev) => [search, ...prev]);
  }, [getAccessToken]);

  const deleteSearch = useCallback(async (id: number) => {
    setSavedSearches((prev) => prev.filter((s) => s.id !== id));
    try {
      const token = await getAccessToken();
      await deleteSavedSearchAPI(token, id);
    } catch {
      // Refetch to restore state
      try {
        const token = await getAccessToken();
        const searches = await getSavedSearches(token);
        setSavedSearches(searches);
      } catch {
        // Give up
      }
    }
  }, [getAccessToken]);

  const renameSearch = useCallback(async (id: number, name: string) => {
    const token = await getAccessToken();
    const updated = await updateSavedSearch(token, id, { name });
    setSavedSearches((prev) => prev.map((s) => (s.id === updated.id ? updated : s)));
  }, [getAccessToken]);

  return { savedSearches, loading, saveCurrentSearch, deleteSearch, renameSearch };
}
