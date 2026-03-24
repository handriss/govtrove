package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/naics"
	"github.com/jackc/pgx/v5/pgxpool"
)

var outputDir string

type setAsideDef struct {
	Code      string // DB value (e.g. SDVOSBC)
	Slug      string // URL slug (e.g. sdvosb)
	ShortName string // Display name for titles (e.g. SDVOSB)
	FullName  string // Long name
	Explainer string
}

var setAsides = []setAsideDef{
	{
		Code:      "SBA",
		Slug:      "sba",
		ShortName: "Small Business",
		FullName:  "Total Small Business Set-Aside",
		Explainer: "The Total Small Business Set-Aside program reserves federal contracts exclusively for small businesses. Contracts under this set-aside are only open to firms meeting the SBA size standards for the relevant NAICS code.",
	},
	{
		Code:      "SDVOSBC",
		Slug:      "sdvosb",
		ShortName: "SDVOSB",
		FullName:  "Service-Disabled Veteran-Owned Small Business (SDVOSB)",
		Explainer: "The SDVOSB program reserves federal contracts for small businesses owned and controlled by service-disabled veterans. The federal government targets at least 5% of contracting dollars for SDVOSB firms.",
	},
	{
		Code:      "8A",
		Slug:      "8a",
		ShortName: "8(a)",
		FullName:  "8(a) Business Development Program",
		Explainer: "The 8(a) Business Development Program helps small, disadvantaged businesses compete for federal contracts. Participating firms can receive sole-source contracts up to $4.5 million ($7 million for manufacturing).",
	},
	{
		Code:      "HZC",
		Slug:      "hubzone",
		ShortName: "HUBZone",
		FullName:  "HUBZone Set-Aside",
		Explainer: "The HUBZone program encourages economic development in historically underutilized business zones. Small businesses headquartered in qualified HUBZone areas receive preferential access to federal contracts.",
	},
	{
		Code:      "WOSB",
		Slug:      "wosb",
		ShortName: "WOSB",
		FullName:  "Women-Owned Small Business (WOSB)",
		Explainer: "The Women-Owned Small Business program reserves certain federal contracts for businesses owned and controlled by women. It targets industries where women-owned firms are underrepresented in federal contracting.",
	},
}

type opportunity struct {
	NoticeID           string
	Title              string
	Department         string
	NAICSCode          *string
	SetAsideCode       *string
	PostedDate         *time.Time
	ResponseDeadline   *time.Time
	PopState           *string
	SolicitationNumber *string
}

type pageData struct {
	SEOTitle        string
	MetaDescription string
	CanonicalPath   string
	Breadcrumb      string
	H1              string
	Subtitle        string
	TotalCount      int
	RecentCount     int
	Explainer       string
	Opportunities   []opportunity
	CTALink         string
	CTACountLabel   string
	UpdatedDate     string
	IsSetAside      bool
	IsNAICS         bool
	IsIndex         bool
	SetAsidePages   []indexEntry
	NAICSPages      []indexEntry
}

type indexEntry struct {
	Slug  string
	Label string
	Count int
}

