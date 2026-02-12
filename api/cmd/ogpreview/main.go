package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type OGMeta struct {
	Title       string
	Description string
	Image       string
	URL         string
	SiteName    string
	ImageWidth  string
	ImageHeight string
	TwitterCard string
	Label1      string
	Data1       string
	Label2      string
	Data2       string
}

type Opportunity struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Type       *string `json:"type"`
	Department *string `json:"department"`
	SetAside   *string `json:"set_aside_description"`
	NAICSCode  *string `json:"naics_code"`
}

type SearchResult struct {
	Opportunities []Opportunity `json:"opportunities"`
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "API base URL")
	ids := flag.String("ids", "", "comma-separated opportunity IDs (if empty, fetches recent)")
	count := flag.Int("n", 6, "number of opportunities to preview (when no IDs given)")
	output := flag.String("o", "/tmp/og-preview.html", "output HTML file")
	flag.Parse()

	var oppIDs []string
	if *ids != "" {
		oppIDs = strings.Split(*ids, ",")
	} else {
		oppIDs = fetchRecentIDs(*baseURL, *count)
	}

	var metas []OGMeta
	for _, id := range oppIDs {
		id = strings.TrimSpace(id)
		meta := fetchOGMeta(*baseURL, id)
		if meta.Title != "" {
			metas = append(metas, meta)
		}
	}

	if len(metas) == 0 {
		fmt.Fprintln(os.Stderr, "No OG meta tags found. Is the API running?")
		os.Exit(1)
	}

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := pageTmpl.Execute(f, metas); err != nil {
		fmt.Fprintf(os.Stderr, "render template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d previews\n", *output, len(metas))
	openBrowser(*output)
}

func fetchRecentIDs(baseURL string, count int) []string {
	resp, err := http.Get(fmt.Sprintf("%s/api/opportunities?limit=%d", baseURL, count))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch opportunities: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	var result SearchResult
	json.NewDecoder(resp.Body).Decode(&result)
	var ids []string
	for _, o := range result.Opportunities {
		ids = append(ids, fmt.Sprintf("%d", o.ID))
	}
	return ids
}

func fetchOGMeta(baseURL, id string) OGMeta {
	resp, err := http.Get(fmt.Sprintf("%s/og/opportunities/%s", baseURL, id))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch OG for %s: %v\n", id, err)
		return OGMeta{}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	meta := OGMeta{
		Image: fmt.Sprintf("%s/og/opportunities/%s/card.png", baseURL, id),
	}
	meta.Title = extractMeta(html, `og:title`)
	meta.Description = extractMeta(html, `og:description`)
	meta.URL = extractMeta(html, `og:url`)
	meta.SiteName = extractMeta(html, `og:site_name`)
	meta.ImageWidth = extractMeta(html, `og:image:width`)
	meta.ImageHeight = extractMeta(html, `og:image:height`)
	meta.TwitterCard = extractMeta(html, `twitter:card`)

	meta.Label1 = extractMeta(html, `twitter:label1`)
	meta.Data1 = extractMeta(html, `twitter:data1`)
	meta.Label2 = extractMeta(html, `twitter:label2`)
	meta.Data2 = extractMeta(html, `twitter:data2`)

	if ogImg := extractMeta(html, `og:image`); ogImg != "" {
		meta.Image = ogImg
	}

	return meta
}

func extractMeta(html, property string) string {
	// Look for property="..." or name="..."
	for _, attr := range []string{"property", "name"} {
		needle := fmt.Sprintf(`%s="%s"`, attr, property)
		idx := strings.Index(html, needle)
		if idx == -1 {
			continue
		}
		rest := html[idx:]
		contentIdx := strings.Index(rest, `content="`)
		if contentIdx == -1 {
			continue
		}
		start := contentIdx + len(`content="`)
		end := strings.Index(rest[start:], `"`)
		if end == -1 {
			continue
		}
		return rest[start : start+end]
	}
	return ""
}

