# Wallet

Bounded context that holds Accounts and an append-only double-entry ledger. The service binary is named wallet; that name is not a domain entity. The HTTP API may expose owned Accounts under `/v1/wallets`; that path is a resource name, not a second domain entity.

## Language

**Account**:
A named balance container in a single Currency with a Normal Balance (debit or credit) and an Account Kind. An owned Account (kind wallet) belongs to an Owner; a System Account does not. Owned Account names are assigned by the service (e.g. `"USD wallet"`), not chosen by the client.
_Avoid_: Wallet (as a domain noun), balance record

**Account Kind**:
Whether an Account is `wallet` (belongs to an Owner) or `system` (platform-side). Kind and Owner are inseparable: wallet Accounts always have an Owner; system Accounts never do.

**System Account**:
An Account with Account Kind `system` and no Owner, used for platform-side ledger positions rather than an end-user's funds.

**Settlement Account**:
The System Account that holds funds at our bank for one Currency. Deposits are funded from it. One per supported Currency; new Currencies are added by migration.
_Avoid_: clearing (as the product name for this account), omnibus (unless that meaning is intended)

**Owner**:
An opaque identity from the auth service, referenced only by UUID. Wallet does not store or validate user records locally.
_Avoid_: User, customer (inside this context)

**Ledger Transaction**:
One atomic movement of value, recorded as one or more Entries. Mistakes are corrected by posting a reversing Ledger Transaction, never by editing past rows.
_Avoid_: Transfer (that is a different service), payment, Wallet transaction

**Entry**:
One line of a Ledger Transaction: a positive Amount in a Currency, either debit or credit, against exactly one Account.
_Avoid_: Line item, posting (as a noun for the row)

**Normal Balance**:
Whether an Account increases on debit or on credit. Owned Accounts are credit-normal (liability to the Owner). Settlement Accounts are debit-normal (asset at the bank).

**Currency**:
An ISO 4217 alphabetic code (three uppercase letters). Fixed for the life of an Account. Supported Currencies for deposits today: USD and ETB (Ethiopian birr).
_Avoid_: Money type name alone when you mean the code

**Amount**:
A positive integer in minor units of a Currency (for example cents), never a floating-point value.
_Avoid_: Decimal money, float

**Account Balance**:
The Amount implied by an Account's Entries relative to its Normal Balance. Derived, never stored as a writable field on the Account.
_Avoid_: Cached balance, wallet balance
