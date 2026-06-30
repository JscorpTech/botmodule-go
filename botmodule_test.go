package botmodule_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	botmodule "github.com/BotSpace/botmodule-go"
)

func newTestModule() *botmodule.Module {
	m := botmodule.New("test", "Test Module")
	m.Version = "0.1.0"
	m.Docs = "# Test"

	m.AddNode(botmodule.Node{
		Type:     "test.Echo",
		Title:    "Echo",
		Category: "integrations",
		Icon:     "sparkles",
		Color:    "blue",
		Content: []botmodule.Field{
			{Type: "text", Key: "input", Label: "Matn"},
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
		Type:        "test.OnKeyword",
		Title:       "Kalit so'z",
		Category:    "triggers",
		Trigger:     true,
		TriggerMode: "event-match",
		Content: []botmodule.Field{
			{Type: "text", Key: "keyword", Label: "Kalit so'z"},
		},
		Match: func(c *botmodule.TriggerCtx) botmodule.MatchResult {
			text := c.MessageText()
			kw := c.String("keyword")
			matched := kw != "" && strings.Contains(strings.ToLower(text), strings.ToLower(kw))
			return botmodule.MatchResult{Matched: matched}
		},
	})

	return m
}

func rpcCall(t *testing.T, handler http.Handler, method string, params any) map[string]any {
	t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "method": method, "id": 1}
	if params != nil {
		body["params"] = params
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response parse error: %v — body: %s", err, rec.Body.String())
	}
	return resp
}

func TestDescribe(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "describe", nil)

	if resp["error"] != nil {
		t.Fatalf("describe error: %v", resp["error"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("result is not object")
	}
	mod, ok := result["module"].(map[string]any)
	if !ok {
		t.Fatal("module is not object")
	}
	if mod["id"] != "test" {
		t.Errorf("module.id = %v, want test", mod["id"])
	}

	nodes, ok := result["nodes"].([]any)
	if !ok {
		t.Fatal("nodes is not array")
	}
	if len(nodes) != 2 {
		t.Errorf("nodes count = %d, want 2", len(nodes))
	}

	// Birinchi node — Echo; manifest maydonlarini tekshir.
	echo, ok := nodes[0].(map[string]any)
	if !ok {
		t.Fatal("nodes[0] is not object")
	}
	if echo["type"] != "test.Echo" {
		t.Errorf("nodes[0].type = %v, want test.Echo", echo["type"])
	}
	if echo["status"] != "runtime" {
		t.Errorf("nodes[0].status = %v, want runtime", echo["status"])
	}
	if echo["trigger"] != false {
		t.Errorf("nodes[0].trigger = %v, want false", echo["trigger"])
	}
	size, _ := echo["size"].(map[string]any)
	if size["width"].(float64) != 200 {
		t.Errorf("nodes[0].size.width = %v, want 200", size["width"])
	}

	// Trigger node.
	trigger, ok := nodes[1].(map[string]any)
	if !ok {
		t.Fatal("nodes[1] is not object")
	}
	if trigger["trigger"] != true {
		t.Errorf("nodes[1].trigger = %v, want true", trigger["trigger"])
	}
	if trigger["triggerMode"] != "event-match" {
		t.Errorf("nodes[1].triggerMode = %v, want event-match", trigger["triggerMode"])
	}
}

func TestNodeExecute(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "node.execute", map[string]any{
		"type": "test.Echo",
		"data": map[string]any{"input": "salom dunyo"},
	})

	if resp["error"] != nil {
		t.Fatalf("node.execute error: %v", resp["error"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("result is not object")
	}
	updates, ok := result["context_updates"].(map[string]any)
	if !ok {
		t.Fatal("context_updates is not object")
	}
	if updates["echo_output"] != "salom dunyo" {
		t.Errorf("echo_output = %v, want salom dunyo", updates["echo_output"])
	}
}

func TestNodeExecuteUnknownType(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "node.execute", map[string]any{
		"type": "test.NoSuch",
		"data": map[string]any{},
	})
	if resp["error"] == nil {
		t.Fatal("expected error for unknown node type")
	}
	rpcError, _ := resp["error"].(map[string]any)
	if rpcError["code"].(float64) != -32601 {
		t.Errorf("error.code = %v, want -32601", rpcError["code"])
	}
}

func TestTriggerMatch(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "trigger.match", map[string]any{
		"type": "test.OnKeyword",
		"data": map[string]any{"keyword": "salom"},
		"update": map[string]any{
			"message": map[string]any{"text": "salom dunyo"},
		},
	})

	if resp["error"] != nil {
		t.Fatalf("trigger.match error: %v", resp["error"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("result is not object")
	}
	if result["matched"] != true {
		t.Errorf("matched = %v, want true", result["matched"])
	}
}

func TestTriggerMatchNoMatch(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "trigger.match", map[string]any{
		"type": "test.OnKeyword",
		"data": map[string]any{"keyword": "xayr"},
		"update": map[string]any{
			"message": map[string]any{"text": "salom dunyo"},
		},
	})

	result, _ := resp["result"].(map[string]any)
	if result["matched"] != false {
		t.Errorf("matched = %v, want false", result["matched"])
	}
}

func TestTriggerMatchUnknownType(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "trigger.match", map[string]any{
		"type": "test.NoSuchTrigger",
		"data": map[string]any{},
	})
	if resp["error"] != nil {
		t.Fatalf("expected no error for unknown trigger (graceful): %v", resp["error"])
	}
	result, _ := resp["result"].(map[string]any)
	if result["matched"] != false {
		t.Errorf("matched = %v, want false", result["matched"])
	}
}