func main() {
	flag.StringVar(&outputDir, "out", "landing/contracts", "Output directory for generated pages")
	flag.Parse()

	dbURL := os.Getenv("NEON_DATABASE_URL")
	if dbURL == "" {
		log.Fatal("NEON_DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7)
	updatedDate := now.Format("January 2, 2006")

	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("Jan 2, 2006")
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"add": func(a, b int) int {
			return a + b
		},
		"commaInt": func(n int) string {
			s := fmt.Sprintf("%d", n)
			if len(s) <= 3 {
				return s
			}
			var parts []string
			for i := len(s); i > 0; i -= 3 {
				start := i - 3
				if start < 0 {
					start = 0
				}
				parts = append([]string{s[start:i]}, parts...)
			}
			return strings.Join(parts, ",")
		},
	}).Parse(pageTmpl)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	totalPages := 0

	// Generate set-aside pages
	var setAsideEntries []indexEntry
	for _, sa := range setAsides {
		opps, totalCount, recentCount, err := queryOpportunities(ctx, pool, "set_aside_code", sa.Code, weekAgo)
		if err != nil {
			log.Fatalf("Failed to query set-aside %s: %v", sa.Code, err)
		}

		data := pageData{
			SEOTitle:        fmt.Sprintf("%s Federal Contract Opportunities — %d+ Active", sa.ShortName, totalCount),
			MetaDescription: fmt.Sprintf("Browse %d+ active %s (%s) federal contract opportunities. Search by NAICS, agency, keyword, and deadline.", totalCount, sa.ShortName, sa.FullName),
			CanonicalPath:   fmt.Sprintf("set-aside/%s", sa.Slug),
			Breadcrumb:      sa.FullName,
			H1:              fmt.Sprintf("%s Federal Contract Opportunities", sa.FullName),
			Subtitle:        fmt.Sprintf("Browse %d+ active federal contract opportunities set aside for %s firms.", totalCount, sa.ShortName),
			TotalCount:      totalCount,
			RecentCount:     recentCount,
			Explainer:       sa.Explainer,
			Opportunities:   opps,
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?setAside=%s", sa.Code),
			CTACountLabel:   fmt.Sprintf("%d+", totalCount),
			UpdatedDate:     updatedDate,
			IsSetAside:      true,
		}

		outPath := filepath.Join(outputDir, "set-aside", sa.Slug+".html")
		if err := writePage(tmpl, outPath, data); err != nil {
			log.Fatalf("Failed to write %s: %v", outPath, err)
		}
		log.Printf("Generated %s (%d opportunities, %d total)", outPath, len(opps), totalCount)
		totalPages++

		setAsideEntries = append(setAsideEntries, indexEntry{
			Slug:  fmt.Sprintf("set-aside/%s", sa.Slug),
			Label: sa.FullName,
			Count: totalCount,
		})
	}

	// Get top 20 NAICS codes by volume
	topNAICS, err := queryTopNAICS(ctx, pool, weekAgo)
	if err != nil {
		log.Fatalf("Failed to query top NAICS: %v", err)
	}

	var naicsEntries []indexEntry
	for _, nc := range topNAICS {
		label := naics.Label(nc.code)
		opps, totalCount, recentCount, err := queryOpportunities(ctx, pool, "naics_code", nc.code, weekAgo)
		if err != nil {
			log.Fatalf("Failed to query NAICS %s: %v", nc.code, err)
		}

		data := pageData{
			SEOTitle:        fmt.Sprintf("NAICS %s – %s Federal Contracts — %d+ Active", nc.code, label, totalCount),
			MetaDescription: fmt.Sprintf("Browse %d+ active federal contract opportunities under NAICS %s (%s). Filter by set-aside, agency, keyword, and deadline.", totalCount, nc.code, label),
			CanonicalPath:   fmt.Sprintf("naics/%s", nc.code),
			Breadcrumb:      fmt.Sprintf("NAICS %s – %s", nc.code, label),
			H1:              fmt.Sprintf("NAICS %s – %s", nc.code, label),
			Subtitle:        fmt.Sprintf("Browse %d+ active federal contract opportunities classified under NAICS %s.", totalCount, nc.code),
			TotalCount:      totalCount,
			RecentCount:     recentCount,
			Opportunities:   opps,
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?naics=%s", nc.code),
			CTACountLabel:   fmt.Sprintf("%d+", totalCount),
			UpdatedDate:     updatedDate,
			IsNAICS:         true,
		}

		outPath := filepath.Join(outputDir, "naics", nc.code+".html")
		if err := writePage(tmpl, outPath, data); err != nil {
			log.Fatalf("Failed to write %s: %v", outPath, err)
		}
		log.Printf("Generated %s (%d opportunities, %d total)", outPath, len(opps), totalCount)
		totalPages++

		naicsEntries = append(naicsEntries, indexEntry{
			Slug:  fmt.Sprintf("naics/%s", nc.code),
			Label: fmt.Sprintf("NAICS %s – %s", nc.code, label),
			Count: totalCount,
		})
	}

	// Generate index page
	indexData := pageData{
		SEOTitle:        "Federal Contract Opportunities by Set-Aside and NAICS Code",
		MetaDescription: "Browse federal contract opportunities by set-aside type (SDVOSB, 8(a), HUBZone, WOSB) and NAICS code. Updated daily from SAM.gov.",
		CanonicalPath:   "",
		Breadcrumb:      "Federal Contracts",
		H1:              "Federal Contract Opportunities",
		Subtitle:        "Browse active federal contract opportunities by set-aside type and NAICS code. Updated daily from SAM.gov.",
		UpdatedDate:     updatedDate,
		IsIndex:         true,
		SetAsidePages:   setAsideEntries,
		NAICSPages:      naicsEntries,
	}

	indexPath := filepath.Join(outputDir, "index.html")
	if err := writePage(tmpl, indexPath, indexData); err != nil {
		log.Fatalf("Failed to write %s: %v", indexPath, err)
	}
	log.Printf("Generated %s", indexPath)
	totalPages++

	fmt.Printf("\nDone. Generated %d pages in %s/\n", totalPages, outputDir)
}

