import { Plus } from 'lucide-react';
import QueryGroup from './QueryGroup';
import type { QueryGroup as QueryGroupType } from '../types/api';

interface QueryBuilderProps {
  groups: QueryGroupType[];
  onChange: (groups: QueryGroupType[]) => void;
  groupOperator: 'AND' | 'OR';
  onGroupOperatorChange: (op: 'AND' | 'OR') => void;
}

function generateId() {
  return Math.random().toString(36).substring(2, 9);
}

function createEmptyGroup(): QueryGroupType {
  return {
    id: generateId(),
    operator: 'AND',
    terms: [{ id: generateId(), value: '', type: 'include' }],
  };
}

export function buildQueryString(groups: QueryGroupType[], groupOperator: 'AND' | 'OR'): string {
  const groupStrings = groups
    .map((group) => {
      const termStrings = group.terms
        .filter((t) => t.value.trim())
        .map((term) => {
          const value = term.value.trim();
          switch (term.type) {
            case 'phrase':
              return `"${value}"`;
            case 'exclude':
              return `-${value}`;
            default:
              return value;
          }
        });

      if (termStrings.length === 0) return '';
      if (termStrings.length === 1) return termStrings[0];

      const joiner = group.operator === 'OR' ? ' or ' : ' ';
      const joined = termStrings.join(joiner);
      return groups.length > 1 ? `(${joined})` : joined;
    })
    .filter(Boolean);

  if (groupStrings.length === 0) return '';
  if (groupStrings.length === 1) return groupStrings[0];

  const joiner = groupOperator === 'OR' ? ' or ' : ' ';
  return groupStrings.join(joiner);
}

export default function QueryBuilder({
  groups,
  onChange,
  groupOperator,
  onGroupOperatorChange,
}: QueryBuilderProps) {
  const addGroup = () => {
    onChange([...groups, createEmptyGroup()]);
  };

  const updateGroup = (groupId: string, updated: QueryGroupType) => {
    onChange(groups.map((g) => (g.id === groupId ? updated : g)));
  };

  const removeGroup = (groupId: string) => {
    onChange(groups.filter((g) => g.id !== groupId));
  };

  return (
    <div className="space-y-4">
      {groups.length > 1 && (
        <div className="flex items-center gap-3">
          <span className="text-xs text-dark-500">Combine groups with</span>
          <button
            onClick={() => onGroupOperatorChange(groupOperator === 'AND' ? 'OR' : 'AND')}
            className={`text-[10px] font-semibold px-2.5 py-1 rounded-md transition-all duration-200 uppercase tracking-wider ${
              groupOperator === 'AND'
                ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20 hover:bg-blue-500/20'
                : 'bg-orange-500/10 text-orange-400 border border-orange-500/20 hover:bg-orange-500/20'
            }`}
          >
            {groupOperator}
          </button>
        </div>
      )}

      <div className="space-y-3">
        {groups.map((group, idx) => (
          <div key={group.id}>
            <QueryGroup
              group={group}
              onChange={(updated) => updateGroup(group.id, updated)}
              onRemove={() => removeGroup(group.id)}
              canRemove={groups.length > 1}
            />
            {idx < groups.length - 1 && (
              <div className="flex justify-center py-2">
                <span className="text-[10px] text-dark-500 bg-dark-900 px-2 py-0.5 rounded uppercase tracking-wider">
                  {groupOperator}
                </span>
              </div>
            )}
          </div>
        ))}
      </div>

      <button
        onClick={addGroup}
        className="py-1.5 px-3 rounded-lg border border-dashed border-dark-700/50
                   text-xs text-dark-500 hover:text-dark-300 hover:border-dark-600/50 hover:bg-dark-800/30
                   flex items-center gap-1.5 transition-all duration-200"
      >
        <Plus size={14} strokeWidth={1.5} />
        Add Group
      </button>
    </div>
  );
}

export { createEmptyGroup };
