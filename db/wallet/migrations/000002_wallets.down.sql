BEGIN;

DELETE FROM accounts
WHERE kind = 'system'
  AND name = 'settlement'
  AND currency IN ('USD', 'ETB');

DROP INDEX entries_account_id_id_idx;

CREATE INDEX entries_account_id_idx ON entries (account_id);

DROP INDEX accounts_wallet_owner_currency_uidx;
DROP INDEX accounts_system_name_currency_uidx;

ALTER TABLE accounts
    DROP CONSTRAINT accounts_kind_owner_check;

ALTER TABLE accounts
    DROP COLUMN kind;

DROP TYPE account_kind;

COMMIT;