type naicsCount struct {
	code  string
	count int
}

func queryTopNAICS(ctx context.Context, pool *pgxpool.Pool, weekAgo time.Time) ([]naicsCount, error) {
	rows, err := pool.Query(ctx, `
		SELECT naics_code, COUNT(*) as cnt
		FROM opportunities
		WHERE posted_date >= $1
		  AND naics_code IS NOT NULL AND naics_code != ''
		  AND is_latest = true
		GROUP BY naics_code
		ORDER BY cnt DESC
		LIMIT 20
	`, weekAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []naicsCount
	for rows.Next() {
		var nc naicsCount
		if err := rows.Scan(&nc.code, &nc.count); err != nil {
			return nil, err
		}
		results = append(results, nc)
	}
	return results, rows.Err()
}

func queryOpportunities(ctx context.Context, pool *pgxpool.Pool, filterCol, filterVal string, weekAgo time.Time) ([]opportunity, int, int, error) {
	// Total count
	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s = $1 AND active = true AND is_latest = true", filterCol)
	if err := pool.QueryRow(ctx, countQuery, filterVal).Scan(&totalCount); err != nil {
		return nil, 0, 0, fmt.Errorf("count query: %w", err)
	}

	// Recent count (posted in last 7 days)
	var recentCount int
	recentQuery := fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s = $1 AND active = true AND is_latest = true AND posted_date >= $2", filterCol)
	if err := pool.QueryRow(ctx, recentQuery, filterVal, weekAgo).Scan(&recentCount); err != nil {
		return nil, 0, 0, fmt.Errorf("recent count query: %w", err)
	}

	// Opportunities list
	var selectCols string
	if filterCol == "set_aside_code" {
		selectCols = "notice_id, title, department, naics_code, posted_date, response_deadline, pop_state, solicitation_number"
	} else {
		selectCols = "notice_id, title, department, set_aside_code, posted_date, response_deadline, pop_state, solicitation_number"
	}
	listQuery := fmt.Sprintf("SELECT %s FROM opportunities WHERE %s = $1 AND active = true AND is_latest = true ORDER BY posted_date DESC LIMIT 25", selectCols, filterCol)

	rows, err := pool.Query(ctx, listQuery, filterVal)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list query: %w", err)
	}
	defer rows.Close()

	var opps []opportunity
	for rows.Next() {
		var o opportunity
		if filterCol == "set_aside_code" {
			if err := rows.Scan(&o.NoticeID, &o.Title, &o.Department, &o.NAICSCode, &o.PostedDate, &o.ResponseDeadline, &o.PopState, &o.SolicitationNumber); err != nil {
				return nil, 0, 0, fmt.Errorf("scan: %w", err)
			}
		} else {
			if err := rows.Scan(&o.NoticeID, &o.Title, &o.Department, &o.SetAsideCode, &o.PostedDate, &o.ResponseDeadline, &o.PopState, &o.SolicitationNumber); err != nil {
				return nil, 0, 0, fmt.Errorf("scan: %w", err)
			}
		}
		opps = append(opps, o)
	}
	return opps, totalCount, recentCount, rows.Err()
}