func TestHealth(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health status = %d, want 200", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Errorf("health ok = %v, want true", body["ok"])
	}
	if body["module"] != "test" {
		t.Errorf("health module = %v, want test", body["module"])
	}
}

func TestDocs(t *testing.T) {
	m := newTestModule()
	h := m.ServeHandler()

	resp := rpcCall(t, h, "docs", nil)
	result, _ := resp["result"].(map[string]any)
	if result["markdown"] != "# Test" {
		t.Errorf("docs markdown = %v, want '# Test'", result["markdown"])
	}
}

// Trigger node'lariga avtomatik "global" switch field qo'shilishi va Global:true
// bo'lganda defaults.global==true bo'lishini tekshiradi.
func TestGlobalTriggerField(t *testing.T) {
	m := botmodule.New("g", "G")
	m.AddNode(botmodule.Node{
		Type: "g.OnEvent", Title: "Ev", Trigger: true, TriggerMode: "event-match",
		Global:  true,
		Content: []botmodule.Field{{Type: "text", Key: "kw"}},
		Match:   func(c *botmodule.TriggerCtx) botmodule.MatchResult { return botmodule.MatchResult{} },
	})
	// Action node global field OLMASLIGI kerak.
	m.AddNode(botmodule.Node{
		Type: "g.Act", Title: "Act",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result { return botmodule.Result{} },
	})

	resp := rpcCall(t, m.ServeHandler(), "describe", nil)
	result, _ := resp["result"].(map[string]any)
	nodes, _ := result["nodes"].([]any)

	for _, n := range nodes {
		node := n.(map[string]any)
		content, _ := node["content"].([]any)
		hasGlobal := false
		for _, f := range content {
			if f.(map[string]any)["key"] == "global" {
				hasGlobal = true
			}
		}
		switch node["type"] {
		case "g.OnEvent":
			if !hasGlobal {
				t.Error("trigger node'da 'global' field yo'q")
			}
			defaults, _ := node["defaults"].(map[string]any)
			if defaults["global"] != true {
				t.Errorf("Global:true uchun defaults.global = %v, want true", defaults["global"])
			}
		case "g.Act":
			if hasGlobal {
				t.Error("action node'ga 'global' field qo'shilmasligi kerak")
			}
		}
	}
}

