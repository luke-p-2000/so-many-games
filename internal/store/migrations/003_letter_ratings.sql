-- Ratings become tier letters to match the tier list (ADR 0020): S, A, B, C, D or F.
-- Old 0-10 scores map 0 = F, 1-2 = D, 3-4 = C, 5-6 = B, 7-8 = A, 9-10 = S.
ALTER TABLE game ADD COLUMN rating_letter TEXT;  -- S, A, B, C, D or F, only when status = 'finished'

UPDATE game SET rating_letter = CASE
    WHEN rating IS NULL THEN NULL
    WHEN rating >= 9 THEN 'S'
    WHEN rating >= 7 THEN 'A'
    WHEN rating >= 5 THEN 'B'
    WHEN rating >= 3 THEN 'C'
    WHEN rating >= 1 THEN 'D'
    ELSE 'F' END;

ALTER TABLE game DROP COLUMN rating;
ALTER TABLE game RENAME COLUMN rating_letter TO rating;

-- Keep the history consistent: past 'rated' events carry the letter too.
UPDATE event SET payload = json_set(payload, '$.rating', CASE
    WHEN json_extract(payload, '$.rating') >= 9 THEN 'S'
    WHEN json_extract(payload, '$.rating') >= 7 THEN 'A'
    WHEN json_extract(payload, '$.rating') >= 5 THEN 'B'
    WHEN json_extract(payload, '$.rating') >= 3 THEN 'C'
    WHEN json_extract(payload, '$.rating') >= 1 THEN 'D'
    ELSE 'F' END)
WHERE kind = 'rated' AND json_type(payload, '$.rating') = 'integer';
