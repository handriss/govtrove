import { useState, useRef, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { LogIn, LogOut, User, Star, Bell } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';

export default function AuthButton() {
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

  if (isLoading) {
    return (
      <div className="w-8 h-8 rounded-full bg-dark-800/50 animate-pulse" />
    );
  }

  if (!isAuthenticated || !user) {
    return (
      <button
        onClick={() => signIn()}
        className="flex items-center gap-2 px-3 py-1.5 text-sm text-dark-300 hover:text-dark-50
                   border border-dark-700/50 hover:border-dark-600/50 rounded-lg
                   bg-dark-800/30 hover:bg-dark-800/50 transition-all duration-200"
      >
        <LogIn size={14} strokeWidth={1.5} />
        <span className="hidden sm:inline">Sign In</span>
      </button>
    );
  }

  const initial = (user.firstName?.[0] || user.email[0] || '?').toUpperCase();

  return (
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
            <Link
              to="/saved"
              onClick={() => setMenuOpen(false)}
              className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-dark-300 hover:text-dark-50 hover:bg-dark-800/50 transition-colors"
            >
              <Star size={14} strokeWidth={1.5} />
              Saved
            </Link>
            <Link
              to="/notifications"
              onClick={() => setMenuOpen(false)}
              className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-dark-300 hover:text-dark-50 hover:bg-dark-800/50 transition-colors"
            >
              <Bell size={14} strokeWidth={1.5} />
              Notifications
              <span className="ml-auto text-[10px] uppercase tracking-wider text-dark-600 bg-dark-800/50 px-1.5 py-0.5 rounded">Soon</span>
            </Link>
            <button
              onClick={() => {
                setMenuOpen(false);
                signOut();
              }}
              className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-dark-300 hover:text-dark-50 hover:bg-dark-800/50 transition-colors"
            >
              <LogOut size={14} strokeWidth={1.5} />
              Sign Out
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
