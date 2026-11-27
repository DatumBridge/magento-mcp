package tools

import (
	"encoding/json"

	"github.com/datumbridge/magento-mcp/internal/mcp"
)

const (
	exchangeOptionsPath = "/weaver/exchange/options"
	exchangeCreatePath  = "/weaver/exchange/create"
	exchangeStatusPath  = "/weaver/exchange/status"
)

func registerPublicExchangeTools(add toolAdder) {
	add(
		"magento_external_public_get_exchange_options",
		"Size exchange step 1, for an order the shopper verified with magento_external_public_lookup_order. "+
			"Returns the sizes of the same product and colour (with stock), the exchange methods (at delivery / return and exchange) and the brand's exchange conditions with pass/fail for this item. "+
			"If the order has several products and the shopper did not say which one, pass nothing: the result lists `items` so you can ask. "+
			"Show the result by copying `card` verbatim inside a ```exchange_options ... ``` block.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
			"item":               map[string]interface{}{"type": "string", "description": "Which product to exchange: its SKU, order item ID or part of its name. Omit when the order has one product."},
		}),
		[]string{"order_access_token"},
		handleMagentoPublicGetExchangeOptions,
	)

	add(
		"magento_external_public_create_exchange",
		"Size exchange step 2: create the exchange request after the shopper chose the new size and the method. "+
			"Only call when the shopper confirmed the item is unused with tags on (`confirm_unused: true`). "+
			"Show the result by copying `card` verbatim inside a ```exchange_request ... ``` block, and tell the shopper the request number.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
			"item":               map[string]interface{}{"type": "string", "description": "SKU, order item ID or name of the product to exchange. Omit when the order has one product."},
			"new_size":           map[string]interface{}{"type": "string", "description": "REQUIRED. New size label from the options, e.g. \"L\"."},
			"method": map[string]interface{}{
				"type":        "string",
				"description": "REQUIRED. at_delivery = exchange when the parcel arrives (only before delivery, if the shipper supports it); return_ship = send the item back and receive the new size.",
				"enum":        []interface{}{"at_delivery", "return_ship"},
			},
			"confirm_unused": map[string]interface{}{"type": "boolean", "description": "REQUIRED true: the shopper confirmed the item is unused with tags on."},
			"note":           map[string]interface{}{"type": "string", "description": "Optional note from the shopper, e.g. \"Áo hơi rộng\"."},
		}),
		[]string{"order_access_token", "new_size", "method", "confirm_unused"},
		handleMagentoPublicCreateExchange,
	)

	add(
		"magento_external_public_get_exchange_status",
		"List the size exchange requests of an order the shopper verified with magento_external_public_lookup_order, with each request's progress "+
			"(Đã tạo yêu cầu → Đang kiểm tra → Chờ nhận hàng → Đổi size thành công). "+
			"Show the most relevant request by copying its `cards` entry verbatim inside a ```exchange_request ... ``` block.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
		}),
		[]string{"order_access_token"},
		handleMagentoPublicGetExchangeStatus,
	)
}

func handleMagentoPublicGetExchangeOptions(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token := strArg(m, "order_access_token")
	if token == "" {
		return mcp.ToolResultError("order_access_token is required: verify the order with magento_external_public_lookup_order first")
	}
	resp, errResult := callOrderEndpoint(args, exchangeOptionsPath, map[string]interface{}{
		"order_access_token": token,
		"item":               firstStrArg(m, "item", "sku"),
	})
	if errResult != nil {
		return errResult
	}
	items, _ := resp["items"].([]interface{})
	if ok, _ := resp["success"].(bool); !ok || len(items) == 0 {
		return jsonResult(failureResult(resp))
	}

	if len(items) > 1 {
		choices := []interface{}{}
		for _, raw := range items {
			if it, ok := raw.(map[string]interface{}); ok {
				choices = append(choices, map[string]interface{}{
					"item_id": it["item_id"], "name": it["name"], "sku": it["sku"],
					"color": it["color"], "current_size": it["current_size"],
				})
			}
		}
		return jsonResult(map[string]interface{}{
			"success":   true,
			"need_item": true,
			"items":     choices,
			"next_step": "Ask the shopper which product to exchange, then call again with `item`.",
		})
	}

	item, _ := items[0].(map[string]interface{})
	card := map[string]interface{}{"order_number": resp["order_number"]}
	for _, k := range []string{"item_id", "name", "sku", "color", "current_size", "price", "image_url", "sizes", "methods", "conditions", "eligible", "reason"} {
		card[k] = item[k]
	}
	return jsonResult(map[string]interface{}{
		"success":  true,
		"eligible": item["eligible"],
		"item":     item,
		"card":     card,
	})
}

func handleMagentoPublicCreateExchange(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token := strArg(m, "order_access_token")
	if token == "" {
		return mcp.ToolResultError("order_access_token is required: verify the order with magento_external_public_lookup_order first")
	}
	if !boolArg(m, "confirm_unused", false) {
		return jsonResult(map[string]interface{}{
			"success":    false,
			"error_code": "confirmation_required",
			"message":    "Bạn vui lòng xác nhận sản phẩm còn nguyên tem, chưa qua sử dụng để tạo yêu cầu đổi size.",
			"next_step":  "Ask the shopper to confirm the item is unused with tags on, then call again with confirm_unused=true.",
		})
	}
	newSize, method := strArg(m, "new_size"), strArg(m, "method")
	if newSize == "" || method == "" {
		return mcp.ToolResultError("new_size and method (at_delivery or return_ship) are required")
	}

	resp, errResult := callOrderEndpoint(args, exchangeCreatePath, map[string]interface{}{
		"order_access_token": token,
		"item":               firstStrArg(m, "item", "sku"),
		"new_size":           newSize,
		"method":             method,
		"note":               strArg(m, "note"),
	})
	if errResult != nil {
		return errResult
	}
	request, _ := resp["request"].(map[string]interface{})
	if ok, _ := resp["success"].(bool); !ok || request == nil {
		out := failureResult(resp)
		if conds, ok := resp["conditions"]; ok {
			out["conditions"] = conds
		}
		return jsonResult(out)
	}
	return jsonResult(map[string]interface{}{
		"success": true,
		"message": resp["message"],
		"request": request,
		"card":    request,
	})
}

func handleMagentoPublicGetExchangeStatus(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token := strArg(m, "order_access_token")
	if token == "" {
		return mcp.ToolResultError("order_access_token is required: verify the order with magento_external_public_lookup_order first")
	}
	resp, errResult := callOrderEndpoint(args, exchangeStatusPath, map[string]interface{}{"order_access_token": token})
	if errResult != nil {
		return errResult
	}
	if ok, _ := resp["success"].(bool); !ok {
		return jsonResult(failureResult(resp))
	}
	requests, _ := resp["requests"].([]interface{})
	return jsonResult(map[string]interface{}{
		"success":      true,
		"order_number": resp["order_number"],
		"count":        len(requests),
		"cards":        requests,
	})
}
