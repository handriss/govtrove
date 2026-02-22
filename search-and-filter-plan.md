# GovTrove Search & Filter UX — Claude Code Implementation Prompts

*Sequenced implementation plan broken into self-contained Claude Code sessions.*
*Each prompt builds on the previous ones. Run them in order.*

---

## Prerequisites & Context for Every Prompt

Paste this context block at the start of any prompt if Claude Code doesn't have access to your CLAUDE.md:

> GovTrove is a government contract opportunity search platform (React frontend, Go backend, Neon PostgreSQL, AWS Fargate). It currently has basic filters for NAICS, agency, set-aside, notice type, dates, and place of performance. We're overhauling the search/filter UX based on patterns from Linear (dismissable filter chips, incremental filter construction), Airtable (workflow constructor for advanced queries), and Booking.com (result counts next to filter options). The frontend uses React. The backend serves a REST API. Data comes from SAM.gov.

---

## Phase 1: Filter Infrastructure & Data Layer

These prompts set up the backend and shared frontend infrastructure that all filter UI components depend on.

### Prompt 1.1 — Facet Count API Endpoint

```
I need a new API endpoint that returns facet counts for filter options,
similar to how Booking.com shows "(234)" next to each filter value.

Create a GET /api/v1/opportunities/facets endpoint that:

1. Accepts the same filter parameters as the existing search endpoint
   (keyword, naics, agency, set_aside, notice_type, deadline_start,
   deadline_end, place_of_performance, active_only)
2. Returns counts for each filterable dimension, respecting all OTHER
   currently-applied filters (cross-filtering). For example, if
   set_aside=SDVOSB is active, the NAICS counts should reflect only
   SDVOSB opportunities.
3. Returns this shape:
   {
     "total": 14832,
     "facets": {
       "set_aside": [
         {"value": "SBA", "label": "Small Business", "count": 2847},
         {"value": "SDVOSB", "label": "Service-Disabled Veteran-Owned", "count": 423},
         ...
       ],
       "notice_type": [
         {"value": "o", "label": "Solicitation", "count": 5234},
         {"value": "p", "label": "Presolicitation", "count": 1822},
         ...
       ],
       "agency": [
         {"value": "DOD", "label": "Department of Defense", "count": 8234},
         ...  // top 50 by count, sorted descending
       ],
       "naics": [
         {"value": "541512", "label": "Computer Systems Design Services", "count": 1247},
         ...  // top 50 by count, sorted descending
       ]
     }
   }
4. For agency and NAICS, return only the top 50 by count (the full
   list is searched client-side via the typeahead). Include a
   "other_count" field showing how many additional options exist.
5. Use a single SQL query with COUNT + GROUP BY + FILTER clauses for
   efficiency. Target <200ms response time on ~100K records.
6. Add appropriate caching headers (Cache-Control: max-age=60) since
   facet counts don't need to be real-time.

Look at the existing search endpoint handler and database query patterns
for conventions on parameter parsing, error handling, and response format.
```

### Prompt 1.2 — Filter State URL Serialization (Frontend)

```
Create a React hook called useFilterState that manages all search filter
state with bidirectional URL synchronization. This is the foundation
that all filter UI components will use.

Requirements:

1. The hook manages this filter state shape:
   {
     keyword: string,
     naics: string[],          // array of NAICS codes
     setAside: string[],       // array of set-aside codes
     agency: string[],         // array of agency codes
     noticeType: string[],     // array of notice type codes
     deadlineRange: string | null,  // "7d" | "14d" | "30d" | "60d" | "quarter" | null
     deadlineStart: string | null,  // ISO date for custom range
     deadlineEnd: string | null,
     placeOfPerformance: string | null,
     activeOnly: boolean,
     sort: string,             // "relevance" | "deadline" | "posted" | "agency"
     sortDir: "asc" | "desc",
     page: number,
   }

2. Serialize to readable URL params:
   ?keyword=cybersecurity&naics=541512,541511&set_aside=SDVOSB&deadline=30d&sort=deadline&page=1

3. On mount, parse URL params into state (handle malformed params
   gracefully — fall back to defaults, don't crash).

4. When state changes:
   - Use replaceState for typing/incremental changes (keyword typing)
   - Use pushState for discrete filter changes (adding/removing a
     filter chip) so back button works
   - Always reset page to 1 when any filter changes

5. Export:
   - filterState: the current state object
   - setFilter(key, value): update a single filter
   - addFilter(key, value): add to an array filter (e.g., add a NAICS code)
   - removeFilter(key, value): remove from an array filter
   - clearFilter(key): clear a specific filter
   - clearAllFilters(): reset everything to defaults
   - filterCount: number of active non-default filters (for badge display)
   - toQueryParams(): serialize current state to API query params

6. Persist the user's last sort preference in localStorage.

7. Do NOT persist filter state in localStorage — URL is the source of
   truth for filters (enables sharing filtered views).

Put this in src/hooks/useFilterState.ts (or .js depending on project convention).
Follow existing project patterns for hooks.
```

