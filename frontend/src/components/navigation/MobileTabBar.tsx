import { Link, useLocation } from 'react-router-dom';
import { Search, Clock, Star, User } from 'lucide-react';
import { useAppAuth } from '../../contexts/AuthContext';

const tabs = [
  { to: '/', label: 'Search', icon: Search },
  { to: '/whats-new', label: "What's New", icon: Clock },
  { to: '/saved', label: 'Saved', icon: Star },
  { to: '/profile', label: 'Profile', icon: User },
] as const;

export default function MobileTabBar() {
  const { pathname } = useLocation();
  const { isAuthenticated, signIn } = useAppAuth();

  if (pathname === '/callback') return null;

  return (
    <nav className="fixed bottom-0 inset-x-0 z-40 md:hidden border-t border-dark-800/50 bg-dark-950/90 backdrop-blur-md safe-area-bottom">
      <div className="flex items-center justify-around h-16">
        {tabs.map(({ to, label, icon: Icon }) => {
          const isProfile = to === '/profile';
          const active = to === '/'
            ? pathname === '/'
            : pathname.startsWith(to);

          if (isProfile && !isAuthenticated) {
            return (
              <button
                key={to}
                onClick={() => signIn()}
                className="flex flex-col items-center justify-center gap-1 flex-1 h-full text-dark-400"
              >
                <Icon size={20} strokeWidth={1.5} />
                <span className="text-[10px] leading-none">Sign In</span>
              </button>
            );
          }

          return (
            <Link
              key={to}
              to={to}
              className={`flex flex-col items-center justify-center gap-1 flex-1 h-full transition-colors ${
                active ? 'text-accent' : 'text-dark-400'
              }`}
            >
              <Icon size={20} strokeWidth={1.5} />
              <span className="text-[10px] leading-none">{label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