// Modul e'lon qilgan credential type describe() chiqishida bo'lishini tekshiradi.
func TestCredentialTypes(t *testing.T) {
	m := botmodule.New("weather", "Weather")
	m.AddCredentialType(botmodule.CredentialType{
		Key:   "weather.apikey",
		Label: "Weather API",
		Mode:  "header",
		Fields: []botmodule.CredentialField{
			{Name: "token", Label: "Token", Type: "text", Required: true, Secret: true},
			{Name: "base_url", Label: "Base URL", Type: "text"},
			{Name: "model", Label: "Model", Type: "select", Options: []botmodule.SelectOption{
				{Value: "v1", Label: "V1"}, {Value: "v2", Label: "V2"},
			}},
		},
	})

	resp := rpcCall(t, m.ServeHandler(), "describe", nil)
	result, _ := resp["result"].(map[string]any)
	cts, _ := result["credentialTypes"].([]any)
	if len(cts) != 1 {
		t.Fatalf("credentialTypes len = %d, want 1", len(cts))
	}
	ct := cts[0].(map[string]any)
	if ct["key"] != "weather.apikey" || ct["mode"] != "header" {
		t.Errorf("credential type key/mode noto'g'ri: %v", ct)
	}
	fields, _ := ct["fields"].([]any)
	if len(fields) != 3 {
		t.Errorf("fields len = %d, want 3", len(fields))
	}
	model := fields[2].(map[string]any)
	if model["type"] != "select" {
		t.Errorf("3-field type = %v, want select", model["type"])
	}
	if opts, _ := model["options"].([]any); len(opts) != 2 {
		t.Errorf("select options len = %d, want 2", len(opts))
	}
}

// credentialTypes yo'q bo'lsa describe() bo'sh massiv qaytarishini tekshiradi.
func TestCredentialTypesEmpty(t *testing.T) {
	m := newTestModule()
	resp := rpcCall(t, m.ServeHandler(), "describe", nil)
	result, _ := resp["result"].(map[string]any)
	if cts, ok := result["credentialTypes"].([]any); !ok || len(cts) != 0 {
		t.Errorf("credentialTypes = %v, want []", result["credentialTypes"])
	}
}

// Result.Error node.execute javobida "error" maydoni sifatida chiqishini tekshiradi.
func TestResultError(t *testing.T) {
	m := botmodule.New("e", "E")
	m.AddNode(botmodule.Node{
		Type: "e.Fail",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			return botmodule.Result{
				ContextUpdates: map[string]any{"x": "1"},
				Error:          "API ishlamadi",
			}
		},
	})
	resp := rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{"type": "e.Fail", "data": map[string]any{}})
	result, _ := resp["result"].(map[string]any)
	if result["error"] != "API ishlamadi" {
		t.Errorf("result.error = %v, want 'API ishlamadi'", result["error"])
	}
	if cu, _ := result["context_updates"].(map[string]any); cu["x"] != "1" {
		t.Errorf("context_updates xato bo'lsa ham qo'llanishi kerak: %v", result["context_updates"])
	}
}

func TestResultAlerts(t *testing.T) {
	m := botmodule.New("a", "A")
	m.AddNode(botmodule.Node{
		Type: "a.Warn",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			return botmodule.Result{
				Alerts: []botmodule.Alert{{
					Level:   botmodule.AlertWarning,
					Message: "quota kam",
					Detail:  "12% qoldi",
				}},
			}
		},
	})
	resp := rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{"type": "a.Warn", "data": map[string]any{}})
	result, _ := resp["result"].(map[string]any)
	alerts, _ := result["alerts"].([]any)
	if len(alerts) != 1 {
		t.Fatalf("alerts len = %d, want 1", len(alerts))
	}
	alert, _ := alerts[0].(map[string]any)
	if alert["level"] != "warning" || alert["message"] != "quota kam" || alert["detail"] != "12% qoldi" {
		t.Errorf("unexpected alert: %v", alert)
	}
}

