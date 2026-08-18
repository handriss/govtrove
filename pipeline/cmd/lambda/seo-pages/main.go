package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/naics"
	"github.com/handriss/govtrove/pipeline/internal/psc"
)

var (
	db       *database.DB
	s3Client *s3.Client
	cfClient *cloudfront.Client
	bucket   string
	distID   string
	logger   *slog.Logger
)

func init() {
	ctx := context.Background()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	secretARN := os.Getenv("DATABASE_URL_SECRET_ARN")
	if secretARN == "" {
		return
	}

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      "production",
			AttachStacktrace: true,
		})
	}

	region := os.Getenv("AWS_REGION_NAME")
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	smClient := secretsmanager.NewFromConfig(awsCfg)
	result, err := smClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretARN,
	})
	if err != nil {
		logger.Error("failed to get database URL from Secrets Manager", "error", err)
		os.Exit(1)
	}

	db, err = database.New(ctx, *result.SecretString)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	s3Client = s3.NewFromConfig(awsCfg)
	cfClient = cloudfront.NewFromConfig(awsCfg)
	bucket = os.Getenv("LANDING_S3_BUCKET")
	distID = os.Getenv("LANDING_DISTRIBUTION_ID")

	logger.Info("cold start complete")
}

type Input struct {
	ExecutionID string `json:"execution_id"`
}

type Output struct {
	Status      string `json:"status"`
	SetAsides   int    `json:"set_aside_pages"`
	NAICSPages  int    `json:"naics_pages"`
	AgencyPages int    `json:"agency_pages"`
	PSCPages    int    `json:"psc_pages"`
	HubPages    int    `json:"hub_pages"`
	SitemapURLs int    `json:"sitemap_urls"`
	PrunedPages int    `json:"pruned_pages"`
	Failures    int    `json:"failures"`
}

type S3Putter interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

type CloudFrontInvalidator interface {
	CreateInvalidation(ctx context.Context, params *cloudfront.CreateInvalidationInput, optFns ...func(*cloudfront.Options)) (*cloudfront.CreateInvalidationOutput, error)
}

type Handler struct {
	Store          database.Store
	S3             S3Putter
	CF             CloudFrontInvalidator
	S3Bucket       string
	DistributionID string
	Logger         *slog.Logger
}

type uploadItem struct {
	key         string
	body        []byte
	contentType string
}

// A page carrying one or two opportunities is near-indistinguishable from every other
// near-empty page, and Google clusters them as duplicates — which drags on the pages that
// do have substance. Below this they are still generated (they are useful to a human who
// lands on one) but noindex'd and kept out of the sitemap.
const indexThreshold = 3

// pruneStalePages deletes contracts/ objects this run did not write. A NAICS code or
// agency that stops appearing in the data is simply skipped by the generator, so without
// this its page stays live in S3 forever: unlinked, absent from the sitemap, and still
// serving 200 to crawlers. Scoped to contracts/ so hand-authored landing files are safe.
func (h *Handler) pruneStalePages(ctx context.Context, writtenKeys map[string]bool) (int, error) {
	const prefix = "contracts/"

	var stale []s3types.ObjectIdentifier
	var token *string
	for {
		page, err := h.S3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            &h.S3Bucket,
			Prefix:            strPtr(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return 0, fmt.Errorf("list %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || writtenKeys[*obj.Key] {
				continue
			}
			stale = append(stale, s3types.ObjectIdentifier{Key: obj.Key})
		}
		if page.IsTruncated == nil || !*page.IsTruncated {
			break
		}
		token = page.NextContinuationToken
	}

	if len(stale) == 0 {
		return 0, nil
	}

	// A run that wrote nothing means the generator failed upstream, not that every page
	// is genuinely gone. Refuse to empty the bucket on the strength of a bad run.
	if len(writtenKeys) == 0 {
		return 0, fmt.Errorf("refusing to prune %d objects: nothing was written this run", len(stale))
	}

	deleted := 0
	for start := 0; start < len(stale); start += 1000 {
		end := start + 1000
		if end > len(stale) {
			end = len(stale)
		}
		batch := stale[start:end]
		out, err := h.S3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &h.S3Bucket,
			Delete: &s3types.Delete{Objects: batch, Quiet: boolPtr(true)},
		})
		if err != nil {
			return deleted, fmt.Errorf("delete stale objects: %w", err)
		}
		deleted += len(batch) - len(out.Errors)
		for _, e := range out.Errors {
			h.Logger.Error("stale object delete failed", "key", deref(e.Key), "error", deref(e.Message))
		}
	}

	h.Logger.Info("pruned stale SEO pages", "deleted", deleted, "candidates", len(stale))
	return deleted, nil
}

