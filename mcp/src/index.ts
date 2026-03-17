import express from "express";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { createRemoteJWKSet, jwtVerify, type JWTPayload } from "jose";
import { z } from "zod";
import { randomUUID } from "node:crypto";
import pg from "pg";

const PORT = process.env.PORT || 3000;
const AUTHKIT_DOMAIN =
  process.env.AUTHKIT_DOMAIN ||
  "https://timely-midnight-01-staging.authkit.app";
const MCP_RESOURCE_URL =
  process.env.MCP_RESOURCE_URL || "http://localhost:3000";
const DATABASE_URL = process.env.DATABASE_URL || "";
const POSTHOG_KEY = process.env.POSTHOG_KEY || "";
const POSTHOG_HOST = process.env.POSTHOG_HOST || "https://us.i.posthog.com";

const FREE_DAILY_LIMIT = 5;
const PRO_DAILY_LIMIT = 500;

const JWKS = createRemoteJWKSet(new URL(`${AUTHKIT_DOMAIN}/oauth2/jwks`));

const pool = new pg.Pool({
  connectionString: DATABASE_URL,
  max: 5,
  ssl: { rejectUnauthorized: false },
});

const WWW_AUTHENTICATE_HEADER = [
  'Bearer error="unauthorized"',
  'error_description="Authorization needed"',
  `resource_metadata="${MCP_RESOURCE_URL}/.well-known/oauth-protected-resource"`,
].join(", ");

// ── Reference data ───────────────────────────────────────────────────

const SET_ASIDE_CODES: Record<string, { name: string; qualifies: string }> = {
  SBA:      { name: "Total Small Business Set-Aside", qualifies: "Any small business meeting SBA size standards for the NAICS code" },
  SBP:      { name: "Partial Small Business Set-Aside", qualifies: "Small businesses for specific line items only" },
  "8A":     { name: "8(a) Competitive", qualifies: "Firms certified under SBA's 8(a) Business Development program" },
  "8AN":    { name: "8(a) Sole Source", qualifies: "Single 8(a) certified firm (non-competitive)" },
  HZC:      { name: "HUBZone Competitive", qualifies: "HUBZone certified small businesses" },
  HZS:      { name: "HUBZone Sole Source", qualifies: "Single HUBZone certified firm (non-competitive)" },
  SDVOSBC:  { name: "Service-Disabled Veteran-Owned Small Business (SDVOSB) Competitive", qualifies: "SDVOSB certified firms" },
  SDVOSBS:  { name: "SDVOSB Sole Source", qualifies: "Single SDVOSB certified firm (non-competitive)" },
  WOSB:     { name: "Women-Owned Small Business (WOSB)", qualifies: "Women-owned small businesses in underrepresented industries" },
  WOSBSS:   { name: "WOSB Sole Source", qualifies: "Single WOSB certified firm (non-competitive)" },
  EDWOSB:   { name: "Economically Disadvantaged WOSB (EDWOSB)", qualifies: "Economically disadvantaged women-owned small businesses" },
  EDWOSBSS: { name: "EDWOSB Sole Source", qualifies: "Single EDWOSB certified firm (non-competitive)" },
  VSA:      { name: "Veteran-Owned Small Business (VOSB) Set-Aside", qualifies: "Veteran-owned small businesses" },
  VSS:      { name: "VOSB Sole Source", qualifies: "Single veteran-owned small business (non-competitive)" },
  ESB:      { name: "Emerging Small Business", qualifies: "Small businesses in early growth stage within designated industries" },
  BICiv:    { name: "Buy Indian - Civilian", qualifies: "Indian-owned economic enterprises under Buy Indian Act (civilian agencies)" },
  LAS:      { name: "Local Area Set-Aside", qualifies: "Small businesses in a specific local geographic area" },
  IEE:      { name: "Indian Economic Enterprise (IEE)", qualifies: "Indian-owned economic enterprises (DoI programs)" },
  ISBEE:    { name: "Indian Small Business Economic Enterprise (ISBEE)", qualifies: "Indian-owned small business economic enterprises" },
};

const SET_ASIDE_ALIASES: Record<string, string> = {
  "small business": "SBA",
  "total small business": "SBA",
  sb: "SBA",
  "partial small business": "SBP",
  "8(a)": "8A",
  "8a": "8A",
  "8(a) sole source": "8AN",
  hubzone: "HZC",
  "hubzone sole source": "HZS",
  sdvosb: "SDVOSBC",
  "service-disabled veteran": "SDVOSBC",
  "service disabled veteran": "SDVOSBC",
  "sdvosb sole source": "SDVOSBS",
  wosb: "WOSB",
  "women-owned": "WOSB",
  "women owned": "WOSB",
  "wosb sole source": "WOSBSS",
  edwosb: "EDWOSB",
  "edwosb sole source": "EDWOSBSS",
  veteran: "VSA",
  "veteran-owned": "VSA",
  "veteran owned": "VSA",
  vosb: "VSA",
  "vosb sole source": "VSS",
};

