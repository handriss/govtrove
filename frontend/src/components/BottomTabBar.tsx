import { Link, useLocation } from 'react-router-dom';
import { Search, Star, Bell, Building2, User, LogIn, Shield } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { useNotificationsCount } from '../hooks/useNotifications';
import { useIsAdmin } from '../hooks/useIsAdmin';

interface Tab {
  label: string;
  icon: typeof Search;
  path: string;
  authRequired?: boolean;
  badge?: number;
  disabled?: boolean;
}

export default function BottomTabBar() {
  const { isAuthenticated, signIn } = useAppAuth();
  const isAdmin = useIsAdmin();
  const { count } = useNotificationsCount();
  const location = useLocation();

  const tabs: Tab[] = isAuthenticated
    ? [
        { label: 'Search', icon: Search, path: '/' },
        { label: 'Saved', icon: Star, path: '/saved' },
        { label: 'Updates', icon: Bell, path: '/notifications', badge: count.unread },
        ...(isAdmin
          ? [{ label: 'Admin', icon: Shield, path: '/admin' } as Tab]
          : [{ label: 'Company', icon: Building2, path: '/company', disabled: true } as Tab]),
        { label: 'Profile', icon: User, path: '/profile' },
      ]
    : [
        { label: 'Search', icon: Search, path: '/' },
      ];

  function isActive(path: string): boolean {
    if (path === '/') return location.pathname === '/';
    return location.pathname.startsWith(path);
  }

  return (
    <nav className="md:hidden fixed bottom-0 left-0 right-0 z-40 bg-dark-900/95 border-t border-dark-700/50 backdrop-blur-md" style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}>
      <div className="flex items-center justify-around h-16">
        {tabs.map((tab) => {
          const active = !tab.disabled && isActive(tab.path);
          const Icon = tab.icon;

          if (tab.disabled) {
            return (
              <span
                key={tab.path}
                className="flex flex-col items-center justify-center gap-0.5 flex-1 h-full text-dark-700 cursor-default"
              >
                <Icon size={20} />
                <span className="text-[10px] leading-tight">{tab.label}</span>
              </span>
            );
          }

          return (
            <Link
              key={tab.path}
              to={tab.path}
              className={`flex flex-col items-center justify-center gap-0.5 flex-1 h-full transition-colors ${
                active ? 'text-accent' : 'text-dark-500'
              }`}
            >
              <span className="relative">
                <Icon size={20} />
                {tab.badge && tab.badge > 0 ? (
                  <span className="absolute -top-1 -right-2 min-w-[14px] h-3.5 px-0.5 bg-red-500 text-white text-[9px] font-bold rounded-full flex items-center justify-center">
                    {tab.badge > 99 ? '99+' : tab.badge}
                  </span>
                ) : null}
              </span>
              <span className="text-[10px] leading-tight">{tab.label}</span>
            </Link>
          );
        })}
        {!isAuthenticated && (
          <button
            onClick={() => signIn()}
            className="flex flex-col items-center justify-center gap-0.5 flex-1 h-full text-dark-500 transition-colors"
          >
            <LogIn size={20} />
            <span className="text-[10px] leading-tight">Sign In</span>
          </button>
        )}
      </div>
    </nav>
  );
}
