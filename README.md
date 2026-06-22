# botmodule-go

Botmother tashqi modul yozish uchun Go SDK.

**Maqsad:** Go dasturchilar (va AI agentlar) shu paket yordamida Botmother engine uchun
yangi node turlari yaratsin — engine kodiga tegmasdan. Modul = JSON-RPC 2.0 HTTP servis.

Faqat stdlib: `net/http`, `encoding/json`, `strings` — tashqi dependency yo'q.

---

## Tez boshlash

```bash
go get github.com/BotSpace/botmodule-go
```

```go
package main

import (
    "strings"
    botmodule "github.com/BotSpace/botmodule-go"
)

func main() {
    m := botmodule.New("demo", "Demo Module")
    m.Version = "0.1.0"

    m.AddNode(botmodule.Node{
        Type:     "demo.Echo",
        Title:    "Echo",
        Category: "integrations",
        Icon:     "sparkles",
        Color:    "blue",
        Content: []botmodule.Field{
            {Type: "text", Key: "input", Label: "Matn", Placeholder: "{{message.text}}"},
        },
        Defaults:      map[string]any{"input": "{{message.text}}"},
        ProducesState: []string{"echo_output"},
        Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
            return botmodule.Result{
                ContextUpdates: map[string]any{"echo_output": c.String("input")},
            }
        },
    })

    m.AddNode(botmodule.Node{
        Type:        "demo.OnKeyword",
        Title:       "Kalit so'z kelganda",
        Category:    "triggers",
        Icon:        "zap",
        Color:       "amber",
        Trigger:     true,
        TriggerMode: "event-match",
        Content: []botmodule.Field{
            {Type: "text", Key: "keyword", Label: "Kalit so'z"},
        },
        Match: func(c *botmodule.TriggerCtx) botmodule.MatchResult {
            text := c.MessageText()
            kw := c.String("keyword")
            ok := kw != "" && strings.Contains(strings.ToLower(text), strings.ToLower(kw))
            return botmodule.MatchResult{Matched: ok}
        },
    })

    m.Docs = "# Demo Module\nEcho va trigger namunasi."
    m.Serve(":8100")
}
```

Server ishga tushgach:
- `GET :8100/health` → `{"ok":true,"module":"demo"}`
- `POST :8100/rpc` — JSON-RPC 2.0 dispatch

---

## API hujjati

### `botmodule.New(id, name string) *Module`

Yangi modul yaratadi.

| Maydon | Tavsif |
|---|---|
| `m.ID` | Global unique slug (= node namespace prefiksi) |
| `m.Name` | Ko'rinadigan nom |
| `m.Version` | Semver (default `"0.1.0"`) |
| `m.Docs` | Markdown hujjat (`docs()` metodi orqali qaytariladi) |

### `m.AddNode(n Node)`

Modulga node qo'shadi. `Type` majburiy, `"moduleId.NodeName"` formatida bo'lishi shart
(masalan `demo.Echo`).

---

### `Node` struct

| Maydon | Tur | Tavsif |
|---|---|---|
| `Type` | `string` | **MAJBURIY.** `"moduleId.NodeName"` namespace |
| `Title` | `string` | Sidebar/canvas sarlavhasi |
| `Description` | `string` | Qisqa tavsif |
| `Category` | `string` | `"triggers"` yoki boshqa (`"integrations"`, ...). Trigger=true bo'lsa default `"triggers"` |
| `Icon` | `string` | lucide-react ikon nomi (masalan `"sparkles"`, `"database"`, `"credit-card"`) |
| `Color` | `string` | Rang tokeni: `blue`, `violet`, `emerald`, `amber`, `rose`, `slate` |
| `Width` | `int` | Node kengligi (default 200, 300 katta) |
| `SortOrder` | `int` | Sidebar tartib raqami |
| `Content` | `[]Field` | Form maydonlari |
| `Defaults` | `map[string]any` | Yangi node default qiymatlari |
| `ProducesState` | `[]string` | UI autocomplete uchun statik maslahat |
| `Trigger` | `bool` | `true` = trigger node |
| `TriggerMode` | `string` | `"event-match"` (trigger.match orqali) |
| `Execute` | `func(*ExecuteCtx) Result` | Action handler |
| `Match` | `func(*TriggerCtx) MatchResult` | Trigger handler |

---

### `Field` struct — content maydonlari

