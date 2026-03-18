import { useState, useEffect } from 'react';
import { useAppAuth } from '../contexts/AuthContext';
import { getAdminUsers } from '../services/api';

const cache = { checked: false, isAdmin: false };

export function useIsAdmin(): boolean {
  const { isAuthenticated, isLoading, getAccessToken } = useAppAuth();
  const [isAdmin, setIsAdmin] = useState(cache.isAdmin);

  useEffect(() => {
    if (isLoading || !isAuthenticated) return;
    if (cache.checked) {
      setIsAdmin(cache.isAdmin);
      return;
    }

    (async () => {
      try {
        const token = await getAccessToken();
        await getAdminUsers(token);
        cache.checked = true;
        cache.isAdmin = true;
        setIsAdmin(true);
      } catch {
        cache.checked = true;
        cache.isAdmin = false;
      }
    })();
  }, [isAuthenticated, isLoading, getAccessToken]);

  return isAdmin;
}