function resolveSetAside(input: string): string | null {
  if (SET_ASIDE_CODES[input]) return input;
  const lower = input.toLowerCase().trim();
  if (SET_ASIDE_ALIASES[lower]) return SET_ASIDE_ALIASES[lower];
  // Check if it's a code with wrong case
  const upper = input.toUpperCase();
  if (SET_ASIDE_CODES[upper]) return upper;
  return null;
}

const NOTICE_TYPES: Record<string, { lifecycle: string; canBid: boolean; action: string }> = {
  "Presolicitation": {
    lifecycle: "Early notice that a solicitation is coming. The agency is signaling intent to solicit.",
    canBid: false,
    action: "Monitor and prepare. Get the solicitation documents when they're released. Use this time to research the requirement and build your team.",
  },
  "Sources Sought": {
    lifecycle: "Market research phase. The agency is gauging industry interest and capabilities before writing a solicitation.",
    canBid: false,
    action: "RESPOND. This is your chance to shape the requirement. Submit a capabilities statement showing why your company is qualified. This is not a bid — it's a conversation starter.",
  },
  "Solicitation": {
    lifecycle: "The formal request for proposals/quotes. This is the actual bidding opportunity.",
    canBid: true,
    action: "Submit a proposal or quote by the response deadline. Read the solicitation carefully, note all evaluation criteria, and follow formatting instructions exactly.",
  },
  "Combined Synopsis/Solicitation": {
    lifecycle: "A combined notice and solicitation in one document. Common for simplified acquisitions under $250K.",
    canBid: true,
    action: "Submit a quote by the response deadline. These are typically simpler procurements with shorter turnaround times.",
  },
  "Award Notice": {
    lifecycle: "The contract has been awarded. Shows who won and for how much (when available).",
    canBid: false,
    action: "Review who won and for what amount. Useful for competitive intelligence and understanding pricing in your market. If you lost, you can request a debrief.",
  },
  "Justification and Approval (J&A)": {
    lifecycle: "Public notice justifying a non-competitive (sole source) award. Required when competition is limited.",
    canBid: false,
    action: "Review to understand why competition was limited. If you believe you could compete, contact the contracting officer before the J&A is finalized.",
  },
  "Special Notice": {
    lifecycle: "Informational notice. Not a solicitation — could be upcoming events, policy changes, or industry days.",
    canBid: false,
    action: "Read for awareness. Industry days and conferences listed here are valuable networking opportunities.",
  },
  "Intent to Bundle": {
    lifecycle: "Notice that the agency plans to bundle multiple requirements into a single contract, which may reduce small business opportunities.",
    canBid: false,
    action: "Review and comment if the bundling would unfairly exclude small businesses. Contact the SBA PCR (Procurement Center Representative) if concerned.",
  },
  "Sale of Surplus Property": {
    lifecycle: "Government surplus property available for purchase.",
    canBid: true,
    action: "Review items and submit a bid if interested in purchasing surplus government property.",
  },
};

const GUIDE_TEXT = `# GovTrove — Federal Contract Opportunities Guide

## What is GovTrove?
GovTrove helps small businesses find and track federal contract opportunities from SAM.gov (the U.S. government's official procurement website). Data is sourced from SAM.gov's public API and updated multiple times daily.

Source: https://sam.gov/content/opportunities

## What is a Federal Contract Opportunity?
When the U.S. government needs to buy goods or services, agencies post "opportunities" (also called notices or solicitations) on SAM.gov. Companies compete for these contracts by submitting proposals.

The federal government is the world's largest buyer — over $700 billion/year in contracts. By law, a percentage must go to small businesses.

## Opportunity Lifecycle
Opportunities progress through stages:

1. **Sources Sought / RFI** — Market research. The agency asks "who can do this?" Respond to shape the requirement.
2. **Presolicitation** — Heads up that a solicitation is coming soon. Prepare your team.
3. **Solicitation / Combined Synopsis** — The actual bidding opportunity. Submit your proposal by the deadline.
4. **Award Notice** — The contract has been awarded. See who won and for how much.

Not all opportunities go through every stage. Some skip straight to solicitation.

## Set-Aside Programs
The government reserves certain contracts for specific types of small businesses. These "set-asides" reduce competition and give qualified firms a better chance at winning. Read the govtrove://set-aside-codes resource for the full list.

Key programs:
- **Total Small Business (SBA)** — Open to any small business meeting size standards
- **8(a)** — For disadvantaged small businesses in the SBA's 8(a) development program
- **HUBZone** — For businesses in Historically Underutilized Business Zones
- **SDVOSB** — For service-disabled veteran-owned small businesses
- **WOSB** — For women-owned small businesses in underrepresented industries

Source: https://www.sba.gov/federal-contracting/contracting-assistance-programs

## NAICS Codes
Every opportunity has a NAICS (North American Industry Classification System) code that describes what industry the work falls under. Your business's size standard (whether you're "small") is determined by the NAICS code on each opportunity.

Example: 541512 = Computer Systems Design Services (small if < $34M annual revenue)

Source: https://www.census.gov/naics/

## Response Deadlines
The response deadline is when proposals must be submitted. Missing it by even one minute means automatic rejection — no exceptions. Plan to submit at least 24 hours early.

## Key Tips
- **Start with Sources Sought** — Responding to these is free, low-risk, and shapes future solicitations in your favor
- **Filter by set-aside** — If you have certifications (8a, HUBZone, SDVOSB, WOSB), filter for those set-asides to find less competitive opportunities
- **Watch the NAICS code** — Make sure you're small under that NAICS code's size standard
- **Track response deadlines** — Set reminders. Late proposals are always rejected
- **Read the full solicitation** — The description on SAM.gov is just a summary. Download and read the actual solicitation documents
`;

