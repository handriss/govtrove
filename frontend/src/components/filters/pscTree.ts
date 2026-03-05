import type { PscCode } from '../../data/pscCodes';
import type { FacetValue } from '../../types/api';

export interface PscTreeNode {
  code: string;
  title: string;
  level: number;
  children: PscTreeNode[];
  leafCodes: string[];
}

interface SearchResult {
  code: PscCode;
  role: 'parent' | 'match' | 'child';
}

// --- Lazy-loaded data ---

let PSC_CODES: PscCode[] = [];
let dataLoaded = false;

export const pscDataReady = import('../../data/pscCodes').then((m) => {
  PSC_CODES = m.PSC_CODES;
  dataLoaded = true;
});

// ---

const NODE_BY_CODE = new Map<string, PscTreeNode>();
const TITLE_BY_CODE = new Map<string, string>();

function isDescendant(childCode: string, parentCode: string): boolean {
  return childCode.startsWith(parentCode);
}

function buildTree(): PscTreeNode[] {
  const roots: PscTreeNode[] = [];
  const stack: PscTreeNode[] = [];

  for (const entry of PSC_CODES) {
    TITLE_BY_CODE.set(entry.code, entry.title);

    const node: PscTreeNode = {
      code: entry.code,
      title: entry.title,
      level: entry.level,
      children: [],
      leafCodes: [],
    };
    NODE_BY_CODE.set(entry.code, node);

    while (stack.length > 0 && !isDescendant(entry.code, stack[stack.length - 1].code)) {
      stack.pop();
    }

    if (stack.length === 0) {
      roots.push(node);
    } else {
      stack[stack.length - 1].children.push(node);
    }
    stack.push(node);
  }

  function computeLeaves(node: PscTreeNode): string[] {
    if (node.children.length === 0) {
      node.leafCodes = [node.code];
    } else {
      node.leafCodes = node.children.flatMap(computeLeaves);
    }
    return node.leafCodes;
  }
  for (const root of roots) computeLeaves(root);

  return roots;
}

let PSC_TREE: PscTreeNode[] | undefined;

function ensureInitialized(): PscTreeNode[] {
  if (!dataLoaded) return [];
  if (!PSC_TREE) PSC_TREE = buildTree();
  return PSC_TREE;
}

export { ensureInitialized as getPscTree, NODE_BY_CODE as PSC_NODE_BY_CODE, TITLE_BY_CODE as PSC_TITLE_BY_CODE };

export function aggregatePscFacetCounts(facets: FacetValue[]): Map<string, number> {
  if (!dataLoaded) return new Map();
  ensureInitialized();
  const counts = new Map<string, number>();
  for (const f of facets) {
    counts.set(f.value, (counts.get(f.value) ?? 0) + f.count);
    for (let len = f.value.length - 1; len >= 1; len--) {
      const ancestor = f.value.slice(0, len);
      if (NODE_BY_CODE.has(ancestor)) {
        counts.set(ancestor, (counts.get(ancestor) ?? 0) + f.count);
      }
    }
  }
  return counts;
}

export function searchPscCodes(query: string): SearchResult[] {
  if (!dataLoaded) return [];
  const q = query.trim();
  if (!q) return [];

  if (/^[A-Za-z0-9]+$/.test(q) && /\d/.test(q)) {
    // Code prefix search (contains digits)
    const upper = q.toUpperCase();
    const results: SearchResult[] = [];
    for (const c of PSC_CODES) {
      if (upper.startsWith(c.code) && c.code !== upper) {
        results.push({ code: c, role: 'parent' });
      } else if (c.code === upper) {
        results.push({ code: c, role: 'match' });
      } else if (c.code.startsWith(upper) && c.code !== upper) {
        results.push({ code: c, role: 'child' });
      }
    }
    return results;
  }

  if (/^[A-Za-z]$/.test(q)) {
    // Single letter: match category and children
    const upper = q.toUpperCase();
    const results: SearchResult[] = [];
    for (const c of PSC_CODES) {
      if (c.code === upper) {
        results.push({ code: c, role: 'match' });
      } else if (c.code.startsWith(upper)) {
        results.push({ code: c, role: 'child' });
      }
    }
    return results;
  }

  // Keyword search
  const lower = q.toLowerCase();
  const results: SearchResult[] = [];
  for (const c of PSC_CODES) {
    if (c.title.toLowerCase().includes(lower) || c.code.toLowerCase().includes(lower)) {
      results.push({ code: c, role: 'match' });
    }
  }
  return results;
}

export function isPscAncestorSelected(code: string, selectedSet: Set<string>): boolean {
  for (let len = 1; len < code.length; len++) {
    const prefix = code.slice(0, len);
    if (selectedSet.has(prefix)) return true;
  }
  return false;
}

export function expandPscToLeafCodes(codes: string[]): string[] {
  if (!dataLoaded) return codes;
  ensureInitialized();
  const result: string[] = [];
  for (const code of codes) {
    const node = NODE_BY_CODE.get(code);
    if (node && node.leafCodes.length > 0 && node.children.length > 0) {
      result.push(...node.leafCodes);
    } else {
      result.push(code);
    }
  }
  return result;
}

export function getPscAncestorCodes(code: string): string[] {
  ensureInitialized();
  const ancestors: string[] = [];
  for (let len = 1; len < code.length; len++) {
    const prefix = code.slice(0, len);
    if (NODE_BY_CODE.has(prefix)) {
      ancestors.push(prefix);
    }
  }
  return ancestors;
}
