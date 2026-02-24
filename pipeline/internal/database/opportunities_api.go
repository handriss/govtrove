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

func (db *DB) upsertOpportunitiesFromAPIBatch(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (affected, failed int, err error) {
	batch := &pgx.Batch{}
	var noticeIDs []string

	for _, o := range opps {
		if o.NoticeID == "" {
			continue
		}
		noticeIDs = append(noticeIDs, o.NoticeID)

		var resourceLinksJSON *string
		if len(o.ResourceLinks) > 0 {
			b, _ := json.Marshal(o.ResourceLinks)
			s := string(b)
			resourceLinksJSON = &s
		}

		batch.Queue(`
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
				'api', $52, $53
			)
			ON CONFLICT (notice_id) DO UPDATE SET
				solicitation_number = COALESCE(EXCLUDED.solicitation_number, opportunities.solicitation_number),
				title = COALESCE(EXCLUDED.title, opportunities.title),
				description = COALESCE(EXCLUDED.description, opportunities.description),
				type = COALESCE(EXCLUDED.type, opportunities.type),
				base_type = COALESCE(EXCLUDED.base_type, opportunities.base_type),
				organization_type = COALESCE(EXCLUDED.organization_type, opportunities.organization_type),
				posted_date = COALESCE(EXCLUDED.posted_date, opportunities.posted_date),
				response_deadline = COALESCE(EXCLUDED.response_deadline, opportunities.response_deadline),
				archive_date = COALESCE(EXCLUDED.archive_date, opportunities.archive_date),
				archive_type = COALESCE(EXCLUDED.archive_type, opportunities.archive_type),
				active = EXCLUDED.active,
				set_aside_code = COALESCE(EXCLUDED.set_aside_code, opportunities.set_aside_code),
				set_aside_description = COALESCE(EXCLUDED.set_aside_description, opportunities.set_aside_description),
				naics_code = COALESCE(EXCLUDED.naics_code, opportunities.naics_code),
				classification_code = COALESCE(EXCLUDED.classification_code, opportunities.classification_code),
				department = COALESCE(EXCLUDED.department, opportunities.department),
				sub_tier = COALESCE(EXCLUDED.sub_tier, opportunities.sub_tier),
				office = COALESCE(EXCLUDED.office, opportunities.office),
				cgac = COALESCE(EXCLUDED.cgac, opportunities.cgac),
				fpds_code = COALESCE(EXCLUDED.fpds_code, opportunities.fpds_code),
				aac_code = COALESCE(EXCLUDED.aac_code, opportunities.aac_code),
				pop_street_address = COALESCE(EXCLUDED.pop_street_address, opportunities.pop_street_address),
				pop_city = COALESCE(EXCLUDED.pop_city, opportunities.pop_city),
				pop_state = COALESCE(EXCLUDED.pop_state, opportunities.pop_state),
				pop_zip = COALESCE(EXCLUDED.pop_zip, opportunities.pop_zip),
				pop_country = COALESCE(EXCLUDED.pop_country, opportunities.pop_country),
				office_city = COALESCE(EXCLUDED.office_city, opportunities.office_city),
				office_state = COALESCE(EXCLUDED.office_state, opportunities.office_state),
				office_zip = COALESCE(EXCLUDED.office_zip, opportunities.office_zip),
				office_country = COALESCE(EXCLUDED.office_country, opportunities.office_country),
				award_number = COALESCE(EXCLUDED.award_number, opportunities.award_number),
				award_date = COALESCE(EXCLUDED.award_date, opportunities.award_date),
				award_amount = COALESCE(EXCLUDED.award_amount, opportunities.award_amount),
				awardee = COALESCE(EXCLUDED.awardee, opportunities.awardee),
				primary_contact_title = COALESCE(EXCLUDED.primary_contact_title, opportunities.primary_contact_title),
				primary_contact_fullname = COALESCE(EXCLUDED.primary_contact_fullname, opportunities.primary_contact_fullname),
				primary_contact_email = COALESCE(EXCLUDED.primary_contact_email, opportunities.primary_contact_email),
				primary_contact_phone = COALESCE(EXCLUDED.primary_contact_phone, opportunities.primary_contact_phone),
				primary_contact_fax = COALESCE(EXCLUDED.primary_contact_fax, opportunities.primary_contact_fax),
				secondary_contact_title = COALESCE(EXCLUDED.secondary_contact_title, opportunities.secondary_contact_title),
				secondary_contact_fullname = COALESCE(EXCLUDED.secondary_contact_fullname, opportunities.secondary_contact_fullname),
				secondary_contact_email = COALESCE(EXCLUDED.secondary_contact_email, opportunities.secondary_contact_email),
				secondary_contact_phone = COALESCE(EXCLUDED.secondary_contact_phone, opportunities.secondary_contact_phone),
				secondary_contact_fax = COALESCE(EXCLUDED.secondary_contact_fax, opportunities.secondary_contact_fax),
				ui_link = COALESCE(EXCLUDED.ui_link, opportunities.ui_link),
				full_parent_path_name = COALESCE(EXCLUDED.full_parent_path_name, opportunities.full_parent_path_name),
				full_parent_path_code = COALESCE(EXCLUDED.full_parent_path_code, opportunities.full_parent_path_code),
				description_url = COALESCE(EXCLUDED.description_url, opportunities.description_url),
				additional_info_link = COALESCE(EXCLUDED.additional_info_link, opportunities.additional_info_link),
				resource_links = COALESCE(EXCLUDED.resource_links, opportunities.resource_links),
				data_sources = CASE
					WHEN opportunities.data_sources LIKE '%csv%' THEN 'csv+api'
					ELSE 'api'
				END,
				last_api_run_id = EXCLUDED.last_api_run_id,
				last_seen_api = EXCLUDED.last_seen_api
		`,
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
			runID, snapshotDate,
		)
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
			if i < len(noticeIDs) {
				noticeID = noticeIDs[i]
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
