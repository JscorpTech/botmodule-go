# Botmother Module Development Guide

> Bu hujjat **yangi tashqi modul** yaratish uchun to'liq qo'llanma. Uni o'qib AI
> agent yoki dasturchi noldan ishlaydigan modul qura oladi. Reference
> implementatsiya: shu repodagi `server.js`.

---

## 1. Modul nima va qanday ishlaydi

Botmother **engine** (Telegram bot flow ijro etuvchi) yangi node turlarini va
ularning logikasini **tashqi servis** (= modul) orqali oladi — engine kodiga
tegmasdan. Modul = alohida HTTP servis, o'z porti, **JSON-RPC 2.0** kontrakti.

```
[Constructor]  ←─ manifest (describe) ──┐
     │  (node sidebar + render)         │
     ▼                                  │
[Engine] ── node.execute / trigger.match (JSON-RPC) ──► [MODUL servis /rpc]
     ▲                                                      │
     └────────────── push trigger (HTTP) ──────────────────┘
```

- Engine noma'lum node turini ko'rsa va u modulга tegishli bo'lsa →
  `POST {endpoint}/rpc` ga `node.execute` yuboradi.
- Modul `describe()` orqali o'z node'larini **constructor manifest formatida**
  tasvirlaydi (UI ularni avtomatik renderlaydi).
- Node turlari **`moduleId.NodeName`** ko'rinishida namespace qilinadi
  (`demo.Echo`) — kolliziya bo'lmaydi.

---

## 2. Tez boshlash

Shu repodan ko'chirib oling va 4 ta narsani o'zgartiring:

1. **`module.yaml`** — `module.id` (global unique slug), `name`, `version`, `port`.
2. **`server.js`** — `MODULE`, `NODES` (manifest), `EXECUTORS` (logika),
   ixtiyoriy `TRIGGERS`, `DOCS`.
3. **`Dockerfile`** — odatda o'zgarmaydi (Node.js zero-dependency). Boshqa til
   ishlatsangiz o'z Dockerfile'ingizni yozing — kontrakt til-agnostik.
4. **`package.json`** — `name`.

Til muhim emas: modul faqat `/rpc` (POST, JSON-RPC) va `/health` (GET, 200)
endpointlarini bersa bo'ldi.

---

## 3. JSON-RPC 2.0 kontrakti

**Endpoint:** `POST {base}/rpc`
**Auth:** ixtiyoriy `Authorization: Bearer <MODULE_AUTH_TOKEN>` (env'dan).
**Envelope:** standart JSON-RPC 2.0 — `{jsonrpc:"2.0", method, params, id}` →
javob `{jsonrpc:"2.0", id, result}` yoki `{jsonrpc:"2.0", id, error:{code,message}}`.

Modul **4 ta metod**ni qo'llab-quvvatlaydi:

| Metod | Maqsad |
|---|---|
| `describe` | Node manifestlari (constructor uchun) |
| `node.execute` | Bitta node logikasi (action) |
| `trigger.match` | Event-match trigger tekshiruvi (ixtiyoriy) |
| `docs` | Modul hujjati (markdown) |

Health: `GET {base}/health` → `200 {"ok":true,"module":"<id>"}`.

### Engine yuboradigan env'lar
Shared pod'ga backend quyidagilarni beradi:
`PORT`, `MODULE_AUTH_TOKEN` (bo'sh bo'lsa auth tekshirilmaydi), `MODULE_SLUG`.
Pod barcha loyihalar uchun umumiy, shuning uchun `PROJECT_ID` berilmaydi —
loyiha konteksti har `node.execute` chaqirig'ida (`chat_id`, `context`,
`credentials`) keladi.

---

## 4. `describe()` — manifest

Parametrsiz. Qaytaradi:

```json
{
  "module": { "id": "demo", "name": "Demo Module", "version": "0.1.0" },
  "nodes": [ /* NodeManifest[] — pastga qarang */ ]
}
```

---

## 5. Node manifest formati

Har node — constructor `NodeManifest` obyekti. **Faqat standart field type'lar**
ishlatiladi → constructor kodiga tegmasdan avtomatik renderlanadi.

