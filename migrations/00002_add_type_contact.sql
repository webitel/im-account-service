-- +goose Up
-- +goose StatementBegin

-- CREATE TABLE IF NOT EXISTS im_account.session_test
--   AS TABLE im_account.session
-- ;

-- IM Contact reference info
CREATE TYPE im_account.refcontact AS
(
  --[ attribute_name data_type [ COLLATE collation ] [, ... ] ] )
  dc int8 -- service (internal) business (domain) identifier
, id uuid -- service (internal) UNIQUE subject identifier ; OPTIONAL
, iss text -- service (external) issuer (provider) identifier ; REQUIRED
, sub text -- service (external) subject identifier at issuer ; REQUIRED

);

-- ALTER TABLE im_account.session
--   ALTER COLUMN contact_id
--     SET DATA TYPE im_account.refcontact
--     USING replace(contact_id, '""', '')::im_account.refcontact
-- ;

-- -- DROP legacy (invalid) session.contact.(reference)
-- DELETE
-- FROM im_account.session
-- WHERE contact_id LIKE '(%,"",%)'
-- ;

ALTER TABLE im_account.session
  DROP CONSTRAINT session_device_id
, DROP CONSTRAINT session_contact_id
, ALTER COLUMN contact_id
    SET DATA TYPE im_account.refcontact
    USING contact_id::im_account.refcontact
-- , ADD CONSTRAINT session_device_id
--     UNIQUE (device_id, ((contact_id).id))
-- , ADD CONSTRAINT session_contact_id
--     UNIQUE (((contact_id).id), device_id)
;

CREATE UNIQUE INDEX session_device_id
  ON im_account.session
  USING btree (device_id, ((contact_id).id))
;

CREATE UNIQUE INDEX session_contact_id
  ON im_account.session
  USING btree (((contact_id).id), device_id)
;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS im_account.session_device_id, im_account.session_contact_id RESTRICT ;

ALTER TABLE im_account.session
  ALTER COLUMN contact_id
    SET DATA TYPE text
    USING contact_id::text
;

ALTER TABLE im_account.session
  ADD CONSTRAINT session_device_id
    UNIQUE (device_id, contact_id)
, ADD CONSTRAINT session_contact_id
    UNIQUE (contact_id, device_id)
;

DROP TYPE im_account.refcontact ;

-- +goose StatementEnd