// ── User context ────────────────────────────────────────────────────

interface GovTroveUser {
  id: number;
  workosId: string;
  email: string;
  plan: string;
  freeForever: boolean;
}

async function lookupUser(workosId: string): Promise<GovTroveUser | null> {
  const result = await pool.query(
    "SELECT id, workos_id, email, plan, free_forever FROM users WHERE workos_id = $1",
    [workosId]
  );
  if (result.rows.length === 0) return null;
  const row = result.rows[0];
  return { id: row.id, workosId: row.workos_id, email: row.email, plan: row.plan, freeForever: row.free_forever };
}

async function getDailyUsageCount(userId: number): Promise<number> {
  const result = await pool.query(
    "SELECT COUNT(*) as cnt FROM mcp_usage WHERE user_id = $1 AND called_at > NOW() - interval '1 day'",
    [userId]
  );
  return parseInt(result.rows[0].cnt, 10);
}

async function logUsage(
  userId: number,
  toolName: string,
  latencyMs: number,
  userEmail: string | null,
  requestParams: Record<string, unknown> | null,
  resultCount: number | null
): Promise<void> {
  await pool.query(
    "INSERT INTO mcp_usage (user_id, tool_name, latency_ms, user_email, request_params, result_count) VALUES ($1, $2, $3, $4, $5, $6)",
    [userId, toolName, latencyMs, userEmail, requestParams ? JSON.stringify(requestParams) : null, resultCount]
  );
}

function capturePosthogEvent(
  distinctId: string,
  event: string,
  properties: Record<string, unknown>
): void {
  if (!POSTHOG_KEY || !distinctId) return;
  const payload = {
    api_key: POSTHOG_KEY,
    event,
    distinct_id: distinctId,
    properties: { ...properties, client: "mcp" },
    timestamp: new Date().toISOString(),
  };
  fetch(`${POSTHOG_HOST}/i/v0/e/`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    signal: AbortSignal.timeout(3000),
  }).catch(() => {});
}

function rateLimitError(plan: string): object {
  const limit = plan === "pro" ? PRO_DAILY_LIMIT : FREE_DAILY_LIMIT;
  const message =
    plan === "pro"
      ? `You've reached your daily limit of ${limit} MCP tool calls. Your limit resets in 24 hours.`
      : `You've used all ${limit} free MCP queries for today. Upgrade to GovTrove Pro for ${PRO_DAILY_LIMIT} queries/day at https://app.govtrove.com/settings`;
  return {
    content: [{ type: "text" as const, text: message }],
  };
}

// ── Auth middleware ──────────────────────────────────────────────────

interface AuthenticatedRequest extends express.Request {
  user?: JWTPayload;
  govtroveUser?: GovTroveUser;
}

async function bearerTokenMiddleware(
  req: AuthenticatedRequest,
  res: express.Response,
  next: express.NextFunction
) {
  const token = req.headers.authorization?.match(/^Bearer (.+)$/)?.[1];
  if (!token) {
    res
      .set("WWW-Authenticate", WWW_AUTHENTICATE_HEADER)
      .status(401)
      .json({ error: "No token provided." });
    return;
  }

  try {
    const { payload } = await jwtVerify(token, JWKS, {
      issuer: AUTHKIT_DOMAIN,
    });
    req.user = payload;

    const workosId = payload.sub;
    if (workosId) {
      const user = await lookupUser(workosId);
      if (user) {
        req.govtroveUser = user;
      }
    }

    next();
  } catch {
    res
      .set("WWW-Authenticate", WWW_AUTHENTICATE_HEADER)
      .status(401)
      .json({ error: "Invalid bearer token." });
    return;
  }
}

