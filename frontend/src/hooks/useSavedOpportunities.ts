import { useState, useCallback, useEffect, useRef } from 'react';
import { useAppAuth } from '../contexts/AuthContext';
import {
  getSavedOpportunities,
  saveOpportunity,
  unsaveOpportunity,
  bulkSaveOpportunities,
} from '../services/api';

const STORAGE_KEY = 'govtrove_saved_opportunities';
const MAX_SAVED = 500;

function loadFromStorage(): Set<number> {
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

function persistToStorage(ids: Set<number>) {
  const arr = [...ids];
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
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const [savedIds, setSavedIds] = useState(loadFromStorage);
  const migratedRef = useRef(false);
  const fetchedRef = useRef(false);

  // Fetch from server when authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      fetchedRef.current = false;
      migratedRef.current = false;
      setSavedIds(loadFromStorage());
      return;
    }

    if (fetchedRef.current) return;
    fetchedRef.current = true;

    let cancelled = false;
    (async () => {
      try {
        const token = await getAccessToken();
        const ids = await getSavedOpportunities(token);
        if (cancelled) return;

        const serverSet = new Set(ids);

        // One-time migration: push localStorage saves to server
        if (!migratedRef.current) {
          migratedRef.current = true;
          const localIds = loadFromStorage();
          const toMigrate = [...localIds].filter((id) => !serverSet.has(id));
          if (toMigrate.length > 0) {
            try {
              await bulkSaveOpportunities(token, toMigrate);
              for (const id of toMigrate) serverSet.add(id);
            } catch {
              // Migration failed — not critical, local items stay in localStorage
            }
          }
          localStorage.removeItem(STORAGE_KEY);
        }

        if (!cancelled) setSavedIds(serverSet);
      } catch {
        // Fall back to localStorage on error
      }
    })();

    return () => { cancelled = true; };
  }, [isAuthenticated, getAccessToken]);

  // Sync across tabs (anonymous only)
  useEffect(() => {
    if (isAuthenticated) return;
    function onStorage(e: StorageEvent) {
      if (e.key === STORAGE_KEY) setSavedIds(loadFromStorage());
    }
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [isAuthenticated]);

  const isSaved = useCallback((id: number) => savedIds.has(id), [savedIds]);

  const toggleSave = useCallback((id: number) => {
    setSavedIds((prev) => {
      const next = new Set(prev);
      const removing = next.has(id);
      if (removing) next.delete(id);
      else next.add(id);

      if (isAuthenticated) {
        getAccessToken().then((token) => {
          const fn = removing ? unsaveOpportunity : saveOpportunity;
          fn(token, id).catch(() => {
            // Revert on failure
            setSavedIds((curr) => {
              const reverted = new Set(curr);
              if (removing) reverted.add(id);
              else reverted.delete(id);
              return reverted;
            });
          });
        });
      } else {
        persistToStorage(next);
      }

      return next;
    });
  }, [isAuthenticated, getAccessToken]);

  const saveAll = useCallback((ids: number[]) => {
    setSavedIds((prev) => {
      const next = new Set(prev);
      for (const id of ids) next.add(id);

      if (isAuthenticated) {
        getAccessToken().then((token) => {
          bulkSaveOpportunities(token, ids).catch(() => {});
        });
      } else {
        persistToStorage(next);
      }

      return next;
    });
  }, [isAuthenticated, getAccessToken]);

  return { savedIds, isSaved, toggleSave, saveAll, count: savedIds.size };
}