### Prompt 1.3 — Facet Count Fetching Hook

```
Create a React hook called useFacetCounts that fetches facet counts
from the API endpoint created in prompt 1.1.

Requirements:

1. Takes the current filterState (from useFilterState) as input.
2. Calls GET /api/v1/opportunities/facets with the current filters.
3. Debounces requests by 300ms to avoid hammering the API during
   rapid filter changes.
4. Returns:
   {
     facets: { set_aside: [...], notice_type: [...], agency: [...], naics: [...] },
     total: number,
     isLoading: boolean,
     error: Error | null
   }
5. Caches the last successful result and shows it (slightly stale)
   while a new request is in flight, with isLoading=true.
6. Aborts in-flight requests when filters change (use AbortController).
7. Does NOT re-fetch if only sort/page/sortDir changed (facet counts
   are independent of sorting and pagination).

Put this in src/hooks/useFacetCounts.ts.
Follow existing patterns for data fetching hooks in the project.
```

---

## Phase 2: Core Filter Components

These prompts build the individual filter UI components.

### Prompt 2.1 — FilterChipBar Component

```
Create a FilterChipBar component that displays all active filters as
dismissable chips, inspired by Linear's filter UI.

Requirements:

1. Renders a horizontal bar between the filter controls and results.
2. Each active filter renders as a chip showing category and value:
   "NAICS: 541512"  "Set-Aside: SDVOSB"  "Deadline: next 30 days"
   "Agency: Department of Defense"
3. Each chip has an × button to remove that specific filter.
4. Include a "Clear all" link at the end when 2+ filters are active.
5. When there are more than 8 chips, wrap to a max of 2 rows, then
   collapse remaining into a "+N more" pill that expands on click.
6. Chip styling: subtle background fill (e.g., blue-50 bg, blue-700
   text), rounded-full, 12-14px text. Active/filled appearance that
   stands out from the page background.
7. Keyboard accessible:
   - Tab navigates between chips
   - Enter/Space on × removes the chip
   - After removing, focus moves to the next chip (or "Clear all"
     if it was the last)
8. Animate chip removal (brief fade-out, ~150ms).
9. The bar should be sticky during scroll (position: sticky, top: X)
   so users always see their active filters.
10. When no filters are active, render nothing (don't show an empty bar).

Props:
- filterState: from useFilterState hook
- onRemoveFilter: (key, value) => void
- onClearAll: () => void

Follow the existing component patterns and styling approach in the project.
Use whatever CSS framework/approach the project already uses.
```

### Prompt 2.2 — SearchableDropdownFilter Component

```
Create a reusable SearchableDropdownFilter component for filter
categories with many options (NAICS codes, agencies, PSC codes).
This is the core multi-select filter dropdown used throughout the app.

Requirements:

1. Trigger button shows the filter label and count of selected items:
   "NAICS (3)" or just "NAICS" if none selected.
   Style the trigger as an outlined button that looks "active" (filled)
   when any values are selected.

2. Clicking the trigger opens a dropdown panel below it containing:
   a. A search input (autofocused on open) with placeholder
      "Search NAICS codes..."
   b. Selected items pinned at the top with checkmarks, separated by
      a thin divider from unselected items
   c. Unselected items below, each showing:
      - Checkbox
      - Label text (e.g., "541512 — Computer Systems Design Services")
      - Result count in muted text: "(1,247)"
      - Highlight matching text when search is active
   d. "Clear selection" link at the bottom when items are selected

3. Options list is virtualized (renders only visible items + buffer)
   to handle 1000+ options without performance issues. Use a simple
   virtual scroll implementation — only render items in the viewport.

4. Search filters options client-side with 100ms debounce. Match on
   both code and label (typing "541" or "computer" both work).

5. Multi-select behavior:
   - Checking/unchecking an item does NOT close the dropdown
   - The dropdown closes when clicking outside or pressing Escape
   - Filter applies when the dropdown closes (not on every click)

6. Sort options: selected first, then by result count descending.

7. Accessibility:
   - role="combobox" on the trigger, aria-expanded
   - role="listbox" on the options list
   - role="option" with aria-selected on each item
   - Arrow keys navigate options
   - Enter toggles selection
   - Escape closes dropdown
   - aria-label on the search input

8. Max height of dropdown: 400px with scroll.
   Max width: 360px (or wider if needed for long labels).

Props:
- label: string ("NAICS", "Agency", etc.)
- options: Array<{ value: string, label: string, count?: number }>
- selected: string[]
- onSelectionChange: (selected: string[]) => void
- searchPlaceholder?: string
- loading?: boolean

Create as src/components/filters/SearchableDropdownFilter.tsx (or .jsx).
```

