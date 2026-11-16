package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleMagentoPublicSearchProducts_usesConnectedStoreURL(t *testing.T) {
	t.Setenv("MAGENTO_BASE_URL", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"products":{"total_count":1,"items":[{"sku":"DRESS-1","name":"Đầm dạ hội"}]}}}`))
	}))
	defer srv.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"search": "đầm dạ hội",
		"credentials_json": fmt.Sprintf(
			`{"base_url":%q,"access_token":"store-token","consumer_key":"ck","consumer_secret":"cs","access_token_secret":"ats"}`,
			srv.URL,
		),
	})
	res := handleMagentoPublicSearchProducts(payload)
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("connected store URL was not used: %+v", res)
	}
}

func TestHandleMagentoPublicSearchProducts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"products": {
					"total_count": 2,
					"items": [
						{"id": 1, "sku": "BLZ-01", "name": "Blazer Basic", "stock_status": "IN_STOCK"},
						{"id": 2, "sku": "BLZ-02", "name": "Blazer Tweed", "stock_status": "OUT_OF_STOCK"}
					]
				}
			}
		}`))
	}))
	defer srv.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"store_url": srv.URL,
		"search":    "blazer",
	})

	res := handleMagentoPublicSearchProducts(payload)
	if errText, ok := res["error_message"].(string); ok && errText != "" {
		t.Fatalf("unexpected tool error: %s", errText)
	}
	if isErr, _ := res["isError"].(bool); isErr {
		t.Fatalf("tool returned error: %+v", res)
	}
}

func TestHandleMagentoPublicGetProduct_ConfigurableColors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"products": {
					"items": [
						{
							"id": 10,
							"sku": "WF-DR-010",
							"name": "Đầm peplum Iris",
							"url_key": "dam-peplum-iris-do-ruou-vang-wf-dr-010",
							"configurable_options": [
								{
									"attribute_code": "color",
									"label": "Màu sắc",
									"values": [
										{
											"label": "Xanh olive",
											"value_index": 138,
											"swatch_data": { "value": "#808000" }
										},
										{
											"label": "Đỏ rượu vang",
											"value_index": 153,
											"swatch_data": { "value": "#7b1e2b" }
										}
									]
								}
							],
							"variants": [
								{
									"attributes": [
										{ "code": "color", "label": "Xanh olive", "value_index": 138 }
									],
									"product": {
										"sku": "WF-DR-010-OLV-S",
										"small_image": {
											"url": "https://example.com/media/wf-dr-010-olv-01.jpg"
										}
									}
								},
								{
									"attributes": [
										{ "code": "color", "label": "Đỏ rượu vang", "value_index": 153 }
									],
									"product": {
										"sku": "WF-DR-010-BUR-S",
										"small_image": {
											"url": "https://example.com/media/wf-dr-010-01.jpg"
										}
									}
								}
							]
						}
					]
				}
			}
		}`))
	}))
	defer srv.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"store_url": srv.URL,
		"sku":       "WF-DR-010",
	})
	res := handleMagentoPublicGetProduct(payload)
	var text string
	if contentArr, ok := res["content"].([]map[string]string); ok && len(contentArr) > 0 {
		text = contentArr[0]["text"]
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	prods := data["products"].(map[string]interface{})
	items := prods["items"].([]interface{})
	first := items[0].(map[string]interface{})

	colorImages, ok := first["color_images"].([]interface{})
	if !ok || len(colorImages) != 2 {
		t.Fatalf("expected 2 color_images, got: %v", first["color_images"])
	}

	colors, ok := first["colors"].([]interface{})
	if !ok || len(colors) != 2 {
		t.Fatalf("expected 2 colors, got: %v", first["colors"])
	}

	// Verify Xanh olive has olv image
	c0 := colors[0].(map[string]interface{})
	if c0["name"] != "Xanh olive" || c0["imageUrl"] != "https://example.com/media/wf-dr-010-olv-01.jpg" {
		t.Errorf("unexpected color[0]: %+v", c0)
	}
}

func TestHandleMagentoPublicGetProduct(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"products": {
					"items": [
						{
							"id": 100,
							"sku": "WF-SK-013",
							"name": "Chân váy midi Cleo",
							"stock_status": "IN_STOCK"
						}
					]
				}
			}
		}`))
	}))
	defer srv.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"store_url": srv.URL,
		"sku":       "WF-SK-013",
	})

	res := handleMagentoPublicGetProduct(payload)
	if errText, ok := res["error_message"].(string); ok && errText != "" {
		t.Fatalf("unexpected tool error: %s", errText)
	}
}

func TestHandleMagentoPublicGetOutfit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"products": {
					"items": [
						{
							"sku": "BLZ-01",
							"name": "Áo khoác Blazer",
							"crosssell_products": [
								{"id": 2, "sku": "PNT-01", "name": "Quần tây ống rộng", "stock_status": "IN_STOCK"},
								{"id": 3, "sku": "SHR-01", "name": "Áo sơ mi lụa", "stock_status": "IN_STOCK"}
							],
							"related_products": [
								{"id": 4, "sku": "BAG-01", "name": "Túi xách tay", "stock_status": "IN_STOCK"}
							],
							"upsell_products": [
								{"id": 5, "sku": "BLZ-02", "name": "Áo khoác blazer dạ", "stock_status": "IN_STOCK"}
							]
						}
					]
				}
			}
		}`))
	}))
	defer srv.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"store_url": srv.URL,
		"sku":       "BLZ-01",
		"type":      "crosssell",
	})

	res := handleMagentoPublicGetOutfit(payload)
	if errText, ok := res["error_message"].(string); ok && errText != "" {
		t.Fatalf("unexpected tool error: %s", errText)
	}
}

func TestHandleMagentoPublicGuestCart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"createEmptyCart": "guest-cart-hash-12345"
			}
		}`))
	}))
	defer srv.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"store_url": srv.URL,
		"action":    "create",
	})

	res := handleMagentoPublicGuestCart(payload)
	if errText, ok := res["error_message"].(string); ok && errText != "" {
		t.Fatalf("unexpected tool error: %s", errText)
	}
}
