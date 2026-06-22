// Package botmodule — Botmother tashqi modul yozish uchun Go SDK.
//
// Botmother engine Telegram bot flow'larini ijro etadi va noma'lum node
// turlarini tashqi modullardan so'raydi (JSON-RPC 2.0). Bu paket modul
// yozishni soddalashtiradi: node'larni ro'yxatdan o'tkazasiz, handler
// yozasiz, [Module.Serve] bilan server ishga tushirasiz.
//
// Minimal misol:
//
//	m := botmodule.New("demo", "Demo Module")
//	m.AddNode(botmodule.Node{
//	    Type: "demo.Echo", Title: "Echo", Category: "integrations",
//	    Icon: "sparkles", Color: "blue",
//	    Content: []botmodule.Field{{Type: "text", Key: "input", Label: "Matn"}},
//	    Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
//	        return botmodule.Result{ContextUpdates: map[string]any{"echo_output": c.String("input")}}
//	    },
//	})
//	m.Serve(":8100")
//
// Autentifikatsiya: [Module.Serve] "MODULE_AUTH_TOKEN" env o'zgaruvchisini
// tekshiradi. Bo'sh bo'lsa — tekshirilmaydi. Port "PORT" env'dan ham olinadi.
//
// Kontrakt: JSON-RPC 2.0, POST /rpc; GET /health → 200.
// To'liq hujjat: README.md.
package botmodule
