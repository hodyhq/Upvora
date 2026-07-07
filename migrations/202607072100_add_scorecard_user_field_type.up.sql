-- Add "user" to the allowed field-type set. Renders as a text input with a
-- datalist autocompleting to the tenant's collaborators + administrators,
-- but any custom name is accepted (someone who doesn't have a Fider account).
ALTER TABLE scorecard_fields DROP CONSTRAINT IF EXISTS scorecard_fields_type_check;
ALTER TABLE scorecard_fields ADD CONSTRAINT scorecard_fields_type_check
    CHECK (type IN ('text', 'note', 'date', 'number', 'url', 'choice', 'score', 'user'));
