import { useState, useCallback, useEffect, useRef } from 'react';
import { useAppAuth } from '../contexts/AuthContext';
import type { UserUpdate, UserUpdateCount } from '../types/api';
import {
  getUpdates,
  getUpdatesCount,
  markUpdateRead as markUpdateReadAPI,
  markAllUpdatesRead as markAllReadAPI,
  deleteUpdate as deleteUpdateAPI,
} from '../services/api';

export function useUpdatesCount() {
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const [count, setCount] = useState<UserUpdateCount>({ unread: 0, total: 0 });
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  const fetchCount = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const token = await getAccessToken();
      const c = await getUpdatesCount(token);
      setCount(c);
    } catch {
      // Silently fail
    }
  }, [isAuthenticated, getAccessToken]);

  useEffect(() => {
    if (!isAuthenticated) {
      setCount({ unread: 0, total: 0 });
      return;
    }

    fetchCount();
    intervalRef.current = setInterval(fetchCount, 60_000);

    function onVisibilityChange() {
      if (document.visibilityState === 'visible') fetchCount();
    }
    document.addEventListener('visibilitychange', onVisibilityChange);

    return () => {
      clearInterval(intervalRef.current);
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
  }, [isAuthenticated, fetchCount]);

  return { count, refetch: fetchCount };
}

export function useUpdates() {
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const [updates, setUpdates] = useState<UserUpdate[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');

  const fetchUpdates = useCallback(async (p: number, unreadOnly: boolean, append: boolean) => {
    if (!isAuthenticated) return;
    setLoading(true);
    try {
      const token = await getAccessToken();
      const data = await getUpdates(token, { unread_only: unreadOnly, page: p, limit: 25 });
      setUpdates((prev) => append ? [...prev, ...data.updates] : data.updates);
      setTotal(data.total);
    } catch {
      // Silently fail
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, getAccessToken]);

  useEffect(() => {
    if (!isAuthenticated) {
      setUpdates([]);
      setTotal(0);
      return;
    }
    setPage(1);
    fetchUpdates(1, filter === 'unread', false);
  }, [isAuthenticated, filter, fetchUpdates]);

  const loadMore = useCallback(() => {
    const next = page + 1;
    setPage(next);
    fetchUpdates(next, filter === 'unread', true);
  }, [page, filter, fetchUpdates]);

  const markRead = useCallback(async (id: string) => {
    setUpdates((prev) => prev.map((u) => (u.id === id ? { ...u, is_read: true } : u)));
    try {
      const token = await getAccessToken();
      await markUpdateReadAPI(token, id);
    } catch {
      // Revert on failure
      setUpdates((prev) => prev.map((u) => (u.id === id ? { ...u, is_read: false } : u)));
    }
  }, [getAccessToken]);

  const markAllRead = useCallback(async () => {
    setUpdates((prev) => prev.map((u) => ({ ...u, is_read: true })));
    try {
      const token = await getAccessToken();
      await markAllReadAPI(token);
    } catch {
      fetchUpdates(1, filter === 'unread', false);
    }
  }, [getAccessToken, fetchUpdates, filter]);

  const remove = useCallback(async (id: string) => {
    setUpdates((prev) => prev.filter((u) => u.id !== id));
    setTotal((prev) => prev - 1);
    try {
      const token = await getAccessToken();
      await deleteUpdateAPI(token, id);
    } catch {
      fetchUpdates(1, filter === 'unread', false);
    }
  }, [getAccessToken, fetchUpdates, filter]);

  const refetch = useCallback(() => {
    setPage(1);
    fetchUpdates(1, filter === 'unread', false);
  }, [filter, fetchUpdates]);

  const hasMore = updates.length < total;

  return { updates, total, loading, hasMore, filter, setFilter, loadMore, markRead, markAllRead, remove, refetch };
}