### Prompt 2.3 — DeadlineFilter Component

```
Create a DeadlineFilter component for filtering by response deadline.

Requirements:

1. Trigger button: "Deadline" (or "Due: next 30 days" when active).

2. Dropdown contains:
   a. Preset relative ranges as selectable pills/radio buttons:
      - "Next 7 days"
      - "Next 14 days"
      - "Next 30 days" (should feel like the default/recommended)
      - "Next 60 days"
      - "This quarter"
      - "All" (no deadline filter)
   b. A "Custom range..." option that reveals a date range picker
      with two date inputs (From / To) using native date inputs.
   c. Display result count next to each preset if available from
      facet data.

3. Selecting a preset applies immediately and closes the dropdown.
   Custom range has an "Apply" button.

4. In the filter chip bar, show both relative and absolute:
   "Deadline: next 30 days (through Mar 23, 2026)"

5. IMPORTANT for saved searches later: the state should store the
   relative preset identifier ("30d") not the computed absolute dates.
   The absolute dates are computed at render/query time.

6. Past dates are disabled in the custom range picker.

Props:
- value: { preset: string | null, start: string | null, end: string | null }
- onChange: (value) => void
- facetCounts?: Array<{ preset: string, count: number }>

Create as src/components/filters/DeadlineFilter.tsx.
```

### Prompt 2.4 — SimpleToggleFilter Component

```
Create a SimpleToggleFilter component for filters with few options
(notice type, active/inactive status).

Requirements:

1. For binary filters (active_only): render as a single toggle/switch
   with label. Apply immediately on toggle.

2. For small multi-select filters (notice_type with ~9 options):
   a. Trigger button: "Notice Type" (or "Notice Type (3)" when active)
   b. Dropdown shows checkboxes for each option with result counts:
      ☑ Solicitation (5,234)
      ☑ Combined Synopsis/Solicitation (3,102)
      ☑ Sources Sought (1,822)
      ☐ Award Notice (8,912)
      ☐ Pre-solicitation (944)
      ...etc
   c. Multi-select, applies on dropdown close
   d. Include "Select all" / "Clear all" links at top

3. For notice type specifically, implement "smart defaults" — pre-select
   the notice types contractors care about (Solicitation, Combined
   Synopsis/Solicitation, Sources Sought, Special Notice) and leave
   Award Notices, Intent to Bundle, etc. unchecked by default.
   Store this as a named constant DEFAULT_NOTICE_TYPES.

4. Result counts from facet data next to each option.

5. Gray out / disable options with count of 0.

Props:
- label: string
- options: Array<{ value: string, label: string, count?: number }>
- selected: string[]
- onSelectionChange: (selected: string[]) => void
- mode: "toggle" | "multi-select"

Create as src/components/filters/SimpleToggleFilter.tsx.
```

---

## Phase 3: Filter Bar Assembly

### Prompt 3.1 — FilterBar Component (Main Assembly)

