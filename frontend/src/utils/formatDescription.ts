/**
 * Heuristically insert paragraph breaks into flat SAM.gov description text.
 *
 * SAM.gov bulk CSVs strip all HTML, producing a single-line wall of text.
 * This function detects likely section boundaries — numbered headings,
 * ALL-CAPS labels, bullets, separators — and inserts newlines before them.
 *
 * Every pattern (except unambiguous bullet markers) requires sentence-ending
 * punctuation or a colon in a lookbehind to avoid false splits mid-sentence.
 */
export function formatDescription(text: string): string {
  let result = text;

  // Strip double-asterisk markdown emphasis markers, keep inner text
  result = result.replace(/\*\*(.+?)\*\*/g, '$1');

  // Separator lines (===== or longer)
  result = result.replace(/\s*([=]{5,})\s*/g, '\n\n$1\n\n');

  // Numbered ALL-CAPS section headings after sentence end
  // e.g. "...standards. 1. INTRODUCTION" or "...scope. 2) REQUIREMENTS"
  result = result.replace(/([.!?:;])\s+(\d+[.)]\s+[A-Z]{2,})/g, '$1\n\n$2');

  // PART + roman numeral after sentence end
  result = result.replace(/([.!?:;])\s+(PART\s+[IVX]+)/g, '$1\n\n$2');

  // Amendment/update markers after sentence end
  result = result.replace(/([.!?])\s+((Amendment|UPDATE)\s+\d)/g, '$1\n\n$2');

  // ALL-CAPS label (5+ chars including spaces) followed by colon/dash after sentence end
  // e.g. "...deleted. DESCRIPTION:" or "...apply. PLACE OF PERFORMANCE:"
  result = result.replace(/([.!?])\s+([A-Z][A-Z\s]{4,}[:–-])/g, '$1\n\n$2');

  // Lettered list items after sentence end
  // e.g. "...below. A) Name of firm"
  result = result.replace(/([.!?:;])\s+([A-Z][.)]\s+)/g, '$1\n\n$2');

  // Bullet points — unambiguous, no lookbehind needed
  result = result.replace(/\s+(•\s)/g, '\n$1');

  // Sub-indent bullets using "o" before a capital letter
  result = result.replace(/\s+(o\s+[A-Z])/g, '\n$1');

  // Collapse any runs of 3+ newlines down to 2
  result = result.replace(/\n{3,}/g, '\n\n');

  return result.trim();
}
