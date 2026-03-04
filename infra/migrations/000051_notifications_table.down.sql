DELETE FROM alert_jobs WHERE job_name IN ('search_notifications', 'opportunity_notifications');
DROP TABLE IF EXISTS notifications;