| Maydon | Tur | Tavsif |
|---|---|---|
| `Type` | `string` | **MAJBURIY.** Quyidagi ro'yxatdan biri |
| `Key` | `string` | **MAJBURIY.** `data[key]` kaliti |
| `Label` | `string` | Ko'rinadigan nom |
| `Placeholder` | `string` | Input placeholder |
| `HelpText` | `string` | Yordam matni |
| `Optional` | `bool` | `true` → "Qo'shimcha sozlamalar" bo'limida |
| `CredentialType` | `string` | `credential` field uchun type filtri |
| `Options` | `[]SelectOption` | `select` field uchun tanlovlar |
| `VisibleWhen` | `*VisibleWhen` | Shartli ko'rsatish |

**Ruxsat etilgan `type`lar:**
`text`, `number`, `textarea`, `select`, `checkbox`, `switch`, `description`,
`widget`, `json`, `color`, `file`, `datetime`, `boolean`, `code`, `credential`

#### `select` misoli:
```go
Field{
    Type:  "select",
    Key:   "mode",
    Label: "Rejim",
    Options: []botmodule.SelectOption{
        {Value: "a", Label: "A"},
        {Value: "b", Label: "B"},
    },
}
```

#### Shartli maydon (`visibleWhen`):
```go
Field{Type: "switch", Key: "save", Label: "Saqlash", Optional: true},
Field{
    Type:        "text",
    Key:         "state_key",
    Label:       "Kalit",
    Optional:    true,
    VisibleWhen: &botmodule.VisibleWhen{Key: "save", Equals: true},
},
```

#### Credential field:
```go
Field{
    Type:           "credential",
    Key:            "api_cred",
    Label:          "API Credential",
    CredentialType: "openai", // ixtiyoriy filtri
}
```

---

### `ExecuteCtx` — action handler konteksti

```go
Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
    // ...
}
```

| Maydon/Metod | Tavsif |
|---|---|
| `c.Type` | Node turi (`"demo.Echo"`) |
| `c.Data` | `map[string]any` — field qiymatlari (engine `{{...}}` ni allaqachon resolve qilgan) |
| `c.Context` | `map[string]any` — flow konteksti (state) |
| `c.ChatID` | `int64` — Telegram chat ID |
| `c.Credentials` | `map[string]*Credential` — decrypted credential'lar |
| `c.String(key)` | `data[key]` string qiymat, topilmasa `""` |
| `c.Int(key)` | `data[key]` int64 qiymat, topilmasa `0` |
| `c.Credential(key)` | `(*Credential, bool)` |

### `Result` — action handler natijasi

```go
botmodule.Result{
    ContextUpdates: map[string]any{"key": "value"},
    ExitOutput:     "",  // ixtiyoriy: nomli chiqish edge'iga yo'naltirish
}
```

---

### `TriggerCtx` — trigger handler konteksti

```go
Match: func(c *botmodule.TriggerCtx) botmodule.MatchResult {
    // ...
}
```

| Maydon/Metod | Tavsif |
|---|---|
| `c.Type` | Node turi |
| `c.Data` | `map[string]any` — field qiymatlari |
| `c.Update` | `map[string]any` — Telegram Update konverti |
| `c.Context` | `map[string]any` — flow konteksti |
| `c.MessageText()` | `update.message.text` |
| `c.CallbackData()` | `update.callback_query.data` |
| `c.String(key)` | `data[key]` string qiymat |

### `MatchResult` — trigger handler natijasi

```go
botmodule.MatchResult{
    Matched:        true,
    ContextUpdates: map[string]any{"matched_keyword": "salom"},
}
```

---

### `Credential` struct

Engine credential'ni decrypt qilib `node.execute` paytida uzatadi.

| Maydon | Tavsif |
|---|---|
| `TypeKey` | Credential turi (`"openai"`, `"http_header"`, ...) |
| `Mode` | `bearer`, `apikey`, `basic`, `header`, `oauth2`, `none` |
| `Data` | `map[string]string` — decrypted ma'lumotlar (`api_key`, `token`, `username`, `password`, ...) |

```go
cred, ok := c.Credential("api_cred")
if !ok {
    // credential tanlanmagan
}
switch cred.Mode {
case "bearer":
    header := "Bearer " + cred.Data["token"]
case "basic":
    // cred.Data["username"], cred.Data["password"]
}
```

> Xavfsizlik: credential ma'lumotlarini `context_updates` orqali flow state'ga to'liq
> oqizmang. Faqat kerakli HTTP chaqiruvda ishlating yoki maskalang.

