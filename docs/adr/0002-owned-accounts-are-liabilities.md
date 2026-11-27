# Owned Accounts are liabilities; Settlement is cash

Deposits move value onto the ledger by debiting the Settlement Account (debit-normal asset: funds at our bank) and crediting the Owner's Account (credit-normal liability: what we owe them). Treating owned Accounts as assets instead would invert deposit semantics and disagree with the product narrative that a deposit increases what we hold and what we owe. Corrections remain reversing Ledger Transactions (ADR-0001), never edits to past Entries.
