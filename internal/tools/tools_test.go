package tools_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datumbridge/magento-mcp/internal/tools"
)

func clearToolFilterEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MAGENTO_TOOLS", "")
}

func TestRegisterFullScopeByDefault(t *testing.T) {
	clearToolFilterEnv(t)
	descs, handlers := tools.Register()
	if len(descs) != 66 {
		t.Fatalf("expected full tool scope (66 tools), got %d", len(descs))
	}

	wantedTools := []string{
		// Products (11)
		"magento_external_list_products", "magento_external_get_product", "magento_external_create_product",
		"magento_external_update_product", "magento_external_delete_product", "magento_external_list_product_types",
		"magento_external_get_product_import_template", "magento_external_import_create_products", "magento_external_import_update_products",
		"magento_external_get_media_import_status", "magento_external_get_related_products",
		// Public Storefront (7)
		"magento_external_public_search_products", "magento_external_public_get_product", "magento_external_public_get_outfit",
		"magento_external_public_list_categories", "magento_external_public_create_cart",
		"magento_external_public_add_to_cart", "magento_external_public_get_cart",
		// Orders (6)
		"magento_external_list_orders", "magento_external_get_order", "magento_external_add_order_comment",
		"magento_external_cancel_order", "magento_external_hold_order", "magento_external_verify_and_get_order",
		// Customers (4)
		"magento_external_list_customers", "magento_external_get_customer", "magento_external_create_customer",
		"magento_external_update_customer",
		// Inventory (4)
		"magento_external_get_stock_item", "magento_external_update_stock_item", "magento_external_list_inventory_sources",
		"magento_external_get_source_items",
		// Categories (4)
		"magento_external_list_categories", "magento_external_get_category", "magento_external_list_category_products",
		"magento_external_assign_category_product",
		// Invoices & Shipments (3)
		"magento_external_list_invoices", "magento_external_get_invoice", "magento_external_create_shipment",
		// Store (2)
		"magento_external_list_store_views", "magento_external_list_store_configs",
		// Promotions & Sales Rules (3)
		"magento_external_list_sales_rules", "magento_external_get_sales_rule", "magento_external_delete_coupon",
		// Credit Memos / Refunds (3)
		"magento_external_list_credit_memos", "magento_external_get_credit_memo", "magento_external_create_credit_memo",
		// CMS (4)
		"magento_external_list_cms_pages", "magento_external_get_cms_page", "magento_external_list_cms_blocks", "magento_external_get_cms_block",
		// Product Media (2)
		"magento_external_list_product_media", "magento_external_get_product_media",
		// Cart (3)
		"magento_external_create_empty_cart", "magento_external_get_cart", "magento_external_add_cart_item",
	}

	for _, want := range wantedTools {
		if _, ok := handlers[want]; !ok {
			t.Fatalf("missing tool handler: %s", want)
		}
	}

	for _, d := range descs {
		doc, ok := d.InputSchema["x-datumbridge-docs"].(string)
		if !ok || doc == "" {
			t.Errorf("tool %s missing x-datumbridge-docs in InputSchema", d.Name)
		}
		caps, ok := d.InputSchema["x-datumbridge-capabilities"].([]string)
		if !ok || len(caps) == 0 {
			t.Errorf("tool %s missing x-datumbridge-capabilities in InputSchema", d.Name)
		}
		if !strings.Contains(d.Description, "Capabilities: magento."+d.Name) {
			t.Errorf("tool %s description missing Capabilities: magento.%s", d.Name, d.Name)
		}
		if !strings.Contains(d.Description, "Outputs:") {
			t.Errorf("tool %s description missing Outputs:", d.Name)
		}
	}
}

func TestRegisterCustomAllowlist(t *testing.T) {
	clearToolFilterEnv(t)
	t.Setenv("MAGENTO_TOOLS", "magento_external_get_product,magento_external_list_orders")
	descs, handlers := tools.Register()
	if len(descs) != 2 {
		t.Fatalf("expected 2 allowlisted tools, got %d", len(descs))
	}
	if _, ok := handlers["magento_external_get_product"]; !ok {
		t.Fatal("magento_external_get_product should be enabled")
	}
	if _, ok := handlers["magento_external_list_orders"]; !ok {
		t.Fatal("magento_external_list_orders should be enabled")
	}
	if _, ok := handlers["magento_external_create_product"]; ok {
		t.Fatal("magento_external_create_product should be filtered out")
	}
}

