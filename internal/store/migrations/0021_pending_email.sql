-- An unconfirmed registration keeps its address here, outside the unique index on email, so nobody can block an
-- address by registering it without access to the mailbox.
ALTER TABLE users ADD COLUMN pending_email text;
UPDATE users SET pending_email = email, email = NULL WHERE verified_at IS NULL;
