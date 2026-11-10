package executor

import (
	"io"
	"testing"

	"github.com/MoeForge/moego/ojson"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

func TestAntigravityArgonCarbonRequestType(t *testing.T) {
	for _, tc := range []struct{ name, model, kind, want string }{
		{"argon_high", "gemini-4-argon-high", "agent", ""},
		{"argon_medium", "gemini-4-argon-medium", "agent", ""},
		{"argon_low", "gemini-4-argon-low", "agent", ""},
		{"carbon_high", "gemini-4-carbon-high", "agent", ""},
		{"carbon_medium", "gemini-4-carbon-medium", "agent", ""},
		{"carbon_low", "gemini-4-carbon-low", "agent", ""},
		{"flash_unchanged", "gemini-3.8-flash-high", "agent", "agent"},
		{"argon_search_unchanged", "gemini-4-argon-high", "web_search", "web_search"},
		{"carbon_search_unchanged", "gemini-4-carbon-high", "web_search", "web_search"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &AntigravityExecutor{}
			auth := &cliproxyauth.Auth{ID: "argon-test", Provider: "antigravity", Metadata: map[string]any{"project_id": "test-project"}}
			doc := ojson.NewDocString(`{"request":{"contents":[{"role":"user","parts":[{"text":"ok"}]}]},"requestType":"agent"}`)
			if err := doc.Set("requestType", tc.kind); err != nil {
				t.Fatal(err)
			}
			req, _, err := e.buildRequestDoc(t.Context(), auth, "test-token", tc.model, doc, false, "", "https://example.invalid")
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(req.Body)
			if closeErr := req.Body.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			out := ojson.NewDoc(raw)
			if got := out.Get("requestType").String(); got != tc.want {
				t.Fatalf("requestType = %q, want %q", got, tc.want)
			}
			if tc.want == "" && out.Get("requestType").Exists() {
				t.Fatal("requestType must be absent")
			}
			if got := out.Get("request.contents.0.parts.0.text").String(); got != "ok" {
				t.Fatalf("content changed: %q", got)
			}
		})
	}
}
