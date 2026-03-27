-- name: GetProfile :one
SELECT * FROM genshin_profiles WHERE id = $1;

-- name: GetProfiles :many
SELECT * FROM genshin_profiles;

-- name: CreateGenshinProfile :one
INSERT INTO genshin_profiles (name, notes)
VALUES ($1, $2) RETURNING *;

-- name: EditGenshinProfile :one
UPDATE genshin_profiles
SET 
    name = $2,
    notes = $3
WHERE id = $1
RETURNING *;

-- name: DeleteGenshinProfile :exec
DELETE FROM genshin_profiles WHERE id = $1;

-- name: GetAllGenshinChars :many
SELECT char_details.*, elements.name AS element_name
FROM char_details
JOIN elements ON char_details.element_id = elements.id;

-- name: GetGenshinChar :one
SELECT char_details.*, elements.name as element_name
FROM char_details
JOIN elements ON elements.id = char_details.element_id
WHERE char_details.id = $1;

-- name: CreateGenshinChar :one
WITH created AS (
    INSERT INTO char_details (name, element_id, stars, icon, notes)
    VALUES(
        sqlc.arg(name),
        (SELECT id FROM elements WHERE elements.name = sqlc.arg(element_name)),
        sqlc.arg(stars), sqlc.arg(icon), sqlc.arg(notes)
    )
    RETURNING *
)
SELECT created.*, elements.name AS element_name
FROM created
JOIN elements ON created.element_id = elements.id;

-- name: EditGenshinChar :one
WITH edited AS (
    UPDATE char_details
    SET
        name = sqlc.arg(name),
        element_id = (SELECT id FROM elements WHERE elements.name = sqlc.arg(element_name)),
        stars = sqlc.arg(stars),
        icon = sqlc.arg(icon),
        notes = sqlc.arg(notes)
    WHERE char_details.id = sqlc.arg(id)
    RETURNING *
)
SELECT edited.*, elements.name AS element_name
from edited
JOIN elements ON edited.element_id = elements.id;

-- name: DeleteGenshinChar :exec
DELETE FROM char_details WHERE id = $1;


-- name: GetAllCharsFromProfile :many
SELECT
    sqlc.embed(profile_chars), 
    sqlc.embed(char_details), 
    sqlc.embed(genshin_profiles),
    sqlc.embed(elements)
FROM profile_chars
JOIN char_details ON profile_chars.char_id = char_details.id
JOIN genshin_profiles ON profile_chars.prof_id = genshin_profiles.id
JOIN elements ON char_details.element_id = elements.id
WHERE profile_chars.prof_id = $1;

-- name: AddCharToProfile :one
WITH inserted AS (
    INSERT INTO profile_chars (
        prof_id, char_id, level, constellation,
        talent_na, talent_e, talent_q, notes
    ) VALUES (
        sqlc.arg(prof_id),
        (SELECT id FROM char_details cd WHERE cd.name = sqlc.arg(char_name)),
        sqlc.arg(level), sqlc.arg(constellation), sqlc.arg(talent_na), sqlc.arg(talent_e), sqlc.arg(talent_q), sqlc.arg(notes) 
    )
    RETURNING *
)
SELECT
    sqlc.embed(profile_chars),
    sqlc.embed(char_details),
    sqlc.embed(elements)
FROM inserted
JOIN profile_chars ON profile_chars.prof_id = inserted.prof_id AND profile_chars.char_id = inserted.char_id
JOIN char_details ON profile_chars.char_id = char_details.id
JOIN elements ON char_details.element_id = elements.id;

-- name: EditCharFromProfile :one
WITH updated AS (
    UPDATE profile_chars
    SET
        level = $3, constellation = $4, talent_na = $5,
        talent_e = $6, talent_q = $7, notes = $8
    WHERE profile_chars.prof_id = $1 AND profile_chars.char_id = $2
    RETURNING *
)
SELECT
    sqlc.embed(profile_chars),
    sqlc.embed(char_details),
    sqlc.embed(elements)
FROM updated
JOIN profile_chars ON profile_chars.prof_id = updated.prof_id AND profile_chars.char_id = updated.char_id
JOIN char_details ON profile_chars.char_id = char_details.id
JOIN elements ON char_details.element_id = elements.id;

-- name: DeleteCharFromProfile :exec
DELETE FROM profile_chars WHERE prof_id = $1 AND char_id = $2;


-- name: GetElementId :one
SELECT id from elements WHERE name = $1;

-- name: GetAllElements :many
SELECT * FROM elements;

-- name: GetElementIconByName :one
SELECT icon_url from elements where name = $1;

-- name: GetProfileCharStats :one
SELECT
    COUNT(pc.char_id)::INT AS total_characters
FROM genshin_profiles gp
LEFT JOIN profile_chars pc on gp.id = pc.prof_id
LEFT JOIN char_details cd on cd.id = pc.char_id
WHERE gp.id = $1
GROUP BY gp.id;

-- name: GetProfileElementCounts :many
SELECT
    e.name AS element_name,
    COUNT(pc.char_id)::INT AS char_count
from profile_chars pc
JOIN char_details cd ON pc.char_id = cd.id
JOIN elements e ON cd.element_id = e.id
WHERE pc.prof_id = $1
GROUP BY e.id, e.name
ORDER BY char_count DESC;
