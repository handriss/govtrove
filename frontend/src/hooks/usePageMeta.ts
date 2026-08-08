import { useEffect } from 'react';

const SITE_URL = 'https://app.govtrove.com';

const DEFAULTS = {
  title: 'GovTrove — Search Federal Contract Opportunities | SAM.gov Alternative',
  description:
    'Find government contracts faster. Search 100,000+ federal opportunities from SAM.gov with boolean search, smart filters, deadline tracking, and set-aside matching.',
} as const;

interface PageMeta {
  title: string;
  description: string;
  /** Path only, e.g. "/glossary". Omit for routes that should not advertise a canonical. */
  canonicalPath?: string;
}

function setMeta(selector: string, attr: string, value: string) {
  const el = document.head.querySelector<HTMLMetaElement | HTMLLinkElement>(selector);
  if (el) el.setAttribute(attr, value);
}

/**
 * index.html ships one hardcoded title/description/canonical, so every SPA route used to
 * declare itself a duplicate of the homepage — Google folded them together and left
 * /guide, /glossary and /whats-new unindexed. Each indexable route sets its own here.
 */
export function usePageMeta({ title, description, canonicalPath }: PageMeta) {
  useEffect(() => {
    document.title = title;
    setMeta('meta[name="description"]', 'content', description);
    setMeta('meta[property="og:title"]', 'content', title);
    setMeta('meta[property="og:description"]', 'content', description);
    setMeta('meta[name="twitter:title"]', 'content', title);
    setMeta('meta[name="twitter:description"]', 'content', description);

    if (canonicalPath) {
      const url = `${SITE_URL}${canonicalPath}`;
      setMeta('link[rel="canonical"]', 'href', url);
      setMeta('meta[property="og:url"]', 'content', url);
    }

    return () => {
      document.title = DEFAULTS.title;
      setMeta('meta[name="description"]', 'content', DEFAULTS.description);
      setMeta('meta[property="og:title"]', 'content', DEFAULTS.title);
      setMeta('meta[property="og:description"]', 'content', DEFAULTS.description);
      setMeta('meta[name="twitter:title"]', 'content', DEFAULTS.title);
      setMeta('meta[name="twitter:description"]', 'content', DEFAULTS.description);
      setMeta('link[rel="canonical"]', 'href', SITE_URL);
      setMeta('meta[property="og:url"]', 'content', SITE_URL);
    };
  }, [title, description, canonicalPath]);
}
