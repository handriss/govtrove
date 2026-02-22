import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, ChevronRight, Search, X } from 'lucide-react';
import { useDropdownPosition } from './useDropdownPosition';
import {
  PSC_TREE,
  PSC_TITLE_BY_CODE,
  aggregatePscFacetCounts,
  searchPscCodes,
  getPscAncestorCodes,
  isPscAncestorSelected,
} from './pscTree';
import type { PscTreeNode } from './pscTree';
import type { FacetValue } from '../../types/api';

interface PscTreeSelectorProps {
  selected: string[];
  onChange: (codes: string[]) => void;
  facets?: FacetValue[];
  label?: string;
  inline?: boolean;
  compact?: boolean;
  loading?: boolean;
}

type CheckState = 'checked' | 'unchecked' | 'indeterminate';

const INDENT: Record<number, string> = {
  1: 'pl-0',
  2: 'pl-4',
  4: 'pl-8',
};

const ROLE_STYLE: Record<string, string> = {
  parent: 'text-dark-500',
  match: 'text-dark-100 font-medium',
  child: 'text-dark-300',
};

function isCovered(code: string, selectedSet: Set<string>): boolean {
  return selectedSet.has(code) || isPscAncestorSelected(code, selectedSet);
}

function getCheckState(node: PscTreeNode, selectedSet: Set<string>): CheckState {
  if (isCovered(node.code, selectedSet)) return 'checked';
  if (node.children.length === 0) return 'unchecked';

  let count = 0;
  for (const leaf of node.leafCodes) {
    if (isCovered(leaf, selectedSet)) count++;
  }
  if (count === 0) return 'unchecked';
  if (count === node.leafCodes.length) return 'checked';
  return 'indeterminate';
}

function HighlightedText({ text, query }: { text: string; query: string }) {
  if (!query) return <>{text}</>;
  const idx = text.toLowerCase().indexOf(query.toLowerCase());
  if (idx === -1) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-accent/20 text-accent">{text.slice(idx, idx + query.length)}</mark>
      {text.slice(idx + query.length)}
    </>
  );
}

