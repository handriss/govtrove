package e2e_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline E2E", Ordered, func() {
	var (
		ctx           context.Context
		snapshotDate1 time.Time
		snapshotDate2 time.Time
		snapshotDate3 time.Time

		activeRunID1   uuid.UUID
		archivedRunID1 uuid.UUID
		activeRunID2   uuid.UUID
		archivedRunID2 uuid.UUID
		activeRunID3   uuid.UUID
		archivedRunID3 uuid.UUID
	)

	BeforeAll(func() {
		ctx = context.Background()
		snapshotDate1 = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
		snapshotDate2 = time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
		snapshotDate3 = time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	})

	// ================================================================
	// RUN 1: Baseline ingestion
	// ================================================================
	Context("Run 1: baseline ingestion", func() {
		It("ingests active CSV with 8 records", func() {
			var err error
			activeRunID1, err = simulateIngest(ctx, db, activeCSVRun1, "snapshot-csv", snapshotDate1)
			Expect(err).NotTo(HaveOccurred())
			Expect(activeRunID1).NotTo(Equal(uuid.Nil))
		})

		It("ingests archived CSV with 3 records", func() {
			var err error
			archivedRunID1, err = simulateIngest(ctx, db, archivedCSV, "ingest-archived", snapshotDate1)
			Expect(err).NotTo(HaveOccurred())
			Expect(archivedRunID1).NotTo(Equal(uuid.Nil))
		})

		It("reconciles and upserts opportunities", func() {
			err := simulateReconcile(ctx, db, activeRunID1, []uuid.UUID{archivedRunID1}, snapshotDate1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("has correct snap_csv row counts", func() {
			activeCount := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_csv WHERE run_id = $1", activeRunID1)
			Expect(activeCount).To(Equal(8))

			archivedCount := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_csv WHERE run_id = $1", archivedRunID1)
			Expect(archivedCount).To(Equal(3))
		})

		It("has 2 completed ingestion runs", func() {
			count := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.ingestion_runs WHERE status = 'completed' AND job_type IN ('snapshot-csv', 'ingest-archived')")
			Expect(count).To(Equal(2))
		})

		It("has no disappearances on first run", func() {
			count := queryCount(ctx, "SELECT COUNT(*) FROM pipeline.snap_disappearances")
			Expect(count).To(Equal(0))
		})

		It("records data quality issues for sentinel and bad dates", func() {
			count := queryCount(ctx, "SELECT COUNT(*) FROM pipeline.snap_data_quality")
			Expect(count).To(BeNumerically(">=", 2))

			sentinelCount := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_data_quality WHERE notice_id = 'SENTINEL-001' AND issue_type = 'sentinel_date'")
			Expect(sentinelCount).To(BeNumerically(">=", 1))

			badDateCount := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_data_quality WHERE notice_id = 'BADDATE-001' AND issue_type = 'unparseable_date'")
			Expect(badDateCount).To(BeNumerically(">=", 1))
		})

		It("creates correct opportunities", func() {
			// Total: 8 active + 3 archived, but HAPPY-001 and WILL-DISAPPEAR overlap → 9 unique
			total := queryCount(ctx, "SELECT COUNT(*) FROM opportunities")
			Expect(total).To(Equal(9))
		})

		It("upserts HAPPY-001 with correct fields", func() {
			var title string
			var active bool
			var postedDate *time.Time
			err := db.Pool().QueryRow(ctx,
				"SELECT title, active, posted_date FROM opportunities WHERE notice_id = 'HAPPY-001'",
			).Scan(&title, &active, &postedDate)
			Expect(err).NotTo(HaveOccurred())
			Expect(title).To(Equal("Test Solicitation"))
			// Archived CSV runs second and overwrites with active=false
			Expect(postedDate).NotTo(BeNil())
		})

		It("upserts HAPPY-002 with award fields", func() {
			var awardAmount *float64
			var active bool
			err := db.Pool().QueryRow(ctx,
				"SELECT award_amount, active FROM opportunities WHERE notice_id = 'HAPPY-002'",
			).Scan(&awardAmount, &active)
			Expect(err).NotTo(HaveOccurred())
			Expect(awardAmount).NotTo(BeNil())
			Expect(*awardAmount).To(BeNumerically("==", 50000))
			Expect(active).To(BeFalse())
		})

		It("strips sentinel archive date from SENTINEL-001", func() {
			var archiveDate *time.Time
			err := db.Pool().QueryRow(ctx,
				"SELECT archive_date FROM opportunities WHERE notice_id = 'SENTINEL-001'",
			).Scan(&archiveDate)
			Expect(err).NotTo(HaveOccurred())
			Expect(archiveDate).To(BeNil())
		})
	})

	// ================================================================
	// RUN 2: Changes and disappearances
	// ================================================================
	Context("Run 2: changes and disappearances", func() {
		It("ingests active CSV run 2", func() {
			var err error
			activeRunID2, err = simulateIngest(ctx, db, activeCSVRun2, "snapshot-csv", snapshotDate2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("ingests archived CSV run 2", func() {
			var err error
			archivedRunID2, err = simulateIngest(ctx, db, archivedCSV, "ingest-archived", snapshotDate2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("reconciles run 2", func() {
			err := simulateReconcile(ctx, db, activeRunID2, []uuid.UUID{archivedRunID2}, snapshotDate2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("detects BRAND-NEW as a new record", func() {
			count := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_csv c LEFT JOIN pipeline.snap_csv p ON c.notice_id = p.notice_id AND p.run_id = $2 WHERE c.run_id = $1 AND c.notice_id = 'BRAND-NEW' AND p.notice_id IS NULL",
				activeRunID2, activeRunID1)
			Expect(count).To(Equal(1))
		})

		It("detects WILL-CHANGE via content hash mismatch", func() {
			// snap_csv should show hash difference between run 1 and run 2
			count := queryCount(ctx,
				`SELECT COUNT(*) FROM pipeline.snap_csv c
				 JOIN pipeline.snap_csv p ON c.notice_id = p.notice_id AND p.run_id = $2
				 WHERE c.run_id = $1 AND c.notice_id = 'WILL-CHANGE' AND c.content_hash != p.content_hash`,
				activeRunID2, activeRunID1)
			Expect(count).To(Equal(1))
		})

		It("creates disappearances for WILL-DISAPPEAR and WILL-GLITCH", func() {
			count := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_disappearances WHERE run_id = $1",
				activeRunID2)
			Expect(count).To(Equal(2))
		})

		It("resolves WILL-DISAPPEAR disappearance as archived", func() {
			var resolution *string
			err := db.Pool().QueryRow(ctx,
				"SELECT resolution FROM pipeline.snap_disappearances WHERE notice_id = 'WILL-DISAPPEAR' AND run_id = $1",
				activeRunID2,
			).Scan(&resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolution).NotTo(BeNil())
			Expect(*resolution).To(Equal("archived"))
		})

		It("leaves WILL-GLITCH disappearance unresolved", func() {
			var resolution *string
			err := db.Pool().QueryRow(ctx,
				"SELECT resolution FROM pipeline.snap_disappearances WHERE notice_id = 'WILL-GLITCH' AND run_id = $1",
				activeRunID2,
			).Scan(&resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolution).To(BeNil())
		})

		It("marks WILL-GLITCH as inactive in opportunities", func() {
			var active bool
			err := db.Pool().QueryRow(ctx,
				"SELECT active FROM opportunities WHERE notice_id = 'WILL-GLITCH' AND is_latest = true",
			).Scan(&active)
			Expect(err).NotTo(HaveOccurred())
			Expect(active).To(BeFalse())
		})

		It("creates WILL-CHANGE version 2 with new title and amount", func() {
			var title string
			var awardAmount *float64
			var version int
			err := db.Pool().QueryRow(ctx,
				"SELECT title, award_amount, version FROM opportunities WHERE notice_id = 'WILL-CHANGE' AND is_latest = true",
			).Scan(&title, &awardAmount, &version)
			Expect(err).NotTo(HaveOccurred())
			Expect(title).To(Equal("Updated Title"))
			Expect(awardAmount).NotTo(BeNil())
			Expect(*awardAmount).To(BeNumerically("==", 2000))
			Expect(version).To(Equal(2))

			// Old version preserved with is_latest=false
			var oldTitle string
			var oldLatest bool
			err = db.Pool().QueryRow(ctx,
				"SELECT title, is_latest FROM opportunities WHERE notice_id = 'WILL-CHANGE' AND version = 1",
			).Scan(&oldTitle, &oldLatest)
			Expect(err).NotTo(HaveOccurred())
			Expect(oldTitle).To(Equal("Original Title"))
			Expect(oldLatest).To(BeFalse())
		})

		It("adds BRAND-NEW to opportunities", func() {
			// 9 original + 1 BRAND-NEW + 1 WILL-CHANGE v2 = 11 total rows
			total := queryCount(ctx, "SELECT COUNT(*) FROM opportunities")
			Expect(total).To(Equal(11))

			// 10 latest versions (WILL-CHANGE v1 is not latest)
			latestCount := queryCount(ctx, "SELECT COUNT(*) FROM opportunities WHERE is_latest = true")
			Expect(latestCount).To(Equal(10))

			var title string
			err := db.Pool().QueryRow(ctx,
				"SELECT title FROM opportunities WHERE notice_id = 'BRAND-NEW'",
			).Scan(&title)
			Expect(err).NotTo(HaveOccurred())
			Expect(title).To(Equal("Brand New Opportunity"))
		})
	})

	// ================================================================
	// RUN 3: Reappearance
	// ================================================================
	Context("Run 3: reappearance", func() {
		It("ingests active CSV run 3 (WILL-GLITCH returns)", func() {
			var err error
			activeRunID3, err = simulateIngest(ctx, db, activeCSVRun3, "snapshot-csv", snapshotDate3)
			Expect(err).NotTo(HaveOccurred())
		})

		It("ingests archived CSV run 3", func() {
			var err error
			archivedRunID3, err = simulateIngest(ctx, db, archivedCSV, "ingest-archived", snapshotDate3)
			Expect(err).NotTo(HaveOccurred())
		})

		It("reconciles run 3", func() {
			err := simulateReconcile(ctx, db, activeRunID3, []uuid.UUID{archivedRunID3}, snapshotDate3)
			Expect(err).NotTo(HaveOccurred())
		})

		It("resolves WILL-GLITCH disappearance as glitch", func() {
			var resolution *string
			var reappearedDate *time.Time
			err := db.Pool().QueryRow(ctx,
				"SELECT resolution, reappeared_date FROM pipeline.snap_disappearances WHERE notice_id = 'WILL-GLITCH'",
			).Scan(&resolution, &reappearedDate)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolution).NotTo(BeNil())
			Expect(*resolution).To(Equal("glitch"))
			Expect(reappearedDate).NotTo(BeNil())
		})

		It("reactivates WILL-GLITCH in opportunities", func() {
			var active bool
			err := db.Pool().QueryRow(ctx,
				"SELECT active FROM opportunities WHERE notice_id = 'WILL-GLITCH' AND is_latest = true",
			).Scan(&active)
			Expect(err).NotTo(HaveOccurred())
			Expect(active).To(BeTrue())
		})

		It("has no new unresolved disappearances", func() {
			count := queryCount(ctx,
				"SELECT COUNT(*) FROM pipeline.snap_disappearances WHERE run_id = $1 AND resolution IS NULL",
				activeRunID3)
			Expect(count).To(Equal(0))
		})

		It("maintains correct total opportunity count", func() {
			// 11 total rows (including WILL-CHANGE v1), 10 with is_latest=true
			total := queryCount(ctx, "SELECT COUNT(*) FROM opportunities")
			Expect(total).To(Equal(11))

			latestCount := queryCount(ctx, "SELECT COUNT(*) FROM opportunities WHERE is_latest = true")
			Expect(latestCount).To(Equal(10))
		})
	})
})
