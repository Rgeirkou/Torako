-- Keys created before scopes became mandatory may hold an empty scope list,
-- which grants every permission (empty scopes allow all). Downgrade any such
-- keys to tw so admin access is not silently granted.
UPDATE api_keys
SET scopes = ARRAY['tw']
WHERE NOT (scopes && ARRAY['tw', 'admin']);