// ── MCP server factory ──────────────────────────────────────────────

// Per-session user context so tool handlers can access the authenticated user
const sessionUsers = new Map<string, GovTroveUser>();

async function withUsageTracking(
  sessionId: string | undefined,
  toolName: string,
  requestParams: Record<string, unknown> | null,
  fn: () => Promise<{ response: object; resultCount: number | null }>
): Promise<object> {
  const user = sessionId ? sessionUsers.get(sessionId) : undefined;

  if (user) {
    const effectivePlan = (user.plan === "pro" || user.freeForever) ? "pro" : "free";
    const dailyLimit = effectivePlan === "pro" ? PRO_DAILY_LIMIT : FREE_DAILY_LIMIT;
    const count = await getDailyUsageCount(user.id);
    if (count >= dailyLimit) {
      return rateLimitError(effectivePlan);
    }
  }

  const start = Date.now();
  try {
    const { response, resultCount } = await fn();
    const latencyMs = Date.now() - start;

    if (user) {
      logUsage(user.id, toolName, latencyMs, user.email ?? null, requestParams, resultCount).catch((e) => {
        console.error("Failed to log usage:", e);
      });
      capturePosthogEvent(user.workosId, `mcp_${toolName}`, {
        ...(requestParams || {}),
        result_count: resultCount,
      });
    }

    return response;
  } catch (err) {
    console.error(`Tool ${toolName} error:`, err);
    throw err;
  }
}

