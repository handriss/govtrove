import { test, expect, Page, APIRequestContext } from '@playwright/test';

// These run against live production data, which changes every day. So nothing
// here asserts a fixed number — only invariants that must hold whatever the
// catalog contains. A test that asserts "coffee returns 12" is a test you will
// delete in a week.

const API = process.env.E2E_API_URL || 'https://api.govtrove.com/api';

/**
 * Reads the settled result count.
 *
 * The on-page headline is split across elements ("10" in one node, "results" in
 * a sibling span), so there is no single element to match. The document title
 * carries the same number — it is what the browser tab shows — and updates from
 * the same state, so it is both user-visible and a stable thing to assert on.
 */
async function headlineCount(page: Page): Promise<number> {
  await expect(page).toHaveTitle(/[\d,]+\s+results? for/, { timeout: 90_000 });

  // The title briefly reads "0 results" while the request is still in flight, so
  // the first match is not necessarily the settled one — reading it directly made
  // this flake. Wait for the same number twice in a row instead. Still returns a
  // real 0 when that is genuinely the answer.
  let last: number | null = null;
  await expect
    .poll(
      async () => {
        const m = (await page.title()).match(/([\d,]+)\s+results?/);
        if (!m) return false;
        const n = parseInt(m[1].replace(/,/g, ''), 10);
        const settled = last === n;
        last = n;
        return settled;
      },
      { timeout: 90_000, intervals: [500] },
    )
    .toBe(true);

  if (last === null) throw new Error(`no result count in title: ${await page.title()}`);
  return last;
}

/** Types a query into the real search box and submits it, as a user would. */
async function typeSearch(page: Page, query: string) {
  const box = page.getByRole('textbox', { name: /search keywords/i });
  await box.waitFor({ state: 'visible' });
  await box.click();
  await box.fill('');
  await box.pressSequentially(query, { delay: 15 });
  await box.press('Enter');
}

/**
 * Captures the `total` from BOTH search and facets. They are separate endpoints
 * with separate param serializers in services/api.ts; when facets silently
 * dropped the `active` filter on 2026-08-14 the page showed "26 results" above
 * a list of 12, and every backend test still passed.
 */
function captureTotals(page: Page) {
  const totals: { search?: number; facets?: number } = {};
  page.on('response', async (res) => {
    const url = res.url();
    if (!url.includes('/api/opportunities')) return;
    if (res.status() !== 200) return;
    try {
      const body = await res.json();
      if (typeof body?.total !== 'number') return;
      if (url.includes('/opportunities/facets')) totals.facets = body.total;
      else totals.search = body.total;
    } catch {
      /* not JSON — ignore */
    }
  });
  return totals;
}

async function apiTotal(request: APIRequestContext, qs: string): Promise<number> {
  const res = await request.get(`${API}/opportunities?${qs}&limit=1`);
  expect(res.ok()).toBeTruthy();
  return (await res.json()).total;
}

