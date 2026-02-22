import { useState, useRef, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { LogIn, LogOut, User, Clock, Star } from 'lucide-react';
import { useAppAuth } from '../../contexts/AuthContext';

const navLinks = [
  { to: '/whats-new', label: "What's New", icon: Clock },
  { to: '/saved', label: 'Saved', icon: Star },
] as const;

export default function DesktopNav({ hideLogo }: { hideLogo: boolean }) {
  const { pathname } = useLocation();
  const { user, isLoading, isAuthenticated, signIn, signOut } = useAppAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    }
    if (menuOpen) document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [menuOpen]);

  if (pathname === '/callback') return null;

  const initial = user ? (user.firstName?.[0] || user.email[0] || '?').toUpperCase() : '';

  return (
    <header className="sticky top-0 z-20 hidden md:block border-b border-dark-800/50 bg-dark-950/80 backdrop-blur-md">
      <div className="max-w-[1400px] mx-auto px-6 h-14 flex items-center">
        {/* Logo */}
        <Link
          to="/"
          className={`text-lg font-semibold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-dark-50 to-dark-200 transition-opacity duration-300 ${
            hideLogo ? 'opacity-0 pointer-events-none' : 'opacity-100'
          }`}
        >
          GovTrove
        </Link>

        <div className="flex-1" />

        {/* Nav links */}
        <nav className="flex items-center gap-1">
          {navLinks.map(({ to, label, icon: Icon }) => {
            const active = pathname.startsWith(to);
            return (
              <Link
                key={to}
                to={to}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors ${
                  active
                    ? 'text-accent bg-accent/10'
                    : 'text-dark-400 hover:text-dark-200'
                }`}
              >
                <Icon size={15} strokeWidth={1.5} />
                {label}
              </Link>
            );
          })}
        </nav>

        {/* Divider */}
        <div className="w-px h-5 bg-dark-700/50 mx-3" />

        {/* Auth area */}
        {isLoading ? (
          <div className="w-8 h-8 rounded-full bg-dark-800/50 animate-pulse" />
        ) : !isAuthenticated || !user ? (
          <button
            onClick={() => signIn()}
            className="flex items-center gap-2 px-3 py-1.5 text-sm text-dark-300 hover:text-dark-50
                       border border-dark-700/50 hover:border-dark-600/50 rounded-lg
                       bg-dark-800/30 hover:bg-dark-800/50 transition-all duration-200"
          >
            <LogIn size={14} strokeWidth={1.5} />
            Sign In
          </button>
        ) : (
          <div className="relative" ref={menuRef}>
            <button
              onClick={() => setMenuOpen(!menuOpen)}
              className="w-8 h-8 rounded-full bg-accent/20 border border-accent/30 text-accent
                         text-sm font-medium flex items-center justify-center
                         hover:bg-accent/30 transition-all duration-200"
            >
              {initial}
            </button>

            {menuOpen && (
              <div className="absolute right-0 top-full mt-2 w-56 bg-dark-900 border border-dark-700/50 rounded-xl shadow-xl z-50 overflow-hidden">
                <div className="px-4 py-3 border-b border-dark-800/50">
                  <p className="text-sm font-medium text-dark-100 truncate">
                    {user.firstName} {user.lastName}
                  </p>
                  <p className="text-xs text-dark-500 truncate">{user.email}</p>
                </div>
                <div className="py-1">
                  <Link
                    to="/profile"
                    onClick={() => setMenuOpen(false)}
                    className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-dark-300 hover:text-dark-50 hover:bg-dark-800/50 transition-colors"
                  >
                    <User size={14} strokeWidth={1.5} />
                    Profile
                  </Link>
                  <button
                    onClick={() => { setMenuOpen(false); signOut(); }}
                    className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-dark-300 hover:text-dark-50 hover:bg-dark-800/50 transition-colors"
                  >
                    <LogOut size={14} strokeWidth={1.5} />
                    Sign Out
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </header>
  );
}
