package ingest

import (
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

func ExtractSnapCSVRow(raw map[string]string) database.SnapCSVRow {
	return database.SnapCSVRow{
		NoticeID:           raw["NoticeId"],
		SolicitationNumber: raw["Sol#"],
		Title:              raw["Title"],
		Type:               raw["Type"],
		BaseType:           raw["BaseType"],
		PostedDate:         samgov.ParseDate(raw["PostedDate"]),
		ResponseDeadline:   samgov.ParseDate(raw["ResponseDeadLine"]),
		ArchiveDate:        raw["ArchiveDate"],
		ArchiveType:        raw["ArchiveType"],
		SetAsideCode:       raw["SetASideCode"],
		NAICSCode:          raw["NaicsCode"],
		ClassificationCode: raw["ClassificationCode"],
		Active:             samgov.ParseActive(raw["Active"]),

		Department: raw["Department/Ind.Agency"],
		SubTier:    raw["Sub-Tier"],
		Office:     raw["Office"],
		CGAC:       raw["CGAC"],
		FPDSCode:   raw["FPDS Code"],
		AACCode:    raw["AAC Code"],

		AwardNumber: raw["AwardNumber"],
		AwardDate:   raw["AwardDate"],
		AwardAmount: samgov.ParseAmount(raw["Award$"]),

		RawData:     raw,
		ContentHash: database.ComputeContentHash(raw),
	}
}
