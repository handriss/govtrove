import { Link } from 'react-router-dom';
import {
  Globe, TreePine, Tag, FileText, ShieldCheck, Building2,
  Hash, CalendarClock, ToggleRight, MapPin, Trophy,
  BookText, ExternalLink, ArrowRight, Search,
} from 'lucide-react';

interface GlossaryEntry {
  id: string;
  icon: typeof Globe;
  title: string;
  content: string;
  externalUrl?: string;
  externalLabel?: string;
  guideLink?: string;
  guideLinkLabel?: string;
}

const entries: GlossaryEntry[] = [
  {
    id: 'sam-gov',
    icon: Globe,
    title: 'SAM.gov',
    content:
      'The System for Award Management (SAM.gov) is the official U.S. government website for federal procurement. It\'s where agencies post contract opportunities, and where businesses register to do business with the government. SAM.gov is free and open to the public. GovTrove pulls all of its opportunity data from SAM.gov\'s public API.',
    externalUrl: 'https://sam.gov',
    externalLabel: 'Visit SAM.gov',
  },
  {
    id: 'naics',
    icon: TreePine,
    title: 'NAICS Codes',
    content:
      'The North American Industry Classification System (NAICS) uses 2- to 6-digit codes to classify businesses by industry. Codes are hierarchical: a 2-digit code represents a broad sector (e.g., 54 = Professional Services), while a 6-digit code is specific (e.g., 541512 = Computer Systems Design Services). Agencies assign NAICS codes to opportunities so the right businesses can find them. Most small businesses know their primary NAICS codes — if you don\'t, search by keyword first and note which codes appear on relevant opportunities.',
    externalUrl: 'https://www.census.gov/naics/',
    externalLabel: 'NAICS reference at Census.gov',
    guideLink: '/guide#naics',
    guideLinkLabel: 'How to filter by NAICS',
  },
  {
    id: 'psc',
    icon: Tag,
    title: 'Product Service Codes (PSC)',
    content:
      'Product and Service Codes classify what the government is buying, while NAICS classifies who can provide it. PSC codes use a letter-number system: letter prefixes indicate service categories (e.g., D = IT Services), number prefixes indicate products (e.g., 7010 = Computer Equipment). Example: D302 = IT Systems Development Services. Using PSC alongside NAICS gives you the most precise results.',
    externalUrl: 'https://www.acquisition.gov/psc-manual',
    externalLabel: 'PSC manual at Acquisition.gov',
    guideLink: '/guide#psc',
    guideLinkLabel: 'How to filter by PSC',
  },
  {
    id: 'notice-types',
    icon: FileText,
    title: 'Notice Types',
    content:
      'Every SAM.gov opportunity has a notice type that indicates its stage in the procurement process:\n\n' +
      '\u2022 Presolicitation — Advance notice that an agency plans to issue a solicitation. Not yet accepting bids, but a signal to start preparing.\n' +
      '\u2022 Solicitation — Actively accepting proposals. This is what most people are looking for.\n' +
      '\u2022 Combined Synopsis/Solicitation — A combined announcement and request for proposals, common for simpler procurements.\n' +
      '\u2022 Sources Sought — Market research by the agency. Not a commitment to buy, but responding helps shape future solicitations.\n' +
      '\u2022 Special Notice — Informational only, no procurement action required (e.g., conference announcements, policy updates).\n' +
      '\u2022 Award Notice — A contract has been awarded. Useful for competitive intelligence.\n' +
      '\u2022 Modification/Amendment/Cancel — Changes to existing notices: revised deadlines, updated requirements, or cancellations.\n' +
      '\u2022 Intent to Bundle — Agency is consolidating multiple requirements into a single contract. Small businesses should pay attention.\n' +
      '\u2022 Sale of Surplus Property — Government selling surplus assets.\n' +
      '\u2022 Fair Opportunity / Limited Sources Justification — Justification for limiting competition, often for task orders under existing contracts.',
    externalUrl: 'https://sam.gov/content/opportunities',
    externalLabel: 'SAM.gov opportunities overview',
    guideLink: '/guide#notice-type',
    guideLinkLabel: 'How to filter by notice type',
  },
  {
    id: 'set-asides',
    icon: ShieldCheck,
    title: 'Set-Aside Types',
    content:
      'Set-asides reserve contract opportunities for specific categories of small businesses. The SBA (Small Business Administration) manages the certification programs:\n\n' +
      '\u2022 Small Business (SBP) — General small business set-aside, based on size standards for your NAICS code.\n' +
      '\u2022 8(a) — For businesses in the SBA\'s 8(a) Business Development program, which helps small disadvantaged businesses compete.\n' +
      '\u2022 HUBZone — For businesses in Historically Underutilized Business Zones.\n' +
      '\u2022 SDVOSB — Service-Disabled Veteran-Owned Small Business.\n' +
      '\u2022 WOSB — Women-Owned Small Business.\n' +
      '\u2022 EDWOSB — Economically Disadvantaged Women-Owned Small Business.\n' +
      '\u2022 VOSB — Veteran-Owned Small Business.\n\n' +
      '"Sole Source" variants (e.g., 8(a) Sole Source, HUBZone Sole Source) mean the agency intends to award directly to a single qualified business without full competition. If your certifications match, filtering by set-aside is one of the best ways to find relevant opportunities.',
    externalUrl: 'https://www.sba.gov/federal-contracting/contracting-assistance-programs',
    externalLabel: 'SBA contracting assistance programs',
    guideLink: '/guide#set-aside',
    guideLinkLabel: 'How to filter by set-aside',
  },
  {
    id: 'agencies',
    icon: Building2,
    title: 'Federal Agencies',
    content:
      'Federal procurement is organized by department hierarchy. Top-level departments (e.g., DEPT OF DEFENSE) contain sub-agencies (e.g., Army, Navy, Air Force), which may have further subdivisions. SAM.gov represents this hierarchy using dot-separated codes. When you filter by a parent agency, GovTrove includes all of its sub-agencies automatically.',
    externalUrl: 'https://www.usa.gov/agency-index',
    externalLabel: 'Federal agency index at USA.gov',
    guideLink: '/guide#agency',
    guideLinkLabel: 'How to filter by agency',
  },
  {
    id: 'solicitation-number',
    icon: Hash,
    title: 'Solicitation Number',
    content:
      'A unique identifier assigned by the contracting agency to each procurement action. The format varies by agency (e.g., W912DY-25-R-0001 for Army Corps of Engineers, FA8532-25-Q-0042 for Air Force). Use this to track a specific opportunity or find related modifications and amendments.',
  },
  {
    id: 'response-deadline',
    icon: CalendarClock,
    title: 'Response Deadline',
    content:
      'The date and time by which proposals or responses must be submitted. After this deadline, the opportunity is considered "closed" but remains in the system as a historical record. Pay attention to time zones — deadlines are typically in the contracting office\'s local time. GovTrove displays all times in UTC.',
    guideLink: '/guide#deadline',
    guideLinkLabel: 'How to filter by deadline',
  },
  {
    id: 'active-vs-archived',
    icon: ToggleRight,
    title: 'Active vs. Archived',
    content:
      'Active opportunities have a response deadline that hasn\'t passed yet (or no deadline set). Archived opportunities have passed their deadline or been explicitly archived by the agency. GovTrove defaults to showing active opportunities only. Toggle "Active Only" off to browse historical data — useful for market research and understanding past award patterns.',
    guideLink: '/guide#active-only',
    guideLinkLabel: 'How to toggle active/archived',
  },
  {
    id: 'place-of-performance',
    icon: MapPin,
    title: 'Place of Performance',
    content:
      'Where the contract work will physically be performed — not where the agency is located. This can be a specific address, city, state, or even "worldwide." If your business operates in a particular region, filtering by place of performance helps find local opportunities.',
    guideLink: '/guide#more-filters',
    guideLinkLabel: 'How to filter by location',
  },
  {
    id: 'award',
    icon: Trophy,
    title: 'Award Information',
    content:
      'Once a contract is awarded, the opportunity record is updated with award details: the award number, dollar amount, awardee name, and their UEI. Award notices are valuable for competitive intelligence — you can see who is winning contracts in your space, at what price points, and from which agencies.',
  },
];