func openBrowser(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return
	}
	cmd.Start()
}

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>OG Preview — GovTrove</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; padding: 32px; color: #1a1a1a; }
  h1 { text-align: center; margin-bottom: 8px; font-size: 24px; }
  .subtitle { text-align: center; color: #666; margin-bottom: 32px; font-size: 14px; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 32px; max-width: 1400px; margin: 0 auto; }
  .card-group { margin-bottom: 48px; }
  .card-group h2 { margin-bottom: 16px; font-size: 14px; text-transform: uppercase; letter-spacing: 1px; color: #666; }

  /* Facebook style */
  .fb-card { border: 1px solid #dadde1; border-radius: 8px; overflow: hidden; background: #fff; max-width: 500px; }
  .fb-card img { width: 100%; display: block; }
  .fb-card .fb-body { padding: 10px 12px; }
  .fb-card .fb-domain { font-size: 12px; color: #606770; text-transform: uppercase; }
  .fb-card .fb-title { font-size: 16px; font-weight: 600; color: #1d2129; margin: 3px 0; line-height: 1.3; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .fb-card .fb-desc { font-size: 14px; color: #606770; line-height: 1.4; display: -webkit-box; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden; }

  /* LinkedIn style */
  .li-card { border: 1px solid #e0e0e0; border-radius: 8px; overflow: hidden; background: #fff; max-width: 500px; }
  .li-card img { width: 100%; display: block; }
  .li-card .li-body { padding: 8px 12px; }
  .li-card .li-title { font-size: 14px; font-weight: 600; color: rgba(0,0,0,0.9); line-height: 1.3; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .li-card .li-domain { font-size: 12px; color: rgba(0,0,0,0.6); margin-top: 2px; }

  /* Slack style */
  .slack-card { border-left: 4px solid #36c5f0; padding: 8px 16px; max-width: 500px; background: #fff; border-radius: 0 4px 4px 0; }
  .slack-card .slack-site { font-size: 12px; font-weight: 700; color: #1d1c1d; margin-bottom: 4px; }
  .slack-card .slack-title { font-size: 15px; font-weight: 700; color: #1264a3; margin-bottom: 4px; line-height: 1.3; }
  .slack-card .slack-desc { font-size: 14px; color: #1d1c1d; margin-bottom: 8px; line-height: 1.4; }
  .slack-card .slack-fields { display: flex; gap: 24px; margin-bottom: 8px; }
  .slack-card .slack-field-label { font-size: 12px; font-weight: 700; color: #616061; }
  .slack-card .slack-field-value { font-size: 14px; color: #1d1c1d; }
  .slack-card img { width: 100%; border-radius: 4px; display: block; margin-top: 4px; }

  /* Twitter/X style */
  .tw-card { border: 1px solid #cfd9de; border-radius: 16px; overflow: hidden; background: #fff; max-width: 500px; }
  .tw-card img { width: 100%; display: block; }
  .tw-card .tw-body { padding: 10px 12px; }
  .tw-card .tw-domain { font-size: 13px; color: #536471; }
  .tw-card .tw-title { font-size: 15px; font-weight: 700; color: #0f1419; line-height: 1.3; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .tw-card .tw-desc { font-size: 15px; color: #536471; line-height: 1.3; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }

  @media (max-width: 900px) { .grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<h1>OG Card Preview</h1>
<p class="subtitle">Simulated social platform previews for GovTrove opportunity cards</p>

{{range .}}
<div class="grid">

  <!-- Facebook -->
  <div class="card-group">
    <h2>Facebook</h2>
    <div class="fb-card">
      <img src="{{.Image}}" alt="">
      <div class="fb-body">
        <div class="fb-domain">app.govtrove.com</div>
        <div class="fb-title">{{.Title}}</div>
        <div class="fb-desc">{{.Description}}</div>
      </div>
    </div>
  </div>

  <!-- LinkedIn -->
  <div class="card-group">
    <h2>LinkedIn</h2>
    <div class="li-card">
      <img src="{{.Image}}" alt="">
      <div class="li-body">
        <div class="li-title">{{.Title}}</div>
        <div class="li-domain">app.govtrove.com</div>
      </div>
    </div>
  </div>

  <!-- Slack -->
  <div class="card-group">
    <h2>Slack</h2>
    <div class="slack-card">
      <div class="slack-site">{{.SiteName}}</div>
      <div class="slack-title">{{.Title}}</div>
      <div class="slack-desc">{{.Description}}</div>
      {{if or .Data1 .Data2}}
      <div class="slack-fields">
        {{if .Data1}}<div><div class="slack-field-label">{{.Label1}}</div><div class="slack-field-value">{{.Data1}}</div></div>{{end}}
        {{if .Data2}}<div><div class="slack-field-label">{{.Label2}}</div><div class="slack-field-value">{{.Data2}}</div></div>{{end}}
      </div>
      {{end}}
      <img src="{{.Image}}" alt="">
    </div>
  </div>

  <!-- Twitter/X -->
  <div class="card-group">
    <h2>Twitter / X</h2>
    <div class="tw-card">
      <img src="{{.Image}}" alt="">
      <div class="tw-body">
        <div class="tw-domain">app.govtrove.com</div>
        <div class="tw-title">{{.Title}}</div>
        <div class="tw-desc">{{.Description}}</div>
      </div>
    </div>
  </div>

</div>
<hr style="margin: 32px 0; border: none; border-top: 1px solid #ddd;">
{{end}}

</body>
</html>`))