func (h *Handler) Handle(ctx context.Context, event json.RawMessage) (_ *Output, retErr error) {
	defer func() {
		if retErr != nil {
			sentry.CaptureException(retErr)
		}
		sentry.Flush(2 * time.Second)
	}()

	start := time.Now()

	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		h.Logger.Warn("failed to parse input", "error", err)
	}

	var stepID uuid.UUID
	if input.ExecutionID != "" {
		execID, err := uuid.Parse(input.ExecutionID)
		if err == nil && h.Store != nil {
			sid, err := h.Store.CreatePipelineStep(ctx, execID, "seo-pages")
			if err != nil {
				h.Logger.Warn("failed to create pipeline step", "error", err)
			} else {
				stepID = sid
			}
		}
	}

	defer func() {
		if retErr != nil && stepID != uuid.Nil {
			_ = h.Store.FailPipelineStep(ctx, stepID, retErr.Error(), int(time.Since(start).Milliseconds()))
		}
	}()

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
		return nil, fmt.Errorf("parse template: %w", err)
	}

	// Batch count query
	counts, err := h.Store.GetSEOPageCounts(ctx, weekAgo)
	if err != nil {
		return nil, fmt.Errorf("get SEO page counts: %w", err)
	}

	countMap := make(map[string]map[string]database.SEOPageCount)
	for _, c := range counts {
		if countMap[c.FilterType] == nil {
			countMap[c.FilterType] = make(map[string]database.SEOPageCount)
		}
		countMap[c.FilterType][c.FilterValue] = c
	}

	uploads := make(chan uploadItem, 100)
	var failures atomic.Int32
	var wg sync.WaitGroup

	// Worker pool for S3 uploads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range uploads {
				_, err := h.S3.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      &h.S3Bucket,
					Key:         &item.key,
					Body:        bytes.NewReader(item.body),
					ContentType: &item.contentType,
				})
				if err != nil {
					failures.Add(1)
					h.Logger.Error("s3 upload failed", "key", item.key, "error", err)
				}
			}
		}()
	}

	// Hoisted above the first page loop: every page type links its opportunity rows'
	// NAICS codes, so this set is needed before the NAICS pages themselves are built.
	allNAICS := naics.Codes()
	allPSCCodes := psc.Codes()
	allSectors := naics.Sectors()

	naicsSiblings := buildSiblingIndex(allNAICS, countMap["naics"],
		func(code string) (string, bool) {
			sec := naics.SectorOf(code)
			return sec, allSectors[sec] != ""
		}, "naics", "NAICS")
	pscSiblings := buildSiblingIndex(allPSCCodes, countMap["psc"],
		func(code string) (string, bool) {
			k, _, ok := psc.ParentOf(code)
			return k, ok
		}, "psc", "PSC")

	// Every key written this run, so stale objects from earlier runs can be pruned.
	// Without this, a NAICS code or agency that goes inactive leaves an orphan page
	// live in S3 forever — unlinked, absent from the sitemap, and still returning 200.
	writtenKeys := make(map[string]bool)
	put := func(key string, body []byte, contentType string) {
		writtenKeys[key] = true
		uploads <- uploadItem{key: key, body: body, contentType: contentType}
	}

	var setAsideCount, naicsCount, agencyCount, pscCount, hubCount int
	var sitemapURLs []string

	sectorTitles := naics.Sectors()
	naicsBySector := make(map[string][]indexEntry)
	sectorCounts := make(map[string]int)

	// Static sitemap entries
	staticPages := []string{
		"https://govtrove.com/",
		"https://govtrove.com/blog/",
		"https://govtrove.com/naics-code-finder/",
		"https://govtrove.com/psc-code-finder/",
		"https://govtrove.com/contracts/",
	}
	sitemapURLs = append(sitemapURLs, staticPages...)

	// Generate set-aside pages
	var setAsideEntries []indexEntry
	for _, sa := range setAsides {
		c := countMap["set_aside"][sa.Code]
		opps, err := h.Store.GetSEOPageOpportunities(ctx, "set_aside_code", sa.Code, 25)
		if err != nil {
			h.Logger.Error("failed to get set-aside opps", "code", sa.Code, "error", err)
			failures.Add(1)
			continue
		}

		data := pageData{
			SEOTitle:        fmt.Sprintf("%s Federal Contract Opportunities — %d+ Active", sa.ShortName, c.TotalCount),
			MetaDescription: fmt.Sprintf("Browse %d+ active %s (%s) federal contract opportunities. Search by NAICS, agency, keyword, and deadline.", c.TotalCount, sa.ShortName, sa.FullName),
			CanonicalPath:   fmt.Sprintf("set-aside/%s", sa.Slug),
			Breadcrumb:      sa.FullName,
			H1:              fmt.Sprintf("%s Federal Contract Opportunities", sa.FullName),
			Subtitle:        fmt.Sprintf("Browse %d+ active federal contract opportunities set aside for %s firms.", c.TotalCount, sa.ShortName),
			TotalCount:      c.TotalCount,
			RecentCount:     c.RecentCount,
			Explainer:       sa.Explainer,
			Opportunities:   toTemplateOpps(opps, allNAICS),
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?setAside=%s", sa.Code),
			CTACountLabel:   fmt.Sprintf("%d+", c.TotalCount),
			UpdatedDate:     updatedDate,
			IsSetAside:      true,
		}

		body, err := renderPage(tmpl, data)
		if err != nil {
			h.Logger.Error("failed to render set-aside page", "code", sa.Code, "error", err)
			failures.Add(1)
			continue
		}

		key := fmt.Sprintf("contracts/set-aside/%s.html", sa.Slug)
		put(key, body, "text/html; charset=utf-8")
		setAsideCount++
		sitemapURLs = append(sitemapURLs, fmt.Sprintf("https://govtrove.com/contracts/set-aside/%s.html", sa.Slug))

		setAsideEntries = append(setAsideEntries, indexEntry{
			Slug:  fmt.Sprintf("set-aside/%s", sa.Slug),
			Label: sa.FullName,
			Count: c.TotalCount,
		})
	}

	// Generate NAICS pages
	naicsCodes := make([]string, 0, len(allNAICS))
	for code := range allNAICS {
		naicsCodes = append(naicsCodes, code)
	}
	sort.Strings(naicsCodes)

	var naicsEntries []indexEntry
	for _, code := range naicsCodes {
		label := allNAICS[code]
		c := countMap["naics"][code]

		var opps []database.SEOOpportunity
		if c.TotalCount > 0 {
			var err error
			opps, err = h.Store.GetSEOPageOpportunities(ctx, "naics_code", code, 25)
			if err != nil {
				h.Logger.Error("failed to get NAICS opps", "code", code, "error", err)
				failures.Add(1)
				continue
			}
		}

		// "naics 335313" is someone asking what the code is, not how many contracts are
		// open under it. These pages ranked 5-10 on those queries and earned almost no
		// clicks because the title answered the wrong question — lead with the definition.
		seoTitle := fmt.Sprintf("NAICS %s – %s: What It Covers, %d Active Contracts", code, label, c.TotalCount)
		metaDesc := fmt.Sprintf("NAICS %s covers %s. See what the code includes, which agencies buy under it, and %d active federal contract opportunities — updated daily from SAM.gov.", code, label, c.TotalCount)
		subtitle := fmt.Sprintf("What NAICS %s (%s) covers, and %d active federal contract opportunities classified under it.", code, label, c.TotalCount)
		if c.TotalCount == 0 {
			seoTitle = fmt.Sprintf("NAICS %s – %s: What This Code Covers", code, label)
			metaDesc = fmt.Sprintf("What NAICS %s (%s) covers in federal contracting, and how to find matching opportunities on SAM.gov — free, daily-updated search from GovTrove.", code, label)
			subtitle = fmt.Sprintf("What NAICS %s (%s) means for federal contracting, and how to find matching opportunities.", code, label)
		}

		data := pageData{
			SEOTitle:        seoTitle,
			MetaDescription: metaDesc,
			CanonicalPath:   fmt.Sprintf("naics/%s", code),
			Breadcrumb:      fmt.Sprintf("NAICS %s – %s", code, label),
			H1:              fmt.Sprintf("NAICS %s – %s", code, label),
			Subtitle:        subtitle,
			TotalCount:      c.TotalCount,
			RecentCount:     c.RecentCount,
			Opportunities:   toTemplateOpps(opps, allNAICS),
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?naics=%s", code),
			CTACountLabel:   fmt.Sprintf("%d+", c.TotalCount),
			UpdatedDate:     updatedDate,
			NoIndex:         c.TotalCount < indexThreshold,
			IsNAICS:         true,
			ParentLink:      sectorLink(code),
			ParentLabel:     sectorLabel(code),
			SiblingPages:    siblingsOf(naicsSiblings, naics.SectorOf(code), fmt.Sprintf("naics/%s", code)),
			SiblingHeading:  "Related NAICS Codes",
		}

		body, err := renderPage(tmpl, data)
		if err != nil {
			h.Logger.Error("failed to render NAICS page", "code", code, "error", err)
			failures.Add(1)
			continue
		}

		key := fmt.Sprintf("contracts/naics/%s.html", code)
		put(key, body, "text/html; charset=utf-8")
		naicsCount++
		// Thin pages are noindex'd — keep them out of the sitemap too.
		if c.TotalCount >= indexThreshold {
			sitemapURLs = append(sitemapURLs, fmt.Sprintf("https://govtrove.com/contracts/naics/%s.html", code))
		}

		if c.TotalCount >= indexThreshold {
			entry := indexEntry{
				Slug:  fmt.Sprintf("naics/%s", code),
				Label: fmt.Sprintf("NAICS %s – %s", code, label),
				Count: c.TotalCount,
			}
			naicsEntries = append(naicsEntries, entry)
			if sec := naics.SectorOf(code); sectorTitles[sec] != "" {
				naicsBySector[sec] = append(naicsBySector[sec], entry)
				sectorCounts[sec] += c.TotalCount
			}
		}
	}

	// Sort NAICS entries by count descending for the index page
	sort.Slice(naicsEntries, func(i, j int) bool {
		return naicsEntries[i].Count > naicsEntries[j].Count
	})

	// Generate agency pages
	agencies, err := h.Store.GetActiveAgencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active agencies: %w", err)
	}

	var agencyEntries []indexEntry
	for _, ag := range agencies {
		slug := slugify(ag.Department)
		c := countMap["department"][ag.Department]

		opps, err := h.Store.GetSEOPageOpportunities(ctx, "department", ag.Department, 25)
		if err != nil {
			h.Logger.Error("failed to get agency opps", "department", ag.Department, "error", err)
			failures.Add(1)
			continue
		}

		data := pageData{
			SEOTitle:        fmt.Sprintf("%s Federal Contract Opportunities — %d+ Active", ag.Department, c.TotalCount),
			MetaDescription: fmt.Sprintf("Browse %d+ active federal contract opportunities from %s. Filter by NAICS, set-aside, keyword, and deadline.", c.TotalCount, ag.Department),
			CanonicalPath:   fmt.Sprintf("agency/%s", slug),
			Breadcrumb:      ag.Department,
			H1:              fmt.Sprintf("%s Contract Opportunities", ag.Department),
			Subtitle:        fmt.Sprintf("Browse %d+ active federal contract opportunities from %s.", c.TotalCount, ag.Department),
			TotalCount:      c.TotalCount,
			RecentCount:     c.RecentCount,
			Opportunities:   toTemplateOpps(opps, allNAICS),
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?agency=%s", ag.Department),
			CTACountLabel:   fmt.Sprintf("%d+", c.TotalCount),
			UpdatedDate:     updatedDate,
			IsAgency:        true,
		}

		body, err := renderPage(tmpl, data)
		if err != nil {
			h.Logger.Error("failed to render agency page", "department", ag.Department, "error", err)
			failures.Add(1)
			continue
		}

		key := fmt.Sprintf("contracts/agency/%s.html", slug)
		put(key, body, "text/html; charset=utf-8")
		agencyCount++
		sitemapURLs = append(sitemapURLs, fmt.Sprintf("https://govtrove.com/contracts/agency/%s.html", slug))

		agencyEntries = append(agencyEntries, indexEntry{
			Slug:  fmt.Sprintf("agency/%s", slug),
			Label: ag.Department,
			Count: c.TotalCount,
		})
	}

	// Generate PSC code pages
	allPSC := psc.Codes()
	pscCodes := make([]string, 0, len(allPSC))
	for code := range allPSC {
		pscCodes = append(pscCodes, code)
	}
	sort.Strings(pscCodes)

	pscByGroup := make(map[string][]indexEntry)
	for _, code := range pscCodes {
		label := allPSC[code]
		c := countMap["psc"][code]

		var opps []database.SEOOpportunity
		if c.TotalCount > 0 {
			var err error
			opps, err = h.Store.GetSEOPageOpportunities(ctx, "classification_code", code, 25)
			if err != nil {
				h.Logger.Error("failed to get PSC opps", "code", code, "error", err)
				failures.Add(1)
				continue
			}
		}

		parentKey, parentTitle, hasParent := psc.ParentOf(code)

		seoTitle := fmt.Sprintf("PSC %s – %s: What It Covers, %d Active Contracts", code, label, c.TotalCount)
		metaDesc := fmt.Sprintf("PSC code %s covers %s. See what the product service code includes, which agencies buy under it, and %d active federal contract opportunities from SAM.gov.", code, label, c.TotalCount)
		subtitle := fmt.Sprintf("What PSC %s (%s) covers, and %d active federal contract opportunities classified under it.", code, label, c.TotalCount)
		if c.TotalCount == 0 {
			seoTitle = fmt.Sprintf("PSC %s – %s: What This Code Covers", code, label)
			metaDesc = fmt.Sprintf("What PSC code %s (%s) covers in federal contracting, and how to find matching opportunities on SAM.gov — free, daily-updated search from GovTrove.", code, label)
			subtitle = fmt.Sprintf("What PSC %s (%s) means for federal contracting, and how to find matching opportunities.", code, label)
		}

		data := pageData{
			SEOTitle:        seoTitle,
			MetaDescription: metaDesc,
			CanonicalPath:   fmt.Sprintf("psc/%s", code),
			Breadcrumb:      fmt.Sprintf("PSC %s – %s", code, label),
			H1:              fmt.Sprintf("PSC %s – %s", code, label),
			Subtitle:        subtitle,
			TotalCount:      c.TotalCount,
			RecentCount:     c.RecentCount,
			Opportunities:   toTemplateOpps(opps, allNAICS),
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?psc=%s", code),
			CTACountLabel:   fmt.Sprintf("%d+", c.TotalCount),
			UpdatedDate:     updatedDate,
			NoIndex:         c.TotalCount < indexThreshold,
			IsPSC:           true,
		}
		if hasParent {
			data.ParentLink = fmt.Sprintf("/contracts/psc/group/%s.html", parentKey)
			data.ParentLabel = fmt.Sprintf("PSC %s – %s", parentKey, parentTitle)
			data.SiblingPages = siblingsOf(pscSiblings, parentKey, fmt.Sprintf("psc/%s", code))
			data.SiblingHeading = "Related PSC Codes"
		}

		body, err := renderPage(tmpl, data)
		if err != nil {
			h.Logger.Error("failed to render PSC page", "code", code, "error", err)
			failures.Add(1)
			continue
		}

		put(fmt.Sprintf("contracts/psc/%s.html", code), body, "text/html; charset=utf-8")
		pscCount++
		if c.TotalCount >= indexThreshold {
			sitemapURLs = append(sitemapURLs, fmt.Sprintf("https://govtrove.com/contracts/psc/%s.html", code))
			pscByGroup[parentKey] = append(pscByGroup[parentKey], indexEntry{
				Slug:  fmt.Sprintf("psc/%s", code),
				Label: fmt.Sprintf("PSC %s – %s", code, label),
				Count: c.TotalCount,
			})
		}
	}

	// Generate PSC group hubs. These carry the "psc code list" style queries that a
	// JS-only finder tool can never rank for, and give the 4-character pages a parent
	// to inherit from instead of hanging off a flat index.
	allGroups := psc.Hubs()
	groupCodes := make([]string, 0, len(allGroups))
	for g := range allGroups {
		groupCodes = append(groupCodes, g)
	}
	sort.Strings(groupCodes)

	var pscGroupEntries []indexEntry
	for _, group := range groupCodes {
		label := allGroups[group]
		countKey := "psc_group"
		if len(group) == 1 {
			countKey = "psc_cat"
		}
		c := countMap[countKey][group]

		opps, err := h.Store.GetSEOPageOpportunitiesByPrefix(ctx, "classification_code", group, 25)
		if err != nil {
			h.Logger.Error("failed to get PSC group opps", "group", group, "error", err)
			failures.Add(1)
			continue
		}

		children := pscByGroup[group]
		sort.Slice(children, func(i, j int) bool { return children[i].Count > children[j].Count })

		data := pageData{
			SEOTitle:        fmt.Sprintf("PSC %s – %s: Full Code List, %d Active Contracts", group, label, c.TotalCount),
			MetaDescription: fmt.Sprintf("Every PSC code under group %s (%s), what each covers, and %d active federal contract opportunities — updated daily from SAM.gov.", group, label, c.TotalCount),
			CanonicalPath:   fmt.Sprintf("psc/group/%s", group),
			Breadcrumb:      fmt.Sprintf("PSC %s – %s", group, label),
			H1:              fmt.Sprintf("PSC Group %s – %s", group, label),
			Subtitle:        fmt.Sprintf("All %d product service codes under PSC group %s, and %d active federal contract opportunities.", len(children), group, c.TotalCount),
			TotalCount:      c.TotalCount,
			RecentCount:     c.RecentCount,
			Opportunities:   toTemplateOpps(opps, allNAICS),
			ChildPages:      children,
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?psc=%s", group),
			CTACountLabel:   fmt.Sprintf("%d+", c.TotalCount),
			UpdatedDate:     updatedDate,
			NoIndex:         c.TotalCount < indexThreshold,
			IsPSC:           true,
			IsHub:           true,
		}

		body, err := renderPage(tmpl, data)
		if err != nil {
			h.Logger.Error("failed to render PSC group page", "group", group, "error", err)
			failures.Add(1)
			continue
		}

		// Every group gets a page even when thin, so the static code directory on
		// /psc-code-finder/ can link all 84 of them without risking a 404.
		put(fmt.Sprintf("contracts/psc/group/%s.html", group), body, "text/html; charset=utf-8")
		hubCount++
		if c.TotalCount < indexThreshold {
			continue
		}
		sitemapURLs = append(sitemapURLs, fmt.Sprintf("https://govtrove.com/contracts/psc/group/%s.html", group))
		pscGroupEntries = append(pscGroupEntries, indexEntry{
			Slug:  fmt.Sprintf("psc/group/%s", group),
			Label: fmt.Sprintf("PSC %s – %s", group, label),
			Count: c.TotalCount,
		})
	}
	sort.Slice(pscGroupEntries, func(i, j int) bool { return pscGroupEntries[i].Count > pscGroupEntries[j].Count })

	// Generate NAICS sector hubs, so /contracts/ links 20 sectors instead of 720 codes.
	sectorPrefixes := make([]string, 0, len(sectorTitles))
	for sec := range sectorTitles {
		sectorPrefixes = append(sectorPrefixes, sec)
	}
	sort.Strings(sectorPrefixes)

	var sectorEntries []indexEntry
	for _, sec := range sectorPrefixes {
		label := sectorTitles[sec]
		children := naicsBySector[sec]
		sort.Slice(children, func(i, j int) bool { return children[i].Count > children[j].Count })

		total := sectorCounts[sec]
		data := pageData{
			SEOTitle:        fmt.Sprintf("NAICS Sector %s – %s: Code List, %d Active Contracts", sec, label, total),
			MetaDescription: fmt.Sprintf("Every NAICS code in sector %s (%s) with active federal contract opportunities, what each covers, and %d open opportunities from SAM.gov.", sec, label, total),
			CanonicalPath:   fmt.Sprintf("naics/sector/%s", sec),
			Breadcrumb:      fmt.Sprintf("NAICS Sector %s", sec),
			H1:              fmt.Sprintf("NAICS Sector %s – %s", sec, label),
			Subtitle:        fmt.Sprintf("All %d NAICS codes in sector %s with active federal contract opportunities.", len(children), sec),
			TotalCount:      total,
			ChildPages:      children,
			CTALink:         fmt.Sprintf("https://app.govtrove.com/?naics=%s", sec),
			CTACountLabel:   fmt.Sprintf("%d+", total),
			UpdatedDate:     updatedDate,
			NoIndex:         total < indexThreshold,
			IsNAICS:         true,
			IsHub:           true,
		}

		body, err := renderPage(tmpl, data)
		if err != nil {
			h.Logger.Error("failed to render NAICS sector page", "sector", sec, "error", err)
			failures.Add(1)
			continue
		}

		put(fmt.Sprintf("contracts/naics/sector/%s.html", sec), body, "text/html; charset=utf-8")
		hubCount++
		if total < indexThreshold {
			continue
		}
		sitemapURLs = append(sitemapURLs, fmt.Sprintf("https://govtrove.com/contracts/naics/sector/%s.html", sec))
		sectorEntries = append(sectorEntries, indexEntry{
			Slug:  fmt.Sprintf("naics/sector/%s", sec),
			Label: fmt.Sprintf("%s – %s", sec, label),
			Count: total,
		})
	}
	sort.Slice(sectorEntries, func(i, j int) bool { return sectorEntries[i].Count > sectorEntries[j].Count })

	// Generate index page
	indexData := pageData{
		SEOTitle:        "Federal Contract Opportunities by NAICS Code, PSC Code, Set-Aside, and Agency",
		MetaDescription: "Browse federal contract opportunities by NAICS code, PSC (product service) code, set-aside type (SDVOSB, 8(a), HUBZone, WOSB), and federal agency. Updated daily from SAM.gov.",
		CanonicalPath:   "",
		Breadcrumb:      "Federal Contracts",
		H1:              "Federal Contract Opportunities",
		Subtitle:        "Browse active federal contract opportunities by NAICS code, PSC code, set-aside type, and federal agency. Updated daily from SAM.gov.",
		UpdatedDate:     updatedDate,
		IsIndex:         true,
		SetAsidePages:   setAsideEntries,
		NAICSPages:      sectorEntries,
		AgencyPages:     agencyEntries,
		PSCPages:        pscGroupEntries,
	}

	indexBody, err := renderPage(tmpl, indexData)
	if err != nil {
		return nil, fmt.Errorf("render index page: %w", err)
	}
	put("contracts/index.html", indexBody, "text/html; charset=utf-8")

	// Generate sitemap
	today := now.Format("2006-01-02")
	sitemapBody := buildSitemap(sitemapURLs, today)
	put("sitemap.xml", sitemapBody, "application/xml; charset=utf-8")

	close(uploads)
	wg.Wait()

	pruned, err := h.pruneStalePages(ctx, writtenKeys)
	if err != nil {
		h.Logger.Error("prune stale pages failed", "error", err)
		failures.Add(1)
	}

	// CloudFront invalidation
	if h.DistributionID != "" {
		callerRef := fmt.Sprintf("seo-pages-%d", now.Unix())
		path := "/*"
		_, err := h.CF.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
			DistributionId: &h.DistributionID,
			InvalidationBatch: &cftypes.InvalidationBatch{
				CallerReference: &callerRef,
				Paths: &cftypes.Paths{
					Quantity: int32Ptr(1),
					Items:    []string{path},
				},
			},
		})
		if err != nil {
			h.Logger.Error("cloudfront invalidation failed", "error", err)
			failures.Add(1)
		}
	}

	durationMs := int(time.Since(start).Milliseconds())
	failCount := int(failures.Load())

	h.Logger.Info("seo pages complete",
		"set_aside_pages", setAsideCount,
		"naics_pages", naicsCount,
		"agency_pages", agencyCount,
		"psc_pages", pscCount,
		"hub_pages", hubCount,
		"sitemap_urls", len(sitemapURLs),
		"pruned_pages", pruned,
		"failures", failCount,
		"duration_ms", durationMs,
	)

	if stepID != uuid.Nil {
		stepStats := map[string]any{
			"set_aside_pages": setAsideCount,
			"naics_pages":     naicsCount,
			"agency_pages":    agencyCount,
			"psc_pages":       pscCount,
			"hub_pages":       hubCount,
			"sitemap_urls":    len(sitemapURLs),
			"pruned_pages":    pruned,
			"failures":        failCount,
		}
		completionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Store.CompletePipelineStep(completionCtx, stepID, stepStats, durationMs); err != nil {
			h.Logger.Warn("failed to complete pipeline step", "error", err)
		}
	}

	return &Output{
		Status:      "ok",
		SetAsides:   setAsideCount,
		NAICSPages:  naicsCount,
		AgencyPages: agencyCount,
		PSCPages:    pscCount,
		HubPages:    hubCount,
		SitemapURLs: len(sitemapURLs),
		PrunedPages: pruned,
		Failures:    failCount,
	}, nil
}

