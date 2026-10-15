# DLX for Godwit

Translate emails and selected text in [Godwit](https://github.com/godwit-email/godwit) into many languages, with nothing to set up. Translation is powered by [DLX](https://github.com/OwO-Network/DLX), called in-process: no server to run, no account, no API key.

DLX is a community plugin. It needs a Godwit release with native plugins ([godwit-email/godwit#68](https://github.com/godwit-email/godwit/pull/68)).

## Install

1. In Godwit, open **Settings › Extensions › Plugin directory**, find **DLX** and choose **Install**.
2. In **Settings › General › Translate with**, choose **DLX**.

Messages, selections and drafts are then translated with DLX. There are no plugin settings. Godwit's proxy environment (`HTTPS_PROXY`, `HTTP_PROXY`, `NO_PROXY`) is used as is.

Builds are available for macOS (Apple silicon and Intel), Linux (arm64 and amd64, glibc of Ubuntu 24.04 or newer) and Windows (x64). There is no Windows on Arm build yet.

## Privacy and availability

- The text you translate is sent to the third-party translation service DLX uses. DLX also visits that service's site to set up cookies, and Apple's public App Store API to look up a client version.
- The plugin does not log or store what it translates, and uses no relay of its own.
- DLX relies on an unofficial interface. It may be rate limited, blocked for a while, or stop working when the service changes.
- Like every native plugin, it runs in a Godwit child process with your user's permissions; that process is not a sandbox. Godwit only runs the exact file reviewed for its plugin directory.

## How it translates

- Language tags map to DLX's languages: `zh-CN` and `zh-SG` to Simplified Chinese, `zh-TW`, `zh-HK` and `zh-MO` to Traditional; English defaults to American and Portuguese to Brazilian.
- Text goes in batches of at most 1,500 UTF-16 units (an emoji counts as two), sent in order. Long paragraphs are split at sentence or word boundaries, which loses some context across sentences. Paragraph count and empty paragraphs are kept.
- On rate limiting, a network error or an incomplete answer, the whole translation fails; there is no partial result and no automatic retry. No new batch starts after 30 seconds, within Godwit's 60-second call limit, so a very long email may need to be translated a selection at a time.
- The detected source language is reported only for a single paragraph whose batches all agree, so Godwit does not skip the other languages of a mixed-language email.

## Development

Requirements: Go 1.27.1, a C compiler and Python 3 (MinGW-w64 GCC on Windows).

```sh
go test -race ./...
go vet ./...
python3 scripts/package.py
(cd tools/host && go mod download)
python3 scripts/smoke.py dist/darwin-arm64/plugin.dylib
python3 scripts/smoke.py dist/darwin-arm64/plugin.dylib --live   # optional: translates one public sample
```

Use your platform's library in the smoke commands. `scripts/package.py` writes `dist/<platform>/` with `plugin.json`, the library (`plugin.dylib`, `plugin.so` or `plugin.dll`) and `dlx-<platform>.zip`. To install a build by hand, unzip that ZIP and install its folder in Godwit's Extensions settings; Godwit asks for consent first, as for any native plugin outside its directory. The smoke test loads the library through Godwit's real plugin host (`ServeNative`) and checks `describe`, `configure`, the translate contract and the host's SHA-256 and ABI checks. The `--live` run sends only a public sample sentence, never mail or credentials.

CI tests, builds and loads the library on native macOS arm64/amd64, Linux arm64/amd64 and Windows amd64 runners; the race detector does not run on Windows.

Notes on the implementation:

- `describe`, `configure` and `translate` follow the ABI v1 JSON envelope (`sdk/godwit_plugin.h`); an unknown method or a bad request is `invalid_request`. Requests are at most 1 MiB and answers 4 MiB; answers are allocated in C and freed by the host.
- The library handles concurrent calls and has no mutable configuration.
- On macOS Intel, a Go host and a Go library share a thread-local storage slot, so initialization is plain C and calls enter the Go runtime on a C thread of their own. Other platforms call in directly. The ABI is the same everywhere.
- DLX is pinned to `0f2b50dd69fea3012d95a8d0e7a4ae8dca9ff5b2` and downloads no code at run time.

## Releasing

Bump `version` in `plugin.json` and push the matching `v<version>` tag. CI builds every platform and creates a draft release with the bare libraries, ZIPs for manual install (manifest, library and dependency licenses), `SHA256SUMS` and `registry-release.json`. Publish the release, then propose `registry-release.json` as a new release of `dlx` in [godwit-email/plugins](https://github.com/godwit-email/plugins).

## License

Plugin code and the Godwit ABI header: Apache-2.0. DLX: MIT; see `THIRD_PARTY_NOTICES`. Each platform's install folder includes the licenses of the linked Go modules in `DEPENDENCY_LICENSES.txt`.
