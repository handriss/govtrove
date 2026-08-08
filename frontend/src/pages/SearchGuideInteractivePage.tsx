import { useState, useEffect, useCallback, type ReactNode } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Search, TreePine, Tag, ShieldCheck, Building2, CalendarClock,
  FileText, ArrowRight, BookOpen,
} from 'lucide-react';
import NaicsTreeSelector from '../components/filters/NaicsTreeSelector';
import PscTreeSelector from '../components/filters/PscTreeSelector';
import SearchableDropdownFilter from '../components/filters/SearchableDropdownFilter';
import SimpleToggleFilter from '../components/filters/SimpleToggleFilter';
import AgencyFilter from '../components/filters/AgencyFilter';
import DeadlineFilter from '../components/filters/DeadlineFilter';
import {
  NOTICE_TYPE_OPTIONS,
  SET_ASIDE_LABELS,
} from '../components/filters/constants';
import { getFacetCounts } from '../services/api';
import type { FacetResult } from '../types/api';
import { usePageMeta } from '../hooks/usePageMeta';

const SET_ASIDE_OPTIONS = Object.entries(SET_ASIDE_LABELS).map(([value, label]) => ({
  value,
  label,
}));

interface DemoSectionProps {
  id: string;
  icon: typeof Search;
  title: string;
  description: string;
  children: ReactNode;
  searchUrl?: string;
}

function DemoSection({ id, icon: Icon, title, description, children, searchUrl }: DemoSectionProps) {
  return (
    <section
      id={id}
      className="rounded-xl border border-dark-700/40 bg-dark-800/20 p-5 hover:border-dark-600/50 transition-colors"
    >
      <div className="flex items-start gap-4">
        <div className="w-8 h-8 rounded-lg bg-dark-800/60 border border-dark-700/30 flex items-center justify-center shrink-0 mt-0.5">
          <Icon size={16} className="text-dark-300" strokeWidth={1.5} />
        </div>
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-medium text-dark-100 mb-1.5">{title}</h2>
          <p className="text-sm text-dark-400 leading-relaxed mb-4">{description}</p>
          <div className="rounded-lg border border-dark-700/30 bg-dark-900/40 p-4">
            {children}
          </div>
          {searchUrl && (
            <Link
              to={searchUrl}
              className="inline-flex items-center gap-1.5 mt-3 text-xs font-medium text-accent hover:text-accent/80 transition-colors"
            >
              Search with these filters
              <ArrowRight size={12} />
            </Link>
          )}
        </div>
      </div>
    </section>
  );
}

