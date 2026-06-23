package botmodule

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// -----------------------------------------------------------------------------
// Public types
// -----------------------------------------------------------------------------

// Field — node manifest'idagi bitta maydon (content[]).
type Field struct {
	Type           string         `json:"type"`
	Key            string         `json:"key"`
	Label          string         `json:"label,omitempty"`
	Placeholder    string         `json:"placeholder,omitempty"`
	HelpText       string         `json:"helpText,omitempty"`
	Optional       bool           `json:"optional,omitempty"`
	CredentialType string         `json:"credentialType,omitempty"`
	Options        []SelectOption `json:"options,omitempty"`
	VisibleWhen    *VisibleWhen   `json:"visibleWhen,omitempty"`
}

// SelectOption — select field uchun tanlov elementi.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// VisibleWhen — shartli ko'rsatish ({key, equals} juftligi).
type VisibleWhen struct {
	Key    string `json:"key"`
	Equals any    `json:"equals"`
}

// Credential — engine yuborgan decrypted credential ma'lumoti.
type Credential struct {
	TypeKey string            `json:"type_key"`
	Mode    string            `json:"mode"`
	Data    map[string]string `json:"data"`
}

// CredentialField — credential type'ining bitta input maydoni.
type CredentialField struct {
	Name        string         `json:"name"`              // data kaliti (masalan "token", "base_url")
	Label       string         `json:"label,omitempty"`   // ko'rinadigan nom
	Type        string         `json:"type"`              // text|password|number|textarea|select|url|switch
	Options     []SelectOption `json:"options,omitempty"` // select uchun
	Required    bool           `json:"required,omitempty"`
	Secret      bool           `json:"secret,omitempty"` // API javobida maskalanadi
	Placeholder string         `json:"placeholder,omitempty"`
	Default     any            `json:"default,omitempty"`
}

// CredentialType — modul e'lon qiladigan credential turi. Foydalanuvchi shu
// turdan credential yaratadi; engine uni decrypt qilib node.execute'ga uzatadi.
type CredentialType struct {
	Key    string            `json:"key"`   // global unique (masalan "weather.apikey")
	Label  string            `json:"label,omitempty"`
	Icon   string            `json:"icon,omitempty"`  // lucide nomi
	Color  string            `json:"color,omitempty"` // hex (#3B82F6)
	Mode   string            `json:"mode"`            // bearer|apikey|basic|header|none
	Fields []CredentialField `json:"fields"`
}

// Result — node.execute qaytaradigan natija.
type Result struct {
	ContextUpdates map[string]any `json:"context_updates"`
	ExitOutput     string         `json:"exit_output,omitempty"` // Node.Outputs dagi Name (qaysi chiqishga ketsin)
	// Error — bo'sh bo'lmasa, bu SURFACE qilinadigan xato: platforma uni debug
	// error ro'yxati + alert'da ko'rsatadi va node'ni qizil (failed) qiladi, flow
	// to'xtaydi. ContextUpdates baribir qo'llanadi (xato detali state'da qoladi).
	Error string `json:"error,omitempty"`
}

// Output — node'ning nomli chiqish edge'i. Handle id "output-<Name>" bo'ladi;
// Result.ExitOutput = Name bo'lsa engine shu edge'ga yo'naltiradi.
type Output struct {
	Name    string `json:"name"`              // exit kaliti (masalan "found", "not_found")
	Label   string `json:"label,omitempty"`   // UI'da ko'rinadigan nom
	Variant string `json:"variant,omitempty"` // default|success|danger|warning|accent (rang)
}

// MatchResult — trigger.match qaytaradigan natija.
type MatchResult struct {
	Matched        bool           `json:"matched"`
	ContextUpdates map[string]any `json:"context_updates,omitempty"`
}

// -----------------------------------------------------------------------------
// Context types — handler'ga uzatiladigan kontekst
// -----------------------------------------------------------------------------

// ExecuteCtx — node.execute handler'iga uzatiladigan parametrlar.
type ExecuteCtx struct {
	Type        string
	Data        map[string]any
	Context     map[string]any
	ChatID      int64
	Credentials map[string]*Credential
}

// String — data ichidan string qiymat oladi. Topilmasa "".
func (c *ExecuteCtx) String(key string) string {
	v, ok := c.Data[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", s)
	}
}

// Int — data ichidan int64 qiymat oladi. Topilmasa 0.
func (c *ExecuteCtx) Int(key string) int64 {
	v, ok := c.Data[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	default:
		return 0
	}
}