export default function PscTreeSelector({
  selected,
  onChange,
  facets,
  label = 'PSC',
  inline = false,
  compact = false,
  loading = false,
}: PscTreeSelectorProps) {
  const [search, setSearch] = useState('');
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const inputRef = useRef<HTMLInputElement>(null);

  const selectedSet = useMemo(() => new Set(selected), [selected]);
  const facetCounts = useMemo(() => (facets ? aggregatePscFacetCounts(facets) : null), [facets]);

  const onClose = useCallback(() => {
    setSearch('');
  }, []);

  const { open, pos, triggerRef, dropdownRef, toggleDropdown } = useDropdownPosition({
    minWidth: compact ? 360 : 400,
    onClose,
  });

  useEffect(() => {
    if (open) {
      setSearch('');
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]);

  const handleToggle = useCallback(
    (node: PscTreeNode) => {
      const state = getCheckState(node, selectedSet);
      if (state === 'checked') {
        const toRemove = new Set([node.code, ...node.leafCodes]);
        onChange(selected.filter((c) => !toRemove.has(c)));
      } else {
        const leafSet = new Set(node.leafCodes);
        const cleaned = selected.filter((c) => !leafSet.has(c));
        onChange([...cleaned, node.code]);
      }
    },
    [selected, selectedSet, onChange],
  );

  const toggleExpand = useCallback((code: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(code)) next.delete(code);
      else next.add(code);
      return next;
    });
  }, []);

  const searchResults = useMemo(() => {
    if (!search.trim()) return null;
    return searchPscCodes(search);
  }, [search]);

  const handleSearchSelect = useCallback(
    (code: string) => {
      if (selectedSet.has(code)) {
        onChange(selected.filter((c) => c !== code));
      } else {
        onChange([...selected, code]);
      }
      const ancestors = getPscAncestorCodes(code);
      if (ancestors.length > 0) {
        setExpanded((prev) => {
          const next = new Set(prev);
          for (const a of ancestors) next.add(a);
          return next;
        });
      }
    },
    [selected, selectedSet, onChange],
  );

  const removeCode = useCallback(
    (code: string) => {
      onChange(selected.filter((c) => c !== code));
    },
    [selected, onChange],
  );

  function TreeNodeRow({ node }: { node: PscTreeNode }) {
    const isLeaf = node.children.length === 0;
    const isExpanded = expanded.has(node.code);
    const state = getCheckState(node, selectedSet);
    const count = facetCounts?.get(node.code);

    return (
      <>
        <div
          role="treeitem"
          aria-expanded={isLeaf ? undefined : isExpanded}
          aria-selected={state === 'checked'}
          className={`flex items-center gap-1.5 ${inline ? 'py-2.5' : 'py-1'} px-2 rounded hover:bg-dark-700/50 cursor-pointer text-xs ${INDENT[node.level] || 'pl-0'}`}
          onClick={() => {
            if (!isLeaf) toggleExpand(node.code);
          }}
        >
          {isLeaf ? (
            <span className="w-4 shrink-0" />
          ) : (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                toggleExpand(node.code);
              }}
              className="w-4 shrink-0 text-dark-500 hover:text-dark-300"
            >
              {isExpanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
            </button>
          )}

          <input
            type="checkbox"
            ref={(el) => {
              if (el) el.indeterminate = state === 'indeterminate';
            }}
            checked={state === 'checked'}
            onChange={(e) => {
              e.stopPropagation();
              handleToggle(node);
            }}
            onClick={(e) => e.stopPropagation()}
            className="accent-accent shrink-0"
          />

          <span className="font-mono text-dark-400 shrink-0">{node.code}</span>
          <span className="truncate text-dark-200">{node.title}</span>

          {count != null && count > 0 && (
            <span className="ml-auto text-dark-500 tabular-nums shrink-0">
              {count.toLocaleString()}
            </span>
          )}
        </div>

        {isExpanded && (
          <div role="group">
            {node.children.map((child) => <TreeNodeRow key={child.code} node={child} />)}
          </div>
        )}
      </>
    );
  }

  const chips = selected.length > 0 && (
    <div
      className={
        inline
          ? 'mt-2 space-y-2'
          : 'p-1.5 border-t border-dark-700/30 space-y-1'
      }
    >
      <div className="flex flex-wrap gap-1 px-1">
        {selected.map((code) => (
          <span
            key={code}
            className={`inline-flex items-center gap-1 bg-dark-900 border border-dark-700/50 rounded text-dark-300
                       ${inline ? 'px-2.5 py-1.5 text-xs' : 'px-1.5 py-0.5 text-[10px] gap-0.5'}`}
          >
            <span className="font-mono">{code}</span>
            <span className="text-dark-600">&mdash;</span>
            <span className="truncate max-w-[160px]">
              {PSC_TITLE_BY_CODE.get(code) || code}
            </span>
            <button
              type="button"
              onClick={() => removeCode(code)}
              className={`text-dark-500 hover:text-dark-200 ${inline ? 'ml-1 p-0.5' : 'ml-0.5'}`}
            >
              &times;
            </button>
          </span>
        ))}
      </div>
      <button
        type="button"
        onClick={() => onChange([])}
        className={`w-full text-dark-400 hover:text-dark-200 ${inline ? 'text-xs py-1' : 'text-[11px] py-0.5'}`}
      >
        Clear all
      </button>
    </div>
  );

  const panelContent = (
    <>
      <div className={inline ? '' : 'p-1.5 border-b border-dark-700/30'}>
        <div className="relative">
          <Search
            size={inline ? 14 : 12}
            className={`absolute left-2 top-1/2 -translate-y-1/2 text-dark-500 ${inline ? 'left-3' : ''}`}
          />
          <input
            ref={inputRef}
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search PSC codes..."
            className={
              inline
                ? `w-full text-sm bg-dark-800 border rounded-lg pl-9 pr-8 py-2.5
                   placeholder:text-dark-500 focus:outline-none focus:border-accent/50
                   ${selected.length > 0 ? 'border-accent/40 text-dark-200' : 'border-dark-700/50 text-dark-200'}`
                : `w-full text-xs bg-dark-900 border border-dark-700/50 rounded pl-6 pr-6 py-1.5 text-dark-200
                   placeholder:text-dark-500 focus:outline-none focus:border-accent/50`
            }
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch('')}
              className={`absolute top-1/2 -translate-y-1/2 text-dark-500 hover:text-dark-300 ${inline ? 'right-3' : 'right-2'}`}
            >
              <X size={inline ? 14 : 12} />
            </button>
          )}
        </div>
      </div>

      {loading ? (
        <p className={`text-dark-500 px-3 ${inline ? 'text-sm py-3' : 'text-xs py-2'}`}>
          Loading...
        </p>
      ) : searchResults ? (
        <div className={`overflow-y-auto ${inline ? 'max-h-[240px] mt-2 rounded-lg border border-dark-700/50 bg-dark-800' : 'flex-1'}`}>
          {searchResults.length === 0 ? (
            <p className={`text-dark-500 px-3 ${inline ? 'text-sm py-3' : 'text-xs py-2'}`}>
              No matches
            </p>
          ) : (
            searchResults.map((item) => {
              const isSelected = selectedSet.has(item.code.code);
              return (
                <div
                  key={item.code.code}
                  onClick={() => handleSearchSelect(item.code.code)}
                  className={`cursor-pointer flex items-center gap-1.5
                             ${INDENT[item.code.level] || 'pl-0'}
                             hover:bg-dark-700/50
                             ${ROLE_STYLE[item.role] || 'text-dark-300'}
                             ${inline ? 'px-3 py-2.5 text-sm' : 'px-2 py-1 text-xs'}`}
                >
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onChange={() => handleSearchSelect(item.code.code)}
                    onClick={(e) => e.stopPropagation()}
                    className="accent-accent shrink-0"
                  />
                  <span className={`font-mono shrink-0 ${inline ? 'text-xs' : 'text-[10px]'}`}>
                    {item.code.code}
                  </span>
                  <span className={`text-dark-600 ${inline ? 'text-xs' : 'text-[10px]'}`}>
                    &mdash;
                  </span>
                  <span className="truncate">
                    <HighlightedText text={item.code.title} query={search} />
                  </span>
                  {facetCounts && facetCounts.get(item.code.code) != null && (
                    <span className="ml-auto text-dark-500 tabular-nums shrink-0">
                      {facetCounts.get(item.code.code)!.toLocaleString()}
                    </span>
                  )}
                </div>
              );
            })
          )}
        </div>
      ) : (
        <div role="tree" aria-label="PSC code hierarchy" className={`overflow-y-auto ${inline ? 'max-h-[300px] mt-2' : 'flex-1'} p-1`}>
          {PSC_TREE.map((root) => (
            <TreeNodeRow key={root.code} node={root} />
          ))}
        </div>
      )}

      {chips}
    </>
  );

  if (inline) {
    return <div>{panelContent}</div>;
  }

  const active = selected.length > 0;
  const triggerLabel = active ? `${label} (${selected.length})` : label;

  if (compact) {
    const display =
      selected.length === 0
        ? 'All'
        : selected.length === 1
          ? selected[0]
          : `${selected.length} codes`;

    return (
      <div>
        <button
          ref={triggerRef}
          type="button"
          onClick={toggleDropdown}
          aria-expanded={open}
          className={`w-full text-xs bg-dark-800 border rounded px-1.5 py-1
                     focus:outline-none cursor-pointer text-left flex items-center justify-between gap-1
                     ${active ? 'border-accent/40 text-accent' : 'border-dark-700/50 text-dark-200'}`}
        >
          <span className="truncate">{display}</span>
          <ChevronDown
            size={10}
            className={`shrink-0 text-dark-500 transition-transform ${open ? 'rotate-180' : ''}`}
          />
        </button>
        {open &&
          createPortal(
            <div
              ref={dropdownRef}
              style={{ position: 'fixed', top: pos.top, left: pos.left, width: pos.width }}
              className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden flex flex-col max-h-[400px]"
            >
              {panelContent}
            </div>,
            document.body,
          )}
      </div>
    );
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={toggleDropdown}
        aria-expanded={open}
        className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border cursor-pointer transition-colors
          ${active
            ? 'bg-accent/10 border-accent/40 text-accent'
            : 'bg-dark-800 border-dark-700/50 text-dark-200 hover:border-dark-600'
          }`}
      >
        <span className="truncate">{triggerLabel}</span>
        <ChevronDown
          size={14}
          className={`shrink-0 text-dark-400 transition-transform ${open ? 'rotate-180' : ''}`}
        />
      </button>
      {open &&
        createPortal(
          <div
            ref={dropdownRef}
            style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: pos.width }}
            className="z-50 bg-dark-800 border border-dark-700/50 rounded-lg shadow-2xl shadow-black/50 overflow-hidden flex flex-col max-h-[400px]"
          >
            {panelContent}
          </div>,
          document.body,
        )}
    </>
  );
}