function createMcpServer(sessionId: string): McpServer {
  const server = new McpServer({
    name: "govtrove",
    version: "0.2.0",
  });

  // ── Resources ───────────────────────────────────────────────────

  server.registerResource(
    "guide",
    "govtrove://guide",
    {
      description: "Introduction to federal contracting, SAM.gov, opportunity lifecycle, set-aside programs, NAICS codes, and tips for small businesses",
    },
    () => ({
      contents: [{ uri: "govtrove://guide", text: GUIDE_TEXT, mimeType: "text/markdown" }],
    })
  );

  server.registerResource(
    "set-aside-codes",
    "govtrove://set-aside-codes",
    {
      description: "Complete list of small business set-aside codes with descriptions and eligibility requirements",
    },
    () => {
      const lines = [
        "# Set-Aside Codes",
        "",
        "Set-asides reserve contracts for specific types of small businesses. Use the code in the `set_aside` parameter when searching.",
        "",
        "Source: https://www.sba.gov/federal-contracting/contracting-assistance-programs",
        "",
        "| Code | Name | Who Qualifies |",
        "|------|------|---------------|",
      ];
      for (const [code, info] of Object.entries(SET_ASIDE_CODES)) {
        lines.push(`| ${code} | ${info.name} | ${info.qualifies} |`);
      }
      lines.push("");
      lines.push("## Common Aliases");
      lines.push("You can also use plain English when searching — the tool will resolve these automatically:");
      lines.push("- \"small business\" → SBA");
      lines.push("- \"8(a)\" or \"8a\" → 8A");
      lines.push("- \"hubzone\" → HZC");
      lines.push("- \"sdvosb\" or \"service-disabled veteran\" → SDVOSBC");
      lines.push("- \"wosb\" or \"women-owned\" → WOSB");
      lines.push("- \"veteran\" or \"veteran-owned\" → VSA");
      return {
        contents: [{ uri: "govtrove://set-aside-codes", text: lines.join("\n"), mimeType: "text/markdown" }],
      };
    }
  );

  server.registerResource(
    "notice-types",
    "govtrove://notice-types",
    {
      description: "Explanation of each federal contract notice type — what it means, where it sits in the procurement lifecycle, and what action to take",
    },
    () => {
      const lines = [
        "# Federal Contract Notice Types",
        "",
        "Source: https://sam.gov/content/opportunities",
        "",
      ];
      for (const [type, info] of Object.entries(NOTICE_TYPES)) {
        lines.push(`## ${type}`);
        lines.push(`**Lifecycle:** ${info.lifecycle}`);
        lines.push(`**Can you bid?** ${info.canBid ? "Yes" : "No"}`);
        lines.push(`**Action:** ${info.action}`);
        lines.push("");
      }
      return {
        contents: [{ uri: "govtrove://notice-types", text: lines.join("\n"), mimeType: "text/markdown" }],
      };
    }
  );

  // ── Prompts ─────────────────────────────────────────────────────

  server.registerPrompt(
    "daily-briefing",
    {
      title: "Daily Briefing",
      description: "Get a summary of federal contract opportunities posted in the last 24 hours",
      argsSchema: {
        keywords: z.string().optional().describe("Keywords to filter by (e.g. 'cybersecurity', 'IT support')"),
        naics_code: z.string().optional().describe("NAICS code to filter by (e.g. '541512')"),
        set_aside: z.string().optional().describe("Set-aside type (e.g. 'small business', 'SBA', '8a', 'hubzone')"),
        state: z.string().optional().describe("Two-letter state code (e.g. 'VA', 'CA')"),
      },
    },
    (args) => {
      const filters: string[] = [];
      if (args.keywords) filters.push(`keywords: "${args.keywords}"`);
      if (args.naics_code) filters.push(`NAICS code: ${args.naics_code}`);
      if (args.set_aside) filters.push(`set-aside: ${args.set_aside}`);
      if (args.state) filters.push(`state: ${args.state}`);
      const filterText = filters.length > 0 ? filters.join(", ") : "no specific filters";

      return {
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `Search for federal contract opportunities posted in the last 1 day with these filters: ${filterText}.

Present results as a daily briefing:
1. Total count of new opportunities matching the filters
2. Group results by department — show department name and count
3. For each opportunity show: title, department, set-aside type, response deadline, and a link
4. Highlight any with response deadlines within the next 7 days as URGENT
5. If there are more than 10 results, show the top 10 most relevant and mention the total count
6. If there are no results, suggest broadening the filters`,
            },
          },
        ],
      };
    }
  );

  server.registerPrompt(
    "opportunity-analysis",
    {
      title: "Opportunity Analysis",
      description: "Get a detailed plain-English analysis of a specific federal contract opportunity",
      argsSchema: {
        id: z.string().describe("The opportunity ID (numeric) or notice ID from a previous search"),
      },
    },
    (args) => ({
      messages: [
        {
          role: "user",
          content: {
            type: "text",
            text: `Get full details for opportunity ${args.id}.

Provide a plain-English analysis:
1. **What they're buying** — Summarize the description in 2-3 sentences that a non-expert can understand
2. **Who can bid** — Set-aside restrictions and what certifications are needed. Mention the NAICS code and what industry it covers
3. **Key dates** — Posted date, response deadline, and how many days remain. Flag if deadline is within 7 days
4. **Estimated value** — Award amount if available, or note if not disclosed
5. **Place of performance** — Where the work will be done
6. **Contacts** — Contracting officer name, email, and phone
7. **Red flags** — Note any concerns: very tight deadline, vague requirements, or signs of an incumbent-friendly solicitation
8. **Links** — SAM.gov page and GovTrove page for full details and documents`,
          },
        },
      ],
    })
  );

  // ── Tools ───────────────────────────────────────────────────────

  server.registerTool(
    "search_opportunities",
    {
      title: "Search Federal Contract Opportunities",
      description:
        "Search federal contract opportunities from SAM.gov. Returns matching opportunities with key details. Read the govtrove://set-aside-codes resource for valid set-aside codes, and govtrove://notice-types for notice type explanations.",
      inputSchema: {
        keywords: z
          .string()
          .max(500)
          .optional()
          .describe(
            'Free text search across title, description, and solicitation number. Use quotes for exact phrases (e.g. "cyber security")'
          ),
        naics_code: z
          .string()
          .max(6)
          .optional()
          .describe("6-digit NAICS code (e.g. 541512 for Computer Systems Design)"),
        set_aside: z
          .string()
          .max(50)
          .optional()
          .describe(
            'Set-aside filter. Use a code (SBA, 8A, HZC, SDVOSBC, WOSB, etc.) or plain English ("small business", "hubzone", "8(a)", "women-owned", "veteran"). See govtrove://set-aside-codes for the full list.'
          ),
        department: z
          .string()
          .max(200)
          .optional()
          .describe(
            'Department or agency name to filter by (partial match, e.g. "defense" or "navy")'
          ),
        state: z
          .string()
          .max(2)
          .optional()
          .describe("Two-letter state code for place of performance (e.g. VA, CA)"),
        type: z
          .string()
          .max(50)
          .optional()
          .describe(
            'Notice type: "Solicitation", "Presolicitation", "Combined Synopsis/Solicitation", "Sources Sought", "Special Notice", "Award Notice", "Justification and Approval (J&A)", "Sale of Surplus Property", "Intent to Bundle". See govtrove://notice-types for explanations.'
          ),
        posted_days_ago: z
          .number()
          .int()
          .min(1)
          .max(365)
          .optional()
          .describe("Only show opportunities posted within the last N days"),
        deadline_within_days: z
          .number()
          .int()
          .min(1)
          .max(365)
          .optional()
          .describe("Only show opportunities with deadlines within the next N days"),
        limit: z
          .number()
          .int()
          .min(1)
          .max(25)
          .optional()
          .describe("Number of results to return (default 10, max 25)"),
        offset: z
          .number()
          .int()
          .min(0)
          .optional()
          .describe("Offset for pagination (default 0)"),
      },
    },
    async (params) => {
      const searchParams = { ...params };
      return withUsageTracking(sessionId, "search_opportunities", searchParams, async () => {
        const conditions: string[] = ["active = true", "is_latest = true"];
        const values: unknown[] = [];
        let paramIdx = 1;

        if (params.keywords) {
          conditions.push(
            `search_vector @@ websearch_to_tsquery('english', $${paramIdx})`
          );
          values.push(params.keywords);
          paramIdx++;
        }

        if (params.naics_code) {
          conditions.push(`naics_code = $${paramIdx}`);
          values.push(params.naics_code);
          paramIdx++;
        }

        if (params.set_aside) {
          const resolved = resolveSetAside(params.set_aside);
          if (!resolved) {
            const validCodes = Object.entries(SET_ASIDE_CODES)
              .map(([code, info]) => `${code} (${info.name})`)
              .join(", ");
            return {
              response: {
                content: [{
                  type: "text" as const,
                  text: `Unknown set-aside "${params.set_aside}". Valid codes: ${validCodes}. You can also use plain English like "small business", "hubzone", "8(a)", "sdvosb", "women-owned", or "veteran".`,
                }],
              },
              resultCount: 0,
            };
          }
          conditions.push(`set_aside_code = $${paramIdx}`);
          values.push(resolved);
          paramIdx++;
        }

        if (params.department) {
          conditions.push(`department ILIKE $${paramIdx}`);
          values.push(`%${params.department}%`);
          paramIdx++;
        }

        if (params.state) {
          conditions.push(`pop_state = $${paramIdx}`);
          values.push(params.state);
          paramIdx++;
        }

        if (params.type) {
          conditions.push(`type = $${paramIdx}`);
          values.push(params.type);
          paramIdx++;
        }

        if (params.posted_days_ago) {
          conditions.push(`posted_date >= NOW() - $${paramIdx}::interval`);
          values.push(`${params.posted_days_ago} days`);
          paramIdx++;
        }

        if (params.deadline_within_days) {
          conditions.push(
            `response_deadline >= NOW() AND response_deadline <= NOW() + $${paramIdx}::interval`
          );
          values.push(`${params.deadline_within_days} days`);
          paramIdx++;
        }

        const limit = Math.min(params.limit ?? 10, 25);
        const offset = params.offset ?? 0;

        const orderBy = params.keywords
          ? `ts_rank(search_vector, websearch_to_tsquery('english', $1)) DESC, posted_date DESC`
          : `posted_date DESC`;

        const query = `
          SELECT id, notice_id, title, LEFT(description, 500) as description,
                 solicitation_number, type, department, sub_tier,
                 naics_code, set_aside_code, set_aside_description,
                 posted_date, response_deadline, pop_state, ui_link
          FROM opportunities
          WHERE ${conditions.join(" AND ")}
          ORDER BY ${orderBy}
          LIMIT ${limit} OFFSET ${offset}
        `;

        const countQuery = `
          SELECT COUNT(*) as total
          FROM opportunities
          WHERE ${conditions.join(" AND ")}
        `;

        const [result, countResult] = await Promise.all([
          pool.query(query, values),
          pool.query(countQuery, values),
        ]);

        const total = parseInt(countResult.rows[0].total, 10);
        const opportunities = result.rows.map((row) => ({
          id: row.id,
          notice_id: row.notice_id,
          title: row.title,
          description: row.description,
          solicitation_number: row.solicitation_number,
          type: row.type,
          department: row.department,
          sub_tier: row.sub_tier,
          naics_code: row.naics_code,
          set_aside: row.set_aside_code,
          set_aside_description: row.set_aside_description,
          posted_date: row.posted_date,
          response_deadline: row.response_deadline,
          state: row.pop_state,
          sam_url: row.ui_link || `https://sam.gov/opp/${row.notice_id}/view`,
          govtrove_url: `https://app.govtrove.com/opportunity/${row.id}`,
        }));

        return {
          response: {
            content: [
              {
                type: "text" as const,
                text: JSON.stringify(
                  { total, showing: opportunities.length, offset, opportunities },
                  null,
                  2
                ),
              },
            ],
          },
          resultCount: total,
        };
      }) as ReturnType<Parameters<typeof server.registerTool>[2]>;
    }
  );

  server.registerTool(
    "get_opportunity",
    {
      title: "Get Opportunity Details",
      description:
        "Fetch full details for a single federal contract opportunity by its ID or notice ID.",
      inputSchema: {
        id: z
          .string()
          .describe(
            "The opportunity ID (numeric) or notice_id (alphanumeric) from a previous search"
          ),
      },
    },
    async ({ id }) => {
      return withUsageTracking(sessionId, "get_opportunity", { id }, async () => {
        const isNumeric = /^\d+$/.test(id);
        const query = `
          SELECT id, notice_id, solicitation_number, title, description, type,
                 department, sub_tier, office,
                 naics_code, classification_code, set_aside_code, set_aside_description,
                 posted_date, response_deadline, archive_date,
                 award_number, award_date, award_amount,
                 awardee_name, awardee_uei,
                 pop_street_address, pop_city, pop_state, pop_zip, pop_country,
                 pop_state_name, pop_country_name,
                 primary_contact_title, primary_contact_fullname,
                 primary_contact_email, primary_contact_phone,
                 secondary_contact_title, secondary_contact_fullname,
                 secondary_contact_email, secondary_contact_phone,
                 ui_link, additional_info_link, resource_links
          FROM opportunities
          WHERE ${isNumeric ? "id = $1" : "notice_id = $1"} AND is_latest = true
          LIMIT 1
        `;

        const result = await pool.query(query, [isNumeric ? parseInt(id, 10) : id]);

        if (result.rows.length === 0) {
          return {
            response: {
              content: [
                {
                  type: "text" as const,
                  text: `No opportunity found with ${isNumeric ? "ID" : "notice ID"} "${id}".`,
                },
              ],
            },
            resultCount: 0,
          };
        }

        const row = result.rows[0];
        const opportunity = {
          id: row.id,
          notice_id: row.notice_id,
          solicitation_number: row.solicitation_number,
          title: row.title,
          description: row.description,
          type: row.type,
          department: row.department,
          sub_tier: row.sub_tier,
          office: row.office,
          naics_code: row.naics_code,
          classification_code: row.classification_code,
          set_aside: row.set_aside_code,
          set_aside_description: row.set_aside_description,
          posted_date: row.posted_date,
          response_deadline: row.response_deadline,
          archive_date: row.archive_date,
          award: row.award_number
            ? { number: row.award_number, date: row.award_date, amount: row.award_amount }
            : null,
          awardee: row.awardee_name
            ? { name: row.awardee_name, uei: row.awardee_uei }
            : null,
          place_of_performance: {
            street: row.pop_street_address,
            city: row.pop_city,
            state: row.pop_state_name || row.pop_state,
            zip: row.pop_zip,
            country: row.pop_country_name || row.pop_country,
          },
          contacts: {
            primary: row.primary_contact_fullname
              ? {
                  title: row.primary_contact_title,
                  name: row.primary_contact_fullname,
                  email: row.primary_contact_email,
                  phone: row.primary_contact_phone,
                }
              : null,
            secondary: row.secondary_contact_fullname
              ? {
                  title: row.secondary_contact_title,
                  name: row.secondary_contact_fullname,
                  email: row.secondary_contact_email,
                  phone: row.secondary_contact_phone,
                }
              : null,
          },
          links: {
            sam_url: row.ui_link || `https://sam.gov/opp/${row.notice_id}/view`,
            govtrove_url: `https://app.govtrove.com/opportunity/${row.id}`,
            additional_info: row.additional_info_link,
          },
          attachments: row.resource_links ?? [],
        };

        return {
          response: {
            content: [
              {
                type: "text" as const,
                text: JSON.stringify(opportunity, null, 2),
              },
            ],
          },
          resultCount: 1,
        };
      }) as ReturnType<Parameters<typeof server.registerTool>[2]>;
    }
  );

  return server;
}

