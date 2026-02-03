import { X, Plus } from 'lucide-react';
import type { QueryGroup as QueryGroupType, QueryTerm } from '../types/api';

interface QueryGroupProps {
  group: QueryGroupType;
  onChange: (group: QueryGroupType) => void;
  onRemove: () => void;
  canRemove: boolean;
}

function generateId() {
  return Math.random().toString(36).substring(2, 9);
}

export default function QueryGroup({ group, onChange, onRemove, canRemove }: QueryGroupProps) {
  const addTerm = () => {
    const newTerm: QueryTerm = { id: generateId(), value: '', type: 'include' };
    onChange({ ...group, terms: [...group.terms, newTerm] });
  };

  const updateTerm = (termId: string, updates: Partial<QueryTerm>) => {
    onChange({
      ...group,
      terms: group.terms.map((t) => (t.id === termId ? { ...t, ...updates } : t)),
    });
  };

  const removeTerm = (termId: string) => {
    onChange({ ...group, terms: group.terms.filter((t) => t.id !== termId) });
  };

  const toggleOperator = () => {
    onChange({ ...group, operator: group.operator === 'AND' ? 'OR' : 'AND' });
  };

  return (
    <div className="rounded-xl border border-dark-800/50 bg-dark-900/30 p-4 space-y-3">
      <div className="flex items-center justify-between">
        <button
          onClick={toggleOperator}
          className={`text-[10px] font-semibold px-2.5 py-1 rounded-md transition-all duration-200 uppercase tracking-wider ${
            group.operator === 'AND'
              ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20 hover:bg-blue-500/20'
              : 'bg-orange-500/10 text-orange-400 border border-orange-500/20 hover:bg-orange-500/20'
          }`}
        >
          {group.operator}
        </button>
        {canRemove && (
          <button
            onClick={onRemove}
            className="p-1.5 text-dark-500 hover:text-dark-300 hover:bg-dark-800/50 rounded-lg transition-all duration-200"
            title="Remove group"
          >
            <X size={14} strokeWidth={1.5} />
          </button>
        )}
      </div>

      <div className="space-y-2">
        {group.terms.map((term) => (
          <div key={term.id} className="flex items-center gap-2">
            <select
              value={term.type}
              onChange={(e) => updateTerm(term.id, { type: e.target.value as QueryTerm['type'] })}
              className="w-24 py-2 px-2.5 text-xs bg-dark-850/50 border border-dark-700/50 rounded-lg
                         text-dark-300 focus:outline-none focus:border-accent/50 transition-colors duration-200"
            >
              <option value="include">Include</option>
              <option value="phrase">Phrase</option>
              <option value="exclude">Exclude</option>
            </select>
            <input
              type="text"
              value={term.value}
              onChange={(e) => updateTerm(term.id, { value: e.target.value })}
              placeholder={
                term.type === 'phrase' ? 'Exact phrase...'
                : term.type === 'exclude' ? 'Exclude term...'
                : 'Keyword...'
              }
              className="flex-1 py-2 px-3 text-sm bg-dark-850/50 border border-dark-700/50 rounded-lg
                         text-dark-100 placeholder:text-dark-500
                         focus:outline-none focus:border-accent/50 transition-colors duration-200"
            />
            {group.terms.length > 1 && (
              <button
                onClick={() => removeTerm(term.id)}
                className="p-1.5 text-dark-500 hover:text-dark-300 hover:bg-dark-800/50 rounded-lg transition-all duration-200"
                title="Remove term"
              >
                <X size={14} strokeWidth={1.5} />
              </button>
            )}
          </div>
        ))}
      </div>

      <button
        onClick={addTerm}
        className="text-xs text-dark-400 hover:text-dark-200 flex items-center gap-1.5 py-1 transition-colors duration-200"
      >
        <Plus size={12} strokeWidth={1.5} />
        Add term
      </button>
    </div>
  );
}
