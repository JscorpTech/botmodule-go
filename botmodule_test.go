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