func main() {
	h := &Handler{
		Store:          db,
		S3:             s3Client,
		CF:             cfClient,
		S3Bucket:       bucket,
		DistributionID: distID,
		Logger:         logger,
	}
	lambda.Start(h.Handle)
}

// --- Helpers ---

func renderPage(tmpl *template.Template, data pageData) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9 -]`)
var multiSpace = regexp.MustCompile(`[ -]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "")
	s = multiSpace.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func int32Ptr(v int32) *int32 { return &v }

func strPtr(v string) *string { return &v }

// sectorLink and sectorLabel point a NAICS code page at its sector hub. Codes outside the
// known sector list (SAM.gov carries a few) get no parent rather than a broken link.
func sectorLink(code string) string {
	sec := naics.SectorOf(code)
	if naics.Sectors()[sec] == "" {
		return ""
	}
	return fmt.Sprintf("/contracts/naics/sector/%s.html", sec)
}

func sectorLabel(code string) string {
	sec := naics.SectorOf(code)
	title := naics.Sectors()[sec]
	if title == "" {
		return ""
	}
	return fmt.Sprintf("NAICS Sector %s – %s", sec, title)
}

func boolPtr(v bool) *bool { return &v }

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// buildSiblingIndex maps each parent key to every published code beneath it.
//
// It is computed up-front rather than accumulated while pages render: the leaf loop
// would otherwise leave the first pages with no siblings and the last with all of them.
// Only codes at or above indexThreshold are included, so a leaf never links to a page
// we deliberately keep out of the index.
func buildSiblingIndex(
	codes map[string]string,
	counts map[string]database.SEOPageCount,
	parentOf func(string) (string, bool),
	slugPrefix, labelPrefix string,
) map[string][]indexEntry {
	idx := make(map[string][]indexEntry)
	for code, label := range codes {
		c := counts[code]
		if c.TotalCount < indexThreshold {
			continue
		}
		parent, ok := parentOf(code)
		if !ok {
			continue
		}
		idx[parent] = append(idx[parent], indexEntry{
			Slug:  fmt.Sprintf("%s/%s", slugPrefix, code),
			Label: fmt.Sprintf("%s %s – %s", labelPrefix, code, label),
			Count: c.TotalCount,
		})
	}
	for parent := range idx {
		sort.Slice(idx[parent], func(i, j int) bool { return idx[parent][i].Count > idx[parent][j].Count })
	}
	return idx
}