```
Create the main FilterBar component that assembles all individual
filter components into the primary search interface.

Layout (top to bottom):

ROW 1: Keyword Search
┌──────────────────────────────────────────────────────┐
│ 🔍 [Search opportunities...                ] [Search]│
└──────────────────────────────────────────────────────┘

ROW 2: Primary Filter Triggers (horizontal, scrollable on mobile)
┌──────────────────────────────────────────────────────┐
│ [NAICS ▾] [Set-Aside ▾] [Agency ▾] [Deadline ▾]     │
│ [Notice Type ▾] [Active Only ⟲]    [+ More Filters] │
└──────────────────────────────────────────────────────┘

ROW 3: Active Filter Chips (FilterChipBar — only when filters active)
┌──────────────────────────────────────────────────────┐
│ NAICS: 541512 × | Set-Aside: SDVOSB × | Clear all   │
└──────────────────────────────────────────────────────┘

ROW 4: Results toolbar
┌──────────────────────────────────────────────────────┐
│ 14,832 results              Sort: Deadline ▾  [☰][▦]│
└──────────────────────────────────────────────────────┘

Requirements:

1. Use useFilterState hook for all state management.
2. Use useFacetCounts hook to pass result counts to each filter component.
3. Wire up all filter components from Phase 2.
4. The keyword search input should:
   - Submit on Enter key press
   - Have a visible Search button
   - Debounce input by 500ms for search-as-you-type (but also
     support explicit Enter/button press for immediate search)
   - Show a clear (×) button inside the input when text is present
5. "+ More Filters" button shows a badge with count of active
   secondary filters. Clicking it opens a slide-out panel from the
   right (or a modal on mobile) containing additional filters:
   - Place of Performance (text input with autocomplete)
   - Award Value Range (min/max number inputs)
   - Posted Date range
   - Solicitation Number (exact text input)
6. The results toolbar row shows:
   - Left: total count "14,832 results" (or "14,832 results for
     'cybersecurity'" if keyword is active). Format with commas.
     Show a skeleton/shimmer while loading.
   - Right: Sort dropdown (Relevance, Deadline, Posted Date, Agency)
     with direction toggle. Default to Relevance when keyword present,
     Deadline (soonest) when browsing.
   - Right: View mode toggle icons (list view / table view)
     — list view is implemented now, table view can be a stub

7. Mobile responsive:
   - On mobile (<768px), primary filters collapse into a single
     "Filters" button that opens a full-screen modal
   - Keyword search stays always visible
   - Filter chips scroll horizontally with overflow
   - Results toolbar stacks count above sort

8. The entire filter bar region should use role="search" with
   aria-label="Search and filter opportunities".

9. Total result count should update the page title:
   document.title = "14,832 results — GovTrove"

Create as src/components/search/FilterBar.tsx.
Wire it into the existing search page, replacing the current filter UI.
Preserve the existing search page's overall layout and routing.
```

---

## Phase 4: Results Display

### Prompt 4.1 — OpportunityCard Component

```
Redesign the search result card for each opportunity. The card should
enable rapid scanning — a contractor should be able to evaluate
"is this worth reading more?" in under 3 seconds per card.

Layout for each card (horizontal list-card):

┌──────────────────────────────────────────────────────┐
│ ☐ ★  Cybersecurity Support Services — IDIQ      🟠  │
│      Department of the Army · NAICS: 541512          │
│      Set-Aside: SDVOSB · Sol#: W911QY-26-R-0042     │
│      $2.4M – $5.0M est.                             │
│      Due: Mar 15, 2026 (22 days)                     │
│                                                      │
│      "The contractor shall provide cybersecurity     │
│      monitoring and incident response services..."    │
│      📍 Fort Liberty, NC                             │
└──────────────────────────────────────────────────────┘

Requirements:

1. Visual hierarchy (3 tiers):
   - PRIMARY (bold, 16px): Title (linked/clickable), urgency badge
   - SECONDARY (14px, muted): Agency, NAICS badge, set-aside badge,
     solicitation number, estimated value
   - TERTIARY (13px, lighter): Description snippet (2-3 lines,
     truncated with ellipsis), place of performance

2. Urgency/freshness badges (top-right corner of card):
   - 🟢 "New" — posted within 48 hours (green)
   - 🔵 "Updated" — modified since last visit (blue, requires auth)
   - 🟠 "Closing Soon" — due within 7 days (amber)
   - 🔴 "Closing Today" — due within 48 hours (red)
   - No badge if none apply (don't clutter)

3. Deadline display shows BOTH relative and absolute:
   "Due: Mar 15, 2026 (22 days)"
   Color the text based on urgency (same thresholds as badges).

4. Keyword highlighting: If a search keyword is active, bold/highlight
   matching terms in the title and description using <mark> with a
   subtle yellow background. Use the keyword from filterState.

5. Left edge of card:
   - Checkbox for bulk selection (hidden by default, shown on hover
     or when any card is selected)
   - Save/bookmark star icon (always visible, toggles on click)

6. NAICS code and set-aside type render as small inline badges
   (pill-shaped, subtle background color).

7. Card has subtle hover state (slight shadow elevation or border
   color change) and is fully clickable (navigates to detail page).
   The star and checkbox should NOT trigger navigation (stopPropagation).

8. Cards separated by a subtle divider line or slight vertical gap.

9. Accessibility:
   - Each card is an article element with aria-label summarizing
     the opportunity
   - Star button has aria-label "Save opportunity" / "Remove saved"
   - Checkbox has aria-label "Select [title] for bulk actions"

10. Description snippet: Show the portion containing matched keywords
    (with ellipsis around it) when a keyword search is active. Fall
    back to first 200 characters when browsing without keywords.

Props:
- opportunity: OpportunityData
- keyword?: string  (for highlighting)
- isSaved: boolean
- isSelected: boolean
- onToggleSave: (id) => void
- onToggleSelect: (id) => void
- onClick: (id) => void

Create as src/components/search/OpportunityCard.tsx.
```

