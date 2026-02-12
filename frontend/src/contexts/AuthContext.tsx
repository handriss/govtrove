import { createContext, useContext, useEffect, useState, useRef, type ReactNode } from 'react';
import { useAuth } from '@workos-inc/authkit-react';
import { syncUser, getMe, type GovTroveUser } from '../services/api';

interface AuthContextValue {
  user: ReturnType<typeof useAuth>['user'];
  govtroveUser: GovTroveUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  signIn: ReturnType<typeof useAuth>['signIn'];
  signUp: ReturnType<typeof useAuth>['signUp'];
  signOut: ReturnType<typeof useAuth>['signOut'];
  getAccessToken: ReturnType<typeof useAuth>['getAccessToken'];
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const [govtroveUser, setGovtroveUser] = useState<GovTroveUser | null>(null);
  const [syncing, setSyncing] = useState(false);
  const syncedForUser = useRef<string | null>(null);

  useEffect(() => {
    const userId = auth.user?.id ?? null;

    if (!userId) {
      setGovtroveUser(null);
      syncedForUser.current = null;
      return;
    }

    if (syncedForUser.current === userId) return;
    syncedForUser.current = userId;

    let cancelled = false;
    setSyncing(true);

    (async () => {
      try {
        const token = await auth.getAccessToken();
        const synced = await syncUser(token, {
          email: auth.user!.email,
          first_name: auth.user!.firstName ?? '',
          last_name: auth.user!.lastName ?? '',
        });
        if (!cancelled) setGovtroveUser(synced);
      } catch {
        try {
          const token = await auth.getAccessToken();
          const me = await getMe(token);
          if (!cancelled) setGovtroveUser(me);
        } catch {
          // Failed to sync — user can still use the app
        }
      } finally {
        if (!cancelled) setSyncing(false);
      }
    })();

    return () => { cancelled = true; };
  }, [auth.user?.id]);

  return (
    <AuthContext.Provider
      value={{
        user: auth.user,
        govtroveUser,
        isLoading: auth.isLoading || syncing,
        isAuthenticated: !!auth.user,
        signIn: auth.signIn,
        signUp: auth.signUp,
        signOut: auth.signOut,
        getAccessToken: auth.getAccessToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAppAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAppAuth must be used within AuthProvider');
  return ctx;
}
