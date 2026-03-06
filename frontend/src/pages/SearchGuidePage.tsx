import { Link } from 'react-router-dom';
import {
  Search, TreePine, Tag, ShieldCheck, Building2, CalendarClock,
  FileText, ToggleRight, SlidersHorizontal, ArrowUpDown, Bookmark,
  Layers, ArrowRight, BookOpen, Star,
} from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';

interface GuideSection {
  id: string;
  icon: typeof Search;
  title: string;
  description: string;
  tryLink?: string;
  tryLabel?: string;
}

const sections: GuideSection[] = [
  {
    id: 'keyword',
    icon: Search,
    title: 'Keyword Search',
    description:
      'Search by title, description, or solicitation number. Tip: press "/" anywhere on the page to focus the search bar instantly.',
    tryLink: '/?q=cybersecurity',
    tryLabel: 'Try "cybersecurity"',
  },
  {
    id: 'naics',
    icon: TreePine,
    title: 'NAICS Codes',
    description:
      'Filter by industry using the North American Industry Classification System. Select parent categories to include all sub-codes, or pick specific 6-digit codes.',
    tryLink: '/?naics=541512',
    tryLabel: 'Try 541512 — IT Consulting',
  },
  {
    id: 'psc',
    icon: Tag,
    title: 'PSC Codes',
    description:
      'Product and Service Codes classify what the government is buying. Browse the tree or search by code/keyword.',
    tryLink: '/?psc=D302',
    tryLabel: 'Try D302 — IT Systems Dev',
  },
  {
    id: 'set-aside',
    icon: ShieldCheck,
    title: 'Set-Aside Type',
    description:
      'Filter for small-business set-asides: 8(a), HUBZone, SDVOSB, WOSB, and more. Great for finding opportunities reserved for your certification.',
    tryLink: '/?set_aside=SBA',
    tryLabel: 'Try SBA set-asides',
  },
  {
    id: 'agency',
    icon: Building2,
    title: 'Agency',
    description:
      'Narrow results to a specific department or sub-agency. Start typing to search across all federal agencies.',
    tryLink: '/?agency=DEPT+OF+DEFENSE',
    tryLabel: 'Try Dept of Defense',
  },
  {
    id: 'deadline',
    icon: CalendarClock,
    title: 'Response Deadline',
    description:
      'Use preset ranges (7, 14, 30, 60 days, this quarter) or pick a custom date range. Only shows opportunities whose deadline falls within the window.',
    tryLink: '/?deadline=14',
    tryLabel: 'Try next 14 days',
  },
  {
    id: 'notice-type',
    icon: FileText,
    title: 'Notice Type',
    description:
      'Filter by opportunity type: Solicitations, Presolicitations, Combined Synopsis, Sources Sought, Special Notices, Award Notices, and more. Defaults exclude awards and modifications.',
    tryLink: '/?type=Presolicitation,Combined+Synopsis%2FSolicitation',
    tryLabel: 'Try Presol + Combined',
  },
  {
    id: 'active-only',
    icon: ToggleRight,
    title: 'Active Only',
    description:
      'Enabled by default — hides opportunities whose response deadline has passed. Turn it off to browse historical data.',
  },
  {
    id: 'more-filters',
    icon: SlidersHorizontal,
    title: 'More Filters',
    description:
      'Additional filters include Posted Date range, Solicitation Number lookup, and Place of Performance (state/city).',
  },
  {
    id: 'sorting',
    icon: ArrowUpDown,
    title: 'Sorting',
    description:
      'Sort results by Relevance (keyword match quality), Deadline (soonest first), Posted Date (newest first), or Agency (alphabetical). Your preference is remembered across sessions.',
  },
  {
    id: 'saved-searches',
    icon: Bookmark,
    title: 'Saved Searches',
    description:
      'Save filter combinations you use often. Saved searches appear as quick-access pills above your results so you can re-run them with one click. Saving searches is free — create an account to get started. Pro users can also enable email alerts on saved searches to get notified when new matches appear.',
  },
  {
    id: 'saved-opportunities',
    icon: Star,
    title: 'Saved Opportunities',
    description:
      'Bookmark individual opportunities by clicking the star icon on any search result or detail page. Access all your saved opportunities from the Saved page in the sidebar. You can also add private notes to any saved opportunity to track your bid progress or key details.',
  },
  {
    id: 'combining',
    icon: Layers,
    title: 'Combining Filters',
    description:
      'Stack multiple filters for precision. Example: find IT services opportunities set aside for small businesses closing within 30 days.',
    tryLink: '/?q=IT+services&set_aside=SBA&deadline=30',
    tryLabel: 'Try combined filters',
  },
];

export default function SearchGuidePage() {
  const { isAuthenticated, signUp } = useAppAuth();

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
            <h1 className="text-2xl font-semibold text-dark-100">Search Guide</h1>
          </div>
          <p className="text-sm text-dark-400 leading-relaxed">
            GovTrove helps you find federal contract opportunities from SAM.gov.
            Use the filters below to narrow results to exactly what you need. Click any "Try it" link to see it in action.
          </p>
          <div className="mt-4 flex flex-col gap-2">
            <Link
              to="/guide/interactive"
              className="inline-flex items-center gap-1.5 text-xs text-accent hover:text-accent/80 transition-colors"
            >
              Want hands-on demos? Try the interactive guide
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
          {sections.map((s) => {
            const Icon = s.icon;
            return (
              <section
                key={s.id}
                id={s.id}
                className="group rounded-xl border border-dark-700/40 bg-dark-800/20 p-5 hover:border-dark-600/50 transition-colors"
              >
                <div className="flex items-start gap-4">
                  <div className="w-8 h-8 rounded-lg bg-dark-800/60 border border-dark-700/30 flex items-center justify-center shrink-0 mt-0.5">
                    <Icon size={16} className="text-dark-300" strokeWidth={1.5} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <h2 className="text-sm font-medium text-dark-100 mb-1.5">{s.title}</h2>
                    <p className="text-sm text-dark-400 leading-relaxed">{s.description}</p>
                    {s.tryLink && (
                      <Link
                        to={s.tryLink}
                        className="inline-flex items-center gap-1.5 mt-3 text-xs font-medium text-accent hover:text-accent/80 transition-colors"
                      >
                        {s.tryLabel}
                        <ArrowRight size={12} />
                      </Link>
                    )}
                  </div>
                </div>
              </section>
            );
          })}
        </div>

        {/* CTA for anonymous users */}
        {!isAuthenticated && (
          <div className="mt-10 rounded-xl border border-accent/20 bg-accent/5 p-6 text-center">
            <h3 className="text-base font-semibold text-dark-100 mb-2">
              Ready to save your searches?
            </h3>
            <p className="text-sm text-dark-400 mb-4 max-w-md mx-auto">
              Create a free account to bookmark opportunities, save searches, and pick up right where you left off.
            </p>
            <button
              onClick={() => signUp()}
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent hover:bg-accent-hover text-white font-medium rounded-xl transition-all duration-200 text-sm"
            >
              Sign Up Free
            </button>
          </div>
        )}

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