// ── Express app ─────────────────────────────────────────────────────

const app = express();
app.use(express.json());

app.get("/.well-known/oauth-protected-resource", (_req, res) => {
  res.json({
    resource: MCP_RESOURCE_URL,
    authorization_servers: [AUTHKIT_DOMAIN],
    bearer_methods_supported: ["header"],
  });
});

app.get("/.well-known/oauth-authorization-server", async (_req, res) => {
  const response = await fetch(
    `${AUTHKIT_DOMAIN}/.well-known/oauth-authorization-server`
  );
  const metadata = await response.json();
  res.json(metadata);
});

app.get("/health", (_req, res) => {
  res.json({ status: "ok" });
});

app.get("/schema", (_req, res) => {
  res.json({
    name: "govtrove",
    version: "0.2.0",
    resources: [
      {
        uri: "govtrove://guide",
        name: "guide",
        description: "Introduction to federal contracting, SAM.gov, opportunity lifecycle, set-aside programs, NAICS codes, and tips for small businesses",
      },
      {
        uri: "govtrove://set-aside-codes",
        name: "set-aside-codes",
        description: "Complete list of small business set-aside codes with descriptions and eligibility requirements",
        data: SET_ASIDE_CODES,
      },
      {
        uri: "govtrove://notice-types",
        name: "notice-types",
        description: "Explanation of each federal contract notice type — lifecycle stage, whether you can bid, and recommended action",
        data: NOTICE_TYPES,
      },
    ],
    prompts: [
      {
        name: "daily-briefing",
        title: "Daily Briefing",
        description: "Get a summary of federal contract opportunities posted in the last 24 hours",
        arguments: [
          { name: "keywords", type: "string", required: false, description: "Keywords to filter by" },
          { name: "naics_code", type: "string", required: false, description: "NAICS code to filter by" },
          { name: "set_aside", type: "string", required: false, description: "Set-aside type (code or plain English)" },
          { name: "state", type: "string", required: false, description: "Two-letter state code" },
        ],
      },
      {
        name: "opportunity-analysis",
        title: "Opportunity Analysis",
        description: "Get a detailed plain-English analysis of a specific federal contract opportunity",
        arguments: [
          { name: "id", type: "string", required: true, description: "Opportunity ID or notice ID" },
        ],
      },
    ],
    tools: [
      {
        name: "search_opportunities",
        title: "Search Federal Contract Opportunities",
        description: "Search federal contract opportunities from SAM.gov with filters",
        parameters: {
          keywords: { type: "string", required: false, description: "Free text search (supports quotes for exact phrases)" },
          naics_code: { type: "string", required: false, description: "6-digit NAICS code" },
          set_aside: { type: "string", required: false, description: "Set-aside code or plain English (e.g. 'small business', 'SBA', '8a', 'hubzone')" },
          department: { type: "string", required: false, description: "Department/agency name (partial match)" },
          state: { type: "string", required: false, description: "Two-letter state code" },
          type: { type: "string", required: false, description: "Notice type filter" },
          posted_days_ago: { type: "number", required: false, description: "Only opportunities posted within N days" },
          deadline_within_days: { type: "number", required: false, description: "Only opportunities with deadlines within N days" },
          limit: { type: "number", required: false, description: "Results to return (default 10, max 25)" },
          offset: { type: "number", required: false, description: "Pagination offset" },
        },
      },
      {
        name: "get_opportunity",
        title: "Get Opportunity Details",
        description: "Fetch full details for a single opportunity by ID or notice ID",
        parameters: {
          id: { type: "string", required: true, description: "Opportunity ID (numeric) or notice_id (alphanumeric)" },
        },
      },
    ],
    set_aside_aliases: SET_ASIDE_ALIASES,
  });
});

