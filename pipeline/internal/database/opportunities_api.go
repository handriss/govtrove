package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/jackc/pgx/v5"
)

func (db *DB) UpsertOpportunitiesFromAPI(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (affected int, err error) {
	for i := 0; i < len(opps); i += upsertBatchSize {
		end := i + upsertBatchSize
		if end > len(opps) {
			end = len(opps)
		}
		batchAffected, batchFailed, batchErr := db.upsertOpportunitiesFromAPIBatch(ctx, runID, snapshotDate, opps[i:end])
		if batchErr != nil {
			return affected, fmt.Errorf("api upsert batch %d-%d: %w", i, end, batchErr)
		}
		if batchFailed > 0 {
			slog.Warn("api batch had row failures", "batch_start", i, "failed", batchFailed)
		}
		affected += batchAffected
	}
	return affected, nil
}

// existingAPIRow holds existing opportunity data needed for COALESCE-then-hash.
type existingAPIRow struct {
	Version     int
	ContentHash string
	Opp         reconcile.Opportunity
}

func (db *DB) upsertOpportunitiesFromAPIBatch(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (affected, failed int, err error) {
	// Collect valid notice_ids
	var noticeIDs []string
	oppMap := make(map[string]reconcile.Opportunity)
	for _, o := range opps {
		if o.NoticeID == "" {
			continue
		}
		noticeIDs = append(noticeIDs, o.NoticeID)
		oppMap[o.NoticeID] = o
	}
	if len(noticeIDs) == 0 {
		return 0, 0, nil
	}

	// Query existing latest rows with content fields for COALESCE merge
	existing := make(map[string]existingAPIRow)
	rows, err := db.pool.Query(ctx, `
		SELECT notice_id, COALESCE(content_hash, ''), version,
			solicitation_number, title, description, type, base_type, organization_type,
			posted_date, response_deadline, archive_date, archive_type, active,
			set_aside_code, set_aside_description, naics_code, classification_code,
			department, sub_tier, office, cgac, fpds_code, aac_code,
			pop_street_address, pop_city, pop_state, pop_zip, pop_country,
			office_city, office_state, office_zip, office_country,
			award_number, award_date, award_amount, awardee,
			primary_contact_title, primary_contact_fullname, primary_contact_email, primary_contact_phone, primary_contact_fax,
			secondary_contact_title, secondary_contact_fullname, secondary_contact_email, secondary_contact_phone, secondary_contact_fax,
			ui_link,
			full_parent_path_name, full_parent_path_code, description_url, additional_info_link
		FROM opportunities
		WHERE notice_id = ANY($1) AND is_latest = true
	`, noticeIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("query existing rows: %w", err)
	}
	for rows.Next() {
		var ex existingAPIRow
		var nid string
		var solNum, title, desc, typ, baseType, orgType *string
		var postedDate, responseDeadline *time.Time
		var archiveDate *time.Time
		var archiveType *string
		var active bool
		var setAsideCode, setAsideDesc, naicsCode, classCode *string
		var dept, subTier, office, cgac, fpdsCode, aacCode *string
		var popStreet, popCity, popState, popZip, popCountry *string
		var offCity, offState, offZip, offCountry *string
		var awardNum *string
		var awardDate *time.Time
		var awardAmt *float64
		var awardee *string
		var pc1Title, pc1Name, pc1Email, pc1Phone, pc1Fax *string
		var pc2Title, pc2Name, pc2Email, pc2Phone, pc2Fax *string
		var uiLink *string
		var fppName, fppCode, descURL, addInfoLink *string

		if err := rows.Scan(
			&nid, &ex.ContentHash, &ex.Version,
			&solNum, &title, &desc, &typ, &baseType, &orgType,
			&postedDate, &responseDeadline, &archiveDate, &archiveType, &active,
			&setAsideCode, &setAsideDesc, &naicsCode, &classCode,
			&dept, &subTier, &office, &cgac, &fpdsCode, &aacCode,
			&popStreet, &popCity, &popState, &popZip, &popCountry,
			&offCity, &offState, &offZip, &offCountry,
			&awardNum, &awardDate, &awardAmt, &awardee,
			&pc1Title, &pc1Name, &pc1Email, &pc1Phone, &pc1Fax,
			&pc2Title, &pc2Name, &pc2Email, &pc2Phone, &pc2Fax,
			&uiLink,
			&fppName, &fppCode, &descURL, &addInfoLink,
		); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan existing row: %w", err)
		}

		ex.Opp = reconcile.Opportunity{
			NoticeID:                 nid,
			SolicitationNumber:      derefStr(solNum),
			Title:                   derefStr(title),
			Description:             derefStr(desc),
			Type:                    derefStr(typ),
			BaseType:                derefStr(baseType),
			OrganizationType:        derefStr(orgType),
			PostedDate:              postedDate,
			ResponseDeadline:        responseDeadline,
			ArchiveDate:             archiveDate,
			ArchiveType:             derefStr(archiveType),
			Active:                  active,
			SetAsideCode:            derefStr(setAsideCode),
			SetAsideDescription:     derefStr(setAsideDesc),
			NAICSCode:               derefStr(naicsCode),
			ClassificationCode:      derefStr(classCode),
			Department:              derefStr(dept),
			SubTier:                 derefStr(subTier),
			Office:                  derefStr(office),
			CGAC:                    derefStr(cgac),
			FPDSCode:                derefStr(fpdsCode),
			AACCode:                 derefStr(aacCode),
			PopStreetAddress:        derefStr(popStreet),
			PopCity:                 derefStr(popCity),
			PopState:                derefStr(popState),
			PopZip:                  derefStr(popZip),
			PopCountry:             derefStr(popCountry),
			OfficeCity:              derefStr(offCity),
			OfficeState:             derefStr(offState),
			OfficeZip:               derefStr(offZip),
			OfficeCountry:           derefStr(offCountry),
			AwardNumber:             derefStr(awardNum),
			AwardDate:               awardDate,
			AwardAmount:             awardAmt,
			Awardee:                 derefStr(awardee),
			PrimaryContactTitle:     derefStr(pc1Title),
			PrimaryContactFullname:  derefStr(pc1Name),
			PrimaryContactEmail:     derefStr(pc1Email),
			PrimaryContactPhone:     derefStr(pc1Phone),
			PrimaryContactFax:       derefStr(pc1Fax),
			SecondaryContactTitle:   derefStr(pc2Title),
			SecondaryContactFullname: derefStr(pc2Name),
			SecondaryContactEmail:   derefStr(pc2Email),
			SecondaryContactPhone:   derefStr(pc2Phone),
			SecondaryContactFax:     derefStr(pc2Fax),
			UILink:                  derefStr(uiLink),
			FullParentPathName:      derefStr(fppName),
			FullParentPathCode:      derefStr(fppCode),
			DescriptionURL:          derefStr(descURL),
			AdditionalInfoLink:      derefStr(addInfoLink),
		}
		existing[nid] = ex
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iterate existing rows: %w", err)
	}

	batch := &pgx.Batch{}
	var batchNoticeIDs []string

	for _, nid := range noticeIDs {
		o := oppMap[nid]
		ex, exists := existing[nid]

		var resourceLinksJSON *string
		if len(o.ResourceLinks) > 0 {
			b, _ := json.Marshal(o.ResourceLinks)
			s := string(b)
			resourceLinksJSON = &s
		}

		if !exists {
			// New notice_id → INSERT version 1
			hash := o.ContentHash()
			batch.Queue(apiInsertSQL,
				o.NoticeID, nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description),
				nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),
				o.PostedDate, o.ResponseDeadline, o.ArchiveDate, nilIfEmpty(o.ArchiveType), o.Active,
				nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription), nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),
				nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),
				nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode),
				nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState), nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),
				nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry),
				nilIfEmpty(o.AwardNumber), o.AwardDate, o.AwardAmount, nilIfEmpty(o.Awardee),
				nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname), nilIfEmpty(o.PrimaryContactEmail),
				nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax),
				nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname), nilIfEmpty(o.SecondaryContactEmail),
				nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax),
				nilIfEmpty(o.UILink),
				nilIfEmpty(o.FullParentPathName), nilIfEmpty(o.FullParentPathCode), nilIfEmpty(o.DescriptionURL),
				nilIfEmpty(o.AdditionalInfoLink), resourceLinksJSON,
				1, true, hash,
				runID, snapshotDate,
			)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else {
			// Merge API values into existing (COALESCE logic)
			merged := coalesceOpp(ex.Opp, o)
			mergedHash := merged.ContentHash()

			if ex.ContentHash == mergedHash {
				// Same content after merge → metadata-only update
				batch.Queue(`
					UPDATE opportunities SET
						last_api_run_id = $1, last_seen_api = $2,
						data_sources = CASE WHEN data_sources LIKE '%csv%' THEN 'csv+api' ELSE 'api' END
					WHERE notice_id = $3 AND is_latest = true
				`, runID, snapshotDate, o.NoticeID)
				batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
			} else if ex.ContentHash == "" {
				// Pre-migration row with no hash — backfill merged content + hash, no new version
				var mergedResourceLinksJSON *string
				if len(merged.ResourceLinks) > 0 {
					b, _ := json.Marshal(merged.ResourceLinks)
					s := string(b)
					mergedResourceLinksJSON = &s
				} else {
					mergedResourceLinksJSON = resourceLinksJSON
				}
				batch.Queue(`
					UPDATE opportunities SET
						solicitation_number = COALESCE($1, solicitation_number),
						title = COALESCE($2, title), description = COALESCE($3, description),
						type = COALESCE($4, type), base_type = COALESCE($5, base_type), organization_type = COALESCE($6, organization_type),
						posted_date = COALESCE($7, posted_date), response_deadline = COALESCE($8, response_deadline),
						archive_date = COALESCE($9, archive_date), archive_type = COALESCE($10, archive_type), active = $11,
						set_aside_code = COALESCE($12, set_aside_code), set_aside_description = COALESCE($13, set_aside_description),
						naics_code = COALESCE($14, naics_code), classification_code = COALESCE($15, classification_code),
						department = COALESCE($16, department), sub_tier = COALESCE($17, sub_tier), office = COALESCE($18, office),
						cgac = COALESCE($19, cgac), fpds_code = COALESCE($20, fpds_code), aac_code = COALESCE($21, aac_code),
						pop_street_address = COALESCE($22, pop_street_address), pop_city = COALESCE($23, pop_city),
						pop_state = COALESCE($24, pop_state), pop_zip = COALESCE($25, pop_zip), pop_country = COALESCE($26, pop_country),
						office_city = COALESCE($27, office_city), office_state = COALESCE($28, office_state),
						office_zip = COALESCE($29, office_zip), office_country = COALESCE($30, office_country),
						award_number = COALESCE($31, award_number), award_date = COALESCE($32, award_date),
						award_amount = COALESCE($33, award_amount), awardee = COALESCE($34, awardee),
						primary_contact_title = COALESCE($35, primary_contact_title), primary_contact_fullname = COALESCE($36, primary_contact_fullname),
						primary_contact_email = COALESCE($37, primary_contact_email), primary_contact_phone = COALESCE($38, primary_contact_phone),
						primary_contact_fax = COALESCE($39, primary_contact_fax),
						secondary_contact_title = COALESCE($40, secondary_contact_title), secondary_contact_fullname = COALESCE($41, secondary_contact_fullname),
						secondary_contact_email = COALESCE($42, secondary_contact_email), secondary_contact_phone = COALESCE($43, secondary_contact_phone),
						secondary_contact_fax = COALESCE($44, secondary_contact_fax),
						ui_link = COALESCE($45, ui_link),
						full_parent_path_name = COALESCE($46, full_parent_path_name), full_parent_path_code = COALESCE($47, full_parent_path_code),
						description_url = COALESCE($48, description_url), additional_info_link = COALESCE($49, additional_info_link),
						resource_links = COALESCE($50, resource_links),
						content_hash = $51,
						data_sources = CASE WHEN data_sources LIKE '%csv%' THEN 'csv+api' ELSE 'api' END,
						last_api_run_id = $52, last_seen_api = $53
					WHERE notice_id = $54 AND is_latest = true
				`,
					nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description),
					nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),
					o.PostedDate, o.ResponseDeadline, o.ArchiveDate, nilIfEmpty(o.ArchiveType), o.Active,
					nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription), nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),
					nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),
					nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode),
					nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState), nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),
					nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry),
					nilIfEmpty(o.AwardNumber), o.AwardDate, o.AwardAmount, nilIfEmpty(o.Awardee),
					nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname), nilIfEmpty(o.PrimaryContactEmail),
					nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax),
					nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname), nilIfEmpty(o.SecondaryContactEmail),
					nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax),
					nilIfEmpty(o.UILink),
					nilIfEmpty(o.FullParentPathName), nilIfEmpty(o.FullParentPathCode), nilIfEmpty(o.DescriptionURL),
					nilIfEmpty(o.AdditionalInfoLink), mergedResourceLinksJSON,
					mergedHash,
					runID, snapshotDate, o.NoticeID,
				)
				batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
			} else {
				// Content changed → demote old, insert new version
				newVersion := ex.Version + 1

				// Merge resource links for insert
				var mergedResourceLinksJSON *string
				if len(merged.ResourceLinks) > 0 {
					b, _ := json.Marshal(merged.ResourceLinks)
					s := string(b)
					mergedResourceLinksJSON = &s
				} else {
					mergedResourceLinksJSON = resourceLinksJSON
				}

				batch.Queue(`
					UPDATE opportunities SET is_latest = false
					WHERE notice_id = $1 AND is_latest = true
				`, o.NoticeID)
				batchNoticeIDs = append(batchNoticeIDs, o.NoticeID+":demote")

				// Determine data_sources for new row
				dataSources := "api"
				// Check if previous had csv
				if ex.Opp.NoticeID != "" {
					// We'll set it based on what the old row likely had; safe default
					dataSources = "csv+api"
				}

				batch.Queue(apiInsertSQL,
					merged.NoticeID, nilIfEmpty(merged.SolicitationNumber), nilIfEmpty(merged.Title), nilIfEmpty(merged.Description),
					nilIfEmpty(merged.Type), nilIfEmpty(merged.BaseType), nilIfEmpty(merged.OrganizationType),
					merged.PostedDate, merged.ResponseDeadline, merged.ArchiveDate, nilIfEmpty(merged.ArchiveType), merged.Active,
					nilIfEmpty(merged.SetAsideCode), nilIfEmpty(merged.SetAsideDescription), nilIfEmpty(merged.NAICSCode), nilIfEmpty(merged.ClassificationCode),
					nilIfEmpty(merged.Department), nilIfEmpty(merged.SubTier), nilIfEmpty(merged.Office),
					nilIfEmpty(merged.CGAC), nilIfEmpty(merged.FPDSCode), nilIfEmpty(merged.AACCode),
					nilIfEmpty(merged.PopStreetAddress), nilIfEmpty(merged.PopCity), nilIfEmpty(merged.PopState), nilIfEmpty(merged.PopZip), nilIfEmpty(merged.PopCountry),
					nilIfEmpty(merged.OfficeCity), nilIfEmpty(merged.OfficeState), nilIfEmpty(merged.OfficeZip), nilIfEmpty(merged.OfficeCountry),
					nilIfEmpty(merged.AwardNumber), merged.AwardDate, merged.AwardAmount, nilIfEmpty(merged.Awardee),
					nilIfEmpty(merged.PrimaryContactTitle), nilIfEmpty(merged.PrimaryContactFullname), nilIfEmpty(merged.PrimaryContactEmail),
					nilIfEmpty(merged.PrimaryContactPhone), nilIfEmpty(merged.PrimaryContactFax),
					nilIfEmpty(merged.SecondaryContactTitle), nilIfEmpty(merged.SecondaryContactFullname), nilIfEmpty(merged.SecondaryContactEmail),
					nilIfEmpty(merged.SecondaryContactPhone), nilIfEmpty(merged.SecondaryContactFax),
					nilIfEmpty(merged.UILink),
					nilIfEmpty(merged.FullParentPathName), nilIfEmpty(merged.FullParentPathCode), nilIfEmpty(merged.DescriptionURL),
					nilIfEmpty(merged.AdditionalInfoLink), mergedResourceLinksJSON,
					newVersion, true, mergedHash,
					runID, snapshotDate,
				)
				batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)

				// Update data_sources on the new row
				batch.Queue(`
					UPDATE opportunities SET data_sources = $1
					WHERE notice_id = $2 AND version = $3
				`, dataSources, merged.NoticeID, newVersion)
				batchNoticeIDs = append(batchNoticeIDs, o.NoticeID+":ds")
			}
		}
	}

	if batch.Len() == 0 {
		return 0, 0, nil
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < batch.Len(); i++ {
		tag, err := results.Exec()
		if err != nil {
			noticeID := ""
			if i < len(batchNoticeIDs) {
				noticeID = batchNoticeIDs[i]
			}
			slog.Warn("api upsert row failed", "notice_id", noticeID, "error", err)
			failed++
			continue
		}
		if tag.RowsAffected() > 0 {
			affected++
		}
	}

	return affected, failed, nil
}