### Prompt 4.2 — Results List & Pagination

```
Create the SearchResults component that renders the list of
OpportunityCards with pagination.

Requirements:

1. Renders a vertical list of OpportunityCard components.
2. Shows loading state: skeleton cards (3-5 placeholders) that match
   the card layout shape during initial load and filter changes.
3. Pagination at the bottom:
   - "« Previous  1  2  3  ...  10  Next »"
   - Show page numbers with ellipsis for large sets
   - 25 results per page (with option for 50, 100 in a small dropdown)
   - Current page highlighted
   - Previous/Next disabled at boundaries
   - Pagination state in URL (?page=3)
   - Scroll to top of results when page changes
4. Results-per-page selector: small dropdown "25 / 50 / 100" right-
   aligned near pagination. Persist preference in localStorage.
5. Zero results state — when filters produce no matches, show:
   - Large, friendly "No opportunities match your filters" heading
   - Suggest which filter to relax: "Try removing [most restrictive
     filter] — that would show ~X results" (use facet count data
     to identify which filter is most restrictive)
   - "Clear all filters" prominent button
   - Do NOT show a blank white page
6. Bulk action bar: when 1+ cards are selected via checkbox, show a
   floating bar at the bottom of the viewport:
   "3 selected: [Save All] [Export CSV] [Clear Selection]"
   This bar should be sticky/fixed at bottom, above pagination.
   Export CSV is a paid feature — show it but with a subtle lock icon
   and tooltip "Pro feature" for now.
7. The list area should show a subtle loading overlay (semi-transparent
   white with spinner) during filter changes while preserving the
   previous results underneath — don't flash blank.

Props:
- opportunities: OpportunityData[]
- total: number
- page: number
- pageSize: number
- isLoading: boolean
- keyword?: string
- filterState: FilterState  (for zero-results suggestions)
- facetCounts: FacetCounts  (for zero-results suggestions)
- onPageChange: (page: number) => void
- onPageSizeChange: (size: number) => void

Create as src/components/search/SearchResults.tsx.
Wire it into the search page alongside the FilterBar.
```

---

## Phase 5: Advanced Filter Builder

### Prompt 5.1 — AdvancedFilterBuilder Component

```
Create an AdvancedFilterBuilder component inspired by Airtable's
filter workflow constructor. This is for power users who need
complex queries.

Layout: Each filter is a horizontal row that reads like a sentence:

  Where [NAICS ▾]        [is any of ▾]     [541512, 541511  ▾]  [×]
  And   [Set-Aside ▾]    [is ▾]            [SDVOSB           ▾]  [×]
  And   [Response Date ▾] [is before ▾]    [2026-04-01         ]  [×]
  [+ Add filter]

Requirements:

1. Toggle access: A text link "Advanced filters" below the primary
   filter bar toggles this panel open/closed. When open, it appears
   between the primary filters and the chip bar.

2. Each row has 4 columns:
   - Conjunction: "Where" (first row, not editable), "And"/"Or"
     (subsequent rows, toggleable)
   - Field selector: dropdown of all filterable fields:
     NAICS Code, Set-Aside Type, Agency, Notice Type, Response Deadline,
     Posted Date, Place of Performance, Award Value, Solicitation Number,
     Active Status
   - Operator selector: context-dependent operators:
     - Text fields: "contains", "does not contain", "is exactly",
       "starts with"
     - Multi-value fields (NAICS, agency): "is any of", "is none of",
       "is exactly"
     - Date fields: "is before", "is after", "is between",
       "in the next X days"
     - Number fields (value): "greater than", "less than", "between"
     - Boolean (active): "is true", "is false"
   - Value input: context-dependent:
     - Multi-value fields: opens the SearchableDropdownFilter
     - Date fields: date picker
     - Number fields: number input
     - Text fields: text input
   - Remove button (×) to delete the row

3. "+ Add filter" button at the bottom adds a new row.

4. "Apply" button (right-aligned, prominent) to execute the query.
   Show preview count on the button: "Apply — 247 results"
   This uses the facet count API to preview before applying.

5. "Reset" link to clear the advanced builder.

6. When the advanced builder is active, its conditions REPLACE the
   primary filter bar state (they're two views of the same state).
   If a user sets NAICS in the primary bar, it appears as a row in
   the advanced builder, and vice versa.

7. Limit to 10 filter rows maximum (show "(10/10 filters)" counter).

8. Animation: rows slide in/out smoothly when added/removed.

9. Accessibility:
   - Each row is a fieldset with legend "Filter condition N"
   - All dropdowns and inputs properly labeled
   - Focus moves to the new row when "+ Add filter" is clicked

Create as src/components/filters/AdvancedFilterBuilder.tsx.
```