```js
{
  type: "demo.Echo",            // MAJBURIY: moduleId.NodeName namespace
  status: "runtime",            // doim "runtime"
  category: "integrations",     // "triggers" → trigger; aks holda action
  titleFallback: "Echo",        // sidebar/canvas sarlavhasi
  descriptionFallback: "Matnni o'zgaruvchiga yozadi",
  titleKey: "module.demo.echo.title",       // ixtiyoriy i18n kaliti
  descriptionKey: "module.demo.echo.desc",
  iconName: "sparkles",         // lucide-react ikon nomi (kebab/lower)
  colorToken: "blue",           // blue|violet|emerald|amber|... (rang tokeni)
  size: { width: 200 },         // standart node = 200; 300 katta ko'rinadi
  sidebar: {
    enabled: true,              // node sidebar ro'yxatida ko'rinsinmi
    groupId: "integrations",    // qaysi guruhга (faqat ko'rinish)
    sortOrder: 100,             // guruh ichidagi tartib
    elementType: "demo.Echo"    // = type
  },
  handles: [
    { preset: "target-default" },  // kiruvchi (yuqorida)
    { preset: "source-default" }   // chiquvchi (pastda)
  ],
  content: [ /* maydonlar — pastga qarang */ ],
  defaults: { input: "{{message.text}}" },  // yangi node default qiymatlari
  producesState: ["echo_output"],           // ixtiyoriy: UI autocomplete uchun
  trigger: false                            // trigger node bo'lsa true
}
```

### 5.1 Field (content) turlari

`content[]` ichidagi har maydon:

```js
{ type: "text", key: "input", label: "Matn", placeholder: "...", helpText: "..." }
```

