# Frontend E2E Test Plan

Comprehensive plan for end-to-end testing of the search and filter system. Covers the full user journey from landing page through search, filtering, pagination, and opportunity interaction.

---

## 1. Landing Page → Search Transition

### 1.1 Hero Mode Rendering
| Test | Expectation |
|------|-------------|
| Page loads at `/` with no URL params | Hero layout: large logo, subtitle, hero count, large search input, "What's New Today?" link, quick filter chips |
| Hero count populates | Facet API fires on mount, first non-zero `total` populates hero count and stays stable even if facets update later |
| Search input has autofocus | Input is focused on load, cursor blinking |
| Search input has submit button | Large size input shows magnifying glass submit button inside the field |

### 1.2 Quick Filter Chips
| Test | Expectation |
|------|-------------|
| Click "IT Services" chip | Chip highlights (accent border/bg), keyword set to "IT services", search fires, page transitions to results mode |
| Click "IT Services" again (toggle off) | Chip deactivates, keyword cleared, if no other filters active → returns to hero mode |
| Click "Small Business" chip | `setAside` includes "SBA", search fires |
| Click "Closing This Week" chip | `deadlinePreset` set to "7", search fires |
| Click keyword chip then different keyword chip | First chip deactivates, second activates, keyword replaced (not appended) |
| Click "Small Business" then "Closing This Week" | Both active simultaneously (different filter types), both reflected in URL |

### 1.3 Hero → Results Transition
| Test | Expectation |
|------|-------------|
| Type "cybersecurity" and press Enter | Page transitions: logo shrinks, hero elements hide, FilterBar + results appear with slide animation |
| Type 1 character and press Enter | No search fires (minimum 2 chars for `hasActiveFilters`) |
| Type "ab" and press Enter | Search fires (2 chars meets minimum) |
| Browser back after search | Returns to hero mode, URL params cleared, results reset |

### 1.4 URL-Driven Initial Load
| Test | Expectation |
|------|-------------|
| Navigate directly to `/?q=test` | Page loads in results mode, search fires immediately with keyword "test" |
| Navigate to `/?naics=541511&set_aside=SBA` | Results mode, both filters active, chips shown, results loaded |
| Navigate to `/?psc=R` | Results mode, PSC filter active, search fires |
| Navigate to `/?active=false` | "Including inactive" chip shown, results include inactive opportunities |
| Navigate to `/?page=3&q=test` | Results load on page 3 |

---

## 2. Keyword Search

### 2.1 Search Input Behavior
| Test | Expectation |
|------|-------------|
| Type keyword and press Enter | Search fires, results appear, URL updates with `q=` param |
| Type keyword and click Search button (desktop) | Same as Enter |
| Clear input with X button | Keyword cleared; if no other filters → returns to hero mode |
| Type while loading | Input remains responsive, previous search completes or new one overwrites |
| Paste long text (>500 chars) | Input accepts it, search fires normally |

### 2.2 Keyword Highlighting
| Test | Expectation |
|------|-------------|
| Search "cybersecurity" | Title text matching "cybersecurity" is highlighted in results cards |
| Search "cyber security" (two words) | Each word highlighted independently in title |
| Search phrase with special regex chars | No crash, characters treated as literals |

### 2.3 Search Help Popover (Desktop)
| Test | Expectation |
|------|-------------|
| Click help icon next to search | Popover appears with search tips: exact phrase, OR, -word, what's searched, filter info |
| Click outside popover | Popover closes |
| Scroll page while popover open | Popover closes |

---

## 3. NAICS Tree Selector

### 3.1 Dropdown Behavior
| Test | Expectation |
|------|-------------|
| Click "NAICS" button | Dropdown opens with search input + tree of root nodes (sectors) |
| Click "NAICS" again | Dropdown closes |
| Click outside dropdown | Dropdown closes |
| Press Escape | Dropdown closes, focus returns to trigger button |
| Scroll page while dropdown open | Dropdown closes |