const transports: Record<string, StreamableHTTPServerTransport> = {};

app.post(
  "/mcp",
  bearerTokenMiddleware,
  async (req: AuthenticatedRequest, res) => {
    const sessionId = req.headers["mcp-session-id"] as string | undefined;

    if (sessionId && transports[sessionId]) {
      await transports[sessionId].handleRequest(req, res, req.body);
      return;
    }

    const newSessionId = randomUUID();
    const transport = new StreamableHTTPServerTransport({
      sessionIdGenerator: () => newSessionId,
    });

    transport.onclose = () => {
      const sid = transport.sessionId;
      if (sid) {
        delete transports[sid];
        sessionUsers.delete(sid);
      }
    };

    // Store user context for this session so tool handlers can access it
    if (req.govtroveUser) {
      sessionUsers.set(newSessionId, req.govtroveUser);
    }

    const server = createMcpServer(newSessionId);
    await server.connect(transport);

    await transport.handleRequest(req, res, req.body);

    const sid = transport.sessionId;
    if (sid) {
      transports[sid] = transport;
    }
  }
);

app.get(
  "/mcp",
  bearerTokenMiddleware,
  async (req: AuthenticatedRequest, res) => {
    const sessionId = req.headers["mcp-session-id"] as string | undefined;
    if (!sessionId || !transports[sessionId]) {
      res.status(404).end();
      return;
    }
    await transports[sessionId].handleRequest(req, res);
  }
);

app.delete(
  "/mcp",
  bearerTokenMiddleware,
  async (req: AuthenticatedRequest, res) => {
    const sessionId = req.headers["mcp-session-id"] as string | undefined;
    if (!sessionId || !transports[sessionId]) {
      res.status(404).end();
      return;
    }
    await transports[sessionId].handleRequest(req, res);
  }
);

app.listen(PORT, () => {
  console.log(`GovTrove MCP server listening on port ${PORT}`);
  console.log(`AuthKit domain: ${AUTHKIT_DOMAIN}`);
  console.log(`Resource URL: ${MCP_RESOURCE_URL}`);
});
