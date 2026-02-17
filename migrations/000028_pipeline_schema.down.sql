DROP TABLE IF EXISTS pipeline.pipeline_runs;

ALTER TABLE IF EXISTS pipeline.archived_csv_download_log SET SCHEMA public;
ALTER TABLE IF EXISTS pipeline.snap_archived_csv SET SCHEMA public;
ALTER TABLE pipeline.snap_data_quality SET SCHEMA public;
ALTER TABLE pipeline.snap_disappearances SET SCHEMA public;
ALTER TABLE pipeline.snap_changes SET SCHEMA public;
ALTER TABLE pipeline.snap_csv SET SCHEMA public;
ALTER TABLE pipeline.csv_download_log SET SCHEMA public;
ALTER TABLE pipeline.bulk_csv_log SET SCHEMA public;
ALTER TABLE pipeline.samgov_requests SET SCHEMA public;
ALTER TABLE pipeline.csv_cache_headers SET SCHEMA public;
ALTER TABLE pipeline.ingestion_runs SET SCHEMA public;

DROP SCHEMA IF EXISTS pipeline;
