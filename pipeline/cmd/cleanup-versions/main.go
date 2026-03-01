package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/jackc/pgx/v5/pgxpool"
)

type versionRow struct {
	id             int64
	version        int
	opp            reconcile.Opportunity
	normalizedHash string
	storedHash     string
}

type keptRow struct {
	id             int64
	newVersion     int
	normalizedHash string
}

const batchSize = 500

func main() {
	dryRun := flag.Bool("dry-run", true, "Print what would change without modifying")
	solicitations := flag.String("solicitations", "", "Comma-separated solicitation numbers")
	noticeIDs := flag.String("notice-ids", "", "Comma-separated notice_ids")
	limit := flag.Int("limit", 0, "Max notice_ids to process (0 = all)")
	verbose := flag.Bool("verbose", false, "Print all versions even when no phantoms found")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	targetIDs, err := findTargetNoticeIDs(ctx, pool, *solicitations, *noticeIDs, *limit)
	if err != nil {
		log.Fatalf("find target notice_ids: %v", err)
	}

	fmt.Printf("Found %d notice_ids with multiple versions\n", len(targetIDs))
	if len(targetIDs) == 0 {
		return
	}

	type cleanupTarget struct {
		noticeID   string
		phantomIDs []int64
		kept       []keptRow
	}

	var totalPhantoms, totalKept int
	var targets []cleanupTarget

	for batchStart := 0; batchStart < len(targetIDs); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(targetIDs) {
			batchEnd = len(targetIDs)
		}
		batchIDs := targetIDs[batchStart:batchEnd]

		allVersions, err := loadVersionsBatch(ctx, pool, batchIDs)
		if err != nil {
			log.Fatalf("load versions batch %d-%d: %v", batchStart, batchEnd, err)
		}

		for _, nid := range batchIDs {
			versions := allVersions[nid]
			if len(versions) < 2 {
				continue
			}

			phantomIDs, kept := identifyPhantoms(versions)

			if *verbose {
				fmt.Printf("notice_id=%s: %d versions\n", nid, len(versions))
				for i, v := range versions {
					fmt.Printf("  v%d (id=%d): normalized=%s  stored=%s\n",
						v.version, v.id, v.normalizedHash[:16], v.storedHash[:16])
					if i > 0 && versions[i].normalizedHash != versions[i-1].normalizedHash {
						diffOpps(versions[i-1].opp, versions[i].opp)
					}
				}
				if len(phantomIDs) == 0 {
					fmt.Println("  -> no phantoms")
				}
				fmt.Println()
			}

			if len(phantomIDs) == 0 {
				continue
			}

			totalPhantoms += len(phantomIDs)
			totalKept += len(kept)

			if *dryRun {
				fmt.Printf("notice_id=%s: %d versions → %d kept, %d phantom\n",
					nid, len(versions), len(kept), len(phantomIDs))
			}

			targets = append(targets, cleanupTarget{
				noticeID:   nid,
				phantomIDs: phantomIDs,
				kept:       kept,
			})
		}

		if batchEnd < len(targetIDs) {
			fmt.Printf("  scanned %d/%d notice_ids...\n", batchEnd, len(targetIDs))
		}
	}

	fmt.Printf("\nSummary: %d phantom versions across %d notice_ids (%d versions kept)\n",
		totalPhantoms, len(targets), totalKept)

	if *dryRun || len(targets) == 0 {
		return
	}

	// Collect all phantom IDs and kept rows for bulk operations
	var allPhantomIDs []int64
	var allKeptIDs []int64
	var allKeptVersions []int
	var allKeptHashes []string
	var allKeptIsLatest []bool

	for _, t := range targets {
		allPhantomIDs = append(allPhantomIDs, t.phantomIDs...)
		for _, k := range t.kept {
			allKeptIDs = append(allKeptIDs, k.id)
			allKeptVersions = append(allKeptVersions, k.newVersion)
			allKeptHashes = append(allKeptHashes, k.normalizedHash)
			allKeptIsLatest = append(allKeptIsLatest, k.newVersion == t.kept[len(t.kept)-1].newVersion)
		}
	}

	fmt.Printf("\nExecuting cleanup: %d deletes, %d updates...\n", len(allPhantomIDs), len(allKeptIDs))

	const deleteBatch = 5000
	for i := 0; i < len(allPhantomIDs); i += deleteBatch {
		end := i + deleteBatch
		if end > len(allPhantomIDs) {
			end = len(allPhantomIDs)
		}
		_, err := pool.Exec(ctx, "DELETE FROM opportunities WHERE id = ANY($1)", allPhantomIDs[i:end])
		if err != nil {
			log.Fatalf("delete batch %d-%d: %v", i, end, err)
		}
		fmt.Printf("  deleted %d/%d phantom rows\n", end, len(allPhantomIDs))
	}

	const updateBatch = 2000
	for i := 0; i < len(allKeptIDs); i += updateBatch {
		end := i + updateBatch
		if end > len(allKeptIDs) {
			end = len(allKeptIDs)
		}
		_, err := pool.Exec(ctx, `
			UPDATE opportunities o SET
				version = v.new_version,
				is_latest = v.is_latest,
				content_hash = v.new_hash
			FROM (
				SELECT unnest($1::bigint[]) AS id,
				       unnest($2::int[]) AS new_version,
				       unnest($3::boolean[]) AS is_latest,
				       unnest($4::text[]) AS new_hash
			) v
			WHERE o.id = v.id
		`, allKeptIDs[i:end], allKeptVersions[i:end], allKeptIsLatest[i:end], allKeptHashes[i:end])
		if err != nil {
			log.Fatalf("update batch %d-%d: %v", i, end, err)
		}
		fmt.Printf("  updated %d/%d kept rows\n", end, len(allKeptIDs))
	}

	fmt.Printf("Done. Removed %d phantom versions from %d notice_ids\n",
		totalPhantoms, len(targets))
}