---

## Phase 6: NAICS Tree Selector

### Prompt 6.1 — NAICS Hierarchy Data & API

```
Create support for hierarchical NAICS code selection.

NAICS codes form a 5-level hierarchy:
  54      Professional, Scientific, and Technical Services
  541     Professional, Scientific, and Technical Services
  5415    Computer Systems Design and Related Services
  54151   Computer Systems Design and Related Services
  541512  Computer Systems Design Services

Backend requirements:

1. Create a GET /api/v1/naics endpoint that returns the NAICS
   hierarchy tree:
   [
     {
       "code": "54",
       "label": "Professional, Scientific, and Technical Services",
       "level": 2,
       "children": [
         {
           "code": "541",
           "label": "Professional, Scientific, and Technical Services",
           "level": 3,
           "children": [...]
         }
       ]
     },
     ...
   ]
   This can be a static JSON file served from the API or S3, since
   NAICS codes change only every 5 years. Use the 2022 NAICS codes.

2. Find or create the NAICS hierarchy data. The Census Bureau publishes
   this. Generate a JSON file from the official NAICS 2022 data.
   Store it in the backend as a static resource.

3. The /api/v1/naics endpoint should support a ?search=query parameter
   that filters the tree to only show branches containing matches
   (by code number or label text). Return ancestors of matched nodes
   so the tree structure is maintained.

4. Add a "naics_label" field to the opportunities API response if
   not already present, so the frontend can show "541512 — Computer
   Systems Design Services" without a separate lookup.

Frontend requirements:

5. Create a NaicsTreeSelector component that provides two interaction
   modes in a single dropdown:

   a. Search mode (default): A search input at the top. Typing "541"
      or "computer" filters the tree to matching nodes with their
      ancestors visible and expanded. Matching text highlighted.

   b. Browse mode: Below the search, show a collapsible tree.
      Start with only 2-digit sectors visible. Click to expand.
      Each node has a checkbox.

6. Selecting a parent does NOT auto-select children (they're different
   NAICS codes and contracting officers use specific ones). But show
   a small "Select all under 5415" action link next to parent nodes.

7. Show result counts at each level from facet data:
   "54 Professional Services (2,341)"

8. Selected codes appear as flat chips in the FilterChipBar:
   "NAICS: 541512" "NAICS: 541511"

9. Support the indeterminate checkbox state (dash icon) on parent
   nodes when some but not all children are selected.

10. Lazy-load children when a 2-digit sector is expanded for the first
    time (don't load the entire tree upfront).

Create the NAICS data file, API endpoint, and NaicsTreeSelector component.
Wire it into the FilterBar to replace the current flat NAICS dropdown.
```

---

## Phase 7: Saved Searches

### Prompt 7.1 — Saved Searches Backend

```
Create the backend infrastructure for saved searches.

Database:

1. Create a saved_searches table:
   - id: UUID primary key
   - user_id: FK to users table
   - name: varchar(255) not null
   - filters: JSONB (stores the full filterState object)
   - alert_frequency: enum('off', 'daily', 'weekly') default 'off'
   - last_alerted_at: timestamp
   - result_count: integer (cached count from last run)
   - created_at: timestamp
   - updated_at: timestamp

2. Create a migration for this table.

API endpoints:

3. POST /api/v1/saved-searches
   - Body: { name, filters, alert_frequency }
   - Creates a new saved search for the authenticated user
   - Returns the created saved search with ID

4. GET /api/v1/saved-searches
   - Returns all saved searches for the authenticated user
   - Include current result_count (re-computed or cached)
   - Ordered by updated_at descending

5. PUT /api/v1/saved-searches/:id
   - Update name, filters, or alert_frequency
   - Verify ownership (user_id matches authenticated user)

6. DELETE /api/v1/saved-searches/:id
   - Soft delete or hard delete (your judgment based on project patterns)
   - Verify ownership

7. POST /api/v1/saved-searches/:id/run
   - Executes the saved search and returns results
   - Updates last_alerted_at and result_count
   - Returns the same shape as the regular search endpoint

Limits:
- Free tier: 3 saved searches max
- Paid tier: unlimited (enforce in middleware based on user tier)
- Return 403 with clear message when limit reached

Follow existing patterns for auth middleware, database access,
error handling, and response format.
```