// Credential — credentials ichidan Credential oladi. Topilmasa (nil, false).
func (c *ExecuteCtx) Credential(key string) (*Credential, bool) {
	if c.Credentials == nil {
		return nil, false
	}
	cred, ok := c.Credentials[key]
	if !ok || cred == nil {
		return nil, false
	}
	return cred, true
}

// TriggerCtx — trigger.match handler'iga uzatiladigan parametrlar.
type TriggerCtx struct {
	Type    string
	Data    map[string]any
	Update  map[string]any
	Context map[string]any
}

// MessageText — update.message.text ni qaytaradi. Telegram Update konverti.
func (c *TriggerCtx) MessageText() string {
	msg, ok := c.Update["message"].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := msg["text"].(string)
	return text
}

// CallbackData — update.callback_query.data ni qaytaradi.
func (c *TriggerCtx) CallbackData() string {
	cq, ok := c.Update["callback_query"].(map[string]any)
	if !ok {
		return ""
	}
	data, _ := cq["data"].(string)
	return data
}

// String — data ichidan string qiymat oladi. Topilmasa "".
func (c *TriggerCtx) String(key string) string {
	v, ok := c.Data[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", s)
	}
}

// -----------------------------------------------------------------------------
// Node definition
// -----------------------------------------------------------------------------

// Node — bitta node turi: manifest ma'lumoti + handler funksiyalari.
type Node struct {
	// Manifest maydonlari.
	Type          string  // MAJBURIY: "moduleId.NodeName" format
	Title         string  // Sidebar/canvas sarlavhasi
	Description   string  // Qisqa tavsif
	Category      string  // "triggers" yoki boshqa (integrations, ...)
	Icon          string  // lucide-react ikon nomi (masalan "sparkles")
	Color         string  // blue|violet|emerald|amber|...
	Width         int     // Node kengligi (default 200)
	SortOrder     int     // Sidebar tartib raqami
	Content       []Field // Form maydonlari
	Defaults      map[string]any
	ProducesState []string // UI autocomplete uchun statik maslahat

	// Outputs — node'ning NOMLI chiqish edge'lari (bir nechta tarmoq). Har biri
	// uchun o'ng tomonda "output-<Name>" handle chiqadi. Execute'da Result.ExitOutput
	// shu Name'lardan birini qaytarsa, engine o'sha edge'ga yo'naltiradi. Bo'sh
	// bo'lsa — bitta oddiy chiqish (default).
	Outputs []Output

	// Trigger-specific.
	Trigger     bool   // true = trigger node
	TriggerMode string // "event-match" yoki ""
	Global      bool   // true = global trigger default ON (har qanday holatda ishlaydi, joriy flowni to'xtatadi)

	// Handler funksiyalari (runtime).
	Execute func(c *ExecuteCtx) Result
	Match   func(c *TriggerCtx) MatchResult
}

// -----------------------------------------------------------------------------
// Module
// -----------------------------------------------------------------------------

// Module — modul registratori va HTTP serveri.
type Module struct {
	ID      string
	Name    string
	Version string
	Docs    string

	nodes           []*Node
	credentialTypes []CredentialType
}

// New — yangi modul yaratadi.
func New(id, name string) *Module {
	return &Module{
		ID:      id,
		Name:    name,
		Version: "0.1.0",
	}
}

// AddNode — modulga node qo'shadi. Type majburiy va "moduleId.NodeName" formatida bo'lishi shart.
func (m *Module) AddNode(n Node) {
	m.nodes = append(m.nodes, &n)
}

// AddCredentialType — modul o'z credential turini e'lon qiladi. describe() uni
// "credentialTypes" ro'yxatida qaytaradi; platforma ro'yxatga oladi va foydalanuvchi
// shu turdan credential yaratadi. Key global unique bo'lsin (masalan "weather.apikey").
func (m *Module) AddCredentialType(ct CredentialType) {
	m.credentialTypes = append(m.credentialTypes, ct)
}

// -----------------------------------------------------------------------------
// describe() — manifest generatsiyasi
// -----------------------------------------------------------------------------

// nodeManifest — describe() chiqishi uchun JSON struct.
type nodeManifest struct {
	Type                string         `json:"type"`
	Status              string         `json:"status"`
	Category            string         `json:"category"`
	TitleFallback       string         `json:"titleFallback"`
	DescriptionFallback string         `json:"descriptionFallback,omitempty"`
	IconName            string         `json:"iconName,omitempty"`
	ColorToken          string         `json:"colorToken,omitempty"`
	Size                map[string]int `json:"size"`
	Sidebar             sidebarDef     `json:"sidebar"`
	Handles             []handleDef    `json:"handles"`
	Content             []Field        `json:"content"`
	Defaults            map[string]any `json:"defaults,omitempty"`
	ProducesState       []string       `json:"producesState,omitempty"`
	Trigger             bool           `json:"trigger"`
	TriggerMode         string         `json:"triggerMode,omitempty"`
}