// Node.Outputs describe()'da har biri uchun "output-<name>" source handle berishini tekshiradi.
func TestNodeOutputs(t *testing.T) {
	m := botmodule.New("r", "R")
	m.AddNode(botmodule.Node{
		Type: "r.Route",
		Outputs: []botmodule.Output{
			{Name: "found", Label: "Topildi", Variant: "success"},
			{Name: "missing", Label: "Yo'q", Variant: "danger"},
		},
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result { return botmodule.Result{ExitOutput: "found"} },
	})
	resp := rpcCall(t, m.ServeHandler(), "describe", nil)
	result, _ := resp["result"].(map[string]any)
	nodes, _ := result["nodes"].([]any)
	node := nodes[0].(map[string]any)
	handles, _ := node["handles"].([]any)
	ids := map[string]bool{}
	for _, h := range handles {
		hm := h.(map[string]any)
		if id, ok := hm["id"].(string); ok {
			ids[id] = true
		}
	}
	for _, want := range []string{"target-handler", "output-found", "output-missing"} {
		if !ids[want] {
			t.Errorf("handle %q yo'q; handles=%v", want, ids)
		}
	}
}

// dynamic_select uchun options.load RPC: loader chaqirilib, params (dependsOn)
// va resource'ga qarab options qaytishini tekshiradi (kaskad).
func TestOptionsLoad(t *testing.T) {
	m := botmodule.New("g", "G")
	m.AddOptionsLoader("sheets", func(c *botmodule.OptionsCtx) []botmodule.SelectOption {
		doc := c.String("doc_id")
		if doc != "DOC1" {
			return nil
		}
		return []botmodule.SelectOption{{Value: "s1", Label: "Sheet 1"}, {Value: "s2", Label: "Sheet 2"}}
	})

	resp := rpcCall(t, m.ServeHandler(), "options.load", map[string]any{
		"resource": "sheets",
		"params":   map[string]any{"doc_id": "DOC1"},
	})
	result, _ := resp["result"].(map[string]any)
	opts, _ := result["options"].([]any)
	if len(opts) != 2 {
		t.Fatalf("options len = %d, want 2 (resp=%v)", len(opts), resp)
	}
	if first := opts[0].(map[string]any); first["value"] != "s1" {
		t.Errorf("first option = %v, want s1", first)
	}

	// Noma'lum resource → xato
	bad := rpcCall(t, m.ServeHandler(), "options.load", map[string]any{"resource": "nope"})
	if bad["error"] == nil {
		t.Error("noma'lum resource uchun error kutilgan edi")
	}
}

// ExecuteCtx.UploadFile/GetFile engine bergan file_api orqali ishlashini tekshiradi.
func TestFileAPI(t *testing.T) {
	var signedURL string
	// Soxta platforma fayl API: upload → {uuid}, metadata → {data.url}, signed → baytlar.
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost { // upload
			if r.Header.Get("Authorization") != "Bearer tok123" {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(`{"uuid":"file-xyz"}`))
			return
		}
		if r.URL.Path == "/file-xyz/" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":true,"data":{"url":"` + signedURL + `","file":"` + signedURL + `","uuid":"file-xyz"}}`))
			return
		}
		if r.URL.Path == "/signed/file-xyz" {
			w.Write([]byte("FILEBYTES"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fileSrv.Close()
	signedURL = fileSrv.URL + "/signed/file-xyz"

	var up, got, fileURL, downloadURL string
	m := botmodule.New("f", "F")
	m.AddNode(botmodule.Node{
		Type: "f.Files",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			id, _ := c.UploadFile("a.txt", []byte("hi"))
			up = id
			fileURL = c.FileURL("file-xyz")
			downloadURL, _ = c.FileDownloadURL("file-xyz")
			b, _ := c.GetFile("file-xyz")
			got = string(b)
			return botmodule.Result{ContextUpdates: map[string]any{"uuid": id}}
		},
	})

	rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{
		"type": "f.Files", "data": map[string]any{},
		"file_api": map[string]any{
			"upload_url": fileSrv.URL, "get_base": fileSrv.URL, "token": "tok123",
		},
	})
	if up != "file-xyz" {
		t.Errorf("UploadFile uuid = %q, want file-xyz", up)
	}
	if got != "FILEBYTES" {
		t.Errorf("GetFile = %q, want FILEBYTES", got)
	}
	if fileURL != fileSrv.URL+"/file-xyz/" {
		t.Errorf("FileURL = %q, want %q", fileURL, fileSrv.URL+"/file-xyz/")
	}
	if downloadURL != signedURL {
		t.Errorf("FileDownloadURL = %q, want %q", downloadURL, signedURL)
	}
}

