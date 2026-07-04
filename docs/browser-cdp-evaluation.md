# Browser CDP Evaluation

Issue #42 is limited to a daemon-owned Chrome/CDP tracer bullet. Whisk does not
embed a browser pane in Wails, does not put browser runtime state in Svelte, and
does not expose raw CDP over the daemon protocol.

The current diagnostic is intentionally local and narrow:

```sh
whisk browser diagnose -cdp-url http://127.0.0.1:9222 --json
```

It accepts only loopback HTTP CDP endpoints, lists browser metadata and page
targets, and can print a launch-command preview for a future dedicated Chrome
profile:

```sh
whisk browser diagnose \
  -chrome-path "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  -user-data-dir /tmp/whisk-browser \
  --json
```

The command does not launch Chrome, persist captures, capture screenshots, add
protocol routes, regenerate SDKs, or create Wails bindings. Those remain
deferred until the product and security gates are closed.

## Deferred Gates

- Choose attach-only, launch-with-dedicated-profile, or both.
- Define the explicit user authorization step before connecting to Chrome CDP.
- Decide whether authenticated default profiles are allowed.
- Set capture caps for text, HTML, CSS, and screenshots with truncation metadata.
- Keep screenshots explicit and default-off.
- Exclude cookies, storage, network bodies, and browser logs from capture payloads.
- Decide whether captures are durable attachments and how users delete them.
- Define daemon restart behavior for browser resources.

Full embedded UI remains deferred because Wails does not provide an Electron
`webview` equivalent, and Whisk's invariant requires runtime resources to be
owned by the daemon before any client renders or invokes them.