func writePage(tmpl *template.Template, path string, data pageData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

const pageTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.SEOTitle}} | GovTrove</title>
    <meta name="description" content="{{.MetaDescription}}">
    {{- if not .IsIndex}}
    <link rel="canonical" href="https://govtrove.com/contracts/{{.CanonicalPath}}">
    {{- else}}
    <link rel="canonical" href="https://govtrove.com/contracts/">
    {{- end}}

    <!-- Open Graph -->
    <meta property="og:title" content="{{.SEOTitle}} | GovTrove">
    <meta property="og:description" content="{{.MetaDescription}}">
    <meta property="og:type" content="website">
    {{- if not .IsIndex}}
    <meta property="og:url" content="https://govtrove.com/contracts/{{.CanonicalPath}}">
    {{- else}}
    <meta property="og:url" content="https://govtrove.com/contracts/">
    {{- end}}
    <meta property="og:image" content="https://govtrove.com/og-image.png">
    <meta property="og:site_name" content="GovTrove">

    <!-- Twitter Card -->
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="{{.SEOTitle}} | GovTrove">
    <meta name="twitter:description" content="{{.MetaDescription}}">
    <meta name="twitter:image" content="https://govtrove.com/og-image.png">

    <link rel="icon" type="image/svg+xml" href="/favicon.svg">
    <script>document.documentElement.setAttribute('data-theme','dark')</script>
    <link rel="preload" href="/theme.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
    <noscript><link rel="stylesheet" href="/theme.css"></noscript>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link rel="preload" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" as="style" onload="this.onload=null;this.rel='stylesheet'">
    <noscript><link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap"></noscript>

    {{- if not .IsIndex}}
    <script type="application/ld+json">
    {
        "@context": "https://schema.org",
        "@type": "ItemList",
        "name": "{{.H1}}",
        "description": "{{.MetaDescription}}",
        "numberOfItems": {{len .Opportunities}},
        "itemListElement": [
            {{- range $i, $opp := .Opportunities}}{{if $i}},{{end}}
            {
                "@type": "ListItem",
                "position": {{add $i 1}},
                "name": "{{js $opp.Title}}",
                "url": "https://app.govtrove.com/?q={{deref $opp.SolicitationNumber}}"
            }
            {{- end}}
        ]
    }
    </script>
    {{- end}}

    <style>
    .seo-page { max-width: 900px; margin: 0 auto; padding: 2rem 1.5rem; }
    .seo-page h1 { font-size: 2rem; margin-bottom: 0.5rem; color: var(--text-primary); }
    .seo-page .subtitle { color: var(--text-secondary); margin-bottom: 2rem; font-size: 1.1rem; }
    .seo-page .stats { display: flex; gap: 2rem; margin-bottom: 2rem; flex-wrap: wrap; }
    .seo-page .stat { background: var(--bg-card); border: 1px solid var(--border-default); border-radius: 8px; padding: 1rem 1.5rem; }
    .seo-page .stat-value { font-size: 1.5rem; font-weight: 700; color: var(--accent); }
    .seo-page .stat-label { font-size: 0.85rem; color: var(--text-secondary); }
    .opp-list { list-style: none; padding: 0; }
    .opp-item { background: var(--bg-card); border: 1px solid var(--border-default); border-radius: 8px; padding: 1.25rem; margin-bottom: 0.75rem; }
    .opp-item:hover { border-color: var(--accent); }
    .opp-title { font-weight: 600; color: var(--text-primary); margin-bottom: 0.5rem; }
    .opp-title a { color: var(--text-primary); text-decoration: none; }
    .opp-title a:hover { color: var(--accent); }
    .opp-meta { display: flex; gap: 1rem; flex-wrap: wrap; font-size: 0.85rem; color: var(--text-secondary); }
    .opp-meta span { white-space: nowrap; }
    .cta-box { background: var(--accent-subtle); border: 1px solid var(--accent); border-radius: 12px; padding: 2rem; text-align: center; margin: 2rem 0; }
    .cta-box h2 { color: var(--text-primary); margin-bottom: 0.5rem; }
    .cta-box p { color: var(--text-secondary); margin-bottom: 1rem; }
    .cta-btn { display: inline-block; background: var(--accent); color: white; padding: 0.75rem 2rem; border-radius: 8px; text-decoration: none; font-weight: 600; }
    .cta-btn:hover { background: var(--accent-hover); }
    .breadcrumb { font-size: 0.85rem; color: var(--text-muted); margin-bottom: 1rem; }
    .breadcrumb a { color: var(--text-secondary); text-decoration: none; }
    .breadcrumb a:hover { color: var(--accent); }
    .source-note { font-size: 0.8rem; color: var(--text-muted); margin-top: 2rem; text-align: center; }
    .source-note a { color: var(--text-secondary); }
    .explainer { background: var(--bg-card); border: 1px solid var(--border-default); border-radius: 8px; padding: 1.25rem; margin-bottom: 2rem; color: var(--text-secondary); line-height: 1.6; }
    .index-section { margin-bottom: 2.5rem; }
    .index-section h2 { color: var(--text-primary); margin-bottom: 1rem; }
    .index-list { list-style: none; padding: 0; }
    .index-list li { border-bottom: 1px solid var(--border-subtle); }
    .index-list li:last-child { border-bottom: none; }
    .index-list a { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem 0.5rem; text-decoration: none; color: var(--text-primary); }
    .index-list a:hover { color: var(--accent); }
    .index-count { background: var(--bg-card); border: 1px solid var(--border-default); border-radius: 20px; padding: 0.2rem 0.75rem; font-size: 0.8rem; color: var(--text-secondary); }
    </style>
