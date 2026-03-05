import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { Link, Navigate, useLocation, useSearchParams } from 'react-router-dom';
import { Users, Key, Globe, BarChart3, Activity, Search, AlertCircle, Megaphone, ArrowLeft, PanelLeftClose, PanelLeftOpen, Mail, Send } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { getAdminUsers, type AdminUser } from '../services/api';

interface AdminContextValue {
  users: AdminUser[];
  setUsers: React.Dispatch<React.SetStateAction<AdminUser[]>>;
  getToken: () => Promise<string>;
}

const AdminContext = createContext<AdminContextValue | null>(null);

export function useAdminContext(): AdminContextValue {
  const ctx = useContext(AdminContext);
  if (!ctx) throw new Error('useAdminContext must be used within AdminLayout');
  return ctx;
}

const STORAGE_KEY = 'admin-sidebar-collapsed';

interface NavItem {
  label: string;
  icon: typeof Users;
  path: string;
  tab?: string;
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    label: 'Users & Analytics',
    items: [
      { label: 'Users', icon: Users, path: '/admin' },
      { label: 'Searches', icon: Search, path: '/admin', tab: 'searches' },
    ],
  },
  {
    label: 'Email',
    items: [
      { label: 'Email Prefs', icon: Mail, path: '/admin', tab: 'email-prefs' },
      { label: 'Sent Emails', icon: Send, path: '/admin', tab: 'sent-emails' },
    ],
  },
  {
    label: 'Pipeline & Data',
    items: [
      { label: 'Pipeline', icon: Activity, path: '/admin', tab: 'pipeline' },
      { label: 'API Keys', icon: Key, path: '/admin', tab: 'api-keys' },
      { label: 'SAM.gov Requests', icon: Globe, path: '/admin', tab: 'samgov-requests' },
      { label: 'Usage Chart', icon: BarChart3, path: '/admin', tab: 'usage' },
    ],
  },
  {
    label: 'Data Quality',
    items: [
      { label: 'Ingestion DQ', icon: AlertCircle, path: '/admin/data-quality' },
      { label: 'Reconcile DQ', icon: AlertCircle, path: '/admin/reconcile-dq' },
    ],
  },
  {
    label: 'Other',
    items: [
      { label: 'Campaigns', icon: Megaphone, path: '/admin/campaigns' },
    ],
  },
];

function isActive(item: NavItem, pathname: string, searchParams: URLSearchParams): boolean {
  if (item.tab === 'pipeline' && pathname.startsWith('/admin/pipeline/')) return true;
  if (item.path !== pathname) return false;
  if (item.path === '/admin' && pathname === '/admin') {
    const currentTab = searchParams.get('tab') || '';
    const itemTab = item.tab || '';
    return currentTab === itemTab;
  }
  return true;
}

export default function AdminLayout({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, getAccessToken } = useAppAuth();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [denied, setDenied] = useState(false);
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(STORAGE_KEY) === 'true');

  const location = useLocation();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      setDenied(true);
      setLoading(false);
      return;
    }
    (async () => {
      try {
        const token = await getAccessToken();
        setUsers(await getAdminUsers(token));
      } catch {
        setDenied(true);
      } finally {
        setLoading(false);
      }
    })();
  }, [isAuthenticated, isLoading, getAccessToken]);

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  if (loading || isLoading) {
    return <div className="min-h-screen flex items-center justify-center" />;
  }

  if (denied) {
    return <Navigate to="/" replace />;
  }

  const sidebarW = collapsed ? 'w-14' : 'w-[220px]';
  const contentML = collapsed ? 'ml-14' : 'ml-[220px]';

  function navLink(item: NavItem) {
    const active = isActive(item, location.pathname, searchParams);
    const Icon = item.icon;
    const to = item.tab ? `${item.path}?tab=${item.tab}` : item.path;

    return (
      <Link
        key={item.label}
        to={to}
        title={collapsed ? item.label : undefined}
        className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
          active
            ? 'bg-accent/10 text-accent'
            : 'text-dark-400 hover:text-dark-200 hover:bg-dark-800/50'
        }`}
      >
        <Icon size={18} className="shrink-0" />
        {!collapsed && <span className="truncate">{item.label}</span>}
      </Link>
    );
  }

  return (
    <AdminContext.Provider value={{ users, setUsers, getToken: getAccessToken }}>
      {/* Sidebar */}
      <aside className={`fixed left-0 top-0 h-screen ${sidebarW} border-r border-dark-700/50 bg-dark-900 flex flex-col transition-all duration-200 z-40`}>
        {/* Header */}
        <div className="flex items-center gap-3 px-3 py-4 border-b border-dark-700/50">
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="p-1 rounded hover:bg-dark-800 text-dark-400 hover:text-dark-200 transition-colors"
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
          </button>
          {!collapsed && <span className="text-sm font-medium text-dark-200">Admin</span>}
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto px-2 py-3">
          {navGroups.map((group, gi) => (
            <div key={group.label}>
              {gi > 0 && <hr className="border-dark-700/50 my-2 mx-1" />}
              {!collapsed && (
                <span className="block px-3 pt-1 pb-1.5 text-[10px] font-semibold uppercase tracking-wider text-dark-600">
                  {group.label}
                </span>
              )}
              <div className="space-y-1">
                {group.items.map(navLink)}
              </div>
            </div>
          ))}
        </nav>

        {/* Footer */}
        <div className="px-2 py-3 border-t border-dark-700/50">
          <Link
            to="/"
            title={collapsed ? 'Back to search' : undefined}
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-dark-400 hover:text-dark-200 hover:bg-dark-800/50 transition-colors"
          >
            <ArrowLeft size={18} className="shrink-0" />
            {!collapsed && <span>Back to search</span>}
          </Link>
        </div>
      </aside>

      {/* Content */}
      <main className={`${contentML} transition-all duration-200 min-h-screen`}>
        {children}
      </main>
    </AdminContext.Provider>
  );
}
