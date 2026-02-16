-- Widen columns that overflow with real SAM.gov data.
-- Per design comment: "use TEXT wherever SAM.gov data has surprised us with length"
ALTER TABLE opportunities ALTER COLUMN secondary_contact_phone TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN award_number TYPE TEXT;
