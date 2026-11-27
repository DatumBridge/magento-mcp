package tools

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// storeStub answers each module path with a canned body and records the requests.
type storeStub struct {
	srv    *httptest.Server
	calls  map[string]map[string]interface{}
	bodies map[string]string
	codes  map[string]int
}

func newStoreStub(t *testing.T) *storeStub {
	t.Helper()
	t.Setenv("MAGENTO_BASE_URL", "")
	s := &storeStub{calls: map[string]map[string]interface{}{}, bodies: map[string]string{}, codes: map[string]int{}}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		s.calls[r.URL.Path] = body
		if code := s.codes[r.URL.Path]; code != 0 {
			w.WriteHeader(code)
		}
		_, _ = w.Write([]byte(s.bodies[r.URL.Path]))
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *storeStub) args(extra map[string]interface{}) json.RawMessage {
	extra["store_url"] = s.srv.URL
	b, _ := json.Marshal(extra)
	return b
}

func resultJSON(t *testing.T, res map[string]interface{}) map[string]interface{} {
	t.Helper()
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(res["content"].([]map[string]string)[0]["text"]), &out); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	return out
}

func TestUpdateDelivery_notAllowedIsRelayed(t *testing.T) {
	s := newStoreStub(t)
	s.codes[orderUpdateDeliveryPath] = http.StatusConflict
	s.bodies[orderUpdateDeliveryPath] = `{"success":false,"error_code":"not_allowed","message":"Đơn hàng đã được đóng gói nên không thể đổi địa chỉ."}`

	out := resultJSON(t, handleMagentoPublicUpdateDelivery(s.args(map[string]interface{}{
		"order_access_token": "dbo1.x.y", "street": "456 Lê Lợi", "district": "Quận 1", "province": "TP.HCM",
	})))
	if out["success"] != false || out["error_code"] != "not_allowed" {
		t.Fatalf("out = %+v", out)
	}
	if got := s.calls[orderUpdateDeliveryPath]; got["order_access_token"] != "dbo1.x.y" || got["district"] != "Quận 1" {
		t.Fatalf("store got %+v", got)
	}
}

func TestUpdateDelivery_successAddsNoticeToCard(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[orderUpdateDeliveryPath] = `{"success":true,"changes":["address"],"message":"Đã cập nhật địa chỉ giao hàng.",
		"order":{"order_number":"000000123","shipping":{"full_address":"456 Lê Lợi, Quận 1, TP.HCM"}}}`
	out := resultJSON(t, handleMagentoPublicUpdateDelivery(s.args(map[string]interface{}{
		"order_access_token": "dbo1.x.y", "street": "456 Lê Lợi", "district": "Quận 1", "province": "TP.HCM",
	})))
	card := out["card"].(map[string]interface{})
	if card["notice"] != "Đã cập nhật địa chỉ giao hàng." {
		t.Fatalf("card = %+v", card)
	}
	if card["shipping"].(map[string]interface{})["address"] != "456 Lê Lợi, Quận 1, TP.HCM" {
		t.Fatalf("card shipping = %+v", card["shipping"])
	}
}

func TestUpdateDelivery_tokenExpiredTellsAgentToLookUpAgain(t *testing.T) {
	s := newStoreStub(t)
	s.codes[orderUpdateDeliveryPath] = http.StatusUnauthorized
	s.bodies[orderUpdateDeliveryPath] = `{"success":false,"error_code":"token_invalid","message":"Phiên xác thực đã hết hạn."}`
	out := resultJSON(t, handleMagentoPublicUpdateDelivery(s.args(map[string]interface{}{"order_access_token": "old", "note": "x"})))
	if out["next_step"] == nil {
		t.Fatalf("expected next_step hint: %+v", out)
	}
}

func TestExchangeOptions_severalItemsAsksWhich(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[exchangeOptionsPath] = `{"success":true,"order_number":"000000123","items":[
		{"item_id":1,"name":"Áo blazer","sku":"BLZ-M","sizes":[]},{"item_id":2,"name":"Quần tây","sku":"QT-S","sizes":[]}]}`
	out := resultJSON(t, handleMagentoPublicGetExchangeOptions(s.args(map[string]interface{}{"order_access_token": "t"})))
	if out["need_item"] != true || len(out["items"].([]interface{})) != 2 {
		t.Fatalf("out = %+v", out)
	}
	if _, hasCard := out["card"]; hasCard {
		t.Fatal("no card until one item is chosen")
	}
}

func TestExchangeOptions_singleItemCard(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[exchangeOptionsPath] = `{"success":true,"order_number":"000000123","items":[{"item_id":1,"name":"Áo blazer","sku":"BLZ-M",
		"color":"Be","current_size":"M","eligible":true,
		"sizes":[{"label":"M","current":true,"in_stock":true},{"label":"L","current":false,"in_stock":true}],
		"methods":[{"value":"at_delivery","available":true}],"conditions":[{"key":"window","status":"pass"}]}]}`
	out := resultJSON(t, handleMagentoPublicGetExchangeOptions(s.args(map[string]interface{}{"order_access_token": "t", "item": "blazer"})))
	card := out["card"].(map[string]interface{})
	if card["order_number"] != "000000123" || card["current_size"] != "M" || len(card["sizes"].([]interface{})) != 2 {
		t.Fatalf("card = %+v", card)
	}
	if s.calls[exchangeOptionsPath]["item"] != "blazer" {
		t.Fatalf("item not forwarded: %+v", s.calls[exchangeOptionsPath])
	}
}

