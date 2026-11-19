# PostgreSQL-enforced ledger invariants

Wallet keeps double-entry correctness and ledger immutability in PostgreSQL (deferred balance/non-empty checks on Ledger Transactions and Entries, plus append-only triggers) rather than relying on application checks alone, so buggy or concurrent writers cannot commit an invalid or rewritten ledger. Corrections are new reversing Ledger Transactions, never updates or deletes of past rows.