// TestUploadFileWithTTL — ttlSeconds > 0 bo'lsa multipart'ga "ttl" field qo'shilishini,
// ttlSeconds <= 0 bo'lsa qo'shilmasligini tekshiradi.
func TestUploadFileWithTTL(t *testing.T) {
	t.Run("with ttl", func(t *testing.T) {
		var gotTTL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				gotTTL = r.FormValue("ttl")
			}
			w.Write([]byte(`{"uuid":"ttl-uuid"}`))
		}))
		defer srv.Close()

		var up string
		m := botmodule.New("f", "F")
		m.AddNode(botmodule.Node{
			Type: "f.TTL",
			Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
				up, _ = c.UploadFileWithTTL("a.txt", []byte("hi"), 3600)
				return botmodule.Result{}
			},
		})
		rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{
			"type": "f.TTL", "data": map[string]any{},
			"file_api": map[string]any{"upload_url": srv.URL, "token": "t"},
		})
		if up != "ttl-uuid" {
			t.Errorf("UploadFileWithTTL uuid = %q, want ttl-uuid", up)
		}
		if gotTTL != "3600" {
			t.Errorf("ttl form field = %q, want 3600", gotTTL)
		}
	})

	t.Run("zero ttl no field", func(t *testing.T) {
		var hasTTL bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				hasTTL = r.FormValue("ttl") != ""
			}
			w.Write([]byte(`{"uuid":"perm-uuid"}`))
		}))
		defer srv.Close()

		m := botmodule.New("f", "F")
		m.AddNode(botmodule.Node{
			Type: "f.Perm",
			Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
				c.UploadFileWithTTL("b.txt", []byte("x"), 0)
				return botmodule.Result{}
			},
		})
		rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{
			"type": "f.Perm", "data": map[string]any{},
			"file_api": map[string]any{"upload_url": srv.URL, "token": "t"},
		})
		if hasTTL {
			t.Error("ttlSeconds=0 bo'lsa 'ttl' form field bo'lmasligi kerak")
		}
	})
}

// TestUploadFileTrailingSlash — DRF APPEND_SLASH regressiyasi: upload_url trailing
// slash'siz bo'lsa server 301 redirect qiladi; SDK URL'ni normallashtirmasa Go
// client POST'ni GET'ga tushiradi (405). Fix: UploadFile trailing slash qo'shadi.
func TestUploadFileTrailingSlash(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// "/up" (slashsiz) → 301 "/up/" (DRF APPEND_SLASH xatti-harakati).
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		gotMethod = r.Method
		w.Write([]byte(`{"uuid":"ok"}`))
	}))
	defer srv.Close()

	var up string
	m := botmodule.New("f", "F")
	m.AddNode(botmodule.Node{
		Type: "f.Up",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			up, _ = c.UploadFile("a.txt", []byte("hi"))
			return botmodule.Result{}
		},
	})
	rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{
		"type": "f.Up", "data": map[string]any{},
		"file_api": map[string]any{"upload_url": srv.URL + "/up", "token": "t"},
	})
	if gotMethod != http.MethodPost {
		t.Errorf("server method = %q, want POST (redirect POST'ni GET'ga tushirmasin)", gotMethod)
	}
	if up != "ok" {
		t.Errorf("UploadFile uuid = %q, want ok", up)
	}
}

