-- Add "multiline" to the allowed field-type set: a hybrid of text and note.
-- Renders at text width (half the form row, so two fit side by side) but as a
-- vertically expandable textarea like note.
ALTER TABLE scorecard_fields DROP CONSTRAINT IF EXISTS scorecard_fields_type_check;
ALTER TABLE scorecard_fields ADD CONSTRAINT scorecard_fields_type_check
    CHECK (type IN ('text', 'note', 'date', 'number', 'url', 'choice', 'score', 'user', 'multiline'));
