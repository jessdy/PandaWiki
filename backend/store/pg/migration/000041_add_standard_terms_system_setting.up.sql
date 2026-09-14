INSERT INTO system_settings (key, value, description)
SELECT
    'standard_terms',
    '{"items":[]}'::jsonb,
    'Global standard terms and aliases'
WHERE NOT EXISTS (
    SELECT 1 FROM system_settings WHERE key = 'standard_terms'
);