func testCreds(t *testing.T, srv *httptest.Server) json.RawMessage {
	t.Helper()
	t.Setenv("MAGENTO_API_BASE", srv.URL)
	creds := map[string]string{
		"base_url":            srv.URL,
		"consumer_key":        "mock_ck",
		"consumer_secret":     "mock_cs",
		"access_token":        "mock_at",
		"access_token_secret": "mock_ats",
	}
	bundleJSON, _ := json.Marshal(creds)
	raw, _ := json.Marshal(map[string]string{"credentials_json": string(bundleJSON)})
	return raw
}

func TestHandleGetProductSuccess(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/graphql" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"products":{"items":[]}}}`))
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "OAuth ") {
			t.Errorf("missing or invalid OAuth header: %q", auth)
		}
		if r.URL.Path != "/rest/V1/products/24-MB01" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sku":"24-MB01","name":"Joust Duffle Bag","price":34.00}`))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["sku"] = "24-MB01"
	raw, _ := json.Marshal(m)

	_, handlers := tools.Register()
	res := handlers["magento_external_get_product"](raw)
	if res["isError"] == true {
		t.Fatalf("unexpected error result: %+v", res)
	}
}

func TestHandleGetProductMissingSKU(t *testing.T) {
	clearToolFilterEnv(t)
	_, handlers := tools.Register()
	res := handlers["magento_external_get_product"](json.RawMessage(`{"credentials_json":"{\"base_url\":\"https://x.com\",\"consumer_key\":\"k\",\"consumer_secret\":\"s\",\"access_token\":\"t\",\"access_token_secret\":\"ts\"}"}`))
	if res["isError"] != true {
		t.Fatalf("expected error result for missing sku and name, got: %+v", res)
	}
}

func TestHandleGetProductByNameSuccess(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/V1/products":
			// Search endpoint returns both simple child variant and parent configurable product
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"items": [
					{"sku": "WF-ST-008-BLK-S", "name": "Set áo peplum và quần tây Sol - Đen - S", "type_id": "simple", "visibility": 1, "price": 1890000},
					{"sku": "WF-ST-008", "name": "Set áo peplum và quần tây Sol", "type_id": "configurable", "visibility": 4, "price": 0}
				],
				"total_count": 2
			}`))
		case "/rest/V1/products/WF-ST-008":
			// Must select the parent configurable product (WF-ST-008), NOT the child variant (WF-ST-008-BLK-S)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"sku": "WF-ST-008",
				"name": "Set áo peplum và quần tây Sol",
				"type_id": "configurable",
				"visibility": 4,
				"price": 0,
				"extension_attributes": {
					"configurable_product_options": [
						{
							"attribute_id": "93",
							"label": "Color",
							"values": [{"value_index": 128}, {"value_index": 129}, {"value_index": 130}]
						}
					]
				}
			}`))
		case "/rest/V1/configurable-products/WF-ST-008/children":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"sku": "WF-ST-008-BLK-S", "price": 1890000, "custom_attributes": [{"attribute_code": "special_price", "value": "1701000"}]},
				{"sku": "WF-ST-008-WIN-S", "price": 1890000}
			]`))
		case "/rest/V1/products/attributes/93/options":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"value": "128", "label": "Đen"},
				{"value": "129", "label": "Rượu vang"},
				{"value": "130", "label": "Xanh rêu"}
			]`))
		case "/graphql":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"products":{"items":[]}}}`))
		default:
			t.Errorf("unexpected path: %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["name"] = "Set áo peplum và quần tây Sol"
	raw, _ := json.Marshal(m)

	_, handlers := tools.Register()
	res := handlers["magento_external_get_product"](raw)
	if res["isError"] == true {
		t.Fatalf("unexpected error result: %+v", res)
	}

	content := res["content"].([]map[string]string)[0]["text"]
	var prod map[string]interface{}
	if err := json.Unmarshal([]byte(content), &prod); err != nil {
		t.Fatalf("failed to parse product result: %v", err)
	}

	if prod["sku"] != "WF-ST-008" {
		t.Errorf("expected parent SKU WF-ST-008, got: %v", prod["sku"])
	}
	if prod["type_id"] != "configurable" {
		t.Errorf("expected type_id configurable, got: %v", prod["type_id"])
	}
	colors, ok := prod["available_colors"].([]interface{})
	if !ok || len(colors) != 3 {
		t.Errorf("expected 3 available_colors, got: %+v", prod["available_colors"])
	}
}

func TestHandleGetProductByNameNotFound(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/rest/V1/products" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items": [], "total_count": 0}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["name"] = "Sản phẩm không tồn tại 12345"
	raw, _ := json.Marshal(m)

	_, handlers := tools.Register()
	res := handlers["magento_external_get_product"](raw)
	if res["isError"] != true {
		t.Fatalf("expected error result for not found product, got: %+v", res)
	}
}

func TestHandleCreateProductSuccess(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		prod := body["product"].(map[string]interface{})
		if prod["sku"] != "SKU-99" {
			t.Errorf("expected SKU-99, got %v", prod["sku"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sku":"SKU-99","name":"Test Product"}`))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["sku"] = "SKU-99"
	m["name"] = "Test Product"
	m["price"] = 99.95
	raw, _ := json.Marshal(m)

	_, handlers := tools.Register()
	res := handlers["magento_external_create_product"](raw)
	if res["isError"] == true {
		t.Fatalf("unexpected error result: %+v", res)
	}
}

