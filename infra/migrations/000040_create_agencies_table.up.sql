CREATE TABLE agencies (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  short_name TEXT,
  level TEXT NOT NULL,
  parent_path TEXT NOT NULL,
  aliases TEXT[] NOT NULL DEFAULT '{}',
  opportunity_count INT NOT NULL DEFAULT 0,
  UNIQUE(parent_path)
);

CREATE INDEX idx_agencies_search ON agencies
  USING gin(to_tsvector('simple', name || ' ' || coalesce(short_name, '') || ' ' || array_to_string(aliases, ' ')));
CREATE INDEX idx_agencies_path ON agencies(parent_path text_pattern_ops);
CREATE INDEX idx_agencies_count ON agencies(opportunity_count DESC);

-- Seed agencies from existing opportunity data.
-- For each full_parent_path_name, generate all prefixes and count matching opportunities.
WITH path_counts AS (
    SELECT full_parent_path_name AS path, COUNT(*) AS cnt
    FROM opportunities
    WHERE active = true AND is_latest = true
      AND full_parent_path_name IS NOT NULL AND full_parent_path_name <> ''
    GROUP BY full_parent_path_name
),
expanded AS (
    SELECT
        array_to_string((string_to_array(path, '.'))[1:n], '.') AS prefix,
        cnt
    FROM path_counts
    CROSS JOIN LATERAL generate_series(1, array_length(string_to_array(path, '.'), 1)) AS n
),
counted AS (
    SELECT prefix, SUM(cnt)::int AS total_count
    FROM expanded
    GROUP BY prefix
)
INSERT INTO agencies (name, level, parent_path, opportunity_count)
SELECT
    split_part(prefix, '.', array_length(string_to_array(prefix, '.'), 1)) AS name,
    CASE array_length(string_to_array(prefix, '.'), 1)
        WHEN 1 THEN 'department'
        WHEN 2 THEN 'sub_tier'
        ELSE 'command'
    END AS level,
    prefix AS parent_path,
    total_count
FROM counted
ORDER BY prefix;

-- Hand-curated aliases for top agencies
UPDATE agencies SET short_name = 'DoD', aliases = '{"DoD","Department of Defense","Pentagon"}' WHERE parent_path = 'DEPT OF DEFENSE' AND level = 'department';
UPDATE agencies SET short_name = 'Army', aliases = '{"Army","US Army"}' WHERE parent_path = 'DEPT OF THE ARMY' AND level = 'department';
UPDATE agencies SET short_name = 'Navy', aliases = '{"Navy","US Navy"}' WHERE parent_path = 'DEPT OF THE NAVY' AND level = 'department';
UPDATE agencies SET short_name = 'USAF', aliases = '{"Air Force","USAF","US Air Force"}' WHERE parent_path = 'DEPT OF THE AIR FORCE' AND level = 'department';

UPDATE agencies SET short_name = 'DLA', aliases = '{"DLA"}' WHERE name = 'DEFENSE LOGISTICS AGENCY' AND level = 'sub_tier';
UPDATE agencies SET short_name = 'USACE', aliases = '{"USACE","Army Corps","Corps of Engineers"}' WHERE name = 'US ARMY CORPS OF ENGINEERS';
UPDATE agencies SET short_name = 'DISA', aliases = '{"DISA"}' WHERE name LIKE 'DEFENSE INFORMATION SYSTEMS AGENCY%';
UPDATE agencies SET short_name = 'DARPA', aliases = '{"DARPA"}' WHERE name LIKE 'DEFENSE ADVANCED RESEARCH PROJECTS AGENCY%';
UPDATE agencies SET short_name = 'MDA', aliases = '{"MDA"}' WHERE name LIKE 'MISSILE DEFENSE AGENCY%';
UPDATE agencies SET short_name = 'DCMA', aliases = '{"DCMA"}' WHERE name LIKE 'DEFENSE CONTRACT MANAGEMENT AGENCY%';
UPDATE agencies SET short_name = 'DCSA', aliases = '{"DCSA"}' WHERE name LIKE 'DEFENSE COUNTERINTELLIGENCE AND SECURITY AGENCY%';
UPDATE agencies SET short_name = 'DHA', aliases = '{"DHA"}' WHERE name LIKE 'DEFENSE HEALTH AGENCY%';

UPDATE agencies SET short_name = 'NAVFAC', aliases = '{"NAVFAC","Naval Facilities"}' WHERE name LIKE 'NAVFACSYSCOM%';
UPDATE agencies SET short_name = 'NAVSUP', aliases = '{"NAVSUP","Naval Supply"}' WHERE name = 'NAVSUP';
UPDATE agencies SET short_name = 'NAVSEA', aliases = '{"NAVSEA","Naval Sea Systems"}' WHERE name LIKE 'NAVSEASYSCOM%' OR (name = 'NAVSEA' AND level = 'command');
UPDATE agencies SET short_name = 'NAVAIR', aliases = '{"NAVAIR","Naval Air Systems"}' WHERE name LIKE 'NAVAIRSYSCOM%' OR (name = 'NAVAIR' AND level = 'command');
UPDATE agencies SET short_name = 'SPAWAR', aliases = '{"SPAWAR","NAVWAR"}' WHERE name LIKE 'SPAWARSYSCOM%' OR name LIKE 'NAVWARSYSCOM%';