type sidebarDef struct {
	Enabled     bool   `json:"enabled"`
	GroupID     string `json:"groupId"`
	SortOrder   int    `json:"sortOrder"`
	ElementType string `json:"elementType"`
}

type handleDef struct {
	Preset       string `json:"preset,omitempty"`
	Kind         string `json:"kind,omitempty"`         // target|source
	Side         string `json:"side,omitempty"`         // left|right
	ID           string `json:"id,omitempty"`           // handle id (output-<name>)
	OffsetClass  string `json:"offsetClass,omitempty"`  // vertikal joylashuv (translate-y-[..%])
	StyleVariant string `json:"styleVariant,omitempty"` // default|success|danger|warning|accent
	Label        string `json:"label,omitempty"`
}

// outputOffsetClass — n ta chiqishni markaz atrofida vertikal teng taqsimlaydi.
func outputOffsetClass(i, n int) string {
	if n <= 1 {
		return ""
	}
	off := (float64(i) - float64(n-1)/2.0) * 170.0
	return fmt.Sprintf("translate-y-[%d%%]", int(off))
}

func (m *Module) buildManifests() []nodeManifest {
	out := make([]nodeManifest, 0, len(m.nodes))
	for i, n := range m.nodes {
		width := n.Width
		if width == 0 {
			width = 200
		}
		sortOrder := n.SortOrder
		if sortOrder == 0 {
			sortOrder = 100 + i
		}
		category := n.Category
		if category == "" {
			if n.Trigger {
				category = "triggers"
			} else {
				category = "integrations"
			}
		}

		var handles []handleDef
		switch {
		case n.Trigger:
			handles = []handleDef{{Preset: "source-default"}}
		case len(n.Outputs) > 0:
			// Bir nechta nomli chiqish: chap target + har output uchun o'ng source.
			handles = []handleDef{{Kind: "target", Side: "left", ID: "target-handler"}}
			for j, o := range n.Outputs {
				variant := o.Variant
				if variant == "" {
					variant = "default"
				}
				handles = append(handles, handleDef{
					Kind:         "source",
					Side:         "right",
					ID:           "output-" + o.Name,
					OffsetClass:  outputOffsetClass(j, len(n.Outputs)),
					StyleVariant: variant,
					Label:        o.Label,
				})
			}
		default:
			handles = []handleDef{{Preset: "target-default"}, {Preset: "source-default"}}
		}

		man := nodeManifest{
			Type:                n.Type,
			Status:              "runtime",
			Category:            category,
			TitleFallback:       n.Title,
			DescriptionFallback: n.Description,
			IconName:            n.Icon,
			ColorToken:          n.Color,
			Size:                map[string]int{"width": width},
			Sidebar: sidebarDef{
				Enabled:     true,
				GroupID:     category,
				SortOrder:   sortOrder,
				ElementType: n.Type,
			},
			Handles:       handles,
			Content:       n.Content,
			Defaults:      n.Defaults,
			ProducesState: n.ProducesState,
			Trigger:       n.Trigger,
			TriggerMode:   n.TriggerMode,
		}
		if man.Content == nil {
			man.Content = []Field{}
		}

		// Trigger node'larga "global" toggle qo'shamiz (built-in triggerlar kabi).
		// Yoqilsa, foydalanuvchi boshqa flow ichida (waiting) bo'lsa ham bu trigger
		// ishlaydi. Engine node.Data["global"]==true ni o'qiydi. Default = n.Global.
		if n.Trigger {
			hasGlobal := false
			for _, f := range man.Content {
				if f.Key == "global" {
					hasGlobal = true
					break
				}
			}
			if !hasGlobal {
				man.Content = append(man.Content, Field{
					Type:     "switch",
					Key:      "global",
					Label:    "Global trigger",
					HelpText: "Yoqilsa, foydalanuvchi boshqa flow ichida bo'lsa ham bu trigger ishlaydi (joriy flowni to'xtatib, yangisini boshlaydi).",
					Optional: true,
				})
			}
			if man.Defaults == nil {
				man.Defaults = map[string]any{}
			}
			if _, ok := man.Defaults["global"]; !ok {
				man.Defaults["global"] = n.Global
			}
		}

		out = append(out, man)
	}
	return out
}

// -----------------------------------------------------------------------------
// JSON-RPC 2.0 types
// -----------------------------------------------------------------------------

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      any             `json:"id"`
}

