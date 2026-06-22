# botmodule-go

Botmother / Botspace **tashqi modul** yozish uchun Go SDK.

**Maqsad:** Go dasturchilar (va AI agentlar) shu paket yordamida Botmother engine uchun
yangi node turlari yaratsin — engine kodiga tegmasdan. Modul = alohida HTTP servis
bo'lib, engine bilan **JSON-RPC 2.0** orqali gaplashadi.

Faqat stdlib: `net/http`, `encoding/json` — tashqi dependency **yo'q**.

Bu hujjat to'liq ma'lumotnoma: har bir tur, har bir field type, ikon va rang
token'lari, JSON-RPC kontrakti, `module.yaml`, deploy — ipidan ignasigacha.

---

## Mundarija

1. [Modul nima va qanday ishlaydi](#1-modul-nima-va-qanday-ishlaydi)
2. [Tez boshlash](#2-tez-boshlash)
3. [API — `Module`, `New`, `AddNode`, `Serve`](#3-api--module-new-addnode-serve)
4. [`Node` struct — to'liq](#4-node-struct--toliq)
5. [`Field` struct — content maydonlari](#5-field-struct--content-maydonlari)
6. [Field `type` ma'lumotnomasi — qaysi type nima beradi](#6-field-type-malumotnomasi)
7. [`visibleWhen` — shartli ko'rsatish](#7-visiblewhen--shartli-korsatish)
8. [Ikon nomlari (`Icon`) — to'liq ro'yxat](#8-ikon-nomlari-icon)
9. [Rang token'lari (`Color`) — to'liq ro'yxat](#9-rang-tokenlari-color)
10. [`ExecuteCtx` / `Result` — action handler](#10-executectx--result--action-handler)
11. [`TriggerCtx` / `MatchResult` — trigger handler](#11-triggerctx--matchresult--trigger-handler)
12. [`Credential` — credential ishlatish](#12-credential--credential-ishlatish)
13. [Dinamik qiymatlar va state (`{{...}}`)](#13-dinamik-qiymatlar-va-state)
14. [`describe()` avtomatik generatsiyasi](#14-describe-avtomatik-generatsiyasi)
15. [JSON-RPC 2.0 kontrakt — simdagi format](#15-json-rpc-20-kontrakt--simdagi-format)
16. [Environment o'zgaruvchilari](#16-environment-ozgaruvchilari)
17. [`module.yaml` — to'liq](#17-moduleyaml--toliq)
18. [Test](#18-test)
19. [Yangi modul yozish — checklist](#19-yangi-modul-yozish--checklist)

---

## 1. Modul nima va qanday ishlaydi

Modul — bu engine'ga **yangi node turlari** beradigan mustaqil HTTP servis. Oqim:

```
[Modul servis]  ── describe() ──►  engine/constructor (node'lar sidebar'da paydo bo'ladi)
       ▲
       │  node.execute (action ishlaganda)
       │  trigger.match (har update'da, event-trigger uchun)
       │
[Engine]  ── JSON-RPC 2.0, POST /rpc, Bearer auth ──►  [Modul servis]
```

- Modul **o'z node'larini** `describe()` orqali constructor manifest formatida tasvirlaydi.
- Foydalanuvchi node'ni canvas'ga qo'ygach va flow ishga tushganda, engine
  `node.execute` chaqiradi → modul biznes-logikani bajaradi → `context_updates` qaytaradi.
- Trigger node'lar uchun engine har kelgan Telegram update'da `trigger.match` chaqiradi.
- Node turi **`moduleId.NodeName`** namespace'da (masalan `weather.Forecast`) — kolliziya bo'lmaydi.

Modul **xabar yubormaydi** (bot token engine'da qoladi). Modul faqat ma'lumot
qaytaradi (`context_updates`), keyin flow'dagi oddiy `Send*` node'lari xabar yuboradi.

---

## 2. Tez boshlash

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

    // Action node.
    m.AddNode(botmodule.Node{
        Type:     "demo.Echo",
        Title:    "Echo",
        Category: "integrations",
        Icon:     "arrow-right",
        Color:    "integration-sky",
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

    // Trigger node.
    m.AddNode(botmodule.Node{
        Type:        "demo.OnKeyword",
        Title:       "Kalit so'z kelganda",
        Category:    "triggers",
        Icon:        "zap",
        Color:       "trigger-blue",
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

Tez sinash:
```bash
go run .
curl -X POST localhost:8100/rpc -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"describe","id":1}'
```

---

## 3. API — `Module`, `New`, `AddNode`, `Serve`

### `botmodule.New(id, name string) *Module`

| Maydon | Tur | Tavsif |
|---|---|---|
| `m.ID` | `string` | Global unique slug (= node namespace prefiksi). Masalan `"weather"` → `weather.Forecast` |
| `m.Name` | `string` | Ko'rinadigan nom |
| `m.Version` | `string` | Semver (default `"0.1.0"`) |
| `m.Docs` | `string` | Markdown hujjat (`docs` RPC metodida qaytariladi) |

### `m.AddNode(n Node)`

Modulga node qo'shadi. `Type` **majburiy** va `"moduleId.NodeName"` formatida bo'lishi shart.

### `m.Serve(addr string)`

`/rpc` va `/health` endpointlarini ochadi va tinglaydi (bloklovchi).
- `addr` bo'sh bo'lsa → `PORT` env tekshiriladi → u ham bo'sh bo'lsa `":8100"`.
- `MODULE_AUTH_TOKEN` env bo'lsa → har `/rpc` so'rovda `Authorization: Bearer <token>` talab qilinadi.

### `m.ServeHandler() http.Handler`

`net/http` `Handler` qaytaradi (test yoki custom server uchun). **Auth tekshiruvi yo'q** — kerak bo'lsa o'zingiz wrap qiling.

---

## 4. `Node` struct — to'liq

```go
type Node struct {
    Type          string   // MAJBURIY: "moduleId.NodeName"
    Title         string
    Description   string
    Category      string
    Icon          string
    Color         string
    Width         int
    SortOrder     int
    Content       []Field
    Defaults      map[string]any
    ProducesState []string
    Trigger       bool
    TriggerMode   string
    Execute       func(c *ExecuteCtx) Result
    Match         func(c *TriggerCtx) MatchResult
}
```

| Maydon | Tur | Default | Tavsif |
|---|---|---|---|
| `Type` | `string` | — | **MAJBURIY.** `"moduleId.NodeName"` namespace (masalan `weather.Forecast`) |
| `Title` | `string` | `""` | Sidebar/canvas sarlavhasi |
| `Description` | `string` | `""` | Qisqa tavsif (sidebar tooltip) |
| `Category` | `string` | trigger bo'lsa `"triggers"`, aks holda `"integrations"` | Sidebar guruhi — pastdagi ro'yxatdan |
| `Icon` | `string` | `""` (→ sparkles) | Ikon nomi — [8-bo'lim](#8-ikon-nomlari-icon) |
| `Color` | `string` | `""` (→ kulrang) | Rang token — [9-bo'lim](#9-rang-tokenlari-color) |
| `Width` | `int` | `200` | Node kengligi (px). Ko'p field bo'lsa ham `200` qoldiring |
| `SortOrder` | `int` | `100+index` | Sidebar tartib raqami (kichik = yuqorida) |
| `Content` | `[]Field` | `[]` | Form maydonlari — [5-bo'lim](#5-field-struct--content-maydonlari) |
| `Defaults` | `map[string]any` | — | Yangi node qo'yilganda default qiymatlar (`data[key]`) |
| `ProducesState` | `[]string` | — | Bu node qaytaradigan state kalitlari (UI autocomplete uchun maslahat) |
| `Trigger` | `bool` | `false` | `true` = trigger node (faqat source handle, `triggers` kategoriya) |
| `TriggerMode` | `string` | `""` | `"event-match"` → engine `trigger.match` chaqiradi |
| `Execute` | `func` | — | Action handler (action node uchun) |
| `Match` | `func` | — | Trigger handler (event-match trigger uchun) |

**`Category` qiymatlari** (sidebar guruhi): `triggers`, `actions`, `messageOperations`,
`flowControl`, `integrations`, `customLogic`, `ai`. Modul node'lari odatda
`integrations` yoki `triggers`.

---

## 5. `Field` struct — content maydonlari

```go
type Field struct {
    Type           string         // MAJBURIY — 6-bo'limdagi ro'yxatdan
    Key            string         // MAJBURIY — data[key] kaliti
    Label          string
    Placeholder    string
    HelpText       string
    Optional       bool
    CredentialType string
    Options        []SelectOption
    VisibleWhen    *VisibleWhen
}

type SelectOption struct {
    Value string
    Label string
}

type VisibleWhen struct {
    Key    string
    Equals any
}
```

| Maydon | Tur | Tavsif |
|---|---|---|
| `Type` | `string` | **MAJBURIY.** [6-bo'lim](#6-field-type-malumotnomasi) |
| `Key` | `string` | **MAJBURIY.** `data[key]` kaliti — handler shu bilan o'qiydi (`c.String("key")`) |
| `Label` | `string` | Field ustidagi yorliq |
| `Placeholder` | `string` | Input placeholder matni |
| `HelpText` | `string` | Input ostidagi kichik yordam matni |
| `Optional` | `bool` | `true` → field "Qo'shimcha maydon qo'shish" bo'limiga tushadi (yashirin, foydalanuvchi qo'shadi). `visibleWhen` bilan birga ishlatilsa shart asosida avtomatik ochiladi |
| `CredentialType` | `string` | `credential` type uchun: qaysi credential turini filtrlash (masalan `"openai"`, `"http_header"`) |
| `Options` | `[]SelectOption` | `select` type uchun tanlovlar |
| `VisibleWhen` | `*VisibleWhen` | Shartli ko'rsatish — [7-bo'lim](#7-visiblewhen--shartli-korsatish) |

---

## 6. Field `type` ma'lumotnomasi

Constructor field'larni runtime renderlaydi. SDK `Field` struct'i quyidagi
xossalarni beradi: `type, key, label, placeholder, helpText, optional,
credentialType, options, visibleWhen`. Shu sabab **SDK orqali to'liq
ishlaydigan** type'lar — ushbu xossalar yetadiganlar:

### To'liq qo'llab-quvvatlanadigan type'lar

| `type` | UI element | Qaysi xossalar ishlaydi | Misol qiymat (`data[key]`) |
|---|---|---|---|
| `text` | bir qatorli input | `label`, `placeholder`, `helpText`, `optional`, `visibleWhen` | `"salom"` (string) |
| `number` | raqamli input | `label`, `placeholder`, `helpText`, `optional`, `visibleWhen` | `42` (number) |
| `textarea` | ko'p qatorli input | `label`, `placeholder`, `helpText`, `optional`, `visibleWhen` | `"uzun matn"` |
| `select` | dropdown | `label`, `placeholder`, **`options[]`**, `helpText`, `visibleWhen` | tanlangan `option.value` |
| `checkbox` | belgilash katakchasi | `label` | `true` / `false` (bool) |
| `switch` | toggle | `label`, `helpText` | `true` / `false` (bool) |
| `credential` | credential tanlovchi | `label`, **`credentialType`**, `optional`, `visibleWhen` | credential ID (engine decrypt qilib `credentials[key]` ga uzatadi) |

**`select` misoli:**
```go
{
    Type:  "select",
    Key:   "units",
    Label: "Birlik",
    Options: []botmodule.SelectOption{
        {Value: "metric", Label: "Selsiy (°C)"},
        {Value: "imperial", Label: "Farengeyt (°F)"},
    },
}
// handler: c.String("units") → "metric" yoki "imperial"
```

**`credential` misoli:**
```go
{Type: "credential", Key: "api_cred", Label: "API kalit", CredentialType: "openai"}
// handler: cred, ok := c.Credential("api_cred")
```

**`switch` / `checkbox` o'qish:** handler'da `c.Data["save"].(bool)` (helper yo'q,
to'g'ridan-to'g'ri map'dan).

### Constructor'da mavjud, lekin SDK `Field` qo'shimcha props talab qiladigan type'lar

Bular constructor renderlaydi, ammo ular qo'shimcha manifest xossalarini talab
qiladi (`itemFields`, `columnFields`, `language`, `resource`, `textFallback`, ...)
— bu xossalar hozirgi SDK `Field` struct'ida **yo'q**. Shu sabab modul yozayotganda
yuqoridagi 7 ta standart type bilan cheklaning. To'liqlik uchun ro'yxat:

`description` (statik matn — `textFallback` kerak), `code-editor` (`language`),
`array-editor` (`itemFields`), `nested-rows` (`columnFields`), `collection-select`,
`condition-list`, `cron-editor`, `trigger-config`, `dynamic-params`, `tool-params`,
`dynamic_select`/`dynamic_inputs` (`resource`), `action-recipients`,
`action-keyboard`, `collection-binding`, `richTextMessage`, `mediaMessage`,
`stringArray`, `widget`, `state-type-select`.

> Agar shulardan biri kerak bo'lsa — SDK `Field` struct'iga maydon qo'shilishi kerak;
> Botspace jamoasiga ayting.

---

## 7. `visibleWhen` — shartli ko'rsatish

Field faqat boshqa field ma'lum qiymatga teng bo'lganda ko'rinadi.

```go
{Type: "switch", Key: "save", Label: "Natijani saqlash", Optional: true},
{
    Type:        "text",
    Key:         "state_key",
    Label:       "State kaliti",
    Optional:    true,
    VisibleWhen: &botmodule.VisibleWhen{Key: "save", Equals: true},
},
```

- `Key` — kuzatiladigan field kaliti.
- `Equals` — `data[Key] == Equals` bo'lganda field ko'rinadi (`any`: `true`, `"foo"`, `3`...).
- `Optional:true` + `VisibleWhen` → field shart bajarilganda **avtomatik** ochiladi/yashirinadi.

> Constructor `notEquals` va `oneOf` ham qo'llab-quvvatlaydi, lekin SDK `VisibleWhen`
> struct'i hozir faqat `Equals` beradi.

---

## 8. Ikon nomlari (`Icon`)

`Icon` qiymati ushbu **registry kalitlaridan biri** bo'lishi kerak. Noma'lum nom →
avtomatik **`sparkles`** (xato emas, neytral ikon). Quyida hammasi (lucide-react):

**Triggerlar:** `keyboard` `clock` `message` `mouse-pointer` `reply` `globe`
`credit-card` `badge-check` `user-plus`

**Yuborish / media:** `arrow-right` `photo` `video` `music` `file` `clapperboard`
`microphone` `location` `contact` `gallery` `payment`

**Xabar amallari:** `edit` `forward` `admin` `copy` `pin` `unpin` `trash` `typing`
`download` `link`

**Flow control:** `split` `repeat` `timer` `shuffle` `loop` `corner-up-left` `form` `users`

**Ma'lumot / state:** `database` `collection` `variable`

**Integratsiya:** `http` `plugin` `door` `callback-answer` `table` `folder`

**Foydalanuvchi:** `user-check` `user-x`

**AI:** `brain-circuit` `settings-2` `wrench` `code-2` `knowledge` `ai-collection` `datetime`

**Boshqa:** `code` `key` `credential` `zap` `plug` `truck`

**Emoji ikonlar:** `sticker` (🎭) `dice` (🎲) `poll` (📊)

> Modullar uchun foydali: `zap`, `globe`, `http`, `database`, `key`, `brain-circuit`,
> `table`, `folder`, `arrow-right`, `settings-2`.

---

## 9. Rang token'lari (`Color`)

`Color` (= `colorToken`) ushbu **registry kalitlaridan biri** bo'lishi kerak.
Noma'lum token → canvas'da **kulrang** (`bg-zinc-600`), sidebar'da rangsiz. Hammasi:

| Guruh | Token'lar |
|---|---|
| Trigger | `trigger-blue` |
| Action | `action-amber` `action-purple` `action-red` `action-pink` `action-gray` `action-fuchsia` `action-violet` `action-emerald` `action-teal` |
| Xabar amallari | `message-op-slate` `message-op-teal` `message-op-slate-light` `message-op-red` `message-op-gray` `message-op-indigo` |
| Custom | `custom-yellow` |
| Integratsiya | `integration-indigo` `integration-green` `integration-sky` |
| Flow | `flow-violet` `flow-indigo` `flow-purple` `flow-amber` `flow-yellow` `flow-green` |
| AI | `ai-violet` `ai-indigo` `ai-sky` `ai-teal` `ai-emerald` `ai-rose` |

> Modul uchun tavsiya: action node'larga `integration-sky` / `integration-green` /
> `action-violet`; trigger node'larga `trigger-blue`; AI node'larga `ai-violet`.

---

## 10. `ExecuteCtx` / `Result` — action handler

```go
Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
    name := c.String("name")
    count := c.Int("count")
    cred, ok := c.Credential("api_cred")
    _ = count; _ = cred; _ = ok
    return botmodule.Result{
        ContextUpdates: map[string]any{"greeting": "Salom, " + name},
        ExitOutput:     "", // ixtiyoriy
    }
}
```

**`ExecuteCtx`:**

| Maydon / metod | Tur | Tavsif |
|---|---|---|
| `c.Type` | `string` | Node turi (`"demo.Echo"`) |
| `c.Data` | `map[string]any` | Field qiymatlari — engine `{{...}}` ni allaqachon resolve qilgan |
| `c.Context` | `map[string]any` | Flow konteksti (state) |
| `c.ChatID` | `int64` | Telegram chat ID |
| `c.Credentials` | `map[string]*Credential` | Decrypted credential'lar (field key bo'yicha) |
| `c.String(key)` | `string` | `data[key]` → string, topilmasa `""`. Non-string'ni `fmt.Sprintf("%v")` qiladi |
| `c.Int(key)` | `int64` | `data[key]` → int64 (`float64`/`int`/`int64`), topilmasa `0` |
| `c.Credential(key)` | `(*Credential, bool)` | Credential, topilmasa `(nil, false)` |

**`Result`:**

| Maydon | Tur | Tavsif |
|---|---|---|
| `ContextUpdates` | `map[string]any` | Flow state'iga qo'shiladigan kalitlar. Flow'da `{{key}}` orqali ishlatiladi. `nil` → bo'sh map |
| `ExitOutput` | `string` | Ixtiyoriy: nomli chiqish edge'iga yo'naltirish (masalan `"success"` / `"error"`) |

Xato bo'lsa: handler'da `panic` qilmang — `ContextUpdates`'ga error qo'ying va
`ExitOutput: "error"` qaytaring (yoki shunga o'xshash konvensiya). Noma'lum node
type uchun engine `node.execute` `-32601` ni o'zi qaytaradi.

---

## 11. `TriggerCtx` / `MatchResult` — trigger handler

```go
Match: func(c *botmodule.TriggerCtx) botmodule.MatchResult {
    if c.MessageText() == c.String("keyword") {
        return botmodule.MatchResult{
            Matched:        true,
            ContextUpdates: map[string]any{"matched_at": "now"},
        }
    }
    return botmodule.MatchResult{Matched: false}
}
```

**`TriggerCtx`:**

| Maydon / metod | Tur | Tavsif |
|---|---|---|
| `c.Type` | `string` | Node turi |
| `c.Data` | `map[string]any` | Field qiymatlari |
| `c.Update` | `map[string]any` | Telegram Update konverti (xom JSON map) |
| `c.Context` | `map[string]any` | Flow konteksti |
| `c.MessageText()` | `string` | `update.message.text` |
| `c.CallbackData()` | `string` | `update.callback_query.data` |
| `c.String(key)` | `string` | `data[key]` → string |

**`MatchResult`:**

| Maydon | Tur | Tavsif |
|---|---|---|
| `Matched` | `bool` | `true` → bu trigger ishlaydi, flow boshlanadi |
| `ContextUpdates` | `map[string]any` | Ixtiyoriy: flow boshlanganda state'ga qo'shiladi |

> Agar node'da `Match` o'rnatilmagan bo'lsa, engine `trigger.match` ga
> avtomatik `{matched:false}` oladi (xato emas).

---

## 12. `Credential` — credential ishlatish

Engine credential'ni decrypt qilib `node.execute` paytida `credentials[fieldKey]` da uzatadi.

```go
type Credential struct {
    TypeKey string            // credential turi: "openai", "http_header", "telegram", ...
    Mode    string            // bearer | apikey | basic | header | oauth2 | none
    Data    map[string]string // decrypted: api_key, token, username, password, ...
}
```

```go
cred, ok := c.Credential("api_cred")
if !ok {
    return botmodule.Result{ContextUpdates: map[string]any{"error": "credential tanlanmagan"}}
}
switch cred.Mode {
case "bearer":
    auth := "Bearer " + cred.Data["token"]
    _ = auth
case "apikey":
    key := cred.Data["api_key"]
    _ = key
case "basic":
    u, p := cred.Data["username"], cred.Data["password"]
    _ = u; _ = p
}
```

Manifestda credential field'i: `{Type: "credential", Key: "api_cred", CredentialType: "openai"}`.
`CredentialType` — foydalanuvchiga faqat shu turdagi credential'larni ko'rsatadi.

---

## 13. Dinamik qiymatlar va state

- **`{{...}}` engine tomonida resolve qilinadi.** Foydalanuvchi field'ga
  `{{message.text}}` yoki `{{state.user_name}}` yozsa, modul `c.Data` ni olganda
  ular allaqachon haqiqiy qiymatga aylangan bo'ladi. Modul `{{...}}` ni o'zi
  parse qilmaydi.
- **`Defaults`** — node birinchi qo'yilganda field'larga tushadigan boshlang'ich
  qiymatlar (`{{message.text}}` kabi shablon ham bo'lishi mumkin).
- **`ProducesState`** — bu node qaytaradigan state kalitlari ro'yxati. UI'da
  keyingi node'larda `{{key}}` autocomplete uchun maslahat. Haqiqiy yozish
  `Result.ContextUpdates` orqali bo'ladi — `ProducesState` faqat ko'rsatma.
- **`ContextUpdates`** dagi kalitlar flow state'iga qo'shiladi va keyingi
  node'larda `{{key}}` bilan ishlatiladi.

---

## 14. `describe()` avtomatik generatsiyasi

Siz `Node` ni soddа to'ldirasiz — SDK to'liq constructor manifestini o'zi quradi:

- **`status: "runtime"`** — har doim (runtime modul node'i).
- **`size.width`** — `Width` yoki `200`.
- **`sidebar`** — `{enabled:true, groupId:Category, sortOrder, elementType:Type}`.
- **`handles`** — action: `[target-default, source-default]`; trigger: faqat `[source-default]`.
- **`category`** — bo'sh bo'lsa trigger→`triggers`, aks holda `integrations`.
- **`content`** — `nil` bo'lsa `[]` ga aylantiriladi.
- **`titleFallback` / `descriptionFallback` / `iconName` / `colorToken`** — `Title`/`Description`/`Icon`/`Color`.

`describe()` chiqishi (qisqartirilgan):
```json
{
  "module": {"id": "demo", "name": "Demo Module", "version": "0.1.0"},
  "nodes": [{
    "type": "demo.Echo", "status": "runtime", "category": "integrations",
    "titleFallback": "Echo", "iconName": "arrow-right", "colorToken": "integration-sky",
    "size": {"width": 200},
    "sidebar": {"enabled": true, "groupId": "integrations", "sortOrder": 100, "elementType": "demo.Echo"},
    "handles": [{"preset": "target-default"}, {"preset": "source-default"}],
    "content": [{"type": "text", "key": "input", "label": "Matn"}],
    "defaults": {"input": "{{message.text}}"},
    "producesState": ["echo_output"],
    "trigger": false
  }]
}
```

---

## 15. JSON-RPC 2.0 kontrakt — simdagi format

Hamma chaqiruv: `POST /rpc`, `Content-Type: application/json`,
`Authorization: Bearer <MODULE_AUTH_TOKEN>` (token o'rnatilgan bo'lsa).

### `describe` — node'lar ro'yxati
```json
→ {"jsonrpc":"2.0","method":"describe","id":1}
← {"jsonrpc":"2.0","id":1,"result":{"module":{...},"nodes":[...]}}
```

### `docs` — markdown hujjat
```json
→ {"jsonrpc":"2.0","method":"docs","id":1}
← {"jsonrpc":"2.0","id":1,"result":{"markdown":"# Demo Module\n..."}}
```

### `node.execute` — action bajarish
```json
→ {"jsonrpc":"2.0","method":"node.execute","id":1,"params":{
     "type":"demo.Echo",
     "data":{"input":"salom"},
     "context":{"user_name":"Ali"},
     "chat_id":12345,
     "credentials":{"api_cred":{"type_key":"openai","mode":"bearer","data":{"token":"..."}}}
   }}
← {"jsonrpc":"2.0","id":1,"result":{"context_updates":{"echo_output":"salom"}}}
```
Noma'lum type → `error {code:-32601, message:"unknown node type: ..."}`.

### `trigger.match` — event-trigger tekshirish
```json
→ {"jsonrpc":"2.0","method":"trigger.match","id":1,"params":{
     "type":"demo.OnKeyword",
     "data":{"keyword":"salom"},
     "update":{"message":{"text":"salom dunyo"}},
     "context":{}
   }}
← {"jsonrpc":"2.0","id":1,"result":{"matched":true}}
```

### Xato kodlari
| Kod | Ma'no |
|---|---|
| `-32700` | parse error (noto'g'ri JSON) |
| `-32601` | method not found / unknown node type |
| `-32602` | invalid params |
| `-32001` | unauthorized (Bearer noto'g'ri) |

---

## 16. Environment o'zgaruvchilari

| Env | Tavsif |
|---|---|
| `PORT` | Tinglanadigan port (`Serve("")` da ishlatiladi). Default `8100` |
| `MODULE_AUTH_TOKEN` | O'rnatilsa, har `/rpc` so'rovda `Authorization: Bearer <token>` talab qilinadi. Bo'sh → auth yo'q. Platforma deploy'da bu token'ni o'zi beradi |

---

## 17. `module.yaml` — to'liq

Modul repo ildizida. Platforma uni o'qib build/run qiladi. `source` — yo `github`
(platforma klonlab image quradi) **yoki** `endpoint` (self-hosted, build yo'q).

```yaml
apiVersion: botmother.module/v1

module:
  id: mymodule              # global unique slug (= node namespace: mymodule.X)
  name: My Module
  icon: zap                 # ixtiyoriy
  description: Modul tavsifi # ixtiyoriy

source:
  # Variant A — platforma build qiladi:
  github: https://github.com/BotSpace/module-template-go
  # Variant B — o'zingiz host qilgansiz (build yo'q):
  # endpoint: https://my-server.com

runtime:                    # faqat source.github uchun
  dockerfile: Dockerfile
  port: 8100
```

---

## 18. Test

`ServeHandler()` `httptest` bilan ishlatiladi (server ko'tarmasdan):

```go
func TestExecute(t *testing.T) {
    m := botmodule.New("demo", "Demo")
    m.AddNode(botmodule.Node{
        Type: "demo.Echo",
        Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
            return botmodule.Result{ContextUpdates: map[string]any{"out": c.String("in")}}
        },
    })
    srv := httptest.NewServer(m.ServeHandler())
    defer srv.Close()

    body := `{"jsonrpc":"2.0","method":"node.execute","id":1,"params":{"type":"demo.Echo","data":{"in":"x"}}}`
    resp, _ := http.Post(srv.URL+"/rpc", "application/json", strings.NewReader(body))
    // resp.Body → result.context_updates.out == "x"
}
```

---

## 19. Yangi modul yozish — checklist

1. `New(id, name)` — `id` global unique (node namespace).
2. Har bir imkoniyat uchun `AddNode` — `Type` = `id.NodeName`.
3. Field'larni 6-bo'limdagi **standart 7 type** bilan tuzing.
4. `Icon` (8-bo'lim) va `Color` (9-bo'lim) registry kalitlaridan tanlang.
5. Action → `Execute` (`ContextUpdates` qaytaring); trigger → `Trigger:true` +
   `TriggerMode:"event-match"` + `Match`.
6. Node turi `id.NodeName` namespace bilan (kolliziya yo'q).
7. `m.Docs` ga markdown yozing (`docs` RPC).
8. Lokal sinang: `go run .` → `curl localhost:8100/rpc -d '{"jsonrpc":"2.0","method":"describe","id":1}'`.
9. `module.yaml` to'ldiring (17-bo'lim) va GitHub'ga push qiling.

---

## Litsenziya

MIT.
