BEGIN;

CREATE TYPE account_kind AS ENUM ('wallet', 'system');

ALTER TABLE accounts
    ADD COLUMN kind account_kind;

UPDATE accounts
SET kind = CASE
    WHEN owner_id IS NULL THEN 'system'::account_kind
    ELSE 'wallet'::account_kind
END;

ALTER TABLE accounts
    ALTER COLUMN kind SET NOT NULL;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_kind_owner_check CHECK (
        (kind = 'wallet' AND owner_id IS NOT NULL)
        OR (kind = 'system' AND owner_id IS NULL)
    );

CREATE UNIQUE INDEX accounts_wallet_owner_currency_uidx
    ON accounts (owner_id, currency)
    WHERE kind = 'wallet';

CREATE UNIQUE INDEX accounts_system_name_currency_uidx
    ON accounts (name, currency)
    WHERE kind = 'system';

DROP INDEX entries_account_id_idx;

CREATE INDEX entries_account_id_id_idx ON entries (account_id, id);

INSERT INTO accounts (owner_id, name, currency, normal_balance, kind)
VALUES
    (NULL, 'settlement', 'USD', 'debit', 'system'),
    (NULL, 'settlement', 'ETB', 'debit', 'system');

COMMIT;