type rpcResponse struct {
	JSONRPC string  `json:"jsonrpc"`
	ID      any     `json:"id"`
	Result  any     `json:"result,omitempty"`
	Error   *rpcErr `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func okResp(id, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func errResp(id any, code int, msg string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcErr{Code: code, Message: msg}}
}

// -----------------------------------------------------------------------------
// RPC method params
// -----------------------------------------------------------------------------

type executeParams struct {
	Type        string                 `json:"type"`
	Data        map[string]any         `json:"data"`
	Context     map[string]any         `json:"context"`
	ChatID      int64                  `json:"chat_id"`
	Credentials map[string]*Credential `json:"credentials"`
}

type triggerParams struct {
	Type    string         `json:"type"`
	Data    map[string]any `json:"data"`
	Update  map[string]any `json:"update"`
	Context map[string]any `json:"context"`
}

// -----------------------------------------------------------------------------
// HTTP handlers
// -----------------------------------------------------------------------------

func (m *Module) handleRPC(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, errResp(nil, -32700, "read error"))
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, errResp(nil, -32700, "parse error"))
		return
	}

	var resp rpcResponse
	switch req.Method {
	case "describe":
		creds := m.credentialTypes
		if creds == nil {
			creds = []CredentialType{}
		}
		resp = okResp(req.ID, map[string]any{
			"module":          map[string]string{"id": m.ID, "name": m.Name, "version": m.Version},
			"nodes":           m.buildManifests(),
			"credentialTypes": creds,
		})

	case "docs":
		resp = okResp(req.ID, map[string]string{"markdown": m.Docs})

	case "node.execute":
		var p executeParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp = errResp(req.ID, -32602, "invalid params")
			break
		}
		node := m.findNode(p.Type)
		if node == nil || node.Execute == nil {
			resp = errResp(req.ID, -32601, fmt.Sprintf("unknown node type: %s", p.Type))
			break
		}
		if p.Data == nil {
			p.Data = map[string]any{}
		}
		result := node.Execute(&ExecuteCtx{
			Type:        p.Type,
			Data:        p.Data,
			Context:     p.Context,
			ChatID:      p.ChatID,
			Credentials: p.Credentials,
		})
		if result.ContextUpdates == nil {
			result.ContextUpdates = map[string]any{}
		}
		resp = okResp(req.ID, result)

	case "trigger.match":
		var p triggerParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp = errResp(req.ID, -32602, "invalid params")
			break
		}
		node := m.findNode(p.Type)
		if node == nil || node.Match == nil {
			resp = okResp(req.ID, MatchResult{Matched: false})
			break
		}
		if p.Data == nil {
			p.Data = map[string]any{}
		}
		if p.Update == nil {
			p.Update = map[string]any{}
		}
		result := node.Match(&TriggerCtx{
			Type:    p.Type,
			Data:    p.Data,
			Update:  p.Update,
			Context: p.Context,
		})
		resp = okResp(req.ID, result)

	default:
		resp = errResp(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}

	writeJSON(w, resp)
}

func (m *Module) findNode(nodeType string) *Node {
	for _, n := range m.nodes {
		if n.Type == nodeType {
			return n
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// -----------------------------------------------------------------------------
// Serve — HTTP server ishga tushiradi
// -----------------------------------------------------------------------------

// Serve — /rpc va /health endpointlarini ochadi va tinglaydi.
// addr bo'sh bo'lsa PORT env tekshiriladi, u ham bo'sh bo'lsa ":8100".
// Bearer autentifikatsiya: MODULE_AUTH_TOKEN env. Bo'sh bo'lsa tekshirilmaydi.
func (m *Module) Serve(addr string) {
	if addr == "" {
		if p := os.Getenv("PORT"); p != "" {
			addr = ":" + p
		} else {
			addr = ":8100"
		}
	}

	authToken := os.Getenv("MODULE_AUTH_TOKEN")

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"module":%q}`, m.ID)
	})

	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if authToken != "" {
			got := r.Header.Get("Authorization")
			if got != "Bearer "+authToken {
				writeJSON(w, errResp(nil, -32001, "unauthorized"))
				return
			}
		}
		m.handleRPC(w, r)
	})

	log.Printf("[%s] JSON-RPC 2.0 listening on %s (/rpc, /health)", m.ID, addr)

	srv := &http.Server{Addr: addr, Handler: mux}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[%s] server error: %v", m.ID, err)
	}
}

// ServeHandler — net/http Handler qaytaradi (test yoki custom server uchun).
// AUTH_TOKEN tekshiruvi kiritilmaydi — xohlasangiz o'zingiz handler'ga wraplang.
func (m *Module) ServeHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"module":%q}`, m.ID)
	})
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.handleRPC(w, r)
	})
	return mux
}