### 3.2 Tree Navigation
| Test | Expectation |
|------|-------------|
| Click chevron on "31-33 Manufacturing" | Node expands showing subsectors (311, 312, ...) |
| Click chevron again | Node collapses |
| Click row text (not chevron) on parent node | Row expands/collapses (same as chevron) |
| Expand deeply: sector → subsector → industry group → industry → national | All 5 levels render with increasing indentation |

### 3.3 Selection Logic
| Test | Expectation |
|------|-------------|
| Check a leaf node (e.g., 541511) | Checkbox checked, chip appears below tree, trigger button shows "NAICS (1)" |
| Check a parent node (e.g., 5415) | Parent checked, all children show checked state, only parent code in `selected` array (not all leaves) |
| Check parent, then uncheck one child | Parent goes indeterminate, unselected child unchecked, remaining children stay checked |
| Check all children of a parent individually | Parent auto-shows as checked (full coverage) |
| Uncheck a fully-checked parent | All children and parent removed from selection |
| Select multiple unrelated codes | Each shows as chip, trigger shows count "NAICS (3)" |

### 3.4 Search Within Dropdown
| Test | Expectation |
|------|-------------|
| Type "541" in search | Shows matching codes: parents (54, 541) + match (541) + children (5411, 5412, ...) |
| Type "information tech" | Keyword search matches titles containing "information" and "tech" |
| Type "999999" (no match) | "No matches" message shown |
| Select code from search results | Code added to selection, ancestor nodes auto-expand in tree view |
| Clear search input | Returns to tree view with current selections preserved |

### 3.5 Facet Counts
| Test | Expectation |
|------|-------------|
| NAICS codes show facet counts | Each node shows count from current search results, aggregated up to parents |
| Change keyword → facets update | NAICS counts in dropdown update after debounce (300ms) |
| Facets loading | Dropdown shows "Loading..." text |

### 3.6 Chips & Clear
| Test | Expectation |
|------|-------------|
| Click X on a chip inside dropdown | Code removed from selection |
| Click "Clear all" inside dropdown | All NAICS selections removed |
| Click X on NAICS chip in FilterChipBar | Individual code removed |

### 3.7 Range Sector Handling (NAICS-specific)
| Test | Expectation |
|------|-------------|
| Expand "31-33 Manufacturing" | Shows subsectors under the combined range (311, 312, 313, ..., 339) |
| Select "31-33" | All manufacturing leaf codes covered, search uses expanded leaf codes in API call |
| Facet count for "31-33" | Aggregates counts from all codes starting with 31, 32, or 33 |

---

## 4. PSC Tree Selector

### 4.1 Dropdown Behavior
Same as NAICS dropdown tests (open, close, escape, outside click, scroll).

### 4.2 Tree Navigation
| Test | Expectation |
|------|-------------|
| Click chevron on "R - Professional..." | Expands showing 2-char groups (R4, R6, R7, ...) |
| Expand "R4" | Shows 4-char leaf codes (R401, R402, ...) |
| Three levels only | Max depth is level 1 → level 2 → level 4, with appropriate indentation |

### 4.3 Selection Logic
| Test | Expectation |
|------|-------------|
| Select "R" (1-char category) | All R-prefixed leaf codes covered, trigger shows "PSC (1)" |
| Select "R4" (2-char group) | All R4xx codes covered |
| Select leaf "R499" | Only that specific code selected |
| Select "R", then deselect "R499" | "R" becomes indeterminate, R499 removed, all other R-children still checked |
| Mix NAICS and PSC selections | Both filters active independently, both appear as chips |

### 4.4 Search Within Dropdown
| Test | Expectation |
|------|-------------|
| Type "R4" | Code prefix search: shows R (parent), R4 (match), R401/R402/... (children) |
| Type "R" (single letter) | Shows R category as match + all R-children |
| Type "professional" | Keyword search matches PSC titles containing "professional" |
| Type "r4" (lowercase) | Case-insensitive: same results as "R4" |

### 4.5 PSC-Specific Edge Cases
| Test | Expectation |
|------|-------------|
| PSC codes with no 2-char parent (e.g., some product codes) | Still render correctly at appropriate tree level |
| Very long PSC title in dropdown | Title truncated with ellipsis, not breaking layout |
| PSC chip label in FilterChipBar | Shows "PSC R499" with title from PSC_TITLE_BY_CODE lookup |