**Ruxsat etilgan `type`lar** (boshqasi manifestни rad etadi):
`text`, `number`, `textarea`, `select`, `checkbox`, `switch`, `description`,
`widget`, `json`, `color`, `file`, `datetime`, `boolean`, `code`,
**`credential`** (7-bo'lim).

Umumiy field xossalari:

| Xossa | Tavsif |
|---|---|
| `type` | Yuqoridagilardan biri (MAJBURIY) |
| `key` | Data kaliti (MAJBURIY) — `data[key]` |
| `label` | Ko'rinadigan nom |
| `placeholder` | Input placeholder |
| `helpText` | Yordam matni |
| `optional` | `true` → "Qo'shimcha sozlamalar" bo'limiga tushadi |
| `visibleWhen` | `{ key, equals }` — shartli ko'rsatish |
| `options` | `select` uchun `[{value,label}]` |
| `credentialType` | `credential` uchun type filtri (masalan `"openai"`) |

`select` misoli:
```js
{ type: "select", key: "mode", label: "Rejim",
  options: [{value:"a",label:"A"},{value:"b",label:"B"}] }
```

Shartli maydon (faqat `save==true` bo'lganda ko'rinadi):
```js
{ type:"switch", key:"save", label:"Saqlash", optional:true },
{ type:"text", key:"state_key", label:"Kalit",
  visibleWhen:{ key:"save", equals:true }, optional:true }
```

---

## 6. `node.execute()` — node logikasi

Engine action node bajarilганда chaqiradi.

**Parametrlar:**
```json
{
  "type": "demo.Echo",
  "data": { "input": "salom" },          // field qiymatlari ({{...}} RESOLVE qilingan)
  "context": { "user_id": 123, "state_x": "..." },  // JSON-safe flow konteksti
  "chat_id": 123456789,
  "credentials": { /* 7-bo'lim — agar credential field bo'lsa */ }
}
```

**Qaytaradi:**
```json
{
  "context_updates": { "echo_output": "salom" },  // state'ga yoziladigan o'zgaruvchilar
  "exit_output": ""                               // ixtiyoriy: nomli chiqish edge'iga yo'naltirish
}
```

Xato bo'lsa JSON-RPC `error` qaytaring (flow to'xtaydi, node "error" bo'ladi).

Namuna:
```js
"demo.Echo": ({ data }) => ({
  context_updates: { echo_output: String(data.input ?? "") },
  exit_output: "",
}),
```

> **MUHIM:** action node turida "Node" so'zi shart EMAS — `demo.Echo` to'g'ri.
> (Engine modul node'larini namespace bo'yicha taniydi.)

---

## 7. Variable substitution

Engine `data` ichidagi **string** qiymatlarda `{{var}}` / `$var` ni
`node.execute` dan OLDIN resolve qiladi (boshqa node'lar kabi). Demak:

- `value: "{{message.text}}"` → modul haqiqiy matnni oladi.
- Nested: `{{collection.items.0.name}}`, `{{message.photo.-1.file_id}}`.

Modul tarafида qo'shimcha hech narsa qilish shart emas — tayyor qiymat keladi.

---

## 8. State'ga saqlash (eng muhim)

Flow **context** = state (Redis'da, updatelar orasида saqlanadi). Modul
`context_updates` qaytarganда engine uni context'ga qo'shadi → keyingi
node'lar `{{key}}` orqali o'qiydi, Variable inspector'да ko'rinadi.

**Statik kalit:**
```js
"demo.Echo": ({ data }) => ({ context_updates: { echo_output: data.input } })
```

**Dinamik kalit** (foydalanuvchi o'zgaruvchi NOMINI o'zi kiritadi):
```js
// manifest: variable_name (text) + value (text)
"demo.SetVariable": ({ data }) => {
  const name = String(data.variable_name ?? "").trim();
  if (!name) return { context_updates: {} };
  return { context_updates: { [name]: String(data.value ?? "") } };
}
```
Natija: `{{<variable_name>}}` keyingi node'lardа ishlatiladi.

`producesState` faqat UI autocomplete uchun statik maslahat — haqiqiy yozish
har doim runtime'da `context_updates` orqali. Dinamik nom uchun bo'sh qoldiring.

---

## 9. Credentials (sirlar)

Modul n8n-uslubidagi credential'larни ishlatishi mumkin. Manifestda
`credential` field so'raysiz — engine credential'ni **o'zи resolve qiladi** va
decrypted sirni `node.execute`ga uzatadi (modulga token sozlash shart emas).

**Manifest:**
```js
content: [
  { type: "credential", key: "api_credential", label: "Credential",
    credentialType: "openai" /* ixtiyoriy filtri */, required: true }
]
```

Constructor credential picker chiqaradi, tanlanganda `data.api_credential` = id.

**`node.execute` paytida engine qo'shadi:**
```json
"credentials": {
  "api_credential": {
    "type_key": "openai",
    "mode": "bearer",
    "data": { "api_key": "sk-..." }
  }
}
```

**Modul ishlatadi:**
```js
"demo.AuthHeader": ({ credentials }) => {
  const cred = credentials?.api_credential;
  if (!cred) return { context_updates: { auth_header: "" } };
  const d = cred.data || {};
  let header = cred.mode === "bearer" ? `Bearer ${d.token || d.api_key}` : (d.api_key || "");
  return { context_updates: { auth_header: header } };
}
```

`mode`lar: `bearer`, `apikey`, `basic`, `header`, `oauth2`, `none`.
Credential type'lari backend registry'sida kodda belgilangan (owner-global).

> Xavfsizlik: sirni to'liq flow context'iga oqizmang (demo `auth_header`ni
> maskalaydi). Faqat kerakli HTTP chaqiruvда ishlating.

---

## 10. Triggerlar

Trigger node = `category: "triggers"`, `trigger: true`. Ikki rejim:

### 10.1 Event-match (`triggerMode: "event-match"`)
Engine har Telegram update'да modulning `trigger.match` ni chaqiradi.

**Parametrlar:**
```json
{
  "type": "demo.OnKeyword",
  "data": { "keyword": "salom" },
  "update": { "message": { "text": "salom", "from": {...}, "chat": {...} } },
  "context": { ... }
}
```
> `update` Telegram **Update** konverti: `update.message.text`,
> `update.callback_query.data` va h.k.

**Qaytaradi:**
```json
{ "matched": true, "context_updates": { "matched_keyword": "salom" } }
```

Namuna:
```js
"demo.OnKeyword": ({ data, update }) => {
  const text = update?.message?.text || "";
  const kw = String(data.keyword ?? "").trim();
  const matched = kw !== "" && text.toLowerCase().includes(kw.toLowerCase());
  return { matched, context_updates: matched ? { matched_keyword: kw } : {} };
}
```

> Latency: har update'да tarmoq chaqirig'i. Iloji bo'lsa push'ni afzal ko'ring.
> Timeout 2s, xato → `matched:false` (graceful).

### 10.2 Push (modul → engine)
Modul tashqi hodisada flow'ni o'zi boshlaydi:
`POST {ENGINE_PUSH_URL}/module-trigger/{module}/{type}[/{chat_id}]`,
header `X-Internal-Token`, body `{chat_id, context, payload}`.

---

## 11. `docs()` — hujjat

Markdown qaytaradi; platforma uni saqlaydi va modul tavsifida ko'rsatadi.
```js
if (method === "docs") return reply({ markdown: DOCS });
```

---

## 12. `module.yaml` + Dockerfile

**`module.yaml`** (repo root) — modul deskriptori:
```yaml
apiVersion: botmother.module/v1
module:
  id: demo                 # global unique slug (= node namespace)
  name: Demo Module
  version: 0.1.0
  icon: sparkles           # lucide ikon
runtime:
  dockerfile: Dockerfile
  port: 8100               # /rpc va /health shu portda
  healthcheck: /health
contract:
  jsonrpc: "2.0"
  endpoint: /rpc
  methods: [describe, node.execute, trigger.match, docs]
provides:
  nodes:
    - { type: demo.Echo, trigger: false }
```

**Dockerfile** — non-root, port'ni expose qiling:
```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package.json server.js ./
ENV PORT=8100
EXPOSE 8100
USER node
CMD ["node", "server.js"]
```

---

## 13. Deploy

Modulni yozib bo'lgach, `module.yaml` orqali platforma uni o'zi build qiladi va
ishga tushiradi — siz faqat `source` ni belgilaysiz:
- `source.github` → platforma reponi klonlaydi, image quradi, ishga tushiradi.
- `source.endpoint` → modulni o'z serveringizda yuritasiz, platforma faqat ulanadi.

(Platforma ichki ishi — pod, registry, manifest cache — sizning vazifangiz emas.)

---

## 14. To'liq minimal namuna (noldan)

`weather` moduli — bitta action:

```js
const MODULE = { id: "weather", name: "Weather", version: "0.1.0" };
const NODES = [{
  type: "weather.Get", status: "runtime", category: "integrations",
  titleFallback: "Ob-havo", descriptionFallback: "Shahar ob-havosini oladi",
  iconName: "cloud", colorToken: "blue", size: { width: 200 },
  sidebar: { enabled: true, groupId: "integrations", sortOrder: 100, elementType: "weather.Get" },
  handles: [{ preset: "target-default" }, { preset: "source-default" }],
  content: [{ type: "text", key: "city", label: "Shahar", placeholder: "Tashkent" }],
  defaults: { city: "Tashkent" }, producesState: ["weather_text"], trigger: false,
}];
const EXECUTORS = {
  "weather.Get": ({ data }) => ({
    context_updates: { weather_text: `${data.city}: +25°C` }, exit_output: "",
  }),
};
```
+ `describe`/`node.execute`/`docs` dispatch (shu repodagi `server.js` naqshi).

---

## 15. Checklist va tuzoqlar

- [ ] Node turi **`moduleId.NodeName`** namespace bilan (kolliziya yo'q).
- [ ] Faqat **standart field type'lar** (12-bo'lim ro'yxati).
- [ ] `category: "triggers"` → trigger; aks holда action.
- [ ] `size.width` ni **200** qiling (standart node bilan bir xil; 300 katta).
- [ ] State'ga yozish **faqat** `context_updates` orqali.
- [ ] `{{...}}` ni modul resolve qilmaydi — engine qiladi (string data'да).
- [ ] Credential sirini context'ga **to'liq oqizmang**.
- [ ] `/health` 200 qaytarsin (aks holда pod Ready bo'lmaydi).
- [ ] Node turi **`moduleId.NodeName`** namespace bilan (kolliziya yo'q).

---

Reference: shu repodagi **`server.js`** (Echo, Upper, AuthHeader, SetVariable,
OnKeyword) — barcha naqshlarning ishlaydigan namunasi.