func TestHandleListOrdersSuccess(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/V1/orders" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"entity_id":1,"status":"pending"}]}`))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	_, handlers := tools.Register()
	res := handlers["magento_external_list_orders"](args)
	if res["isError"] == true {
		t.Fatalf("unexpected error result: %+v", res)
	}
}

func TestHandleListStoreViewsSuccess(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/V1/store/storeViews" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":1,"code":"default"}]`))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	_, handlers := tools.Register()
	res := handlers["magento_external_list_store_views"](args)
	if res["isError"] == true {
		t.Fatalf("unexpected error result: %+v", res)
	}
}

func getToolResultText(res map[string]interface{}) string {
	switch c := res["content"].(type) {
	case []map[string]string:
		if len(c) > 0 {
			return c[0]["text"]
		}
	case []interface{}:
		if len(c) > 0 {
			if m, ok := c[0].(map[string]interface{}); ok {
				return fmt.Sprint(m["text"])
			}
		}
	}
	return ""
}

func TestHandleMagentoVerifyAndGetOrder(t *testing.T) {
	clearToolFilterEnv(t)
	orderJSON := `{"items":[{"increment_id":"100000025","customer_email":"customer@example.com","status":"processing","created_at":"2026-10-25 10:00:00","grand_total":550000,"order_currency_code":"VND","billing_address":{"firstname":"Van","lastname":"Nguyen","telephone":"0987654321","city":"Ho Chi Minh","region":"District 1"},"items":[{"name":"Ao so mi trang","sku":"SHIRT-WHT-M","qty_ordered":1,"price":550000,"row_total":550000}]}],"total_count":1}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/V1/orders" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(orderJSON))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	_, handlers := tools.Register()

	// 1. Success with email
	{
		var m map[string]interface{}
		_ = json.Unmarshal(args, &m)
		m["order_number"] = "#100000025"
		m["verification"] = "customer@example.com"
		raw, _ := json.Marshal(m)

		res := handlers["magento_external_verify_and_get_order"](raw)
		if res["isError"] == true {
			t.Fatalf("unexpected error: %+v", res)
		}
		text := getToolResultText(res)
		if !strings.Contains(text, `"verified": true`) && !strings.Contains(text, `"verified":true`) {
			t.Fatalf("expected verified=true, got: %s", text)
		}
		if !strings.Contains(text, "SHIRT-WHT-M") {
			t.Fatalf("expected item SKU SHIRT-WHT-M, got: %s", text)
		}
	}

	// 2. Success with phone
	{
		var m map[string]interface{}
		_ = json.Unmarshal(args, &m)
		m["order_number"] = "100000025"
		m["verification"] = "+84987654321"
		raw, _ := json.Marshal(m)

		res := handlers["magento_external_verify_and_get_order"](raw)
		if res["isError"] == true {
			t.Fatalf("unexpected error: %+v", res)
		}
		text := getToolResultText(res)
		if !strings.Contains(text, `"verified": true`) && !strings.Contains(text, `"verified":true`) {
			t.Fatalf("expected verified=true with phone, got: %s", text)
		}
	}

	// 3. Mismatch verification
	{
		var m map[string]interface{}
		_ = json.Unmarshal(args, &m)
		m["order_number"] = "100000025"
		m["verification"] = "wrong@example.com"
		raw, _ := json.Marshal(m)

		res := handlers["magento_external_verify_and_get_order"](raw)
		if res["isError"] == true {
			t.Fatalf("unexpected error: %+v", res)
		}
		text := getToolResultText(res)
		if !strings.Contains(text, `"verified": false`) && !strings.Contains(text, `"verified":false`) {
			t.Fatalf("expected verified=false, got: %s", text)
		}
	}
}