// maxSiblings caps the lateral block. Enough to spread crawl depth and give a reader
// somewhere to go; not so many that the page turns into a link farm.
const maxSiblings = 12

// siblingsOf returns the busiest peers of code under parent, excluding code itself.
func siblingsOf(idx map[string][]indexEntry, parent, selfSlug string) []indexEntry {
	all := idx[parent]
	out := make([]indexEntry, 0, maxSiblings)
	for _, e := range all {
		if e.Slug == selfSlug {
			continue
		}
		out = append(out, e)
		if len(out) == maxSiblings {
			break
		}
	}
	return out
}

// toTemplateOpps converts store rows for rendering. The NAICS code printed on every
// opportunity row becomes a link when we actually publish a page for that code — it is
// the only lateral path between the PSC and NAICS page sets, which are otherwise two
// disconnected islands. Codes we do not generate (retired, 5-digit, malformed) stay
// plain text rather than linking to a 403.
func toTemplateOpps(opps []database.SEOOpportunity, naicsPages map[string]string) []opportunity {
	result := make([]opportunity, len(opps))
	for i, o := range opps {
		result[i] = opportunity{
			NoticeID:           o.NoticeID,
			Title:              o.Title,
			Department:         o.Department,
			NAICSCode:          o.NAICSCode,
			SetAsideCode:       o.SetAsideCode,
			PostedDate:         o.PostedDate,
			ResponseDeadline:   o.ResponseDeadline,
			PopState:           o.PopState,
			SolicitationNumber: o.SolicitationNumber,
		}
		if code := deref(o.NAICSCode); code != "" {
			if _, ok := naicsPages[code]; ok {
				result[i].NAICSLink = fmt.Sprintf("/contracts/naics/%s.html", code)
			}
		}
	}
	return result
}