const apiInsertSQL = `
	INSERT INTO opportunities (
		notice_id, solicitation_number, title, description, type, base_type, organization_type,
		posted_date, response_deadline, archive_date, archive_type, active,
		set_aside_code, set_aside_description, naics_code, classification_code,
		department, sub_tier, office, cgac, fpds_code, aac_code,
		pop_street_address, pop_city, pop_state, pop_zip, pop_country,
		office_city, office_state, office_zip, office_country,
		award_number, award_date, award_amount, awardee,
		primary_contact_title, primary_contact_fullname, primary_contact_email, primary_contact_phone, primary_contact_fax,
		secondary_contact_title, secondary_contact_fullname, secondary_contact_email, secondary_contact_phone, secondary_contact_fax,
		ui_link,
		full_parent_path_name, full_parent_path_code, description_url, additional_info_link, resource_links,
		version, is_latest, content_hash,
		data_sources, last_api_run_id, last_seen_api
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11, $12,
		$13, $14, $15, $16,
		$17, $18, $19, $20, $21, $22,
		$23, $24, $25, $26, $27,
		$28, $29, $30, $31,
		$32, $33, $34, $35,
		$36, $37, $38, $39, $40,
		$41, $42, $43, $44, $45,
		$46,
		$47, $48, $49, $50, $51,
		$52, $53, $54,
		'api', $55, $56
	)`