func TestUploadFilePreservesPostAcrossRedirect(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/up/" {
			http.Redirect(w, r, "/final/", http.StatusMovedPermanently)
			return
		}
		if r.URL.Path != "/final/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotMethod = r.Method
		w.Write([]byte(`{"uuid":"redirect-ok"}`))
	}))
	defer srv.Close()

	var up string
	m := botmodule.New("f", "F")
	m.AddNode(botmodule.Node{
		Type: "f.Up",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			up, _ = c.UploadFile("a.txt", []byte("hi"))
			return botmodule.Result{}
		},
	})
	rpcCall(t, m.ServeHandler(), "node.execute", map[string]any{
		"type": "f.Up", "data": map[string]any{},
		"file_api": map[string]any{"upload_url": srv.URL + "/up", "token": "t"},
	})
	if gotMethod != http.MethodPost {
		t.Errorf("redirected method = %q, want POST", gotMethod)
	}
	if up != "redirect-ok" {
		t.Errorf("UploadFile uuid = %q, want redirect-ok", up)
	}
}

func TestExecuteCtxHelpers(t *testing.T) {
	var capturedCtx *botmodule.ExecuteCtx
	m := botmodule.New("h", "H")
	m.AddNode(botmodule.Node{
		Type: "h.Cap",
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			capturedCtx = c
			return botmodule.Result{ContextUpdates: map[string]any{}}
		},
	})

	h := m.ServeHandler()
	rpcCall(t, h, "node.execute", map[string]any{
		"type": "h.Cap",
		"data": map[string]any{
			"name":  "ali",
			"count": float64(42),
		},
		"credentials": map[string]any{
			"my_cred": map[string]any{
				"type_key": "openai",
				"mode":     "bearer",
				"data":     map[string]any{"api_key": "sk-123"},
			},
		},
	})

	if capturedCtx == nil {
		t.Fatal("handler was not called")
	}
	if capturedCtx.String("name") != "ali" {
		t.Errorf("String(name) = %q, want ali", capturedCtx.String("name"))
	}
	if capturedCtx.Int("count") != 42 {
		t.Errorf("Int(count) = %d, want 42", capturedCtx.Int("count"))
	}
	cred, ok := capturedCtx.Credential("my_cred")
	if !ok {
		t.Fatal("Credential(my_cred) not found")
	}
	if cred.TypeKey != "openai" {
		t.Errorf("cred.TypeKey = %q, want openai", cred.TypeKey)
	}
	if cred.Data["api_key"] != "sk-123" {
		t.Errorf("cred.Data[api_key] = %q, want sk-123", cred.Data["api_key"])
	}
}

func TestToolDescribeAndInvoke(t *testing.T) {
	m := botmodule.New("mymodule", "My Module")
	m.AddTool(botmodule.Tool{
		Name:        "mymodule.search",
		Description: "Search the catalog",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"query": map[string]any{"type": "string"}},
			"required":   []any{"query"},
		},
		Invoke: func(c *botmodule.ToolCtx) (string, error) {
			return "found: " + c.String("query"), nil
		},
	})
	h := m.ServeHandler()

	// describe() tools ro'yxatida chiqsin
	resp := rpcCall(t, h, "describe", nil)
	result, _ := resp["result"].(map[string]any)
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %v, want 1", result["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "mymodule.search" {
		t.Errorf("tool.name = %v", tool["name"])
	}
	if tool["rpcMethod"] != "mymodule.search" {
		t.Errorf("tool.rpcMethod = %v, want fallback to name", tool["rpcMethod"])
	}

	// engine kontrakti: method=rpcMethod, params=args
	resp = rpcCall(t, h, "mymodule.search", map[string]any{"query": "kitob"})
	if resp["error"] != nil {
		t.Fatalf("tool invoke error: %v", resp["error"])
	}
	if resp["result"] != "found: kitob" {
		t.Errorf("tool result = %v, want 'found: kitob'", resp["result"])
	}

	// noma'lum metod hali ham 404
	resp = rpcCall(t, h, "nope.x", nil)
	if resp["error"] == nil {
		t.Error("unknown method should error")
	}
}