func findTargetNoticeIDs(ctx context.Context, pool *pgxpool.Pool, solicitations, noticeIDList string, limit int) ([]string, error) {
	var conditions []string
	var args []any
	argNum := 1

	if solicitations != "" {
		solList := strings.Split(solicitations, ",")
		for i := range solList {
			solList[i] = strings.TrimSpace(solList[i])
		}
		conditions = append(conditions, fmt.Sprintf("solicitation_number = ANY($%d)", argNum))
		args = append(args, solList)
		argNum++
	}

	if noticeIDList != "" {
		nids := strings.Split(noticeIDList, ",")
		for i := range nids {
			nids[i] = strings.TrimSpace(nids[i])
		}
		conditions = append(conditions, fmt.Sprintf("notice_id = ANY($%d)", argNum))
		args = append(args, nids)
		argNum++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT notice_id FROM opportunities
		%s
		GROUP BY notice_id
		HAVING COUNT(*) > 1
		ORDER BY notice_id
	`, where)

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query multi-version notice_ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

const versionColumns = `
	id, notice_id, solicitation_number, title, description,
	type, base_type, organization_type,
	posted_date, response_deadline, archive_date, archive_type, active,
	set_aside_code, set_aside_description, naics_code, classification_code,
	department, sub_tier, office, cgac, fpds_code, aac_code, middle_tier,
	pop_street_address, pop_city, pop_state, pop_zip, pop_country,
	pop_city_code, pop_state_code, pop_country_code,
	office_city, office_state, office_zip, office_country,
	award_number, award_date, award_amount, awardee,
	awardee_name, awardee_uei, awardee_street_address,
	awardee_city, awardee_city_code, awardee_state, awardee_state_code,
	awardee_country, awardee_country_code, awardee_zip,
	primary_contact_title, primary_contact_fullname,
	primary_contact_email, primary_contact_phone, primary_contact_fax,
	secondary_contact_title, secondary_contact_fullname,
	secondary_contact_email, secondary_contact_phone, secondary_contact_fax,
	ui_link, additional_info_link, description_url, resource_links,
	full_parent_path_name, full_parent_path_code,
	version, content_hash
`

func loadVersionsBatch(ctx context.Context, pool *pgxpool.Pool, noticeIDs []string) (map[string][]versionRow, error) {
	query := fmt.Sprintf(`SELECT %s FROM opportunities WHERE notice_id = ANY($1) ORDER BY notice_id, version ASC`, versionColumns)

	rows, err := pool.Query(ctx, query, noticeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]versionRow, len(noticeIDs))
	for rows.Next() {
		v, err := scanVersionRow(rows)
		if err != nil {
			return nil, err
		}
		result[v.opp.NoticeID] = append(result[v.opp.NoticeID], v)
	}
	return result, rows.Err()
}

func scanVersionRow(rows interface{ Scan(dest ...any) error }) (versionRow, error) {
	var v versionRow
	var (
		solNum, title, desc, typ, baseType, orgType      *string
		archiveType                                       *string
		setAsideCode, setAsideDesc, naicsCode, classCode *string
		dept, subTier, office, cgac, fpdsCode, aacCode   *string
		middleTier                                        *string
		popStreet, popCity, popState, popZip, popCountry *string
		popCityCode, popStateCode, popCountryCode        *string
		offCity, offState, offZip, offCountry            *string
		awardNum, awardee                                 *string
		awardeeName, awardeeUei, awardeeStreet           *string
		awardeeCity, awardeeCityCode                      *string
		awardeeState, awardeeStateCode                    *string
		awardeeCountry, awardeeCountryCode, awardeeZip  *string
		pc1Title, pc1Name, pc1Email, pc1Phone, pc1Fax    *string
		pc2Title, pc2Name, pc2Email, pc2Phone, pc2Fax    *string
		uiLink, addInfoLink, descURL                      *string
		resourceLinksJSON                                 *string
		fullParentPathName, fullParentPathCode            *string
		storedHash                                        *string
	)

	err := rows.Scan(
		&v.id, &v.opp.NoticeID,
		&solNum, &title, &desc, &typ, &baseType, &orgType,
		&v.opp.PostedDate, &v.opp.ResponseDeadline, &v.opp.ArchiveDate,
		&archiveType, &v.opp.Active,
		&setAsideCode, &setAsideDesc, &naicsCode, &classCode,
		&dept, &subTier, &office, &cgac, &fpdsCode, &aacCode, &middleTier,
		&popStreet, &popCity, &popState, &popZip, &popCountry,
		&popCityCode, &popStateCode, &popCountryCode,
		&offCity, &offState, &offZip, &offCountry,
		&awardNum, &v.opp.AwardDate, &v.opp.AwardAmount, &awardee,
		&awardeeName, &awardeeUei, &awardeeStreet,
		&awardeeCity, &awardeeCityCode, &awardeeState, &awardeeStateCode,
		&awardeeCountry, &awardeeCountryCode, &awardeeZip,
		&pc1Title, &pc1Name, &pc1Email, &pc1Phone, &pc1Fax,
		&pc2Title, &pc2Name, &pc2Email, &pc2Phone, &pc2Fax,
		&uiLink, &addInfoLink, &descURL, &resourceLinksJSON,
		&fullParentPathName, &fullParentPathCode,
		&v.version, &storedHash,
	)
	if err != nil {
		return v, fmt.Errorf("scan version row: %w", err)
	}

	v.storedHash = deref(storedHash)

	v.opp.SolicitationNumber = deref(solNum)
	v.opp.Title = deref(title)
	v.opp.Description = deref(desc)
	v.opp.Type = deref(typ)
	v.opp.BaseType = deref(baseType)
	v.opp.OrganizationType = deref(orgType)
	v.opp.ArchiveType = deref(archiveType)
	v.opp.SetAsideCode = deref(setAsideCode)
	v.opp.SetAsideDescription = deref(setAsideDesc)
	v.opp.NAICSCode = deref(naicsCode)
	v.opp.ClassificationCode = deref(classCode)
	v.opp.Department = deref(dept)
	v.opp.SubTier = deref(subTier)
	v.opp.Office = deref(office)
	v.opp.CGAC = deref(cgac)
	v.opp.FPDSCode = deref(fpdsCode)
	v.opp.AACCode = deref(aacCode)
	v.opp.MiddleTier = deref(middleTier)
	v.opp.PopStreetAddress = deref(popStreet)
	v.opp.PopCity = deref(popCity)
	v.opp.PopState = deref(popState)
	v.opp.PopZip = deref(popZip)
	v.opp.PopCountry = deref(popCountry)
	v.opp.PopCityCode = deref(popCityCode)
	v.opp.PopStateCode = deref(popStateCode)
	v.opp.PopCountryCode = deref(popCountryCode)
	v.opp.OfficeCity = deref(offCity)
	v.opp.OfficeState = deref(offState)
	v.opp.OfficeZip = deref(offZip)
	v.opp.OfficeCountry = deref(offCountry)
	v.opp.AwardNumber = deref(awardNum)
	v.opp.Awardee = deref(awardee)
	v.opp.AwardeeName = deref(awardeeName)
	v.opp.AwardeeUeiSAM = deref(awardeeUei)
	v.opp.AwardeeStreetAddress = deref(awardeeStreet)
	v.opp.AwardeeCity = deref(awardeeCity)
	v.opp.AwardeeCityCode = deref(awardeeCityCode)
	v.opp.AwardeeState = deref(awardeeState)
	v.opp.AwardeeStateCode = deref(awardeeStateCode)
	v.opp.AwardeeCountry = deref(awardeeCountry)
	v.opp.AwardeeCountryCode = deref(awardeeCountryCode)
	v.opp.AwardeeZip = deref(awardeeZip)
	v.opp.PrimaryContactTitle = deref(pc1Title)
	v.opp.PrimaryContactFullname = deref(pc1Name)
	v.opp.PrimaryContactEmail = deref(pc1Email)
	v.opp.PrimaryContactPhone = deref(pc1Phone)
	v.opp.PrimaryContactFax = deref(pc1Fax)
	v.opp.SecondaryContactTitle = deref(pc2Title)
	v.opp.SecondaryContactFullname = deref(pc2Name)
	v.opp.SecondaryContactEmail = deref(pc2Email)
	v.opp.SecondaryContactPhone = deref(pc2Phone)
	v.opp.SecondaryContactFax = deref(pc2Fax)
	v.opp.UILink = deref(uiLink)
	v.opp.AdditionalInfoLink = deref(addInfoLink)
	v.opp.DescriptionURL = deref(descURL)
	v.opp.FullParentPathName = deref(fullParentPathName)
	v.opp.FullParentPathCode = deref(fullParentPathCode)

	if resourceLinksJSON != nil {
		json.Unmarshal([]byte(*resourceLinksJSON), &v.opp.ResourceLinks)
	}

	// Normalize dates to midnight UTC before hashing
	v.opp.PostedDate = truncateToMidnight(v.opp.PostedDate)
	v.opp.ResponseDeadline = truncateToMidnight(v.opp.ResponseDeadline)
	v.opp.ArchiveDate = truncateToMidnight(v.opp.ArchiveDate)
	v.opp.AwardDate = truncateToMidnight(v.opp.AwardDate)

	v.normalizedHash = v.opp.ContentHash()

	return v, nil
}

func identifyPhantoms(versions []versionRow) (phantomIDs []int64, kept []keptRow) {
	if len(versions) == 0 {
		return nil, nil
	}

	lastHash := versions[0].normalizedHash
	newVersion := 1
	kept = append(kept, keptRow{
		id:             versions[0].id,
		newVersion:     newVersion,
		normalizedHash: lastHash,
	})

	for _, v := range versions[1:] {
		if v.normalizedHash == lastHash {
			phantomIDs = append(phantomIDs, v.id)
		} else {
			newVersion++
			lastHash = v.normalizedHash
			kept = append(kept, keptRow{
				id:             v.id,
				newVersion:     newVersion,
				normalizedHash: v.normalizedHash,
			})
		}
	}

	return phantomIDs, kept
}

func diffOpps(a, b reconcile.Opportunity) {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	t := va.Type()
	for i := 0; i < va.NumField(); i++ {
		fa := va.Field(i)
		fb := vb.Field(i)
		if !reflect.DeepEqual(fa.Interface(), fb.Interface()) {
			fmt.Printf("    DIFF %s: %v → %v\n", t.Field(i).Name, fa.Interface(), fb.Interface())
		}
	}
}

func truncateToMidnight(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	midnight := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	return &midnight
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
