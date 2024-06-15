---------------------
------ Session ------
---------------------

-- name: AllSessions :many
SELECT *
FROM auth.session
ORDER BY created_at ASC;

-- name: FindSessionsByUserID :many
SELECT *
FROM auth.session
WHERE user_id = $1
ORDER BY created_at;

-- name: FindSessionDataByKey :one
SELECT data
FROM auth.session
WHERE key = $1;

-- name: DeleteSessionByKey :exec
DELETE
FROM auth.session
WHERE key = $1;

-- name: UpsertSessionData :exec
INSERT INTO auth.session (key, data, expires_at_utc)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET (data, expires_at_utc) = ($2, $3);

-- name: UpsertNewSession :exec
INSERT INTO auth.session (key, user_id, user_agent)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET (user_id, user_agent) = ($2, $3);



------------------
------ User ------
------------------

-- name: AllUsers :many
SELECT *
FROM auth.user
WHERE TRUE
     AND (CASE WHEN @login::TEXT <> '' THEN @login < login ELSE TRUE END)
ORDER BY login
LIMIT $1;

-- name: AllUsersByIDs :many
SELECT *
FROM auth.user
WHERE id = ANY ($1::uuid[]);

-- name: FindUserByID :one
SELECT *
FROM auth.user
WHERE id = $1;

-- name: FindUserByLogin :one
SELECT *
FROM auth.user
WHERE login = $1;

-- name: UserExistsByID :one
SELECT EXISTS(SELECT 1 FROM auth.user WHERE id = $1) AS "exists";

-- name: UserExistsByLogin :one
SELECT EXISTS(SELECT 1 FROM auth.user WHERE login = $1) AS "exists";

-- name: CountUsers :one
SELECT COUNT(*)
FROM auth.user;

-- name: CreateUser :one
INSERT
INTO auth.user (id, login, password_hash, verified_at_utc, blocked_at_utc)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpsertUser :one
INSERT INTO auth.user(id, created_at, login, password_hash, name_firstname, name_lastname, name_displayname, birthday,
                      locale, time_zone,
                      picture_url, profile, verified_at_utc, blocked_at_utc, superuser_at_utc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (id) DO UPDATE SET (login, password_hash, name_firstname, name_lastname, name_displayname, birthday, locale,
                                time_zone,
                                picture_url, profile, verified_at_utc, blocked_at_utc,
                                superuser_at_utc) = ($3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: DeleteUser :exec
DELETE
FROM auth.user
WHERE id = ANY ($1::uuid[]);

-- name: DeleteAllUsers :exec
DELETE
FROM auth.user;

-- name: CreateVerificationToken :exec
INSERT INTO auth.user_verification(token, user_id, valid_until_utc)
VALUES ($1, $2, $3);

-- name: VerificationTokenByToken :one
SELECT *
FROM auth.user_verification
WHERE token = $1;