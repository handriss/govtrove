import { useEffect } from 'react';
import { useAppAuth } from '../contexts/AuthContext';

declare global {
  interface Window {
    Tawk_API?: {
      setAttributes?: (attrs: Record<string, string>, cb?: (err: unknown) => void) => void;
      onLoad?: () => void;
    };
    Tawk_LoadStart?: Date;
  }
}

const TAWK_PROPERTY_ID = import.meta.env.VITE_TAWK_PROPERTY_ID || '69a7180d7b02b21c3601dced/1jiqbbtmn';

export default function useTawk() {
  const { user } = useAppAuth();

  useEffect(() => {
    if (!TAWK_PROPERTY_ID) return;

    window.Tawk_API = window.Tawk_API || {};
    window.Tawk_LoadStart = new Date();

    const script = document.createElement('script');
    script.async = true;
    script.src = `https://embed.tawk.to/${TAWK_PROPERTY_ID}`;
    script.charset = 'UTF-8';
    script.setAttribute('crossorigin', '*');
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
  }, []);

  useEffect(() => {
    if (!TAWK_PROPERTY_ID || !user) return;

    const name = [user.firstName, user.lastName].filter(Boolean).join(' ');
    const attrs: Record<string, string> = {};
    if (name) attrs.name = name;
    if (user.email) attrs.email = user.email;
    if (!Object.keys(attrs).length) return;

    const setAttrs = () => {
      window.Tawk_API?.setAttributes?.(attrs);
    };

    if (window.Tawk_API?.setAttributes) {
      setAttrs();
    } else {
      const prev = window.Tawk_API?.onLoad;
      window.Tawk_API = window.Tawk_API || {};
      window.Tawk_API.onLoad = () => {
        prev?.();
        setAttrs();
      };
    }
  }, [user]);
}
