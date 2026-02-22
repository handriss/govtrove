-- Reverse migration 023: Drop snapshot tables, restore renames

DROP TABLE IF EXISTS snap_data_quality CASCADE;
DROP TABLE IF EXISTS snap_disappearances CASCADE;
DROP TABLE IF EXISTS snap_changes CASCADE;
DROP TABLE IF EXISTS snap_archived_csv CASCADE;
DROP TABLE IF EXISTS snap_csv CASCADE;
DROP TABLE IF EXISTS archived_csv_download_log CASCADE;
DROP TABLE IF EXISTS csv_download_log CASCADE;
DROP TABLE IF EXISTS ingestion_runs CASCADE;

-- Restore archive log table renames
ALTER TABLE archived_csv_s3_archive_log RENAME TO archived_csv_download_log;
ALTER INDEX idx_archived_csv_s3_archive_fy_result RENAME TO idx_archived_csv_dl_fy_result;

ALTER TABLE csv_s3_archive_log RENAME TO csv_download_log;
ALTER INDEX idx_csv_s3_archive_log_checked_at RENAME TO idx_csv_download_log_checked_at;
ALTER INDEX idx_csv_s3_archive_log_sha256 RENAME TO idx_csv_download_log_sha256;