### Prompt 7.2 — Saved Searches Frontend

```
Create the frontend UI for saved searches.

Requirements:

1. "Save Search" button: appears in the FilterBar next to the
   keyword search when any filters are active. Icon: bookmark or
   floppy disk.

2. Clicking "Save Search" opens a popover/small modal:
   - Name input (pre-populated with smart suggestion based on active
     filters, e.g., "DOD IT Services, SDVOSB" or "Army Cybersecurity")
   - Alert frequency: radio buttons for Off / Daily / Weekly
   - "Save" button
   - Show limit warning for free tier: "2 of 3 saved searches used"

3. Saved search pills: Below the keyword search (or above the primary
   filters), show the user's saved searches as clickable pills:
   [DOD IT Services ▾] [Army Cybersecurity ▾] [My Daily Check ▾]
   - Clicking a pill loads its filters into the FilterBar
   - Small dropdown on each pill with: "Edit name", "Update filters"
     (saves current filters to this search), "Change alerts",
     "Delete"
   - Active/highlighted state when a saved search is currently loaded

4. Saved searches page/section (accessible from user menu):
   - List of all saved searches with:
     - Name
     - Filter summary (brief text like "NAICS: 541512, SDVOSB, Army")
     - Current match count
     - Alert frequency setting
     - Last run date
     - "Run" button to execute and navigate to results
     - Edit/Delete actions
   - Empty state: explain what saved searches do with a CTA to
     "Set up your first search"

5. When a saved search with a relative deadline preset ("next 30 days")
   is loaded, compute the actual dates from today. Show a subtle
   indicator that dates are relative: "Deadline: next 30 days
   (recalculated daily)".

6. Free tier limit enforcement:
   - When at limit (3/3), the "Save Search" button shows a tooltip
     "Upgrade to Pro for unlimited saved searches" and the save
     modal shows an upgrade CTA
   - Don't disable the button entirely — let them click and see the
     limit message

Create as src/components/search/SavedSearches.tsx and supporting components.
Wire into the FilterBar and search page.
```

---

## Phase 8: Integration & Polish

### Prompt 8.1 — Search Page Assembly & Wiring

```
Wire together all the components from Phases 1-7 into the main
search page, replacing the existing search/filter UI.

Requirements:

1. The search page layout should be:
   - FilterBar (keyword + filters + chips + results toolbar)
   - SearchResults (cards + pagination)
   - All connected via useFilterState and useFacetCounts hooks

2. Data flow:
   - useFilterState manages all filter state and URL sync
   - useFacetCounts provides counts to all filter dropdowns
   - Search results are fetched via the existing search API endpoint
     using the filter state from useFilterState.toQueryParams()
   - Changing any filter triggers: facet count refetch + search
     results refetch (with debouncing per prompt 1.3)

3. Loading states:
   - Initial page load: skeleton cards + shimmer on filter counts
   - Filter change: semi-transparent overlay on results + shimmer
     on result count, while keeping previous results visible
   - Keyword search: show spinner in the search input

4. Error states:
   - API error: show inline error message with "Retry" button
   - Network offline: show "You appear to be offline" banner

5. Performance:
   - Memoize filter components to avoid re-rendering on every
     keystroke in the search input
   - Use React.memo or useMemo on OpportunityCard list
   - Ensure only the affected components re-render when a single
     filter changes

6. Preserve any existing functionality that users depend on:
   - Direct links to search with filters (URL params)
   - "View on SAM.gov" links on results
   - SAM.gov attribution
   - Any existing saved opportunity functionality

7. Page title updates: "14,832 results for 'cybersecurity' — GovTrove"
   or "Search Opportunities — GovTrove" when no search is active.

8. Make sure the existing route/URL structure isn't broken.
   If the current search page is at /search or /opportunities,
   keep that path and update the component.

Test the full flow:
- Load page with no filters → see all results
- Add keyword → results filter, counts update
- Add NAICS filter → chip appears, counts update, results filter
- Add deadline preset → chip appears, results filter
- Remove a chip → results expand
- Clear all → back to unfiltered
- Change sort → results reorder, page resets to 1
- Navigate to page 3 → URL updates, scroll to top
- Copy URL and open in new tab → same filters/page/sort applied
- Back button → previous filter state restored
```