func buildSitemap(urls []string, today string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, u := range urls {
		priority := "0.7"
		changefreq := "daily"
		if u == "https://govtrove.com/" {
			priority = "1.0"
			changefreq = "weekly"
		} else if strings.HasSuffix(u, "/contracts/") {
			priority = "0.9"
		} else if strings.Contains(u, "/set-aside/") {
			priority = "0.9"
		} else if strings.Contains(u, "/naics/") {
			priority = "0.8"
		} else if strings.Contains(u, "/agency/") {
			priority = "0.8"
		}
		fmt.Fprintf(&buf, "  <url>\n    <loc>%s</loc>\n    <lastmod>%s</lastmod>\n    <changefreq>%s</changefreq>\n    <priority>%s</priority>\n  </url>\n",
			u, today, changefreq, priority)
	}
	buf.WriteString("</urlset>\n")
	return buf.Bytes()
}

// --- Types ---

type setAsideDef struct {
	Code      string
	Slug      string
	ShortName string
	FullName  string
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
	NAICSLink          string
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
	NoIndex         bool
	IsSetAside      bool
	IsNAICS         bool
	IsAgency        bool
	IsIndex         bool
	IsPSC           bool
	IsHub           bool
	ParentLink      string
	ParentLabel     string
	ChildPages      []indexEntry
	SiblingPages    []indexEntry
	SiblingHeading  string
	SetAsidePages   []indexEntry
	NAICSPages      []indexEntry
	AgencyPages     []indexEntry
	PSCPages        []indexEntry
}

