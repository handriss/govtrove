import { useState, useEffect } from 'react';
import { useParams, Navigate, Link } from 'react-router-dom';
import { Loader2, Gift, Check, AlertTriangle } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { redeemGiftCode } from '../services/api';

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
}

export default function RedeemPage() {
  const { code } = useParams<{ code: string }>();
  const { isAuthenticated, isLoading, signIn, getAccessToken, govtroveUser } = useAppAuth();
  const [status, setStatus] = useState<'idle' | 'redeeming' | 'success' | 'error'>('idle');
  const [grantedUntil, setGrantedUntil] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    if (!code || !isAuthenticated || isLoading || !govtroveUser) return;
    if (status !== 'idle') return;

    setStatus('redeeming');
    (async () => {
      try {
        const token = await getAccessToken();
        const result = await redeemGiftCode(token, code);
        setGrantedUntil(result.granted_until);
        setStatus('success');
      } catch (e) {
        setErrorMsg(e instanceof Error ? e.message : 'Failed to redeem gift code');
        setStatus('error');
      }
    })();
  }, [code, isAuthenticated, isLoading, govtroveUser, getAccessToken, status]);

  if (!code) return <Navigate to="/" replace />;

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 size={24} className="animate-spin text-dark-400" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="max-w-md text-center px-6">
          <Gift size={48} className="text-accent mx-auto mb-4" strokeWidth={1.5} />
          <h1 className="text-2xl font-semibold text-dark-50 mb-2">You've received a gift!</h1>
          <p className="text-dark-400 mb-6">Sign in or create an account to activate your free Pro access.</p>
          <button
            onClick={() => { sessionStorage.setItem('govtrove_redeem_code', code!); signIn(); }}
            className="px-6 py-3 text-sm font-semibold text-dark-950 rounded-lg bg-accent hover:bg-accent/90 transition-colors"
          >
            Sign in to redeem
          </button>
        </div>
      </div>
    );
  }

  if (status === 'redeeming') {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <Loader2 size={32} className="animate-spin text-accent mx-auto mb-3" />
          <p className="text-dark-400">Activating your gift code...</p>
        </div>
      </div>
    );
  }

  if (status === 'success') {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="max-w-md text-center px-6">
          <div className="w-16 h-16 rounded-full bg-green-500/10 border border-green-500/20 flex items-center justify-center mx-auto mb-4">
            <Check size={32} className="text-green-400" strokeWidth={1.5} />
          </div>
          <h1 className="text-2xl font-semibold text-dark-50 mb-2">Pro access activated!</h1>
          <p className="text-dark-400 mb-1">Your GovTrove Pro access is active until:</p>
          <p className="text-lg text-dark-100 font-medium mb-6">{formatDate(grantedUntil)}</p>
          <Link
            to="/"
            className="inline-flex px-6 py-3 text-sm font-semibold text-dark-950 rounded-lg bg-accent hover:bg-accent/90 transition-colors"
          >
            Start exploring
          </Link>
        </div>
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="max-w-md text-center px-6">
          <div className="w-16 h-16 rounded-full bg-red-500/10 border border-red-500/20 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle size={32} className="text-red-400" strokeWidth={1.5} />
          </div>
          <h1 className="text-2xl font-semibold text-dark-50 mb-2">Unable to redeem</h1>
          <p className="text-dark-400 mb-6">{errorMsg}</p>
          <Link
            to="/"
            className="inline-flex px-6 py-3 text-sm font-medium text-dark-300 border border-dark-700/50 rounded-lg hover:bg-dark-800/50 transition-colors"
          >
            Go to search
          </Link>
        </div>
      </div>
    );
  }

  return null;
}