### Prompt 8.2 — Mobile Responsive Pass

```
Do a mobile-responsive pass on all the search/filter components
created in the previous prompts.

Requirements for screens <768px:

1. Keyword search: full width, always visible at top.

2. Primary filters: collapse into a single "Filters" button with
   badge showing count of active filters. Tapping opens a full-screen
   modal with all filters stacked vertically:
   - Each filter category as an expandable accordion section
   - "Apply filters" sticky button at bottom showing result count:
     "Show 247 results"
   - "Clear all" link at top of modal
   - Close (×) button at top-right

3. Filter chips: horizontal scroll with overflow, not wrapping.
   Touch-friendly chip sizes (min 44px height for tap targets).

4. Results toolbar: stack into two rows:
   - Row 1: "14,832 results"
   - Row 2: Sort dropdown + view toggle

5. OpportunityCards: adjust for narrow screens:
   - Full width cards
   - Stack metadata vertically instead of inline
   - Larger tap targets for star/checkbox
   - Deadline and urgency badge remain prominent

6. Pagination: simplify to "Previous / Page X of Y / Next" on mobile.
   Hide individual page number links.

7. Advanced filter builder: on mobile, each filter row stacks its
   4 columns vertically (Field, Operator, Value, Remove) instead of
   horizontal.

8. Saved search pills: horizontal scroll, not wrapping.

9. Test on 375px width (iPhone SE) and 414px (iPhone 14).
   No horizontal scrolling on the page body.
   All interactive elements have minimum 44x44px touch targets.

Review all components created in Phases 2-7 and apply mobile styles.
Use the existing responsive breakpoint approach in the project.
```

### Prompt 8.3 — Accessibility Audit & Fixes

```
Do an accessibility audit of all search/filter components and fix
any issues. GovTrove serves government contractors who may need
Section 508 compliance.

Audit checklist:

1. Keyboard navigation (test without a mouse):
   - Tab order follows visual order through all interactive elements
   - Filter dropdowns open with Enter/Space, close with Escape
   - Arrow keys navigate within dropdown lists
   - Filter chips can be dismissed with Enter/Space on their × button
   - Focus is properly trapped inside modals
   - Focus returns to trigger when dropdowns/modals close
   - Pagination navigable via keyboard
   - Cards navigable and activatable via keyboard

2. Screen reader testing (test with VoiceOver or NVDA):
   - Search region has role="search" and aria-label
   - Filter dropdowns use role="combobox" with aria-expanded
   - Option lists use role="listbox" with role="option"
   - Active filter count announced when filters change:
     aria-live="polite" region: "247 opportunities found"
   - Each card is an <article> with meaningful aria-label
   - Urgency badges have aria-label (not just color/emoji)
   - Loading states announced: "Loading results..."

3. Visual accessibility:
   - All text meets 4.5:1 contrast ratio (WCAG AA)
   - UI components (buttons, inputs, badges) meet 3:1 contrast
   - Focus rings visible on all interactive elements (never hidden)
   - Nothing relies on color alone — icons/text/borders supplement
   - Urgency badges have text labels, not just color

4. Specific fixes:
   - Add skip link: "Skip to search results" at top of page
   - NAICS tree: role="tree" with role="treeitem", aria-expanded
   - Facet counts: "(1,247)" should be aria-label="1247 results"
     not just visual text
   - Filter chip bar: role="list" with role="listitem"
   - Bulk action bar: role="toolbar" with aria-label

Run through every component and fix any issues found. List the
fixes made for review.
```

---

## Execution Notes

**Estimated sessions:** 12 Claude Code sessions (one per prompt)

**Dependency order:**
- Phase 1 (1.1 → 1.2 → 1.3) must be done first — everything depends on these
- Phase 2 (2.1-2.4) can be done in parallel after Phase 1
- Phase 3 (3.1) depends on all of Phase 2
- Phase 4 (4.1 → 4.2) can be done in parallel with Phase 2-3
- Phase 5 (5.1) depends on Phase 2 components
- Phase 6 (6.1) depends on Phase 1 & 2
- Phase 7 (7.1 → 7.2) depends on Phase 1 & 3
- Phase 8 (8.1 → 8.2 → 8.3) is the final integration, depends on everything

**What's intentionally deferred:**
- Table view / map view (stubs only in this implementation)
- Split-pane detail preview (separate feature)
- Email alerts execution (Phase 2 from the feature timeline — separate)
- "Popular filters" quick-filter row (you mentioned deferring this)
- Export to CSV (paid feature, button placed but not wired)