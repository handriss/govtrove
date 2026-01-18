import type { OpportunityListItem } from '../types/opportunity';
import { OpportunityCard } from './OpportunityCard';

interface OpportunityListProps {
  opportunities: OpportunityListItem[] | undefined;
  isLoading: boolean;
  error: Error | null;
  onSelect: (id: number, index: number) => void;
}

export function OpportunityList({ opportunities, isLoading, error, onSelect }: OpportunityListProps) {
  if (isLoading) {
    return (
      <div className="space-y-4">
        {[...Array(5)].map((_, i) => (
          <div key={i} className="bg-white border border-gray-200 rounded-lg p-4 animate-pulse">
            <div className="h-5 bg-gray-200 rounded w-3/4 mb-2"></div>
            <div className="h-4 bg-gray-200 rounded w-1/2 mb-3"></div>
            <div className="flex gap-4">
              <div className="h-4 bg-gray-200 rounded w-24"></div>
              <div className="h-4 bg-gray-200 rounded w-24"></div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
        <p className="text-red-800 font-medium">Failed to load opportunities</p>
        <p className="text-red-600 text-sm mt-1">{error.message}</p>
      </div>
    );
  }

  if (!opportunities || opportunities.length === 0) {
    return (
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
        <p className="text-gray-600 font-medium">No opportunities found</p>
        <p className="text-gray-500 text-sm mt-1">Try adjusting your search or filters</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {opportunities.map((opp, index) => (
        <OpportunityCard key={opp.id} opportunity={opp} onClick={() => onSelect(opp.id, index)} />
      ))}
    </div>
  );
}