</head>
<body>
    <nav style="background: var(--nav-bg); backdrop-filter: blur(8px); border-bottom: 1px solid var(--border-subtle); padding: 1rem 1.5rem; position: sticky; top: 0; z-index: 100;">
        <div style="max-width: 900px; margin: 0 auto; display: flex; align-items: center; justify-content: space-between;">
            <a href="/" style="text-decoration: none; font-size: 1.25rem; font-weight: 700; color: var(--text-primary);">GovTrove</a>
            <a href="https://app.govtrove.com" class="cta-btn" style="padding: 0.5rem 1.25rem; font-size: 0.9rem;">Search Contracts &rarr;</a>
        </div>
    </nav>

    <main class="seo-page">
        <div class="breadcrumb">
            {{- if .IsIndex}}
            <a href="/">Home</a> / Federal Contracts
            {{- else}}
            <a href="/">Home</a> / <a href="/contracts/">Federal Contracts</a> / {{.Breadcrumb}}
            {{- end}}
        </div>

        <h1>{{.H1}}</h1>
        <p class="subtitle">{{.Subtitle}}</p>

        {{- if .IsIndex}}
        <div class="index-section">
            <h2>By Set-Aside Type</h2>
            <ul class="index-list">
                {{- range .SetAsidePages}}
                <li><a href="/contracts/{{.Slug}}"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
        </div>

        <div class="index-section">
            <h2>By NAICS Code (Top 20 This Week)</h2>
            <ul class="index-list">
                {{- range .NAICSPages}}
                <li><a href="/contracts/{{.Slug}}"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
        </div>

        <div class="cta-box">
            <h2>Search All Federal Contracts</h2>
            <p>Filter by keyword, NAICS, set-aside, agency, and more. Free to use.</p>
            <a href="https://app.govtrove.com" class="cta-btn">Search on GovTrove &rarr;</a>
        </div>
        {{- else}}

        <div class="stats">
            <div class="stat">
                <div class="stat-value">{{commaInt .TotalCount}}</div>
                <div class="stat-label">Active Opportunities</div>
            </div>
            <div class="stat">
                <div class="stat-value">{{commaInt .RecentCount}}</div>
                <div class="stat-label">Posted This Week</div>
            </div>
        </div>

        {{- if .Explainer}}
        <div class="explainer">{{.Explainer}}</div>
        {{- end}}

        <h2>Recent Opportunities</h2>
        <ul class="opp-list">
            {{- range .Opportunities}}
            <li class="opp-item">
                <div class="opp-title"><a href="https://app.govtrove.com/?q={{deref .SolicitationNumber}}">{{.Title}}</a></div>
                <div class="opp-meta">
                    <span>Agency: {{.Department}}</span>
                    {{- if .PostedDate}}
                    <span>Posted: {{formatDate .PostedDate}}</span>
                    {{- end}}
                    {{- if .ResponseDeadline}}
                    <span>Due: {{formatDate .ResponseDeadline}}</span>
                    {{- end}}
                    {{- if and .NAICSCode (ne (deref .NAICSCode) "")}}
                    <span>NAICS: {{deref .NAICSCode}}</span>
                    {{- end}}
                    {{- if and .SetAsideCode (ne (deref .SetAsideCode) "")}}
                    <span>Set-Aside: {{deref .SetAsideCode}}</span>
                    {{- end}}
                    {{- if and .PopState (ne (deref .PopState) "")}}
                    <span>Location: {{deref .PopState}}</span>
                    {{- end}}
                </div>
            </li>
            {{- end}}
        </ul>

        <div class="cta-box">
            <h2>Search All {{commaInt .TotalCount}}+ Opportunities</h2>
            <p>Filter by keyword, NAICS, set-aside, agency, and more. Free to use.</p>
            <a href="{{.CTALink}}" class="cta-btn">Search on GovTrove &rarr;</a>
        </div>
        {{- end}}

        <p class="source-note">
            Data sourced from <a href="https://sam.gov/content/opportunities" target="_blank" rel="noopener">SAM.gov</a>.
            Updated daily. Last updated: {{.UpdatedDate}}.
        </p>
    </main>

    <script src="/posthog.js" defer></script>
    <script src="/utm.js" defer></script>
</body>
</html>
`
