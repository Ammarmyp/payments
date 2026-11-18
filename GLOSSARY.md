# Wallet

Bounded context that holds Accounts and an append-only double-entry ledger. The service binary is named wallet; that name is not a domain entity.

## Language

**Account**:
A named balance container in a single Currency with a Normal Balance (debit or credit). May belong to an Owner or be a System Account.
_Avoid_: Wallet, balance record

**System Account**:
An Account with no Owner (`owner_id` null), used for platform-side ledger positions rather than an end-user's funds.

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
Whether an Account increases on debit or on credit. Asset-like Accounts are typically debit-normal; liability/equity-like Accounts are typically credit-normal.

**Currency**:
An ISO 4217 alphabetic code (three uppercase letters). Fixed for the life of an Account.
_Avoid_: Money type name alone when you mean the code

**Amount**:
A positive integer in minor units of a Currency (for example cents), never a floating-point value.
_Avoid_: Decimal money, float

**Account Balance**:
The Amount implied by an Account's Entries relative to its Normal Balance. Derived, never stored as a writable field on the Account.
_Avoid_: Cached balance, wallet balance