test.describe('search regressions fixed 2026-08-14', () => {
  // "IT" is entirely stopwords, so phraseto_tsquery produces an EMPTY tsquery.
  // That empty query used to veto the literal re-check it was ANDed with, and
  // the search returned zero despite tens of thousands of matches.
  test('a quoted all-stopword phrase returns results', async ({ page }) => {
    await page.goto('/');
    await typeSearch(page, '"IT"');

    const count = await headlineCount(page);
    expect(count, 'quoted "IT" must not return zero').toBeGreaterThan(0);
  });

  // The headline count comes from /opportunities/facets, the list from
  // /opportunities. Drift between their param serializers is invisible to
  // backend tests — this is the check that would have caught it.
  test('headline count agrees with the search results', async ({ page }) => {
    const totals = captureTotals(page);

    await page.goto('/');
    await typeSearch(page, 'services');
    await headlineCount(page);
    await page.waitForTimeout(3000);

    expect(totals.search, 'no search response captured').toBeDefined();
    expect(totals.facets, 'no facets response captured').toBeDefined();
    expect(
      totals.facets,
      `facets total ${totals.facets} != search total ${totals.search} — a filter is missing from one serializer`,
    ).toBe(totals.search);
  });

  // Sort is persisted in localStorage. Every prior visitor had "posted_date"
  // written there automatically by the old default, which pinned them to it and
  // hid the switch to relevance. A brand-new visitor must get relevance.
  test('a first-time visitor gets relevance ranking, not newest-first', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.goto('/');
    await typeSearch(page, 'drone');
    await headlineCount(page);

    await expect(
      page.getByRole('button', { name: 'Relevance', exact: true }),
      'a typed query should default to Relevance, not newest-first',
    ).toBeVisible();
  });

  // Someone pasting a notice number wants THAT notice, even though it closed
  // and its type sits outside the default filter. Discovered from live data so
  // the test does not rot when a fixed notice ages out of the catalog.
  test('a pasted notice number is found regardless of filters or dashes', async ({ page, request }) => {
    // Sorted by soonest deadline so the sample is dominated by CLOSED notices —
    // newest-first would mostly return open ones and the test would skip.
    const res = await request.get(`${API}/opportunities?limit=100&sort=deadline&order=asc`);
    expect(res.ok()).toBeTruthy();
    const items = (await res.json()).opportunities as Array<{
      solicitation_number?: string | null;
      response_deadline?: string | null;
    }>;

    const candidate = items.find(
      (o) =>
        o.solicitation_number &&
        o.solicitation_number.length >= 8 &&
        /[A-Za-z]/.test(o.solicitation_number) &&
        /\d/.test(o.solicitation_number) &&
        o.response_deadline &&
        new Date(o.response_deadline) < new Date(),
    );
    test.skip(!candidate, 'no closed notice with a usable notice number in the sample');

    const solNum = candidate!.solicitation_number!;
    const bare = solNum.replace(/[^A-Za-z0-9]/g, '');
    // Same number with dashes inserted — users paste whichever form they were given.
    const dashed = `${bare.slice(0, 6)}-${bare.slice(6, 8)}-${bare.slice(8)}`;

    for (const form of [bare, dashed]) {
      await page.goto('/');
      await typeSearch(page, form);
      const count = await headlineCount(page);
      expect(count, `notice number ${form} should find the notice despite the open-only filter`).toBeGreaterThan(0);
    }
  });

  // Quotes are an exact-phrase match and can cut thousands of hits to one with
  // no visible reason. The banner has to say so, and its promised count has to
  // be real — clicking it must actually produce that number.
  test('quote collapse is explained and the offered count is honest', async ({ page, request }) => {
    // The hint fires below quoteRelaxThreshold (10) — INCLUDING zero, which is
    // the case that matters most: an exact phrase that matches nothing while
    // its words match thousands. Word pairs that rarely sit adjacent verbatim.
    const candidates = [
      'Utilization Management',
      'Beneficiary Travel',
      'Wildfire Fuel Treatment',
      'Vocational Rehabilitation',
      'Unarmed Security Services',
      'Case Management System',
    ];
    let phrase: string | undefined;
    let expected = 0;
    for (const c of candidates) {
      const quoted = await apiTotal(request, `q=${encodeURIComponent(`"${c}"`)}&active=true`);
      const plain = await apiTotal(request, `q=${encodeURIComponent(c)}&active=true`);
      if (quoted < 10 && plain > quoted && plain > 0) {
        phrase = c;
        expected = plain;
        break;
      }
    }
    test.skip(!phrase, 'no phrase in the sample currently collapses under quotes');
    expect(expected).toBeGreaterThan(0);

    await page.goto('/');
    await typeSearch(page, `"${phrase}"`);
    await headlineCount(page);

    const banner = page.locator('text=/Quotes match the exact phrase/i');
    await expect(banner, 'a collapsed quoted search must explain itself').toBeVisible();

    const bannerText = (await banner.textContent()) ?? '';
    const promised = parseInt((bannerText.match(/([\d,]+)\s+results/) ?? ['', '0'])[1].replace(/,/g, ''), 10);
    expect(promised).toBeGreaterThan(0);

    await page.getByRole('button', { name: /search without quotes/i }).click();
    await page.waitForTimeout(3000);

    expect(await headlineCount(page), 'the relaxed count offered must be the count delivered').toBe(promised);
  });
});

