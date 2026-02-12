import { ChevronLeft, ChevronRight, Loader2 } from 'lucide-react';
import ResultRow from './ResultRow';
import ResultCard from './ResultCard';
import NoResults from './NoResults';
import type { OpportunityListItem } from '../types/api';

interface ResultsListProps {
  results: OpportunityListItem[];
  page: number;
  totalPages: number;
  loading: boolean;
  query?: string;
  onPageChange: (page: number) => void;
}

export default function ResultsList({
  results,
  page,
  totalPages,
  loading,
  query,
  onPageChange,
}: ResultsListProps) {
  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-accent/70" strokeWidth={1.5} />
          <span className="text-sm text-dark-400">Searching...</span>
        </div>
      </div>
    );
  }

  if (results.length === 0) {
    return <NoResults query={query} />;
  }

  return (
    <div>
      {/* Mobile: card layout */}
      <div className="md:hidden space-y-3">
        {results.map((opp, index) => (
          <ResultCard key={opp.id} opportunity={opp} index={index} />
        ))}
      </div>

      {/* Desktop: table layout */}
      <div className="hidden md:block rounded-xl border border-dark-800/50 overflow-hidden bg-dark-900/30 backdrop-blur-sm">
        <table className="w-full">
          <thead>
            <tr className="text-[11px] text-dark-400 uppercase tracking-wider bg-dark-850/50">
              <th className="px-4 py-3 w-10"></th>
              <th className="px-4 py-3 text-left font-medium">Title</th>
              <th className="px-4 py-3 text-left font-medium">Agency</th>
              <th className="px-4 py-3 text-left font-medium">Set-Aside</th>
              <th className="px-4 py-3 text-left font-medium">Due</th>
              <th className="px-4 py-3 text-left font-medium">NAICS</th>
              <th className="px-4 py-3 text-left font-medium">State</th>
              <th className="px-4 py-3 w-12"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-dark-800/30">
            {results.map((opp, index) => (
              <ResultRow key={opp.id} opportunity={opp} index={index} />
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 mt-6">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="p-2 rounded-lg border border-dark-700/50 bg-dark-900/50
                       text-dark-400 hover:text-dark-100 hover:border-dark-600/50 hover:bg-dark-800/50
                       disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:bg-dark-900/50
                       transition-all duration-200"
          >
            <ChevronLeft size={18} strokeWidth={1.5} />
          </button>
          <span className="text-sm text-dark-400 px-4 tabular-nums">
            <span className="text-dark-200">{page}</span>
            <span className="mx-2 text-dark-600">/</span>
            <span>{totalPages}</span>
          </span>
          <button
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="p-2 rounded-lg border border-dark-700/50 bg-dark-900/50
                       text-dark-400 hover:text-dark-100 hover:border-dark-600/50 hover:bg-dark-800/50
                       disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:bg-dark-900/50
                       transition-all duration-200"
          >
            <ChevronRight size={18} strokeWidth={1.5} />
          </button>
        </div>
      )}
    </div>
  );
}
