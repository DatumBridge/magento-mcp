package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func lookupPayload(t *testing.T, baseURL, order, verification string) json.RawMessage {
	t.Helper()
	b, _ := json.Marshal(map[string]interface{}{
		"order_number":     order,
		"verification":     verification,
		"credentials_json": fmt.Sprintf(`{"base_url":%q,"access_token":"must-not-be-sent"}`, baseURL),
	})
	return b
}

func TestPublicLookupOrder_verifiedBuildsCard(t *testing.T) {
	t.Setenv("MAGENTO_BASE_URL", "")
	var gotPath, gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"success":true,"verified":true,
			"order":{"order_number":"000000123","status_label":"Đang giao","current_step":"shipping","is_canceled":false,
			"grand_total":1790000,"currency":"VND","items":[{"name":"Áo khoác blazer","sku":"BLZ-M-BE","size":"M","color":"Be","qty":1,"price":1790000,"options":[]}],
			"timeline":[{"key":"shipping","label":"Đang giao","date":"16/10","status":"current"}],"estimated_delivery":"18/10/2026",
			"tracking":[],"can_change_address":false,"change_address_reason":"Đã đóng gói",
			"shipping":{"method":"Tiêu chuẩn","recipient_name":"Nguyen *** An","phone":"090****567","full_address":"123 Nguyễn Văn Linh, Quận 7"}},
			"order_access_token":"dbo1.abc.def","token_expires_at":"2026-10-16T10:00:00+00:00"}`))
	}))
	defer srv.Close()

	res := handleMagentoPublicLookupOrder(lookupPayload(t, srv.URL, "#123", "0901234567"))
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("unexpected error: %+v", res)
	}
	if gotPath != orderLookupPath {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "" {
		t.Fatalf("store token leaked in Authorization header: %q", gotAuth)
	}
	if gotBody["order_number"] != "#123" || gotBody["verification"] != "0901234567" {
		t.Fatalf("body = %+v", gotBody)
	}

	var out map[string]interface{}
	text := res["content"].([]map[string]string)[0]["text"]
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	card, ok := out["card"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing card: %s", text)
	}
	if card["order_number"] != "000000123" || card["estimated_delivery"] != "18/10/2026" {
		t.Fatalf("card = %+v", card)
	}
	if _, leaked := card["order_access_token"]; leaked {
		t.Fatal("card must not carry the access token")
	}
	if ship := card["shipping"].(map[string]interface{}); ship["address"] != "123 Nguyễn Văn Linh, Quận 7" {
		t.Fatalf("shipping = %+v", ship)
	}
	if out["order_access_token"] != "dbo1.abc.def" {
		t.Fatal("agent must receive the access token for follow-up actions")
	}
}

func TestPublicLookupOrder_notVerifiedIsNotAnError(t *testing.T) {
	t.Setenv("MAGENTO_BASE_URL", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"success":false,"verified":false,"error_code":"rate_limited","message":"Thử lại sau"}`))
	}))
	defer srv.Close()

	res := handleMagentoPublicLookupOrder(lookupPayload(t, srv.URL, "123", "0900000000"))
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("rate limit should be a normal result the agent relays: %+v", res)
	}
	text := res["content"].([]map[string]string)[0]["text"]
	var out map[string]interface{}
	_ = json.Unmarshal([]byte(text), &out)
	if out["verified"] != false || out["error_code"] != "rate_limited" {
		t.Fatalf("out = %+v", out)
	}
}

func TestPublicLookupOrder_moduleMissing(t *testing.T) {
	t.Setenv("MAGENTO_BASE_URL", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<html>404</html>`))
	}))
	defer srv.Close()

	res := handleMagentoPublicLookupOrder(lookupPayload(t, srv.URL, "123", "0900000000"))
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Fatalf("expected tool error when module is missing: %+v", res)
	}
}

func TestPublicLookupOrder_requiresBothArgs(t *testing.T) {
	res := handleMagentoPublicLookupOrder(json.RawMessage(`{"order_number":"123","store_url":"https://x.test"}`))
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Fatal("expected error without verification")
	}
}
