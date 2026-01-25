-- +goose Up
-- +goose StatementBegin
--------------------------------------------------------------------------------

CREATE SCHEMA im_account ;

--------------------------------------------------------------------------------

-- im_account.app DEFINITION

-- DROP TABLE im_account.app ;

CREATE TABLE im_account.app
(
  dc int8 NOT NULL -- Business Account ID
, id uuid DEFAULT gen_random_uuid() NOT NULL -- Application ID [client_id]
, "name" text NOT NULL -- Application friendly name
, about text NULL -- Short description
, config jsonb -- bytea -- configuration source

, created_at timestamptz DEFAULT timezone('utc', NOW()) NOT NULL
, updated_at timestamptz NULL
, revoked_at timestamptz NULL

, CONSTRAINT app_id PRIMARY KEY (id)
, CONSTRAINT business_app_id UNIQUE (dc, id)

);

--------------------------------------------------------------------------------

-- im_account.session DEFINITION

-- DROP TABLE im_account.session ;

CREATE TABLE im_account.session
(
  dc int8 NOT NULL -- Business ID
, id uuid DEFAULT gen_random_uuid() NOT NULL -- Session ID

, ip inet NOT NULL -- Device (Client) [FROM] IP Address. [ Forwarded-From | X-Real-IP ]
, "name" text NOT NULL -- Display (custom) name. Default: device.user_agent info

-- Authorization

, app_id uuid NULL -- App (Client) ID ; [VIA]
, device_id text NOT NULL -- Device (Client) ID ; [FROM]
, user_agent text NULL -- Device (Client) App info
, contact_id text NOT NULL -- Signed-In (Account) ID
, push_token jsonb NULL -- Device PUSH token registration

, metadata jsonb NULL -- Session extra metadata

, created_at timestamptz DEFAULT timezone('utc', now()) NOT NULL -- Creation date

, CONSTRAINT session_id PRIMARY KEY (id)
, CONSTRAINT session_device_id UNIQUE (device_id, contact_id)
, CONSTRAINT session_contact_id UNIQUE (contact_id, device_id)

-- , CONSTRAINT session_dc_fk FOREIGN KEY (dc) REFERENCES directory.wbt_domain(dc) ON DELETE CASCADE
-- , CONSTRAINT session_user_fk FOREIGN KEY (dc, user_id) REFERENCES im_account.user(dc, id) ON DELETE CASCADE
, CONSTRAINT session_app_fk FOREIGN KEY (dc, app_id) REFERENCES im_account.app(dc, id)
-- , CONSTRAINT session_device_fk FOREIGN KEY (device_id) REFERENCES im_account.device(id)
);

COMMENT ON TABLE im_account.session IS 'Session. Authorization';

COMMENT ON COLUMN im_account.session.dc IS 'Business Account ID';
COMMENT ON COLUMN im_account.session.id IS 'Session Internal ID';
COMMENT ON COLUMN im_account.session.ip IS 'Device (Client) [FROM] IP Address. [ Forwarded-From | X-Real-IP ]';
COMMENT ON COLUMN im_account.session.name IS 'Custom name. Default: device.user_agent info';
COMMENT ON COLUMN im_account.session.app_id IS 'App (Client) ID ; [VIA]';
COMMENT ON COLUMN im_account.session.device_id IS 'Device (Client) ID ; [FROM]';
COMMENT ON COLUMN im_account.session.user_agent IS 'Device (Client) App ; [FROM]';
COMMENT ON COLUMN im_account.session.contact_id IS 'Signed-In (Account) ID';
COMMENT ON COLUMN im_account.session.metadata IS 'Session extra metadata';
COMMENT ON COLUMN im_account.session.created_at IS 'Created date';


-- im_account.session_token DEFINITION

-- DROP TABLE im_account.session_token ;

CREATE TABLE im_account.session_token
(
  id uuid NOT NULL -- Session ID

, "type" name NULL -- token_type ; DEFAULT 'bearer'
, "token" text COLLATE "C" NOT NULL -- Opaque access_token
, "refresh" text COLLATE "C" NULL -- Opaque refresh_token

-- , scope text[] COLLATE "C" NULL -- 
, scope name[] NULL -- 

, rotated_at timestamptz DEFAULT timezone('utc', NOW()) NOT NULL -- Token [RE]generation date
, expires_at timestamptz NULL -- Token expiration date

, revoked_at timestamptz NULL -- Session revoked date
, revoked_by int8 NULL -- Session revoked by admin

, CONSTRAINT session_token_id PRIMARY KEY (id) -- session [1:1] token --
, CONSTRAINT session_access_token UNIQUE (token) INCLUDE (id)
, CONSTRAINT session_refresh_token UNIQUE (refresh) INCLUDE (id)

, CONSTRAINT session_token_fk FOREIGN KEY (id) REFERENCES im_account.session(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED
);

-- CREATE INDEX session_token_id ON im_account.session_token (id) ;

COMMENT ON TABLE im_account.session_token IS 'Authorization Token';

COMMENT ON COLUMN im_account.session_token.id IS 'Session ID';
COMMENT ON COLUMN im_account.session_token.type IS 'token_type ; default: ''bearer''';
COMMENT ON COLUMN im_account.session_token.token IS 'Opaque [access_token] ; REQUIRED';
COMMENT ON COLUMN im_account.session_token.refresh IS 'Opaque [refresh_token] ; OPTIONAL';
COMMENT ON COLUMN im_account.session_token.rotated_at IS 'Token [RE]generation date ; [not_before] ; REQUIRED';
COMMENT ON COLUMN im_account.session_token.expires_at IS 'Token expiration date ; [expiry] ; OPTIONAL';
COMMENT ON COLUMN im_account.session_token.revoked_at IS 'Session revoked date ; OPTIONAL';
COMMENT ON COLUMN im_account.session_token.revoked_by IS 'Session revoked by admin';


--------------------------------------------------------------------------------

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE im_account.session_token ;
DROP TABLE im_account.session ;
DROP TABLE im_account.app ;

DROP SCHEMA im_account CASCADE ;

-- +goose StatementEnd