---

## 5. Set-Aside Filter (SearchableDropdownFilter)

### 5.1 Dropdown Behavior
| Test | Expectation |
|------|-------------|
| Click "Set-Aside" button | Dropdown opens with search + list of set-aside options |
| Selected items pinned to top | Previously selected set-asides appear above unselected ones |
| Unselected items sorted by count | Higher-count set-asides appear first |

### 5.2 Selection
| Test | Expectation |
|------|-------------|
| Check "SBA" | Checkbox checked, pending selection shown |
| Close dropdown | `onSelectionChange` fires with new selection, search auto-fires |
| Check multiple set-asides | Multiple selections, trigger shows "Set-Aside (2)" |
| Uncheck a selected item | Removed from pending, finalized on close |

### 5.3 Search
| Test | Expectation |
|------|-------------|
| Type "small" | Filters to set-asides matching "small" in label |
| Type search, select item, clear search | Selected item persists, full list restored |

### 5.4 Facet Counts
| Test | Expectation |
|------|-------------|
| Set-aside options show counts | Each option shows count from `facets.set_aside` |
| Option with count 0 | Option still visible (not hidden) |

---

## 6. Agency Filter (SearchableDropdownFilter)

### 6.1 Current State (Disabled)
| Test | Expectation |
|------|-------------|
| Agency filter button | Shows as disabled (greyed out, not clickable) in both desktop and mobile |
| Agency filter in AdvancedFilterBuilder | Selectable as field, renders SearchableDropdownFilter |

### 6.2 When Enabled (Future)
| Test | Expectation |
|------|-------------|
| Agency options from facets | Options populated from `facets.agency`, single-select (not multi) |
| Select agency | `department` filter set to single value, search fires |

---

## 7. Deadline Filter (EnhancedDeadlineFilter)

### 7.1 Preset Buttons
| Test | Expectation |
|------|-------------|
| Click "7d" preset | Button highlights, `deadlinePreset` = "7", API gets `deadline_from=today, deadline_to=today+7` |
| Click "30d" while "7d" active | "7d" deactivates, "30d" activates |
| Click active preset again | Preset cleared (toggle off), "All" re-selected |
| Click "quarter" | Deadline computed as end of current calendar quarter |

### 7.2 Custom Range
| Test | Expectation |
|------|-------------|
| Toggle "Custom range..." | Date inputs appear for from/to |
| Set "from" date | `deadlineFrom` set, preset cleared |
| Set "from" and "to" dates | Both set, search uses exact range |
| Set "to" before "from" | "to" input has `min` constraint set to "from" value |
| Both dates in past | Allowed (user might want historical search with activeOnly off) |

### 7.3 Display Label
| Test | Expectation |
|------|-------------|
| Preset "7" active | Trigger shows "Due: next 7 days" |
| Custom from+to | Trigger shows "Due: Jan 15 – Feb 28" format |
| Only "from" set | Trigger shows "Due: from Jan 15" |
| Only "to" set | Trigger shows "Due: until Feb 28" |
| No deadline set | Trigger shows "Deadline" |

---

## 8. Notice Type Filter (SimpleToggleFilter)

### 8.1 Toggle Behavior
| Test | Expectation |
|------|-------------|
| Click "Solicitation" checkbox | Added to `noticeType` array |
| Click again | Removed from array |
| "Select all" button | All notice types selected |
| "Clear all" button | All notice types cleared |
| Options with count 0 | Shown but disabled (not clickable) |

### 8.2 Facet Counts
| Test | Expectation |
|------|-------------|
| Notice types show counts | Each option shows count from `facets.notice_type` |
| Selecting a type updates other facet counts | Non-notice-type facets (NAICS, set-aside, etc.) update to reflect the filtered set |

---

## 9. Active Only Toggle

