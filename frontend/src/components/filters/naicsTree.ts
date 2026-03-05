import type { NaicsCode } from '../../data/naicsCodes';
import type { FacetValue } from '../../types/api';

export interface NaicsTreeNode {
  code: string;
  title: string;
  level: number;
  children: NaicsTreeNode[];
  leafCodes: string[];
}

interface SearchResult {
  code: NaicsCode;
  role: 'parent' | 'match' | 'child';
}

// --- Lazy-loaded data ---

let NAICS_CODES: NaicsCode[] = [];
let dataLoaded = false;

export const naicsDataReady = import('../../data/naicsCodes').then((m) => {
  NAICS_CODES = m.NAICS_CODES;
  dataLoaded = true;
});

// --- Module-level constants (built once on first access) ---

const NODE_BY_CODE = new Map<string, NaicsTreeNode>();
const TITLE_BY_CODE = new Map<string, string>();

// NAICS sectors that span multiple 2-digit prefixes
const RANGE_SECTORS = [
  { code: '31-33', title: 'Manufacturing', prefixes: ['31', '32', '33'] },
  { code: '44-45', title: 'Retail Trade', prefixes: ['44', '45'] },
  { code: '48-49', title: 'Transportation and Warehousing', prefixes: ['48', '49'] },
];

const PREFIX_TO_RANGE = new Map<string, string>();
for (const sector of RANGE_SECTORS) {
  for (const prefix of sector.prefixes) {
    PREFIX_TO_RANGE.set(prefix, sector.code);
  }
}

function isDescendant(childCode: string, parentCode: string): boolean {
  if (childCode.startsWith(parentCode)) return true;
  const rangeCode = PREFIX_TO_RANGE.get(childCode.slice(0, 2));
  return rangeCode === parentCode;
}

function buildTree(): NaicsTreeNode[] {
  const roots: NaicsTreeNode[] = [];
  const stack: NaicsTreeNode[] = [];
  const rangeSectorNodes = new Map<string, NaicsTreeNode>();

  for (const entry of NAICS_CODES) {
    TITLE_BY_CODE.set(entry.code, entry.title);

    const node: NaicsTreeNode = {
      code: entry.code,
      title: entry.title,
      level: entry.level,
      children: [],
      leafCodes: [],
    };
    NODE_BY_CODE.set(entry.code, node);

    // Find parent: walk stack from top until we find an ancestor
    while (stack.length > 0 && !isDescendant(entry.code, stack[stack.length - 1].code)) {
      stack.pop();
    }

    if (stack.length === 0) {
      // Check if this level-3 code belongs under a range sector
      const rangeCode = PREFIX_TO_RANGE.get(entry.code.slice(0, 2));
      if (rangeCode && entry.level === 3) {
        let rangeNode = rangeSectorNodes.get(rangeCode);
        if (!rangeNode) {
          const sector = RANGE_SECTORS.find((s) => s.code === rangeCode)!;
          rangeNode = {
            code: sector.code,
            title: sector.title,
            level: 2,
            children: [],
            leafCodes: [],
          };
          rangeSectorNodes.set(rangeCode, rangeNode);
          NODE_BY_CODE.set(sector.code, rangeNode);
          TITLE_BY_CODE.set(sector.code, sector.title);
          roots.push(rangeNode);
        }
        rangeNode.children.push(node);
        stack.push(rangeNode, node);
      } else {
        roots.push(node);
        stack.push(node);
      }
    } else {
      stack[stack.length - 1].children.push(node);
      stack.push(node);
    }
  }

  // Compute leafCodes bottom-up
  function computeLeaves(node: NaicsTreeNode): string[] {
    if (node.level === 6) {
      node.leafCodes = [node.code];
    } else {
      node.leafCodes = node.children.flatMap(computeLeaves);
    }
    return node.leafCodes;
  }
  for (const root of roots) computeLeaves(root);

  return roots;
}

let NAICS_TREE: NaicsTreeNode[] | undefined;

function ensureInitialized(): NaicsTreeNode[] {
  if (!dataLoaded) return [];
  if (!NAICS_TREE) NAICS_TREE = buildTree();
  return NAICS_TREE;
}

export { ensureInitialized as getNaicsTree, NODE_BY_CODE, TITLE_BY_CODE };

// --- Exported functions ---

export function aggregateFacetCounts(facets: FacetValue[]): Map<string, number> {
  if (!dataLoaded) return new Map();
  ensureInitialized();
  const counts = new Map<string, number>();
  for (const f of facets) {
    counts.set(f.value, (counts.get(f.value) ?? 0) + f.count);
    // Roll up to range sector parent if applicable
    const rangeCode = PREFIX_TO_RANGE.get(f.value.slice(0, 2));
    if (rangeCode) {
      counts.set(rangeCode, (counts.get(rangeCode) ?? 0) + f.count);
    }
    // Walk up ancestors
    for (let len = f.value.length - 1; len >= 2; len--) {
      const ancestor = f.value.slice(0, len);
      if (NODE_BY_CODE.has(ancestor)) {
        counts.set(ancestor, (counts.get(ancestor) ?? 0) + f.count);
      }
    }
  }
  return counts;
}

export function searchNaicsCodes(query: string): SearchResult[] {
  if (!dataLoaded) return [];
  const q = query.trim();
  if (!q) return [];

  if (/^\d+$/.test(q)) {
    // Digit query: code prefix search
    const results: SearchResult[] = [];
    for (const c of NAICS_CODES) {
      if (q.startsWith(c.code) && c.code !== q) {
        results.push({ code: c, role: 'parent' });
      } else if (c.code === q) {
        results.push({ code: c, role: 'match' });
      } else if (c.code.startsWith(q) && c.code !== q) {
        results.push({ code: c, role: 'child' });
      }
    }
    return results;
  }

  // Keyword search
  const lower = q.toLowerCase();
  const results: SearchResult[] = [];
  for (const c of NAICS_CODES) {
    if (c.title.toLowerCase().includes(lower) || c.code.includes(q)) {
      results.push({ code: c, role: 'match' });
    }
  }
  return results;
}

/** Check if any ancestor of a code is in the selected set. */
export function isAncestorSelected(code: string, selectedSet: Set<string>): boolean {
  // Check range sector parent (e.g., "31-33" covers codes starting with 31/32/33)
  const rangeCode = PREFIX_TO_RANGE.get(code.slice(0, 2));
  if (rangeCode && selectedSet.has(rangeCode)) return true;
  // Check numeric prefix ancestors
  for (let len = 2; len < code.length; len++) {
    const prefix = code.slice(0, len);
    if (selectedSet.has(prefix)) return true;
  }
  return false;
}

/** Expand any parent codes in the selection to their leaf codes (for API queries). */
export function expandToLeafCodes(codes: string[]): string[] {
  if (!dataLoaded) return codes;
  ensureInitialized();
  const result: string[] = [];
  for (const code of codes) {
    const node = NODE_BY_CODE.get(code);
    if (node && node.leafCodes.length > 0 && node.level !== 6) {
      result.push(...node.leafCodes);
    } else {
      result.push(code);
    }
  }
  return result;
}

export function getAncestorCodes(code: string): string[] {
  ensureInitialized();
  const ancestors: string[] = [];
  const rangeCode = PREFIX_TO_RANGE.get(code.slice(0, 2));
  if (rangeCode && NODE_BY_CODE.has(rangeCode)) {
    ancestors.push(rangeCode);
  }
  for (let len = 2; len < code.length; len++) {
    const prefix = code.slice(0, len);
    if (NODE_BY_CODE.has(prefix)) {
      ancestors.push(prefix);
    }
  }
  return ancestors;
}
