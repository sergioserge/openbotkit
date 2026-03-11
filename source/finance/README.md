# Finance Source

Real-time stock prices and exchange rates via Yahoo Finance.

## Why Yahoo Finance

There is no official, free, public API for stock prices. Stock price data is proprietary (owned by exchanges, licensed to redistributors). Yahoo Finance is the de facto standard used by virtually every open-source finance tool — Python's `yfinance`, Go's `ticker`, `finance-go`, etc.

For exchange rates, central banks publish free data (ECB), but Yahoo handles both stocks and forex in a single endpoint, which is simpler.

## Why TLS fingerprinting (utls)

Yahoo actively blocks non-browser HTTP clients by inspecting TLS ClientHello fingerprints. We verified this empirically: curl (LibreSSL) gets HTTP 429, while Go's standard `net/http` works today but is fragile (the `ticker` project's CI broke on March 10, 2026 for this reason).

We use `refraction-networking/utls` to present Chrome's TLS fingerprint. This is the same core library used by `go-yfinance` (via CycleTLS) and is maintained by the censorship-circumvention research community.

## Why utls and not CycleTLS

CycleTLS (`Danny-Dasilva/CycleTLS`) wraps utls but pulls in 27 dependencies including QUIC, gopacket, ginkgo, etc. We only need TLS fingerprint control on the handshake — everything else uses Go stdlib. utls adds only 2 new deps (`utls` + `andybalholm/brotli`).

## Why stateless

This source serves a personal assistant use case: "What's AAPL at?" is a real-time question, not a data pipeline. Storing historical prices would add complexity with no benefit. This makes the source fundamentally different from iMessage/Gmail/WhatsApp which sync and index historical data.

## Yahoo Finance Terms of Service & Disclaimer

Yahoo Finance data is provided by Yahoo and its data providers. The `/v7/finance/quote` endpoint is undocumented and not part of any official public API. Yahoo does not offer a public API for programmatic access to financial data. This implementation accesses Yahoo Finance in the same manner as a web browser visiting the site.

Users should be aware that:

- Yahoo may change or restrict access to this endpoint at any time without notice
- Data may be delayed (typically 15-20 minutes for US equities)
- This should not be used for trading decisions or any financial application requiring guaranteed data accuracy or availability
- Usage should comply with Yahoo's [Terms of Service](https://legal.yahoo.com/us/en/yahoo/terms/otos/index.html)
- This project is not affiliated with, endorsed by, or sponsored by Yahoo

## Prior art

- [achannarasappa/ticker](https://github.com/achannarasappa/ticker) — 4k+ stars, same Yahoo approach with stdlib
- [wnjoon/go-yfinance](https://github.com/wnjoon/go-yfinance) — CycleTLS approach
- [ranaroussi/yfinance](https://github.com/ranaroussi/yfinance) — Python, most popular finance library, same endpoints
