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

// ── User context ────────────────────────────────────────────────────

interface GovTroveUser {
  id: number;
  workosId: string;
  email: string;
  plan: string;
}

async function lookupUser(workosId: string): Promise<GovTroveUser | null> {
  const result = await pool.query(
    "SELECT id, workos_id, email, plan FROM users WHERE workos_id = $1",
    [workosId]
  );
  if (result.rows.length === 0) return null;
  const row = result.rows[0];
  return { id: row.id, workosId: row.workos_id, email: row.email, plan: row.plan };
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
    const dailyLimit = user.plan === "pro" ? PRO_DAILY_LIMIT : FREE_DAILY_LIMIT;
    const count = await getDailyUsageCount(user.id);
    if (count >= dailyLimit) {
      return rateLimitError(user.plan);
    }
  }

  const start = Date.now();
  const { response, resultCount } = await fn();
  const latencyMs = Date.now() - start;

  if (user) {
    logUsage(user.id, toolName, latencyMs, user.email ?? null, requestParams, resultCount).catch(() => {});
  }

  return response;
}

function createMcpServer(sessionId: string): McpServer {
  const server = new McpServer({
    name: "govtrove",
    version: "0.1.0",
  });

  server.registerTool(
    "search_opportunities",
    {
      title: "Search Federal Contract Opportunities",
      description:
        "Search federal contract opportunities from SAM.gov. Use when the user wants to find contracts, solicitations, RFPs, or procurement notices. Supports filtering by keywords, NAICS codes, set-aside types, agencies, deadlines, and more.",
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
          .enum([
            "SBA",
            "SBP",
            "8A",
            "8AN",
            "HZC",
            "HZS",
            "SDVOSBC",
            "SDVOSBS",
            "WOSB",
            "WOSBSS",
            "EDWOSB",
            "EDWOSBSS",
            "VSA",
            "VSB",
          ])
          .optional()
          .describe("Small business set-aside code"),
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
          .enum([
            "Solicitation",
            "Presolicitation",
            "Combined Synopsis/Solicitation",
            "Sources Sought",
            "Special Notice",
            "Award Notice",
            "Justification and Approval (J&A)",
            "Sale of Surplus Property",
            "Intent to Bundle",
          ])
          .optional()
          .describe("Notice type filter"),
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
          conditions.push(`set_aside_code = $${paramIdx}`);
          values.push(params.set_aside);
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
            state: row.pop_state,
            zip: row.pop_zip,
            country: row.pop_country,
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
      res.status(400).json({ error: "Invalid or missing session ID." });
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
      res.status(400).json({ error: "Invalid or missing session ID." });
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