func TestCreateExchange_requiresConfirmation(t *testing.T) {
	s := newStoreStub(t)
	out := resultJSON(t, handleMagentoPublicCreateExchange(s.args(map[string]interface{}{
		"order_access_token": "t", "new_size": "L", "method": "at_delivery",
	})))
	if out["error_code"] != "confirmation_required" {
		t.Fatalf("out = %+v", out)
	}
	if _, called := s.calls[exchangeCreatePath]; called {
		t.Fatal("store must not be called before the shopper confirms")
	}
}

func TestCreateExchange_success(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[exchangeCreatePath] = `{"success":true,"message":"Đã tạo yêu cầu đổi size #EX000001.",
		"request":{"request_number":"EX000001","status":"created","timeline":[{"key":"created","status":"current"}]}}`
	out := resultJSON(t, handleMagentoPublicCreateExchange(s.args(map[string]interface{}{
		"order_access_token": "t", "new_size": "L", "method": "return_ship", "confirm_unused": true, "note": "rộng",
	})))
	if out["card"].(map[string]interface{})["request_number"] != "EX000001" {
		t.Fatalf("out = %+v", out)
	}
	got := s.calls[exchangeCreatePath]
	if got["new_size"] != "L" || got["method"] != "return_ship" || got["note"] != "rộng" {
		t.Fatalf("store got %+v", got)
	}
}

func TestMyOrders_cardAndToken(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[customerOrdersPath] = `{"success":true,"total_count":2,"page":1,"page_size":5,"orders":[{"order_number":"000000123"},{"order_number":"000000120"}]}`
	out := resultJSON(t, handleMagentoPublicMyOrders(s.args(map[string]interface{}{"customer_access_token": "dbc1.a.b"})))
	if out["card"].(map[string]interface{})["total_count"].(float64) != 2 {
		t.Fatalf("out = %+v", out)
	}
	got := s.calls[customerOrdersPath]
	if got["customer_access_token"] != "dbc1.a.b" || got["page_size"].(float64) != 5 {
		t.Fatalf("store got %+v", got)
	}
}

func TestMyOrders_expiredTokenSuggestsGuestLookup(t *testing.T) {
	s := newStoreStub(t)
	s.codes[customerOrdersPath] = http.StatusUnauthorized
	s.bodies[customerOrdersPath] = `{"success":false,"error_code":"token_invalid","message":"Phiên đăng nhập đã hết hạn."}`
	out := resultJSON(t, handleMagentoPublicMyOrders(s.args(map[string]interface{}{"customer_access_token": "dbc1.old"})))
	if out["error_code"] != "token_invalid" || out["next_step"] == nil {
		t.Fatalf("out = %+v", out)
	}
}

func TestMyOrderDetail_returnsOrderAccessToken(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[customerOrderPath] = `{"success":true,"verified":true,"order":{"order_number":"000000123","status_label":"Đang giao"},
		"order_access_token":"dbo1.x.y","token_expires_at":"2026-01-01T00:00:00Z"}`
	out := resultJSON(t, handleMagentoPublicMyOrderDetail(s.args(map[string]interface{}{"customer_access_token": "dbc1.a.b", "order_number": "123"})))
	if out["order_access_token"] != "dbo1.x.y" || out["card"].(map[string]interface{})["order_number"] != "000000123" {
		t.Fatalf("out = %+v", out)
	}
}

func TestMyOrderDetail_requiresToken(t *testing.T) {
	res := handleMagentoPublicMyOrderDetail(json.RawMessage(`{"order_number":"123","store_url":"https://x.test"}`))
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Fatal("expected error without customer_access_token")
	}
}

func TestCreateTicket_cardCarriesUpload(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[ticketCreatePath] = `{"success":true,"message":"Đã tiếp nhận khiếu nại #TK000001.",
		"ticket":{"ticket_number":"TK000001","status":"received"},
		"upload":{"url":"https://store.test/weaver/ticket/upload","token":"dbu1.a.b","remaining":5}}`
	out := resultJSON(t, handleMagentoPublicCreateTicket(s.args(map[string]interface{}{
		"order_access_token": "dbo1.x", "category": "defective", "item": "BLZ", "description": "Áo bị bung chỉ",
	})))
	card := out["card"].(map[string]interface{})
	if card["ticket_number"] != "TK000001" || card["upload"].(map[string]interface{})["token"] != "dbu1.a.b" {
		t.Fatalf("card = %+v", card)
	}
	if _, leaked := out["ticket"].(map[string]interface{})["upload"]; leaked {
		t.Fatal("upload must only be in the card")
	}
	if got := s.calls[ticketCreatePath]; got["category"] != "defective" || got["item"] != "BLZ" {
		t.Fatalf("store got %+v", got)
	}
}

func TestCreateTicket_requiresCategoryAndDescription(t *testing.T) {
	res := handleMagentoPublicCreateTicket(json.RawMessage(`{"order_access_token":"dbo1.x","category":"defective","store_url":"https://x.test"}`))
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Fatal("expected error without description")
	}
}

func TestTicketOptions_andStatus(t *testing.T) {
	s := newStoreStub(t)
	s.bodies[ticketOptionsPath] = `{"success":true,"order_number":"000000123","categories":[{"value":"defective"}],"items":[{"sku":"BLZ"}],"tickets":[]}`
	s.bodies[ticketStatusPath] = `{"success":true,"order_number":"000000123","tickets":[{"ticket_number":"TK000001"}]}`
	opt := resultJSON(t, handleMagentoPublicGetTicketOptions(s.args(map[string]interface{}{"order_access_token": "dbo1.x"})))
	if opt["card"].(map[string]interface{})["order_number"] != "000000123" {
		t.Fatalf("options = %+v", opt)
	}
	st := resultJSON(t, handleMagentoPublicGetTicketStatus(s.args(map[string]interface{}{"order_access_token": "dbo1.x"})))
	if st["count"].(float64) != 1 {
		t.Fatalf("status = %+v", st)
	}
}