export default function SearchGuideInteractivePage() {
  usePageMeta({
    title: "Interactive Search Tutorial — Practice Boolean Contract Search | GovTrove",
    description: "Practice building federal contract searches step by step. Try boolean operators, quoted phrases, and filters against live SAM.gov opportunity data.",
    canonicalPath: '/guide/interactive',
  });

  const navigate = useNavigate();
  const [facets, setFacets] = useState<FacetResult['facets'] | null>(null);

  // Local filter states (not URL-synced)
  const [keyword, setKeyword] = useState('');
  const [naics, setNaics] = useState<string[]>([]);
  const [psc, setPsc] = useState<string[]>([]);
  const [setAside, setSetAside] = useState<string[]>([]);
  const [agency, setAgency] = useState<string[]>([]);
  const [deadline, setDeadline] = useState({ deadlinePreset: '', deadlineFrom: '', deadlineTo: '' });
  const [noticeType, setNoticeType] = useState<string[]>([]);

  useEffect(() => {
    getFacetCounts({}).then((r) => setFacets(r.facets)).catch(() => {});
  }, []);

  const buildUrl = useCallback((params: Record<string, string>) => {
    const sp = new URLSearchParams(params);
    return `/?${sp.toString()}`;
  }, []);

  const handleKeywordSearch = useCallback(() => {
    if (keyword.trim()) navigate(buildUrl({ q: keyword.trim() }));
  }, [keyword, navigate, buildUrl]);

  const setAsideOptions = SET_ASIDE_OPTIONS.map((o) => {
    const facet = facets?.set_aside?.find((f) => f.value === o.value);
    return { ...o, count: facet?.count ?? 0 };
  });

  const noticeTypeOptions = NOTICE_TYPE_OPTIONS.map((o) => {
    const facet = facets?.notice_type?.find((f) => f.value === o.value);
    return { ...o, count: facet?.count ?? 0 };
  });

  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-3xl mx-auto px-6 pt-10 pb-20">
        {/* Header */}
        <div className="mb-10">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-accent/10 border border-accent/20 flex items-center justify-center">
              <BookOpen size={20} className="text-accent" strokeWidth={1.5} />
            </div>
            <h1 className="text-2xl font-semibold text-dark-100">Interactive Search Guide</h1>
          </div>
          <p className="text-sm text-dark-400 leading-relaxed">
            Try each filter below with real data. Make your selections, then click "Search with these filters" to jump to search results.
          </p>
          <div className="mt-4 flex flex-col gap-2">
            <Link
              to="/guide"
              className="inline-flex items-center gap-1.5 text-xs text-accent hover:text-accent/80 transition-colors"
            >
              Prefer a quick overview? See the static guide
              <ArrowRight size={12} />
            </Link>
            <Link
              to="/glossary"
              className="inline-flex items-center gap-1.5 text-xs text-dark-500 hover:text-dark-300 transition-colors"
            >
              Not sure what a term means? See the Glossary
              <ArrowRight size={12} />
            </Link>
          </div>
        </div>

        {/* Sections */}
        <div className="space-y-6">
          {/* Keyword Search */}
          <DemoSection
            id="keyword"
            icon={Search}
            title="Keyword Search"
            description='Searches title, description, and solicitation number. Several words means all of them must appear, so each extra word narrows the results. Use "quotes" for an exact phrase, OR for either word, and -word to exclude. Press "/" anywhere to focus the search bar.'
            searchUrl={keyword.trim() ? buildUrl({ q: keyword.trim() }) : undefined}
          >
            <div className="flex gap-2">
              <input
                type="text"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleKeywordSearch(); }}
                placeholder="Try: cybersecurity, IT modernization, construction..."
                className="flex-1 px-3 py-2 text-sm bg-dark-800/50 border border-dark-700/50 rounded-lg
                           text-dark-100 placeholder-dark-500 focus:outline-none focus:border-accent/50"
              />
              <button
                onClick={handleKeywordSearch}
                disabled={!keyword.trim()}
                className="px-3 py-2 text-sm bg-accent/20 text-accent rounded-lg hover:bg-accent/30
                           disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              >
                Search
              </button>
            </div>
          </DemoSection>

          {/* NAICS Codes */}
          <DemoSection
            id="naics"
            icon={TreePine}
            title="NAICS Codes"
            description="Filter by industry classification. Select parent categories to include all sub-codes."
            searchUrl={naics.length ? buildUrl({ naics: naics.join(',') }) : undefined}
          >
            <NaicsTreeSelector
              selected={naics}
              onChange={setNaics}
              facets={facets?.naics}
              inline
            />
          </DemoSection>

          {/* PSC Codes */}
          <DemoSection
            id="psc"
            icon={Tag}
            title="PSC Codes"
            description="Product and Service Codes classify what the government is buying."
            searchUrl={psc.length ? buildUrl({ psc: psc.join(',') }) : undefined}
          >
            <PscTreeSelector
              selected={psc}
              onChange={setPsc}
              facets={facets?.psc}
              inline
            />
          </DemoSection>

          {/* Set-Aside */}
          <DemoSection
            id="set-aside"
            icon={ShieldCheck}
            title="Set-Aside Type"
            description="Find opportunities reserved for specific small-business certifications."
            searchUrl={setAside.length ? buildUrl({ set_aside: setAside.join(',') }) : undefined}
          >
            <SearchableDropdownFilter
              label="Set-Aside"
              options={setAsideOptions}
              selected={setAside}
              onSelectionChange={setSetAside}
              searchPlaceholder="Search set-asides..."
              showSelectedValues
            />
          </DemoSection>

          {/* Agency */}
          <DemoSection
            id="agency"
            icon={Building2}
            title="Agency"
            description="Narrow results to a specific department or sub-agency."
            searchUrl={agency.length ? buildUrl({ agency: agency.join(',') }) : undefined}
          >
            <AgencyFilter
              selected={agency}
              onChange={setAgency}
              showSelectedValues
            />
          </DemoSection>

          {/* Deadline */}
          <DemoSection
            id="deadline"
            icon={CalendarClock}
            title="Response Deadline"
            description="Use presets or pick a custom date range."
            searchUrl={
              deadline.deadlinePreset
                ? buildUrl({ deadline: deadline.deadlinePreset })
                : deadline.deadlineFrom || deadline.deadlineTo
                  ? buildUrl({
                      ...(deadline.deadlineFrom ? { deadline_from: deadline.deadlineFrom } : {}),
                      ...(deadline.deadlineTo ? { deadline_to: deadline.deadlineTo } : {}),
                    })
                  : undefined
            }
          >
            <DeadlineFilter
              deadlinePreset={deadline.deadlinePreset}
              deadlineFrom={deadline.deadlineFrom}
              deadlineTo={deadline.deadlineTo}
              onChange={setDeadline}
            />
          </DemoSection>

          {/* Notice Type */}
          <DemoSection
            id="notice-type"
            icon={FileText}
            title="Notice Type"
            description="Filter by opportunity type: Solicitations, Presolicitations, Sources Sought, and more."
            searchUrl={noticeType.length ? buildUrl({ type: noticeType.join(',') }) : undefined}
          >
            <SimpleToggleFilter
              label="Notice Type"
              options={noticeTypeOptions}
              selected={noticeType}
              onSelectionChange={setNoticeType}
              showSelectedValues
            />
          </DemoSection>
        </div>

        {/* Footer */}
        <div className="mt-12 text-center">
          <Link
            to="/"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-accent/10 border border-accent/30 text-sm text-accent hover:bg-accent/20 transition-colors"
          >
            <Search size={14} />
            Start Searching
          </Link>
        </div>
      </div>
    </div>
  );
}