type indexEntry struct {
	Slug  string
	Label string
	Count int
}

const pageTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.SEOTitle}} | GovTrove</title>
    <meta name="description" content="{{.MetaDescription}}">
    {{- if .NoIndex}}
    <meta name="robots" content="noindex,follow">
    {{- end}}
    {{- if not .IsIndex}}
    <link rel="canonical" href="https://govtrove.com/contracts/{{.CanonicalPath}}.html">
    {{- else}}
    <link rel="canonical" href="https://govtrove.com/contracts/">
    {{- end}}

    <!-- Open Graph -->
    <meta property="og:title" content="{{.SEOTitle}} | GovTrove">
    <meta property="og:description" content="{{.MetaDescription}}">
    <meta property="og:type" content="website">
    {{- if not .IsIndex}}
    <meta property="og:url" content="https://govtrove.com/contracts/{{.CanonicalPath}}.html">
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
    .meta-link { color: var(--text-secondary); text-decoration: underline; text-underline-offset: 2px; }
    .meta-link:hover { color: var(--accent); }
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
                <li><a href="/contracts/{{.Slug}}.html"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
        </div>

        <div class="index-section">
            <h2>By NAICS Code</h2>
            <p class="subtitle">Browse by industry sector. Each sector lists every NAICS code with active opportunities.</p>
            <ul class="index-list">
                {{- range .NAICSPages}}
                <li><a href="/contracts/{{.Slug}}.html"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
            <p><a href="/naics-code-finder/">Not sure which NAICS code fits? Use the NAICS code finder &rarr;</a></p>
        </div>

        <div class="index-section">
            <h2>By PSC Code (Product and Service Codes)</h2>
            <p class="subtitle">Every PSC group with active opportunities. Each group lists the full set of 4-character codes beneath it.</p>
            <ul class="index-list">
                {{- range .PSCPages}}
                <li><a href="/contracts/{{.Slug}}.html"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
            <p><a href="/psc-code-finder/">Not sure which PSC code fits? Use the PSC code finder &rarr;</a></p>
        </div>

        <div class="index-section">
            <h2>By Agency</h2>
            <ul class="index-list">
                {{- range .AgencyPages}}
                <li><a href="/contracts/{{.Slug}}.html"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
        </div>

        <div class="cta-box">
            <h2>Search All Federal Contracts</h2>
            <p>Filter by keyword, NAICS, set-aside, agency, and more. Free to use.</p>
            <a href="https://app.govtrove.com" class="cta-btn">Search on GovTrove &rarr;</a>
        </div>
        {{- else}}

        {{- if .ParentLink}}
        <p class="subtitle">Part of <a href="{{.ParentLink}}">{{.ParentLabel}}</a></p>
        {{- end}}

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

        {{- if .ChildPages}}
        <div class="index-section">
            <h2>All Codes in This Group</h2>
            <ul class="index-list">
                {{- range .ChildPages}}
                <li><a href="/contracts/{{.Slug}}.html"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
        </div>
        {{- end}}

        <h2>Recent Opportunities</h2>
        <ul class="opp-list">
            {{- range .Opportunities}}
            <li class="opp-item">
                <div class="opp-title"><a href="https://app.govtrove.com/?q={{deref .SolicitationNumber}}">{{.Title}}</a></div>
                <div class="opp-meta">
                    {{- if not $.IsAgency}}
                    <span>Agency: {{.Department}}</span>
                    {{- end}}
                    {{- if .PostedDate}}
                    <span>Posted: {{formatDate .PostedDate}}</span>
                    {{- end}}
                    {{- if .ResponseDeadline}}
                    <span>Due: {{formatDate .ResponseDeadline}}</span>
                    {{- end}}
                    {{- if and .NAICSCode (ne (deref .NAICSCode) "")}}
                    {{- if and .NAICSLink (ne .NAICSLink (printf "/contracts/%s.html" $.CanonicalPath))}}
                    <span>NAICS: <a href="{{.NAICSLink}}" class="meta-link">{{deref .NAICSCode}}</a></span>
                    {{- else}}
                    <span>NAICS: {{deref .NAICSCode}}</span>
                    {{- end}}
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

        {{- if .SiblingPages}}
        <div class="index-section">
            <h2>{{.SiblingHeading}}</h2>
            <ul class="index-list">
                {{- range .SiblingPages}}
                <li><a href="/contracts/{{.Slug}}.html"><span>{{.Label}}</span><span class="index-count">{{commaInt .Count}} active</span></a></li>
                {{- end}}
            </ul>
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