---

### `m.Serve(addr string)`

HTTP server ishga tushiradi. `/rpc` (POST, JSON-RPC 2.0) + `/health` (GET).

```go
m.Serve(":8100")      // aniq port
m.Serve("")           // PORT env → yo'q bo'lsa :8100
```

Autentifikatsiya: `MODULE_AUTH_TOKEN` env. Bo'sh bo'lsa tekshirilmaydi.
Port: `PORT` env yoki `addr` argument.

### `m.ServeHandler() http.Handler`

Test yoki custom server uchun `http.Handler` qaytaradi (auth tekshiruvsiz).

---

## describe() avtomatik generatsiyasi

`m.Serve()` / `describe` RPC chaqirilganda SDK quyidagilarni **avtomatik** to'ldiradi:

| Manifest maydoni | Qanday to'ldiriladi |
|---|---|
| `status` | doim `"runtime"` |
| `category` | `Node.Category` yoki Trigger=true → `"triggers"` |
| `size.width` | `Node.Width` yoki default `200` |
| `sidebar.groupId` | = `category` |
| `sidebar.sortOrder` | `Node.SortOrder` yoki `100 + index` |
| `sidebar.elementType` | = `Node.Type` |
| `sidebar.enabled` | doim `true` |
| `handles` | Trigger: `[source-default]`; Action: `[target-default, source-default]` |

---

## Trigger turlari

### Event-match (trigger.match)

Engine har Telegram update'da `trigger.match` chaqiradi.

```go
m.AddNode(botmodule.Node{
    Type:        "demo.OnKeyword",
    Trigger:     true,
    TriggerMode: "event-match",
    // ...
    Match: func(c *botmodule.TriggerCtx) botmodule.MatchResult {
        return botmodule.MatchResult{Matched: strings.Contains(c.MessageText(), "salom")}
    },
})
```

Latency: har update'da tarmoq round-trip. Timeout 2s, xato → `matched:false` (graceful).

### Push trigger (modul → engine)

Modul tashqi hodisada flow'ni o'zi boshlaydi (webhook, cronjob, ...):

```
POST {ENGINE_PUSH_URL}/module-trigger/{module}/{type}[/{chat_id}]
Header: X-Internal-Token: <token>
Body: {"chat_id": 123, "context": {}, "payload": {}}
```

Push trigger uchun bu SDK'da alohida funksiya yo'q — to'g'ri HTTP so'rov yuboring.

---

## Dinamik state

Foydalanuvchi o'zgaruvchi nomini o'zi kiritsa:

```go
m.AddNode(botmodule.Node{
    Type: "demo.SetVar",
    Content: []botmodule.Field{
        {Type: "text", Key: "var_name", Label: "O'zgaruvchi nomi"},
        {Type: "text", Key: "value",    Label: "Qiymat"},
    },
    Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
        name := strings.TrimSpace(c.String("var_name"))
        if name == "" {
            return botmodule.Result{ContextUpdates: map[string]any{}}
        }
        return botmodule.Result{
            ContextUpdates: map[string]any{name: c.String("value")},
        }
    },
})
```

Engine `value` ichidagi `{{...}}` ni allaqachon resolve qilib beradi.

---

## JSON-RPC 2.0 kontrakt

Endpoint: `POST /rpc`  
Auth: `Authorization: Bearer <MODULE_AUTH_TOKEN>` (bo'sh bo'lsa tekshirilmaydi)

| Metod | Tavsif |
|---|---|
| `describe` | Node manifestlari (constructor uchun) |
| `node.execute` | Node logikasi — `{type, data, context, chat_id, credentials}` |
| `trigger.match` | Trigger tekshiruv — `{type, data, update, context}` |
| `docs` | Markdown hujjat |

Health: `GET /health` → `200 {"ok":true,"module":"<id>"}`

---

## Test

```bash
cd botmodule-go
go test ./...
```

---

## Yangi modul yozish qadamlari

1. `module-template-go` reposidan klonlang yoki `go get` qiling.
2. `module.yaml` — `module.id`, `name`, `icon` o'zgartiring.
3. `main.go` — `botmodule.New(id, name)` bilan modul yarating.
4. `m.AddNode(...)` — har node uchun `Execute` yoki `Match` yozing.
5. `m.Serve(":8100")` — server ishga tushiradi.
6. `docker build` + registry'ga push → admin panel → Modullar → qo'shish.

---

## Litsenziya

MIT