func TestHandleMagentoGetRelatedProducts(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/rest/V1/products/SHIRT-01/links/crosssell" {
			_, _ = w.Write([]byte(`[{"sku":"SHIRT-01","link_type":"crosssell","linked_product_sku":"PANT-01"}]`))
			return
		}
		if r.URL.Path == "/rest/V1/products" {
			_, _ = w.Write([]byte(`{"items":[{"sku":"PANT-01","name":"Quan tay cong so","price":450000,"status":1,"type_id":"simple"}],"total_count":1}`))
			return
		}
		t.Errorf("unexpected path: %q", r.URL.Path)
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	_, handlers := tools.Register()

	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["sku"] = "SHIRT-01"
	m["link_type"] = "crosssell"
	raw, _ := json.Marshal(m)

	res := handlers["magento_external_get_related_products"](raw)
	if res["isError"] == true {
		t.Fatalf("unexpected error: %+v", res)
	}
	text := getToolResultText(res)
	if !strings.Contains(text, "PANT-01") || !strings.Contains(text, "Quan tay cong so") {
		t.Fatalf("expected PANT-01 in result, got: %s", text)
	}
}

func TestHandleGetSourceItemsSuccess(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/V1/inventory/source-items" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		skuFound := false
		sourceFound := false
		for i := 0; i < 5; i++ {
			field := q.Get(fmt.Sprintf("searchCriteria[filterGroups][%d][filters][0][field]", i))
			val := q.Get(fmt.Sprintf("searchCriteria[filterGroups][%d][filters][0][value]", i))
			if field == "sku" && val == "BLZ-01" {
				skuFound = true
			}
			if field == "source_code" && val == "vincom_dk" {
				sourceFound = true
			}
		}
		if !skuFound {
			t.Errorf("expected sku filter BLZ-01, got query: %v", q)
		}
		if !sourceFound {
			t.Errorf("expected source_code filter vincom_dk, got query: %v", q)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"sku":"BLZ-01","source_code":"vincom_dk","quantity":3,"status":1}],"total_count":1}`))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["sku"] = "BLZ-01"
	m["source_code"] = "vincom_dk"
	raw, _ := json.Marshal(m)

	_, handlers := tools.Register()
	res := handlers["magento_external_get_source_items"](raw)
	if res["isError"] == true {
		t.Fatalf("unexpected error: %+v", res)
	}
	text := getToolResultText(res)
	if !strings.Contains(text, "vincom_dk") || !strings.Contains(text, `"quantity": 3`) {
		t.Fatalf("expected source item details in result, got: %s", text)
	}
}

func TestHandleGetSourceItemsMissingSKU(t *testing.T) {
	clearToolFilterEnv(t)
	_, handlers := tools.Register()
	res := handlers["magento_external_get_source_items"](json.RawMessage(`{"credentials_json":"{\"base_url\":\"https://x.com\",\"consumer_key\":\"k\",\"consumer_secret\":\"s\",\"access_token\":\"t\",\"access_token_secret\":\"ts\"}"}`))
	if res["isError"] != true {
		t.Fatalf("expected error result for missing sku, got: %+v", res)
	}
}

func TestHandleListProductsWithCategoryID(t *testing.T) {
	clearToolFilterEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/graphql" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"products":{"items":[]}}}`))
			return
		}
		if r.URL.Path != "/rest/V1/products" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		q := r.URL.Query()
		categoryFound := false
		for i := 0; i < 5; i++ {
			field := q.Get(fmt.Sprintf("searchCriteria[filterGroups][%d][filters][0][field]", i))
			val := q.Get(fmt.Sprintf("searchCriteria[filterGroups][%d][filters][0][value]", i))
			if field == "category_id" && val == "15" {
				categoryFound = true
			}
		}
		if !categoryFound {
			t.Errorf("expected category_id filter 15, got query: %v", q)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"sku":"DRESS-01","name":"Vay da hoi den","price":1500000}],"total_count":1}`))
	}))
	defer srv.Close()

	args := testCreds(t, srv)
	var m map[string]interface{}
	_ = json.Unmarshal(args, &m)
	m["category_id"] = "15"
	raw, _ := json.Marshal(m)

	_, handlers := tools.Register()
	res := handlers["magento_external_list_products"](raw)
	if res["isError"] == true {
		t.Fatalf("unexpected error: %+v", res)
	}
	text := getToolResultText(res)
	if !strings.Contains(text, "DRESS-01") {
		t.Fatalf("expected DRESS-01 in result, got: %s", text)
	}
}

// The agent planner only fills arguments marked required; without this the
// public search tool was called with no input at all.
func TestPublicSearchProductsRequiresSearch(t *testing.T) {
	clearToolFilterEnv(t)
	descs, _ := tools.Register()
	for _, d := range descs {
		if d.Name != "magento_external_public_search_products" {
			continue
		}
		req, _ := d.InputSchema["required"].([]string)
		if len(req) != 1 || req[0] != "search" {
			t.Fatalf("expected required [search], got %v", d.InputSchema["required"])
		}
		props := d.InputSchema["properties"].(map[string]interface{})
		cat := props["category_id"].(map[string]interface{})
		if got := cat["x-datumbridge-related-tool"]; got != "magento_external_public_list_categories" {
			t.Fatalf("public category_id should point to public tool, got %v", got)
		}
		return
	}
	t.Fatal("magento_external_public_search_products not registered")
}

// Every public tool whose call is useless without an argument must mark it
// required, otherwise the agent planner sends no input.
func TestPublicToolsRequiredArgs(t *testing.T) {
	clearToolFilterEnv(t)
	want := map[string]string{
		"magento_external_public_search_products": "search",
		"magento_external_public_get_product":     "sku",
		"magento_external_public_get_outfit":      "sku",
		"magento_external_public_add_to_cart":     "sku",
		"magento_external_public_get_cart":        "cart_id",
	}
	descs, _ := tools.Register()
	for _, d := range descs {
		field, ok := want[d.Name]
		if !ok {
			continue
		}
		req, _ := d.InputSchema["required"].([]string)
		if len(req) != 1 || req[0] != field {
			t.Errorf("%s: expected required [%s], got %v", d.Name, field, d.InputSchema["required"])
		}
		delete(want, d.Name)
	}
	for name := range want {
		t.Errorf("%s not registered", name)
	}
}

func TestPublicAddToCartCreatesCartWhenMissing(t *testing.T) {
	clearToolFilterEnv(t)
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(body.Query, "createEmptyCart") {
			calls = append(calls, "create")
			fmt.Fprint(w, `{"data":{"createEmptyCart":"cart-123"}}`)
			return
		}
		calls = append(calls, "add:"+fmt.Sprint(body.Variables["cartId"]))
		fmt.Fprint(w, `{"data":{"addProductsToCart":{"cart":{"id":"cart-123","total_quantity":1},"user_errors":[]}}}`)
	}))
	defer srv.Close()

	_, handlers := tools.Register()
	res := handlers["magento_external_public_add_to_cart"](json.RawMessage(`{"sku":"WF-SK-013","store_url":"` + srv.URL + `"}`))
	out, _ := json.Marshal(res)
	if strings.Contains(string(out), `"isError":true`) {
		t.Fatalf("unexpected error: %s", out)
	}
	if len(calls) != 2 || calls[0] != "create" || calls[1] != "add:cart-123" {
		t.Fatalf("expected create then add to cart-123, got %v", calls)
	}
	if !strings.Contains(string(out), "cart-123") {
		t.Fatalf("result should carry cart_id: %s", out)
	}
}

// fakeCatalogServer mimics Magento full-text search: every word of `search`
// must appear in the same field (name or description).
func fakeCatalogServer(t *testing.T, searches *[]string) *httptest.Server {
	t.Helper()
	type prod struct{ sku, name, desc string }
	catalog := []prod{
		{"SK-1", "Chân váy tulip Ayla", "dáng tulip trẻ trung"},
		{"SK-2", "Chân váy đuôi cá Elodie", "tôn dáng duyên dáng nữ tính"},
		{"ST-1", "Set blouse và chân váy Iris", "thanh lịch đi làm"},
		{"DR-1", "Đầm wrap Emilia", "duyên dáng"},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		s, _ := body.Variables["search"].(string)
		*searches = append(*searches, s)
		words := strings.Fields(strings.ToLower(s))
		inField := func(f string) bool {
			for _, wd := range words {
				if !strings.Contains(strings.ToLower(f), wd) {
					return false
				}
			}
			return true
		}
		items := []map[string]interface{}{}
		for _, p := range catalog {
			if inField(p.name) || inField(p.desc) {
				items = append(items, map[string]interface{}{"sku": p.sku, "name": p.name})
			}
		}
		out, _ := json.Marshal(map[string]interface{}{"data": map[string]interface{}{"products": map[string]interface{}{
			"total_count": len(items), "items": items, "page_info": map[string]interface{}{"current_page": 1, "total_pages": 1},
		}}})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}))
}

func publicSearch(t *testing.T, srvURL string, args map[string]interface{}) map[string]interface{} {
	t.Helper()
	args["store_url"] = srvURL
	raw, _ := json.Marshal(args)
	_, handlers := tools.Register()
	res, _ := json.Marshal(handlers["magento_external_public_search_products"](raw))
	var env struct {
		IsError bool                    `json:"isError"`
		Content []struct{ Text string } `json:"content"`
	}
	_ = json.Unmarshal(res, &env)
	if env.IsError || len(env.Content) == 0 {
		t.Fatalf("unexpected tool result: %s", res)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(env.Content[0].Text), &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, env.Content[0].Text)
	}
	return out
}

func itemSKUs(out map[string]interface{}) []string {
	var skus []string
	for _, it := range out["products"].(map[string]interface{})["items"].([]interface{}) {
		skus = append(skus, it.(map[string]interface{})["sku"].(string))
	}
	return skus
}

func TestPublicSearchRelaxesDescriptiveWords(t *testing.T) {
	clearToolFilterEnv(t)
	var searches []string
	srv := fakeCatalogServer(t, &searches)
	defer srv.Close()

	out := publicSearch(t, srv.URL, map[string]interface{}{"search": "chân váy duyên dáng"})
	meta := out["search_meta"].(map[string]interface{})
	if meta["relaxed"] != true || meta["search_used"] != "chân váy" {
		t.Fatalf("expected relaxed search on \"chân váy\", got %v (searches %v)", meta, searches)
	}
	skus := itemSKUs(out)
	if len(skus) != 3 || skus[0] != "SK-2" {
		t.Fatalf("expected 3 skirts with SK-2 (duyên dáng in description) first, got %v", skus)
	}
	first := out["products"].(map[string]interface{})["items"].([]interface{})[0].(map[string]interface{})
	if first["matches_style"] != true || meta["style_matched"].(float64) != 1 {
		t.Fatalf("expected SK-2 flagged matches_style, got %v / %v", first, meta)
	}
}

func TestPublicSearchSetKeepsBothProductTypes(t *testing.T) {
	clearToolFilterEnv(t)
	var searches []string
	srv := fakeCatalogServer(t, &searches)
	defer srv.Close()

	out := publicSearch(t, srv.URL, map[string]interface{}{"search": "set chân váy duyên dáng"})
	meta := out["search_meta"].(map[string]interface{})
	if meta["search_used"] != "set chân váy" {
		t.Fatalf("expected search_used \"set chân váy\", got %v", meta["search_used"])
	}
	if skus := itemSKUs(out); len(skus) != 1 || skus[0] != "ST-1" {
		t.Fatalf("expected only the set, got %v", skus)
	}
}

func TestPublicSearchExactMatchIsUntouched(t *testing.T) {
	clearToolFilterEnv(t)
	var searches []string
	srv := fakeCatalogServer(t, &searches)
	defer srv.Close()

	out := publicSearch(t, srv.URL, map[string]interface{}{"search": "chân váy"})
	if meta := out["search_meta"].(map[string]interface{}); meta["relaxed"] != false {
		t.Fatalf("exact match should not relax: %v", meta)
	}
	if len(searches) != 1 {
		t.Fatalf("exact match should cost one query, got %v", searches)
	}
}

func TestPublicSearchExplicitStyleRanks(t *testing.T) {
	clearToolFilterEnv(t)
	var searches []string
	srv := fakeCatalogServer(t, &searches)
	defer srv.Close()

	out := publicSearch(t, srv.URL, map[string]interface{}{"search": "chân váy", "style": "duyên dáng"})
	if skus := itemSKUs(out); len(skus) != 3 || skus[0] != "SK-2" {
		t.Fatalf("style should move SK-2 first, got %v", skus)
	}
}
