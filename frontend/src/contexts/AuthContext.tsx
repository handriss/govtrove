import { createContext, useContext, useCallback, useEffect, useState, useRef, type ReactNode } from 'react';
import { useAuth } from '@workos-inc/authkit-react';
import { usePostHog } from '@posthog/react';
import { syncUser, getMe, AUTH_ERROR_EVENT, type GovTroveUser } from '../services/api';
import { trackSignIn, trackSignUp } from '../lib/analytics';
import { capturedUTM } from '../hooks/useUTMCapture';

interface AuthContextValue {
  user: ReturnType<typeof useAuth>['user'];
  govtroveUser: GovTroveUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  signIn: ReturnType<typeof useAuth>['signIn'];
  signUp: ReturnType<typeof useAuth>['signUp'];
  signOut: () => void;
  getAccessToken: ReturnType<typeof useAuth>['getAccessToken'];
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const posthog = usePostHog();
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
        if (!cancelled) {
          setGovtroveUser(synced);
          const utmOnce: Record<string, string> = {};
          if (capturedUTM.source) utmOnce.$initial_utm_source = capturedUTM.source;
          if (capturedUTM.medium) utmOnce.$initial_utm_medium = capturedUTM.medium;
          if (capturedUTM.campaign) utmOnce.$initial_utm_campaign = capturedUTM.campaign;
          posthog?.identify(auth.user!.id, {
            email: auth.user!.email,
            name: `${auth.user!.firstName ?? ''} ${auth.user!.lastName ?? ''}`.trim(),
            plan: synced.plan,
            created_at: synced.created_at,
          }, utmOnce);
          const isNew = Date.now() - new Date(synced.created_at).getTime() < 60_000;
          if (isNew) trackSignUp(posthog); else trackSignIn(posthog);
        }
      } catch {
        try {
          const token = await auth.getAccessToken();
          const me = await getMe(token);
          if (!cancelled) {
            setGovtroveUser(me);
            const utmOnceFb: Record<string, string> = {};
            if (capturedUTM.source) utmOnceFb.$initial_utm_source = capturedUTM.source;
            if (capturedUTM.medium) utmOnceFb.$initial_utm_medium = capturedUTM.medium;
            if (capturedUTM.campaign) utmOnceFb.$initial_utm_campaign = capturedUTM.campaign;
            posthog?.identify(auth.user!.id, {
              email: auth.user!.email,
              name: `${auth.user!.firstName ?? ''} ${auth.user!.lastName ?? ''}`.trim(),
              plan: me.plan,
              created_at: me.created_at,
            }, utmOnceFb);
          }
        } catch {
          // Failed to sync — user can still use the app
        }
      } finally {
        if (!cancelled) setSyncing(false);
      }
    })();

    return () => { cancelled = true; };
  }, [auth.user?.id]);

  const signOut = useCallback(() => {
    posthog?.reset();
    auth.signOut({ returnTo: window.location.origin });
  }, [auth.signOut, posthog]);

  const handleAuthError = useCallback(() => {
    setGovtroveUser(null);
    syncedForUser.current = null;
    signOut();
  }, [signOut]);

  useEffect(() => {
    window.addEventListener(AUTH_ERROR_EVENT, handleAuthError);
    return () => window.removeEventListener(AUTH_ERROR_EVENT, handleAuthError);
  }, [handleAuthError]);

  return (
    <AuthContext.Provider
      value={{
        user: auth.user,
        govtroveUser,
        isLoading: auth.isLoading || syncing,
        isAuthenticated: !!auth.user,
        signIn: auth.signIn,
        signUp: auth.signUp,
        signOut,
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