// coalesceOpp merges API opportunity values into existing, preferring non-empty API values.
func coalesceOpp(existing, api reconcile.Opportunity) reconcile.Opportunity {
	m := existing
	m.NoticeID = api.NoticeID
	m.Active = api.Active

	if api.SolicitationNumber != "" {
		m.SolicitationNumber = api.SolicitationNumber
	}
	if api.Title != "" {
		m.Title = api.Title
	}
	if api.Description != "" {
		m.Description = api.Description
	}
	if api.Type != "" {
		m.Type = api.Type
	}
	if api.BaseType != "" {
		m.BaseType = api.BaseType
	}
	if api.OrganizationType != "" {
		m.OrganizationType = api.OrganizationType
	}
	if api.PostedDate != nil {
		m.PostedDate = api.PostedDate
	}
	if api.ResponseDeadline != nil {
		m.ResponseDeadline = api.ResponseDeadline
	}
	if api.ArchiveDate != nil {
		m.ArchiveDate = api.ArchiveDate
	}
	if api.ArchiveType != "" {
		m.ArchiveType = api.ArchiveType
	}
	if api.SetAsideCode != "" {
		m.SetAsideCode = api.SetAsideCode
	}
	if api.SetAsideDescription != "" {
		m.SetAsideDescription = api.SetAsideDescription
	}
	if api.NAICSCode != "" {
		m.NAICSCode = api.NAICSCode
	}
	if api.ClassificationCode != "" {
		m.ClassificationCode = api.ClassificationCode
	}
	if api.Department != "" {
		m.Department = api.Department
	}
	if api.SubTier != "" {
		m.SubTier = api.SubTier
	}
	if api.Office != "" {
		m.Office = api.Office
	}
	if api.CGAC != "" {
		m.CGAC = api.CGAC
	}
	if api.FPDSCode != "" {
		m.FPDSCode = api.FPDSCode
	}
	if api.AACCode != "" {
		m.AACCode = api.AACCode
	}
	if api.PopStreetAddress != "" {
		m.PopStreetAddress = api.PopStreetAddress
	}
	if api.PopCity != "" {
		m.PopCity = api.PopCity
	}
	if api.PopState != "" {
		m.PopState = api.PopState
	}
	if api.PopZip != "" {
		m.PopZip = api.PopZip
	}
	if api.PopCountry != "" {
		m.PopCountry = api.PopCountry
	}
	if api.OfficeCity != "" {
		m.OfficeCity = api.OfficeCity
	}
	if api.OfficeState != "" {
		m.OfficeState = api.OfficeState
	}
	if api.OfficeZip != "" {
		m.OfficeZip = api.OfficeZip
	}
	if api.OfficeCountry != "" {
		m.OfficeCountry = api.OfficeCountry
	}
	if api.AwardNumber != "" {
		m.AwardNumber = api.AwardNumber
	}
	if api.AwardDate != nil {
		m.AwardDate = api.AwardDate
	}
	if api.AwardAmount != nil {
		m.AwardAmount = api.AwardAmount
	}
	if api.Awardee != "" {
		m.Awardee = api.Awardee
	}
	if api.PrimaryContactTitle != "" {
		m.PrimaryContactTitle = api.PrimaryContactTitle
	}
	if api.PrimaryContactFullname != "" {
		m.PrimaryContactFullname = api.PrimaryContactFullname
	}
	if api.PrimaryContactEmail != "" {
		m.PrimaryContactEmail = api.PrimaryContactEmail
	}
	if api.PrimaryContactPhone != "" {
		m.PrimaryContactPhone = api.PrimaryContactPhone
	}
	if api.PrimaryContactFax != "" {
		m.PrimaryContactFax = api.PrimaryContactFax
	}
	if api.SecondaryContactTitle != "" {
		m.SecondaryContactTitle = api.SecondaryContactTitle
	}
	if api.SecondaryContactFullname != "" {
		m.SecondaryContactFullname = api.SecondaryContactFullname
	}
	if api.SecondaryContactEmail != "" {
		m.SecondaryContactEmail = api.SecondaryContactEmail
	}
	if api.SecondaryContactPhone != "" {
		m.SecondaryContactPhone = api.SecondaryContactPhone
	}
	if api.SecondaryContactFax != "" {
		m.SecondaryContactFax = api.SecondaryContactFax
	}
	if api.UILink != "" {
		m.UILink = api.UILink
	}
	if api.FullParentPathName != "" {
		m.FullParentPathName = api.FullParentPathName
	}
	if api.FullParentPathCode != "" {
		m.FullParentPathCode = api.FullParentPathCode
	}
	if api.DescriptionURL != "" {
		m.DescriptionURL = api.DescriptionURL
	}
	if api.AdditionalInfoLink != "" {
		m.AdditionalInfoLink = api.AdditionalInfoLink
	}
	if len(api.ResourceLinks) > 0 {
		m.ResourceLinks = api.ResourceLinks
	}
	return m
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
