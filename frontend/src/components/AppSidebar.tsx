import { Link, useLocation } from 'react-router-dom';
import { Search, Star, Bell, BookOpen, BookText, PanelLeftClose, PanelLeftOpen, Shield } from 'lucide-react';
import { useNotificationsCount } from '../hooks/useNotifications';
import { useSidebar } from '../contexts/SidebarContext';
import { useIsAdmin } from '../hooks/useIsAdmin';

interface NavItem {
  label: string;
  icon: typeof Search;
  path: string;
  badge?: number;
  disabled?: boolean;
  tag?: string;
}

export default function AppSidebar() {
  const { collapsed, toggle } = useSidebar();
  const { count } = useNotificationsCount();
  const isAdmin = useIsAdmin();
  const location = useLocation();

  const navItems: NavItem[] = [
    { label: 'Search', icon: Search, path: '/' },
    { label: 'Saved', icon: Star, path: '/saved' },
    { label: 'Notifications', icon: Bell, path: '/notifications', badge: count.unread },
    { label: 'Guide', icon: BookOpen, path: '/guide' },
    { label: 'Glossary', icon: BookText, path: '/glossary' },
  ];

  function isActive(path: string): boolean {
    if (path === '/') return location.pathname === '/';
    return location.pathname.startsWith(path);
  }

  const sidebarW = collapsed ? 'w-14' : 'w-56';

  return (
    <aside className={`hidden md:flex fixed left-0 top-0 h-screen ${sidebarW} border-r border-dark-700/50 bg-dark-900 flex-col transition-all duration-200 z-40`}>
      {/* Logo */}
      <div className="flex items-center gap-3 px-3 py-4 border-b border-dark-700/50">
        <Link to="/" className="flex items-center gap-2 min-w-0">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-accent shrink-0"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/></svg>
          {!collapsed && <span className="text-sm font-semibold text-dark-200 tracking-tight truncate">GovTrove</span>}
        </Link>
      </div>

      {/* Collapse toggle */}
      <div className="px-2 py-2">
        <button
          onClick={toggle}
          className="w-full flex items-center justify-center p-1.5 rounded hover:bg-dark-800 text-dark-400 hover:text-dark-200 transition-colors"
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
        </button>
      </div>

      {/* Nav items */}
      <nav className="flex-1 overflow-y-auto px-2">
        <div className="space-y-1">
          {navItems.map((item) => {
            const active = !item.disabled && isActive(item.path);
            const Icon = item.icon;

            if (item.disabled) {
              return (
                <span
                  key={item.path}
                  title={collapsed ? `${item.label} (${item.tag})` : undefined}
                  className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-dark-600 cursor-default"
                >
                  <Icon size={18} className="shrink-0" />
                  {!collapsed && (
                    <>
                      <span className="truncate">{item.label}</span>
                      {item.tag && (
                        <span className="ml-auto text-[10px] font-medium text-dark-600 bg-dark-800/50 px-1.5 py-0.5 rounded-full shrink-0">
                          {item.tag}
                        </span>
                      )}
                    </>
                  )}
                </span>
              );
            }

            return (
              <Link
                key={item.path}
                to={item.path}
                title={collapsed ? item.label : undefined}
                className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                  active
                    ? 'bg-accent/10 text-accent'
                    : 'text-dark-400 hover:text-dark-200 hover:bg-dark-800/50'
                }`}
              >
                <span className="relative shrink-0">
                  <Icon size={18} />
                  {item.badge && item.badge > 0 ? (
                    <span className="absolute -top-1.5 -right-1.5 min-w-[16px] h-4 px-1 bg-red-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center">
                      {item.badge > 99 ? '99+' : item.badge}
                    </span>
                  ) : null}
                </span>
                {!collapsed && <span className="truncate">{item.label}</span>}
              </Link>
            );
          })}
        </div>
      </nav>

      {isAdmin && (
        <div className="px-2 py-3 border-t border-dark-700/50">
          <Link
            to="/admin"
            title={collapsed ? 'Admin' : undefined}
            className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
              location.pathname.startsWith('/admin')
                ? 'bg-accent/10 text-accent'
                : 'text-dark-400 hover:text-dark-200 hover:bg-dark-800/50'
            }`}
          >
            <Shield size={18} className="shrink-0" />
            {!collapsed && <span className="truncate">Admin</span>}
          </Link>
        </div>
      )}
    </aside>
  );
}