test.describe('search rescue', () => {
  // Rescue fires on a zero-result search and may only offer suggestions whose
  // counts it verified server-side. The contract worth testing is that the
  // number on the chip is the number you get when you click it.
  test('a zero-result search is rescued with a count that holds up', async ({ page }) => {
    // Coffee wholesaler's real search: a term that exists, under a NAICS that has none.
    await page.goto('/?q=coffee&naics=424490');
    await page.waitForLoadState('networkidle', { timeout: 90_000 }).catch(() => {});

    const count = await headlineCount(page);
    test.skip(count > 0, 'this filter combination now returns results; nothing to rescue');

    const chip = page.getByRole('button', { name: /remove the naics filter/i });
    await expect(chip, 'rescue should offer to drop the over-narrow filter').toBeVisible({ timeout: 60_000 });

    const promisedText = (await chip.locator('..').textContent()) ?? '';
    const promised = parseInt((promisedText.match(/([\d,]+)\s+results/) ?? ['', '0'])[1].replace(/,/g, ''), 10);
    expect(promised, 'a rescue suggestion must carry a verified count').toBeGreaterThan(0);

    await chip.click();
    await page.waitForTimeout(3000);

    expect(await headlineCount(page), 'rescue counts are server-verified and must match').toBe(promised);
  });
});

test.describe('still-open filter', () => {
  // "Still open" includes notices with no stated deadline. Before the fix,
  // response_deadline >= today silently dropped ~8k of them because NULL >= x
  // is NULL. Asserted against the API so the invariant is exact.
  test('active-only keeps undated notices', async ({ request }) => {
    const activeOnly = await apiTotal(request, 'q=services&active=true');
    const futureDeadlineOnly = await apiTotal(
      request,
      `q=services&deadline_from=${new Date().toISOString().slice(0, 10)}`,
    );

    expect(activeOnly).toBeGreaterThan(0);
    expect(
      activeOnly,
      'active=true must be broader than a bare deadline range: undated notices are open, not expired',
    ).toBeGreaterThan(futureDeadlineOnly);
  });
});

test.describe('fixes 2026-08-17', () => {
  // naics_code is always 6 digits, so a sector/subsector code from a deep link
  // (?naics=5415) matched nothing under equality and returned a silent zero.
  // Short codes have to widen to a prefix instead.
  test('a short NAICS code from a deep link returns results', async ({ request }) => {
    const sector = await apiTotal(request, 'naics=54');
    const subsector = await apiTotal(request, 'naics=5415');
    const full = await apiTotal(request, 'naics=541512');

    expect(full, 'the exact 6-digit code should have results to nest inside').toBeGreaterThan(0);
    expect(subsector, 'a 4-digit NAICS must widen to a prefix, not return zero').toBeGreaterThanOrEqual(full);
    expect(sector, 'a 2-digit NAICS must be broader still').toBeGreaterThanOrEqual(subsector);
  });

  // PSC codes are natively 4 characters. The NAICS widening must not leak across
  // and turn every exact PSC lookup into a prefix scan.
  test('a 4-character PSC code stays an exact match', async ({ request }) => {
    const exact = await apiTotal(request, 'psc=R425');
    const prefix = await apiTotal(request, 'psc_prefixes=R4');

    expect(exact).toBeGreaterThan(0);
    expect(prefix, 'R4* must be strictly broader than R425 — if equal, PSC went prefix too').toBeGreaterThan(exact);
  });

  // The headline count comes from /opportunities/facets, which has its own param
  // serializer. A NAICS change that lands in one and not the other shows a count
  // above a list that disagrees with it.
  test('a short NAICS deep link agrees between facets and results', async ({ page }) => {
    const totals = captureTotals(page);

    await page.goto('/?naics=5415');
    await headlineCount(page);
    await page.waitForTimeout(3000);

    expect(totals.search, 'no search response captured').toBeDefined();
    expect(totals.search, 'a 4-digit NAICS deep link must not return zero').toBeGreaterThan(0);
    expect(totals.facets, `facets ${totals.facets} != search ${totals.search}`).toBe(totals.search);
  });

  // Opening a notice used to send no Authorization header, so every view was
  // logged anonymously and the search -> view funnel was unmeasurable. The page
  // must still render for a signed-out visitor, which is what this pins.
  test('an opportunity page opens from a search result', async ({ page }) => {
    await page.goto('/');
    await typeSearch(page, 'services');
    await headlineCount(page);

    const firstResult = page.getByRole('link', { name: /view details|open notice/i }).first();
    const fallback = page.locator('a[href*="/opportunities/"]').first();
    const link = (await firstResult.count()) > 0 ? firstResult : fallback;

    await link.click();
    await expect(page, 'clicking a result must land on the notice page').toHaveURL(/\/opportunities\/\d+/, {
      timeout: 60_000,
    });
    await expect(
      page.getByText(/sam\.gov/i).first(),
      'the notice page must render, not blank out on a stale chunk',
    ).toBeVisible({ timeout: 60_000 });
  });
});
