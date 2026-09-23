# notifyutils

Go common notify utils.

`notifyutils` is the **core module** of the notifyutils ecosystem — a pluggable, multi-channel
notification library for Go. It defines the message model, the driver interface, the routing
rules and the error type that every channel driver implements. Applications depend on this core
plus one or more driver modules, and push notifications through a single API no matter whether
the message ends up in a DingTalk group, an inbox, a Telegram chat or on a handset as an SMS.

The core talks to **nothing**: it holds no SDK and no HTTP call of its own. Everything concrete
lives in a driver module, and the SMS drivers in turn delegate to
[smsutils](https://go.gh.ink/smsutils), this family's SMS equivalent.

### Why no provider SDK in this family

Drivers use the Go standard library, not vendor SDKs, and that is a deliberate rule rather than
an omission: every channel this family wraps is exposed as a webhook or an SMTP conversation, and
the official Go SDKs either do not exist (Telegram, Discord, WeCom) or cover a *different*
integration surface than the one a notification needs (the DingTalk and Lark SDKs serve
application robots behind `accessToken`, not the custom group robots that alerting uses; Gotify's
is a generated swagger client whose price is the whole `go-openapi` runtime for one POST). Where
an official SDK genuinely covers the flow — the cloud SMS providers — the driver uses it, by way
of smsutils. The result is a family whose only third-party dependencies are `go.gh.ink/toolbox`
and the smsutils modules.

## Architecture

The ecosystem follows the same plugin-driver pattern as smsutils and payutils. The core exposes
interfaces and a global driver registry; each channel ships as a separate module that registers
itself through an `init()` function when imported.

```
                     ┌──────────────────────────┐
                     │      Application code    │
                     └────────────┬─────────────┘
                                  │ client.NewClient(config) / c.Send(ctx, msg)
                                  ▼
          ┌───────────────────────────────────────────────────────┐
          │                   notifyutils (core)                  │
          │  model/    Message, Client, Driver, Config, Routes    │
          │  client/   NewClient factory, Send, SendTo, Names     │
          │  driver/   Register()                                 │
          │  errors/   NotifyutilsError + sentinels               │
          │  utils/    text, sign, httpx shared by the drivers    │
          │  internal/state/  global driver registry              │
          └───────────────────────────────────────────────────────┘
                                  ▲ driver.Register(Name, Driver{})
     ┌──────────┬──────────┬──────┴───────┬──────────┬──────────┬───────────┐
     │          │          │              │          │          │           │
  dingtalk   wecom       lark         telegram    discord    gotify   email  webhook
     └──────────┴──────────┴──────────────┴───────┬──────────┴──────────────┘
                                                  │
                              sms/{aliyun,qcloud,bce,ucloud,volc}
                                                  │ wraps
                                        go.gh.ink/smsutils/<provider>
```

### Module map

| Module | Import path | Channel |
|--------|-------------|---------|
| notifyutils (this) | `go.gh.ink/notifyutils` | Core library |
| notifyutils-dingtalk | `go.gh.ink/notifyutils/dingtalk` | DingTalk group robot |
| notifyutils-wecom | `go.gh.ink/notifyutils/wecom` | WeCom (企业微信) group robot |
| notifyutils-lark | `go.gh.ink/notifyutils/lark` | Lark / Feishu group robot |
| notifyutils-telegram | `go.gh.ink/notifyutils/telegram` | Telegram bot |
| notifyutils-discord | `go.gh.ink/notifyutils/discord` | Discord webhook |
| notifyutils-gotify | `go.gh.ink/notifyutils/gotify` | Gotify server |
| notifyutils-email | `go.gh.ink/notifyutils/email` | SMTP mail |
| notifyutils-webhook | `go.gh.ink/notifyutils/webhook` | Arbitrary HTTP endpoint |
| notifyutils-sms-aliyun | `go.gh.ink/notifyutils/sms/aliyun` | Alibaba Cloud SMS |
| notifyutils-sms-qcloud | `go.gh.ink/notifyutils/sms/qcloud` | Tencent Cloud SMS |
| notifyutils-sms-bce | `go.gh.ink/notifyutils/sms/bce` | Baidu Cloud SMS |
| notifyutils-sms-ucloud | `go.gh.ink/notifyutils/sms/ucloud` | UCloud SMS |
| notifyutils-sms-volc | `go.gh.ink/notifyutils/sms/volc` | Volcengine SMS |

Modules publish v1, so the import path carries no `/vN` suffix and the Go sources sit at the
repository root rather than in a versioned subdirectory.

A driver registers under the **last element of its import path**, which is also the key to use in
`Config.Credentials` and `Config.Routes`. The SMS modules are the case to watch: the module is
`go.gh.ink/notifyutils/sms/aliyun` but the channel is `aliyun`, so a config that says `sms-aliyun`
fails with `ErrDriverNotRegistered`.

## Installation

```bash
go get go.gh.ink/notifyutils
# plus one or more channels, e.g.
go get go.gh.ink/notifyutils/dingtalk
```

`go.gh.ink` is a public vanity host that maps each module path to its repository, so the
standard proxies work with no configuration:

```bash
env GOPROXY=https://goproxy.cn,direct go build ./...
```

## Usage

1. Blank-import the channels you want so they self-register.
2. Build a `model.Config` mapping each channel name to its credential map.
3. Send messages; the core routes them by level.

```go
package main

import (
	"context"
	"log"

	"go.gh.ink/notifyutils/client"
	"go.gh.ink/notifyutils/model"

	_ "go.gh.ink/notifyutils/dingtalk"
	_ "go.gh.ink/notifyutils/telegram"
)

func main() {
	c, err := client.NewClient(model.Config{
		Credentials: model.C{
			"dingtalk": {
				"webhook": "https://oapi.dingtalk.com/robot/send?access_token=…",
				"secret":  "SEC…",
			},
			"telegram": {
				"botToken": "123456:ABC-DEF…",
				"chatID":   "-1001234567890",
			},
		},
		Routes: model.Routes{
			// An outage notice also reaches the on-call's handset; everything
			// else stays in the group chat.
			model.LevelCritical: {"dingtalk", "telegram"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	err = c.Send(context.Background(), model.Message{
		Title:  "Deploy finished",
		Text:   "release ${version} is live on ${host}",
		Format: model.FormatMarkdown,
		Level:  model.LevelInfo,
		Vars:   model.Vars{{Key: "version", Value: "1.4.2"}, {Key: "host", Value: "web-01"}},
	})
	if err != nil {
		log.Print(err) // every channel was still attempted; err lists the failures
	}
}
```

## Core API

### `client.NewClient`

```go
func NewClient(config model.Config) (*Client, error)
```

Walks `config.Credentials`, looks each name up in the global registry and builds one channel
client per entry. A name that was never registered — you forgot the import — yields
`errors.ErrDriverNotRegistered` carrying that name.

### `model.Config`

```go
type Config struct {
	Routes      Routes             // map[Level][]driverName
	HTTPClient  *http.Client       // optional; defaults to one with HTTPTimeout
	Credentials C                  // map[driverName]map[credentialKey]credentialValue
	Marshal     func(v any) ([]byte, error)
	Unmarshal   func(data []byte, v any) error
}
```

`HTTPClient` is handed to every driver, so one proxy setting and one timeout cover the whole fan
out. When it is nil the core substitutes `&http.Client{Timeout: model.HTTPTimeout}` — drivers
never see a nil client. `Marshal`/`Unmarshal` default to `encoding/json`.

### `(*Client) Send`, `SendTo`, `Names`

```go
func (c *Client) Send(ctx context.Context, msg model.Message) error
func (c *Client) SendTo(ctx context.Context, drivers []string, msg model.Message) error
func (c *Client) Names() []string
```

`Send` resolves the target channels from `Routes[msg.Level]`, falling back to every configured
channel in sorted order. `SendTo` bypasses routing and names the channels explicitly.

Both attempt **every** selected channel even after one fails, and return the failures joined
(`errors.Join`), each prefixed with the channel that produced it. `errors.Is` and `errors.As`
still see through the join and the prefix, so a caller can ask "did anything fail with
`ErrDriverSendFailed`?" without parsing strings.

Before any driver sees the message the core fills in what the channel contracts assume: an empty
`msg.Format` becomes `FormatPlain`, an empty `msg.Level` becomes `LevelInfo`, and `msg.Vars` are
bound into `msg.Title` and `msg.Text` as `${key}`. Rendering lives here, once, so that one message
fans out as the same text to every channel — a driver must not call `Subst` itself, which would
expand a substituted value that happens to contain `${...}` a second time. A placeholder with no
matching var is left visible rather than erased, and the caller's own `Message` is never mutated
(rendering happens on a copy). Drivers additionally treat an empty `Format` as plain, so a client
built straight from `Driver{}.NewClient` — in a test, or by an application that wants one channel
and no routing — behaves the same as one reached through the core.

### `model.Message`

```go
type Message struct {
	Title      string
	Text       string
	Format     Format   // FormatPlain | FormatMarkdown | FormatHTML
	Level      Level    // LevelDebug | Info | Warn | Error | Critical
	Recipients []string // meaning per channel: phone, chat ID, mailbox, mention list…
	Template   string   // provider-side template id (SMS)
	Vars       Vars     // ordered ${key} bindings, rendered into Title/Text by the core
	Extras     map[string]any
}
```

`Message` deliberately holds only what **every** channel can express. Anything a channel can do
but others cannot — DingTalk buttons, Telegram thread ids, email attachments — travels in
`Extras` under a key that the owning driver exports as a constant, and its value keeps whatever
Go type that driver documents. The core forwards `Extras` untouched and never reads it.

`Recipients` is per-message addressing. It may stay empty: each channel documents a default
target in its credentials, which is the normal setup for a push notification (a fixed group, a
fixed inbox). Leaving both empty is `errors.ErrRecipientRequired`.

Typed readers are provided for the common shapes: `Extra`, `ExtraString`, `ExtraStrings`,
`ExtraBool`, `ExtraInt`.

### `model.Driver` (for driver authors)

```go
type Driver interface {
	NewClient(params DriverClientParam) (Client, error)
}

type DriverClientParam struct {
	Credential map[string]string
	HTTPClient *http.Client
	Marshal    func(v any) ([]byte, error)
	Unmarshal  func(data []byte, v any) error
}

type Client interface {
	Send(ctx context.Context, msg Message) error
}
```

## Shared driver utilities

The core exports the plumbing that several channels need identically, so it appears once:

| Package | Provides |
|---------|----------|
| `utils/httpx` | `Do(ctx, client, Request)` — one HTTP call with JSON or raw body, returning status and full response bytes. Only transport failures are errors; a `4xx`/`5xx` is data, because every channel reports failure in its own body shape. |
| `utils/sign`  | `HMACSHA256Base64`, `HMACSHA256Hex`, `UnixSeconds`, `UnixMilli`. The robot channels differ in what they use as key and as message, so each composes these itself and documents its own recipe. |
| `utils/text`  | `Subst` for `${key}` binding (applied by the core before dispatch, not by drivers), `Chunk` for splitting an over-long body at the last whitespace that fits a channel's per-message ceiling, `Runes` for character counts. |

## Errors

The `errors` package provides `NotifyutilsError`, a rich error type carrying channel context.
`Error()` reports the sentinel text only; the diagnostics come from the accessor methods. Every
`With*` call clones the error, so the exported sentinels stay immutable and `errors.Is` keeps
working on a decorated copy.

| Sentinel | Meaning |
|----------|---------|
| `ErrDriverNotRegistered` | Credentials referenced a channel that was not imported |
| `ErrDriverNotFound` | A route or `SendTo` named a channel that was not configured |
| `ErrNoDriverConfigured` | Nothing to deliver to: no route matched and no channel is configured |
| `ErrDriverCredentialInvalid` | Required credential fields were missing or empty |
| `ErrDriverSendFailed` | The channel returned a non-success response |
| `ErrRecipientRequired` | Neither the message nor the credentials supply a target |
| `ErrTemplateRequired` | A template-delivering channel (SMS) got a message with no template to send |
| `ErrUnsupportedFormat` | The channel cannot render the requested `Format` |
| `ErrMessageTooLong` | The channel has a hard ceiling this message exceeds and cannot split |

Builder / accessor pairs: `WithDriverName`/`DriverName`, `WithDriverCode`/`DriverCode`,
`WithDriverMessage`/`DriverMessage`, `WithDriverRequestID`/`DriverRequestID`,
`WithDriverResponse`/`DriverResponse`.

```go
err := c.Send(ctx, msg)

var nerr *errors.NotifyutilsError
if stderrors.As(err, &nerr) && stderrors.Is(err, errors.ErrDriverSendFailed) {
	log.Printf("channel=%s code=%s message=%s requestID=%s",
		nerr.DriverName(), nerr.DriverCode(), nerr.DriverMessage(), nerr.DriverRequestID())
}
```

## Driver development

A channel module is a separate Go module in its own repository, sources at the repository root:

1. Module path `go.gh.ink/notifyutils/<channel>`; package named after the channel.
2. Define `const Name = "<channel>"` plus one credential-key constant per key.
3. Implement `model.Driver.NewClient`: validate required credentials
   (`ErrDriverCredentialInvalid.WithDriverName(Name)`), read the default target, build the
   channel client from `params.HTTPClient`.
4. Implement `model.Client.Send(ctx, msg)`: resolve targets (`msg.Recipients` or the configured
   default), check `msg.Format` and the size ceiling, lay out the payload (`Title`/`Text` arrive with variables already bound), call the endpoint and
   map a failure onto `ErrDriverSendFailed` with the channel's code, message and request id.
5. Register from `init()`: `driver.Register(Name, Driver{})`.
6. Read `go.gh.ink/notifyutils/utils/{httpx,sign,text}` before hand-rolling any of it.

A driver must be safe for concurrent `Send` calls, must not mutate the `Message` it is handed,
and must not reach the network except through the injected `HTTPClient`.

## Requirements

- Go 1.26.0+
- Dependencies: `go.gh.ink/toolbox` (the core needs nothing else)

## License

[Apache License 2.0](LICENSE)