| Test | Expectation |
|------|-------------|
| Default state | Active Only is ON (true), no chip shown (it's the default) |
| Toggle off | Switch flips, "Including inactive" chip appears in FilterChipBar |
| Toggle off → API behavior | `deadline_from` no longer auto-set to today, `active=false` in URL |
| Toggle back on | Chip removed, default behavior restored |
| URL `?active=false` | Toggle shows OFF, inactive results included |

---

## 10. More Filters Panel (Posted Date)

### 10.1 Panel Behavior
| Test | Expectation |
|------|-------------|
| Click "More" button | Side panel slides in from right |
| Click backdrop or X | Panel closes |
| Escape key | Panel closes |
| Body scroll locked | Background page doesn't scroll while panel open |

### 10.2 Posted Date Range
| Test | Expectation |
|------|-------------|
| Set posted "from" date | `postedFrom` set, chip appears: "Posted: from Jan 1" |
| Set posted "from" and "to" | Both set, chip: "Posted: Jan 1 – Feb 28" |
| Click "Reset" in panel | Posted date fields cleared |
| Click "Apply" | Panel closes, filters applied |

### 10.3 Stub Fields
| Test | Expectation |
|------|-------------|
| Solicitation Number input | Visible but disabled / marked as coming soon |
| Place of Performance | Visible but disabled |
| Award Value Range | Visible but disabled |

---

## 11. Advanced Filter Builder

### 11.1 Visibility Toggle
| Test | Expectation |
|------|-------------|
| Click "Advanced filters" link | Builder panel appears with chevron flipped |
| Click again | Panel hides |

### 11.2 Row Management
| Test | Expectation |
|------|-------------|
| Initial state with active filters | Builder pre-populates rows matching active normal filters (e.g., if NAICS is set, a NAICS row appears) |
| Click "Add filter" | New empty row added with "Select field..." dropdown |
| Add 10 rows | "Add filter" button disabled at max |
| Click X on a row | Row animates out, associated filter cleared |
| Click "Reset" | All rows cleared, all filters reset to defaults |

### 11.3 Field Selection
| Test | Expectation |
|------|-------------|
| Select "NAICS Code" field | Operator dropdown shows "is any of", value shows NaicsTreeSelector |
| Select "PSC Code" field | Operator shows "is any of", value shows PscTreeSelector |
| Select "Set-Aside Type" | Value shows SearchableDropdownFilter |
| Select "Notice Type" | Value shows SimpleToggleFilter |
| Select "State" | Value shows state SearchableDropdownFilter |
| Select "Response Deadline" | Operator shows 4 options (in_next_x_days, is_after, is_before, is_between), appropriate date inputs |
| Select "Posted Date" | Similar to deadline but 3 operators |
| Select "Active Status" | Radio buttons: "Active only" / "All (including inactive)" |
| Select "Award Value" (coming soon) | "Coming soon" text shown |
| Select "Solicitation Number" (coming soon) | "Coming soon" text shown |

### 11.4 Sync with Normal Filters
| Test | Expectation |
|------|-------------|
| Set NAICS in normal FilterBar, open Advanced | NAICS row pre-populated in builder |
| Set PSC in normal FilterBar, open Advanced | PSC row pre-populated |
| Change filter in Advanced builder | Normal FilterBar reflects the change (shared state) |
| Change filter in normal FilterBar while Advanced open | Advanced builder rows stay in sync |
| Remove filter via chip while Advanced open | Corresponding builder row cleared or removed on next derive |

### 11.5 Operator Changes
| Test | Expectation |
|------|-------------|
| Deadline: switch from "in_next_x_days" to "is_after" | Preset cleared, date input appears |
| Deadline: switch from "is_between" to "is_after" | Both date fields cleared, single input shown |
| Posted date: switch operators | Date fields cleared on operator change |

### 11.6 Field Exclusivity
| Test | Expectation |
|------|-------------|
| Row 1 selects "NAICS", Row 2 dropdown | "NAICS Code" is disabled/greyed in Row 2's field list |
| Remove Row 1 (NAICS) | "NAICS Code" becomes available in remaining rows |

---

## 12. Filter Chip Bar

### 12.1 Chip Rendering
| Test | Expectation |
|------|-------------|
| Keyword filter active | Chip shows: `"cybersecurity"` (quoted) |
| NAICS 541511 selected | Chip shows: `NAICS 541511 — Custom Computer Programming` |
| PSC R499 selected | Chip shows: `PSC R499 — [PSC title]` |
| Set-Aside SBA selected | Chip shows human-readable label from SET_ASIDE_LABELS |
| Deadline preset "7" | Chip shows: `Deadline: next 7 days` |
| Custom deadline from+to | Chip shows: `Deadline: Jan 15 – Feb 28` |
| Posted from date | Chip shows: `Posted: from Jan 15` |
| Active Only off | Chip shows: `Including inactive` |
| State filter "VA" | Chip shows: `State: VA` |
| Multiple notice types | One chip per selected type |

### 12.2 Chip Interactions
| Test | Expectation |
|------|-------------|
| Click X on keyword chip | Keyword cleared, chip removed |
| Click X on NAICS chip | That specific NAICS code removed (not all NAICS) |
| Click X on "Including inactive" chip | activeOnly set back to true (default) |
| Click "Clear all" | All filters reset to defaults |
| "Clear all" only visible with 2+ filters | Hidden with 0-1 active filters |

### 12.3 Overflow Behavior
| Test | Expectation |
|------|-------------|
| 10+ filter chips | Chip bar caps at ~2 rows, shows "+N more" button |
| Click "+N more" | Chip bar expands to show all chips, "Show less" button appears |
| Click "Show less" | Collapses back to 2 rows |

### 12.4 Stickiness
| Test | Expectation |
|------|-------------|
| Scroll down in results | Chip bar sticks to top of viewport (sticky positioning) |
| Semi-transparent background | Content scrolling behind chips is slightly visible |

---

## 13. Filter ↔ URL Synchronization

### 13.1 State → URL
| Test | Expectation |
|------|-------------|
| Set keyword "test" | URL becomes `/?q=test` |
| Set NAICS 541511 | URL becomes `/?naics=541511` |
| Set PSC R,R4 | URL becomes `/?psc=R,R4` (comma-separated) |
| Set set-aside SBA,WOSB | URL becomes `/?set_aside=SBA,WOSB` |
| Set deadline preset 30 | URL becomes `/?deadline=30` |
| Set activeOnly false | URL becomes `/?active=false` |
| Set state VA | URL becomes `/?state=VA` |
| Set multiple filters | All params present in URL |
| Clear all filters | URL becomes `/` (all params removed) |

### 13.2 URL → State
| Test | Expectation |
|------|-------------|
| Navigate to `/?q=hello&naics=541511` | Both filters populated, search fires |
| Navigate to `/?set_aside=SBA,WOSB` | Both set-asides selected in filter |
| Navigate to `/?psc=R` | PSC "R" selected |
| Navigate to `/?active=false` | Active Only toggle shows OFF |
| Navigate to `/?deadline=30` | Deadline shows "30d" preset active |
| Navigate to `/?page=2&sort=relevance&order=desc&q=test` | Page 2, sort by relevance desc |
| Invalid param values (e.g., `?page=abc`) | Graceful fallback to defaults |

### 13.3 History Management
| Test | Expectation |
|------|-------------|
| First search (hero → results) | URL pushed (browser back returns to hero) |
| Subsequent filter changes | URL replaced (browser back goes to pre-search state, not each filter tweak) |
| Browser back from results | Returns to hero/landing page |
| Browser forward | Returns to results with filters |

### 13.4 Secondary Params
| Test | Expectation |
|------|-------------|
| Sort/page only in URL when other params exist | `/?sort=relevance` alone does NOT appear; `/?q=test&sort=relevance` does |
| Clear keyword but keep sort | Sort param removed from URL (secondary field rule) |

---

## 14. Search Execution & Auto-Search

### 14.1 Search Triggers
| Test | Expectation |
|------|-------------|
| Press Enter in search input | Immediate search |
| Click Search button | Immediate search |
| Click quick filter chip | Immediate search |
| Toggle any non-keyword filter | Auto-search after 300ms debounce |
| Change NAICS selection | Auto-search fires |
| Change PSC selection | Auto-search fires |
| Change set-aside | Auto-search fires |
| Change deadline preset | Auto-search fires |

### 14.2 Debounce & Dedup
| Test | Expectation |
|------|-------------|
| Toggle filter A, then B within 300ms | Only one search fires (with both filters applied) |
| Same params as last search | No duplicate API call (lastSearchedRef check) |
| Change filter, then change back within 300ms | Debounce cancels, params same as before → no search |

### 14.3 Page Reset
| Test | Expectation |
|------|-------------|
| On page 3, change any filter | Page resets to 1 |
| On page 3, change sort | Page resets to 1 |
| On page 3, change page size | Page resets to 1 |

---

## 15. Facet Counts

### 15.1 Loading & Updates
| Test | Expectation |
|------|-------------|
| Initial load | Facets fetched for default filter state |
| Change keyword | Facets refresh (debounced 300ms) |
| Change filter | Facets refresh with new params |
| Facets loading | Filter dropdowns show loading indicator |
| Stale data while loading | Previous facet counts visible until new ones arrive |

### 15.2 Cross-Filter Facet Behavior
| Test | Expectation |
|------|-------------|
| Select NAICS 541511, check PSC facets | PSC facet counts reflect opportunities with NAICS 541511 |
| Select Set-Aside SBA, check NAICS facets | NAICS counts reflect SBA-filtered set |
| Facets exclude own filter | API uses exclude pattern: NAICS facets computed without NAICS filter applied (shows all NAICS, not just selected) |

### 15.3 Abort Handling
| Test | Expectation |
|------|-------------|
| Rapidly toggle filters 5 times | Intermediate facet requests aborted, only final one completes |
| AbortError not shown to user | No error banner for cancelled requests |

---

## 16. Search Results Display

### 16.1 Results Rendering
| Test | Expectation |
|------|-------------|
| Results returned | Cards render with title, agency, NAICS badge, set-aside badge, solicitation number, deadline, description snippet |
| Keyword highlighting in title | Matched text highlighted |
| Keyword highlighting in description | Snippet extracted around first keyword match, max ~200 chars |
| Long title | Truncated or wrapped properly |
| Missing optional fields | Card renders without crashing (no agency, no NAICS, no set-aside, etc.) |

### 16.2 Result Count & Toolbar
| Test | Expectation |
|------|-------------|
| Results found | Toolbar shows "X,XXX results" |
| Zero results | ZeroResults component shown with message, tips, "Clear all filters" button |
| Loading state | Skeleton cards shown during initial load; overlay on subsequent loads |

### 16.3 Urgency Badges
| Test | Expectation |
|------|-------------|
| Opportunity posted in last 48 hours | Green "New" badge |
| Deadline in 0-2 days | Red "Closing Today/Tomorrow" badge |
| Deadline in 3-7 days | Amber "Closing Soon" badge |
| Deadline > 7 days | No urgency badge |

### 16.4 Deadline Display
| Test | Expectation |
|------|-------------|
| Deadline in 1 day | Red text, "1 day left" |
| Deadline in 5 days | Amber text, "5 days left" |
| Deadline in 30 days | Normal text, "30 days left" |
| No deadline | "No deadline" shown |
| Deadline passed | "Expired" or similar indication |

---

## 17. Pagination

### 17.1 Page Navigation
| Test | Expectation |
|------|-------------|
| Click page 2 | Page 2 results load, URL updates with `page=2` |
| Click "Next" | Next page loads |
| Click "Previous" | Previous page loads |
| First page | "Previous" disabled |
| Last page | "Next" disabled |
| Total < pageSize | No pagination shown |

### 17.2 Page Size
| Test | Expectation |
|------|-------------|
| Change page size to 50 | Results reload with 50 items, page resets to 1, stored in localStorage |
| Reload page | Page size restored from localStorage |
| Available sizes | 25, 50, 100 |

### 17.3 Mobile Pagination
| Test | Expectation |
|------|-------------|
| Mobile viewport | Simplified pagination: prev/next buttons + "X/Y" display |
| Page size selector on mobile | Still accessible |

---

## 18. Sort

### 18.1 Sort Dropdown
| Test | Expectation |
|------|-------------|
| Default sort (no keyword) | "Posted Date" (desc) |
| Search with keyword | Auto-selects "Relevance" sort |
| Click sort option | Results re-sort, URL updates |
| Click direction toggle | asc↔desc flips, results reload |
| Clear keyword while sorted by relevance | Sort resets to "Posted Date" |
| "Relevance" option only when keyword present | Option hidden from dropdown when no keyword |

### 18.2 Sort Persistence
| Test | Expectation |
|------|-------------|
| Set sort to "Response Deadline", close tab, reopen | Sort preference restored from localStorage |
| URL sort param overrides localStorage | `?sort=relevance` takes precedence over stored preference |

---

## 19. Save / Unsave Opportunities

### 19.1 Save Toggle
| Test | Expectation |
|------|-------------|
| Click star icon on unsaved opportunity | Star turns yellow, ID added to localStorage |
| Click star on saved opportunity | Star turns gray, ID removed |
| Reload page | Saved state persists |

### 19.2 Cross-Tab Sync
| Test | Expectation |
|------|-------------|
| Save in Tab A, check Tab B | Tab B shows opportunity as saved (storage event) |

### 19.3 Bulk Save
| Test | Expectation |
|------|-------------|
| "Save All" from bulk action bar | All visible results saved |
| Exceeds 500 limit | Oldest saved items evicted (FIFO), most recent 500 kept |

---

## 20. Mobile Filters Sheet

### 20.1 Sheet Behavior
| Test | Expectation |
|------|-------------|
| Tap "Filters" button (mobile) | Bottom sheet slides up, backdrop visible |
| Tap backdrop | Sheet closes |
| Press Escape | Sheet closes |
| Swipe interactions | Sheet stays open (no swipe-to-dismiss implemented) |
| Body scroll locked | Background page doesn't scroll |

### 20.2 Filter Sections
| Test | Expectation |
|------|-------------|
| NAICS section | NaicsTreeSelector in inline mode (full tree, no dropdown) |
| PSC section | PscTreeSelector in inline mode |
| Set-Aside section | SearchableDropdownFilter inline |
| Agency section | Disabled (matches desktop) |
| Deadline section | EnhancedDeadlineFilter |
| Notice Type section | SimpleToggleFilter |
| Active Only section | Toggle button |
| Posted Date section | Two date inputs (from/to) |

### 20.3 Footer Actions
| Test | Expectation |
|------|-------------|
| "Show X results" button | Shows current total, closes sheet on tap |
| "Reset" button | Clears all filters |
| "Reset" only visible when filters active | Hidden when filterCount = 0 |

---

## 21. Opportunity Card Interactions

### 21.1 Card Click
| Test | Expectation |
|------|-------------|
| Click card title | Navigates to `/opportunity/{id}` |
| Click card body (not title) | Also navigates to detail page |
| Middle-click / Ctrl+click | Opens in new tab |

### 21.2 Badge Display
| Test | Expectation |
|------|-------------|
| NAICS badge | Shows NAICS code in monospace |
| Set-aside badge | Shows set-aside code with appropriate color |
| No set-aside | Badge not rendered (no empty space) |

### 21.3 Description Snippet
| Test | Expectation |
|------|-------------|
| Long description | Truncated to ~200 chars with context around keyword |
| No description | Empty or "No description available" |
| Description with HTML entities | Properly escaped/rendered |

---

## 22. Zero Results State

| Test | Expectation |
|------|-------------|
| Search returns 0 results | ZeroResults component: icon, "No opportunities match" message |
| Keyword shown in message | If keyword active, displayed in the message |
| Filter suggestion | If filters active, suggests "Try removing some filters" |
| "Clear all filters" button | Clears all, returns to hero mode |
| Search tips | Displayed below zero results message |

---

## 23. Error Handling

| Test | Expectation |
|------|-------------|
| API returns 500 | Error message displayed, retry button shown |
| Click retry | Search re-fires with same params |
| Network offline | Error message, graceful degradation |
| Facet API fails | Filter counts may show stale data, no crash |
| API returns malformed JSON | Error caught, error message shown (not crash) |

---

## 24. Responsive / Viewport Tests

### 24.1 Desktop (≥768px)
| Test | Expectation |
|------|-------------|
| All filter triggers visible | NAICS, PSC, Set-Aside, Agency, Deadline, Notice Type, Active Only, More, Advanced |
| Dropdowns portal-positioned | Rendered at correct screen position via `createPortal` |
| Dropdown near right edge | Right-aligned to not overflow viewport |

### 24.2 Mobile (<768px)
| Test | Expectation |
|------|-------------|
| Filter row hidden | Only "Filters" button visible (with count badge) |
| Search button hidden | No separate "Search" button (Enter to submit) |
| Search help icon hidden | Not shown on mobile |
| Tap "Filters" | Mobile sheet opens with all filter sections inline |
| Result cards | Full-width, touch-friendly tap targets |

---

## 25. Cross-Filter Interactions

These tests verify that different filters work correctly together.

| Test | Expectation |
|------|-------------|
| NAICS + PSC | Both applied as AND condition, results match both |
| NAICS + Set-Aside + Keyword | All three combined, facets reflect the intersection |
| Deadline preset + activeOnly off | Custom deadline range used, inactive results included |
| Clear one filter while others active | Only that filter removed, others remain, results update |
| All filters set, then "Clear all" | Everything resets, returns to hero mode |
| Quick chip (keyword) + manual NAICS selection | Both active, both in URL, both in chips |

---

## 26. Leaf Expansion & API Params

These tests verify that UI selections translate correctly to API parameters.

| Test | Expectation |
|------|-------------|
| Select NAICS "5415" (parent) | API receives all 5415xx leaf codes (541511, 541512, ...) via `naics` param |
| Select PSC "R" (1-char parent) | API receives all R-prefixed leaf codes via `psc` param |
| Select PSC "R499" (leaf) | API receives just "R499" |
| Select NAICS "31-33" (range sector) | API receives all leaf codes under 31xxxx, 32xxxx, 33xxxx |
| Deadline preset "7" | API receives `deadline_from=today, deadline_to=today+7days` |
| Deadline preset "quarter" | API receives `deadline_to=end-of-quarter` |
| activeOnly true (default) + no deadline | API receives `deadline_from=today` (active filter) |

---

## 27. Known Bugs to Verify/Fix

| Bug | Location | Issue |
|-----|----------|-------|
| `hasActiveFilters` missing PSC | `SimpleSearchPage.tsx:56-68` | PSC filter not included in active filter check — selecting only PSC won't trigger search or transition to results mode |
| Dropdown scroll inside list may close dropdown | `useDropdownPosition.ts` scroll handler | Scrolling inside a dropdown's scrollable list (e.g., long NAICS tree) may trigger the scroll-close listener |

---

## 28. Performance Considerations

| Test | Expectation |
|------|-------------|
| NAICS tree with 1,000+ nodes | Tree renders without lag, expand/collapse smooth |
| PSC tree with 1,400+ nodes | Same as NAICS |
| Facet debounce (300ms) | No excessive API calls during rapid filter changes |
| Virtual scrolling in SearchableDropdownFilter | Long option lists (agencies) scroll smoothly, only visible items rendered |
| Large result set (100 items per page) | Cards render without jank |
| Memory: open/close dropdown 50 times | No memory leak from portal mounting/unmounting |

---

## Test Environment Notes

- **Framework recommendation**: Playwright (cross-browser, good mobile emulation)
- **API mocking**: Use Playwright route interception to mock API responses for deterministic tests
- **Viewport sizes**: Test at 1440px (desktop), 768px (tablet), 375px (mobile)
- **Browsers**: Chrome, Firefox, Safari (at minimum)
- **Test data**: Mock responses with known NAICS/PSC codes, facet counts, and result sets