export default function GlossaryPage() {
  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-3xl mx-auto px-6 pt-10 pb-20">
        {/* Header */}
        <div className="mb-10">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-accent/10 border border-accent/20 flex items-center justify-center">
              <BookText size={20} className="text-accent" strokeWidth={1.5} />
            </div>
            <h1 className="text-2xl font-semibold text-dark-100">Glossary</h1>
          </div>
          <p className="text-sm text-dark-400 leading-relaxed">
            Government contracting has its own vocabulary. This glossary explains the key terms you'll encounter when searching for federal contract opportunities on GovTrove.
          </p>
          <div className="mt-4">
            <Link
              to="/guide"
              className="inline-flex items-center gap-1.5 text-xs text-accent hover:text-accent/80 transition-colors"
            >
              Want to learn how to use the search filters? See the Search Guide
              <ArrowRight size={12} />
            </Link>
          </div>
        </div>

        {/* Sections */}
        <div className="space-y-6">
          {entries.map((entry) => {
            const Icon = entry.icon;
            return (
              <section
                key={entry.id}
                id={entry.id}
                className="group rounded-xl border border-dark-700/40 bg-dark-800/20 p-5 hover:border-dark-600/50 transition-colors"
              >
                <div className="flex items-start gap-4">
                  <div className="w-8 h-8 rounded-lg bg-dark-800/60 border border-dark-700/30 flex items-center justify-center shrink-0 mt-0.5">
                    <Icon size={16} className="text-dark-300" strokeWidth={1.5} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <h2 className="text-sm font-medium text-dark-100 mb-1.5">{entry.title}</h2>
                    <p className="text-sm text-dark-400 leading-relaxed whitespace-pre-line">{entry.content}</p>
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 mt-3">
                      {entry.externalUrl && (
                        <a
                          href={entry.externalUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1.5 text-xs text-dark-500 hover:text-dark-300 transition-colors"
                        >
                          <ExternalLink size={11} />
                          {entry.externalLabel}
                        </a>
                      )}
                      {entry.guideLink && (
                        <Link
                          to={entry.guideLink}
                          className="inline-flex items-center gap-1.5 text-xs font-medium text-accent hover:text-accent/80 transition-colors"
                        >
                          {entry.guideLinkLabel}
                          <ArrowRight size={12} />
                        </Link>
                      )}
                    </div>
                  </div>
                </div>
              </section>
            );
          })}
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