UPDATE agencies SET short_name = 'AFMC', aliases = '{"AFMC"}' WHERE name = 'AIR FORCE MATERIEL COMMAND';
UPDATE agencies SET short_name = 'MICC', aliases = '{"MICC"}' WHERE name LIKE 'MISSION INSTALLATION CONTRACTING COMMAND%' OR name = 'MICC';
UPDATE agencies SET short_name = 'AMC', aliases = '{"AMC"}' WHERE name = 'AMC' AND level = 'command';

UPDATE agencies SET short_name = 'VA', aliases = '{"VA"}' WHERE name LIKE 'VETERANS AFFAIRS%' AND level = 'department';
UPDATE agencies SET short_name = 'DHS', aliases = '{"DHS"}' WHERE name LIKE 'HOMELAND SECURITY%' AND level = 'department';
UPDATE agencies SET short_name = 'GSA', aliases = '{"GSA"}' WHERE name = 'GENERAL SERVICES ADMINISTRATION' AND level = 'department';
UPDATE agencies SET short_name = 'NASA', aliases = '{"NASA"}' WHERE name = 'NATIONAL AERONAUTICS AND SPACE ADMINISTRATION' AND level = 'department';
UPDATE agencies SET short_name = 'HHS', aliases = '{"HHS"}' WHERE name LIKE 'HEALTH AND HUMAN SERVICES%' AND level = 'department';
UPDATE agencies SET short_name = 'DOE', aliases = '{"DOE"}' WHERE name LIKE 'ENERGY, DEPARTMENT OF%' AND level = 'department';
UPDATE agencies SET short_name = 'DOT', aliases = '{"DOT"}' WHERE name LIKE 'TRANSPORTATION, DEPARTMENT OF%' AND level = 'department';
UPDATE agencies SET short_name = 'DOJ', aliases = '{"DOJ"}' WHERE name LIKE 'JUSTICE, DEPARTMENT OF%' AND level = 'department';
UPDATE agencies SET short_name = 'DOI', aliases = '{"DOI"}' WHERE name LIKE 'INTERIOR, DEPARTMENT OF THE%' AND level = 'department';
UPDATE agencies SET short_name = 'USDA', aliases = '{"USDA"}' WHERE name LIKE 'AGRICULTURE, DEPARTMENT OF%' AND level = 'department';
UPDATE agencies SET short_name = 'DOC', aliases = '{"DOC"}' WHERE name LIKE 'COMMERCE, DEPARTMENT OF%' AND level = 'department';
UPDATE agencies SET short_name = 'DOL', aliases = '{"DOL"}' WHERE name LIKE 'LABOR, DEPARTMENT OF%' AND level = 'department';
UPDATE agencies SET short_name = 'EPA', aliases = '{"EPA"}' WHERE name = 'ENVIRONMENTAL PROTECTION AGENCY' AND level = 'department';
UPDATE agencies SET short_name = 'SBA', aliases = '{"SBA"}' WHERE name = 'SMALL BUSINESS ADMINISTRATION' AND level = 'department';
UPDATE agencies SET short_name = 'SSA', aliases = '{"SSA"}' WHERE name LIKE 'SOCIAL SECURITY ADMINISTRATION%' AND level = 'department';
UPDATE agencies SET short_name = 'FEMA', aliases = '{"FEMA"}' WHERE name LIKE 'FEDERAL EMERGENCY MANAGEMENT AGENCY%';
UPDATE agencies SET short_name = 'CBP', aliases = '{"CBP"}' WHERE name LIKE 'U.S. CUSTOMS AND BORDER PROTECTION%';
UPDATE agencies SET short_name = 'ICE', aliases = '{"ICE"}' WHERE name LIKE 'U.S. IMMIGRATION AND CUSTOMS ENFORCEMENT%';
UPDATE agencies SET short_name = 'USCG', aliases = '{"USCG","Coast Guard"}' WHERE name LIKE 'UNITED STATES COAST GUARD%';
UPDATE agencies SET short_name = 'TSA', aliases = '{"TSA"}' WHERE name LIKE 'TRANSPORTATION SECURITY ADMINISTRATION%';
UPDATE agencies SET short_name = 'FAA', aliases = '{"FAA"}' WHERE name LIKE 'FEDERAL AVIATION ADMINISTRATION%';
UPDATE agencies SET short_name = 'OPM', aliases = '{"OPM"}' WHERE name = 'OFFICE OF PERSONNEL MANAGEMENT' AND level = 'department';
