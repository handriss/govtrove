import { useState, useCallback, useEffect, useRef } from 'react';
import { useAppAuth } from '../contexts/AuthContext';
import type { Notification, NotificationCount } from '../types/api';
import {
  getNotifications,
  getNotificationsCount,
  markNotificationRead as markNotificationReadAPI,
  markAllNotificationsRead as markAllReadAPI,
  deleteNotification as deleteNotificationAPI,
} from '../services/api';

const NOTIFICATIONS_CHANGED = 'govtrove:notifications-changed';

export function notifyNotificationsChanged() {
  window.dispatchEvent(new Event(NOTIFICATIONS_CHANGED));
}

export function useNotificationsCount() {
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const [count, setCount] = useState<NotificationCount>({ unread: 0, total: 0 });
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  const fetchCount = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const token = await getAccessToken();
      const c = await getNotificationsCount(token);
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
    window.addEventListener(NOTIFICATIONS_CHANGED, fetchCount);

    return () => {
      clearInterval(intervalRef.current);
      document.removeEventListener('visibilitychange', onVisibilityChange);
      window.removeEventListener(NOTIFICATIONS_CHANGED, fetchCount);
    };
  }, [isAuthenticated, fetchCount]);

  return { count, refetch: fetchCount };
}

export function useNotifications() {
  const { isAuthenticated, getAccessToken } = useAppAuth();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');

  const fetchNotifications = useCallback(async (p: number, unreadOnly: boolean, append: boolean) => {
    if (!isAuthenticated) return;
    setLoading(true);
    try {
      const token = await getAccessToken();
      const data = await getNotifications(token, { unread_only: unreadOnly, page: p, limit: 25 });
      setNotifications((prev) => append ? [...prev, ...data.notifications] : data.notifications);
      setTotal(data.total);
    } catch {
      // Silently fail
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, getAccessToken]);

  useEffect(() => {
    if (!isAuthenticated) {
      setNotifications([]);
      setTotal(0);
      return;
    }
    setPage(1);
    fetchNotifications(1, filter === 'unread', false);
  }, [isAuthenticated, filter, fetchNotifications]);

  const loadMore = useCallback(() => {
    const next = page + 1;
    setPage(next);
    fetchNotifications(next, filter === 'unread', true);
  }, [page, filter, fetchNotifications]);

  const markRead = useCallback(async (id: string) => {
    setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, is_read: true } : n)));
    try {
      const token = await getAccessToken();
      await markNotificationReadAPI(token, id);
      notifyNotificationsChanged();
    } catch {
      setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, is_read: false } : n)));
    }
  }, [getAccessToken]);

  const markAllRead = useCallback(async () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
    try {
      const token = await getAccessToken();
      await markAllReadAPI(token);
      notifyNotificationsChanged();
    } catch {
      fetchNotifications(1, filter === 'unread', false);
    }
  }, [getAccessToken, fetchNotifications, filter]);

  const remove = useCallback(async (id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
    setTotal((prev) => prev - 1);
    try {
      const token = await getAccessToken();
      await deleteNotificationAPI(token, id);
      notifyNotificationsChanged();
    } catch {
      fetchNotifications(1, filter === 'unread', false);
    }
  }, [getAccessToken, fetchNotifications, filter]);

  const refetch = useCallback(() => {
    setPage(1);
    fetchNotifications(1, filter === 'unread', false);
  }, [filter, fetchNotifications]);

  const hasMore = notifications.length < total;

  return { notifications, total, loading, hasMore, filter, setFilter, loadMore, markRead, markAllRead, remove, refetch };
}
