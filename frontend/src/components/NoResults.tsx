import { SearchX } from 'lucide-react';

interface NoResultsProps {
  query?: string;
}

export default function NoResults({ query }: NoResultsProps) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="w-16 h-16 rounded-2xl bg-dark-800/50 border border-dark-700/30 flex items-center justify-center mb-6">
        <SearchX size={28} className="text-dark-500" strokeWidth={1.5} />
      </div>
      <h3 className="text-lg font-medium text-dark-200 mb-2">No results found</h3>
      {query ? (
        <p className="text-dark-400 text-sm max-w-md leading-relaxed">
          No opportunities match "<span className="text-dark-300">{query}</span>".
          <br />
          Try different keywords or adjust your filters.
        </p>
      ) : (
        <p className="text-dark-400 text-sm max-w-md leading-relaxed">
          Enter a search term to find government contracting opportunities.
        </p>
      )}

      <div className="mt-8 text-left bg-dark-900/50 border border-dark-800/50 rounded-xl p-5 max-w-sm">
        <p className="text-xs font-medium text-dark-300 mb-3 uppercase tracking-wider">Search tips</p>
        <ul className="space-y-2.5 text-sm text-dark-400">
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">space</kbd>
            <span>All words must appear &mdash; more words, fewer results</span>
          </li>
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">"..."</kbd>
            <span>Exact phrase, e.g. &ldquo;zero trust&rdquo;</span>
          </li>
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">OR</kbd>
            <span>Either word, e.g. cyber OR cloud</span>
          </li>
          <li className="flex items-start gap-2">
            <kbd className="px-1.5 py-0.5 bg-dark-800/50 rounded text-[10px] text-dark-400 font-mono mt-0.5">-</kbd>
            <span>Exclude unwanted terms</span>
          </li>
        </ul>
      </div>
    </div>
  );
}
