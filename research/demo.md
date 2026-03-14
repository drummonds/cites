# APR vs AER — A Comparison of UK Interest Rate Standards

## Overview

UK financial regulation uses two distinct standardised interest rate measures:

- **APR (Annual Percentage Rate)** — the cost of borrowing, defined by the Consumer Credit Act 1974
- **AER (Annual Equivalent Rate)** — the return on savings, defined by UK Finance industry practice

Both aim to give consumers a single comparable number, but they apply to opposite sides of the same transaction.

## APR — Annual Percentage Rate

The Consumer Credit Act 1974 establishes the legal framework for APR disclosure.
The total charge for credit must be expressed as an annual rate to allow
comparison between different credit products.

Key provisions:

- [CCA s.20 — Total charge for credit](sources/cca-1974.yaml#L142) defines what must be included in the APR calculation
- [CCA Part V — Entry into credit agreements](sources/cca-1974.yaml#L200) requires APR disclosure before agreement signing

The APR formula accounts for the timing and amount of all payments, including fees, insurance, and ancillary charges — not just the headline interest rate.

## AER — Annual Equivalent Rate

The AER is an industry standard maintained by UK Finance (formerly the British Bankers' Association).
The [AER Practice Note](../r2/) sets out the calculation methodology.

Key points from the practice note:

- [Scope — which products must show AER](../r2/p01.html#l29) — all interest bearing accounts
- [Advertising requirements](../r2/p01.html#l37) — how interest rates must be described
- AER shows what the interest rate would be if interest were compounded annually
- [Calculation examples](../r2/p09.html#l444) — worked examples for different compounding frequencies

## Comparison

| Aspect | APR | AER |
|--------|-----|-----|
| Applies to | Credit (borrowing) | Savings (deposits) |
| Defined by | Consumer Credit Act 1974 (statute) | UK Finance Practice Note (industry) |
| Includes fees | Yes — total charge for credit | No — interest only |
| Compounding | Accounts for payment timing | Annualises the compounding frequency |
| Purpose | Cost of borrowing | Return on savings |

## Why Both Matter

A consumer taking out a loan and opening a savings account needs both numbers to make informed decisions. The APR tells them what they'll pay; the AER tells them what they'll earn. Without standardisation, comparing products across providers would be impractical.

## Sources

- [Consumer Credit Act 1974](sources/cca-1974.yaml) — full text capture from legislation.gov.uk
- [AER Practice Note, January 2024](../r2/) — UK Finance ([source YAML](sources/aer-practice-note-2025.yaml))
