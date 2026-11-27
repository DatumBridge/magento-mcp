package tools

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/datumbridge/magento-mcp/internal/mcp"
)

// orderLookupPath is served by the DatumBridge_Chatbot Magento module. It is
// public by design: the shopper proves ownership with the email or phone on the
// order, so no store token is needed and the gateway does not inject one.
const (
	orderLookupPath         = "/weaver/order/lookup"
	orderUpdateDeliveryPath = "/weaver/order/updateDelivery"
)

const tokenParamDesc = "REQUIRED. `order_access_token` returned by magento_external_public_lookup_order for this order. If it is missing or expired, call the lookup again."

func registerPublicOrderTools(add toolAdder) {
	add(
		"magento_external_public_lookup_order",
		"Look up a shopper's order without any store token. The shopper must give the order number AND the email or phone number used on that order; ask for whichever is missing before calling. "+
			"Returns status, items (size/color), delivery timeline (Đã xác nhận → Đang đóng gói → Đang giao → Dự kiến đến), estimated delivery date, tracking numbers, the delivery address, and whether the address can still be changed. "+
			"Show the result by copying the `card` field verbatim, as JSON, inside a ```order_status ... ``` block, then add one or two sentences. "+
			"Keep `order_access_token` for follow-up actions on this order; never show it to the shopper. "+
			"If `verified` is false, relay `message` politely and do not reveal whether the order exists.",
		publicProps(map[string]interface{}{
			"order_number": map[string]interface{}{"type": "string", "description": "REQUIRED. Order number exactly as the shopper typed it, e.g. \"#000000123\" or \"123\"."},
			"verification": map[string]interface{}{"type": "string", "description": "REQUIRED. Email address or phone number the shopper used when placing the order, e.g. \"0901234567\" or \"an@example.com\"."},
		}),
		[]string{"order_number", "verification"},
		handleMagentoPublicLookupOrder,
	)

	add(
		"magento_external_public_update_delivery",
		"Change the delivery address and/or add a delivery note for an order the shopper already verified with magento_external_public_lookup_order. "+
			"The address can only change before the warehouse starts packing (lookup `can_change_address`). Order cancellation is not supported. "+
			"Before changing the address, read the full new address back to the shopper and get a yes, unless they submitted it from the chat form. "+
			"On success, show the updated order by copying `card` verbatim inside a ```order_status ... ``` block.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
			"street":             map[string]interface{}{"type": "string", "description": "New house number, street and ward, e.g. \"456 Lê Lợi, P. Bến Nghé\". Omit to only add a note."},
			"district":           map[string]interface{}{"type": "string", "description": "New district (quận/huyện), e.g. \"Quận 1\". Required with street."},
			"province":           map[string]interface{}{"type": "string", "description": "New province or city (tỉnh/thành phố), e.g. \"TP.HCM\". Required with street."},
			"note":               map[string]interface{}{"type": "string", "description": "Optional delivery note for the shipper, e.g. \"Giao giờ hành chính\" (max 500 characters)."},
		}),
		[]string{"order_access_token"},
		handleMagentoPublicUpdateDelivery,
	)
}

// callOrderEndpoint POSTs to a DatumBridge_Chatbot storefront endpoint. It
// returns the decoded body, or a tool error result when the call itself failed
// (module missing, server error, unreadable answer).
func callOrderEndpoint(args json.RawMessage, path string, body map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	client, _, err := graphqlClientFrom(args)
	if err != nil {
		return nil, mcp.ToolResultError(err.Error())
	}
	c, cancel := ctx()
	defer cancel()
	status, raw, err := client.PostStorefrontJSON(c, path, body)
	if err != nil {
		return nil, mcp.ToolResultError(err.Error())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(raw, &resp); err != nil {
		if status == http.StatusNotFound {
			return nil, mcp.ToolResultError("order support is not available on this store: install and enable the DatumBridge_Chatbot module (Stores → Configuration → DatumBridge → AI Customer Service)")
		}
		return nil, mcp.ToolResultError(fmt.Sprintf("store http %d: unexpected response", status))
	}
	if status >= 500 || resp["error_code"] == "disabled" {
		return nil, mcp.ToolResultError(fmt.Sprintf("%v", resp["message"]))
	}
	return resp, nil
}

// failureResult relays a business failure (not verified, not allowed, token
// expired...) as a normal result so the agent explains it to the shopper.
func failureResult(resp map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{
		"success":    false,
		"error_code": resp["error_code"],
		"message":    resp["message"],
	}
	if resp["error_code"] == "token_invalid" {
		out["next_step"] = "Call magento_external_public_lookup_order again with the order number and the email/phone the shopper already gave, then retry."
	}
	return out
}

func handleMagentoPublicLookupOrder(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	orderNumber := strArg(m, "order_number")
	verification := firstStrArg(m, "verification", "phone", "email")
	if orderNumber == "" || verification == "" {
		return mcp.ToolResultError("order_number and verification (email or phone used on the order) are required; ask the shopper for the missing one")
	}

	resp, errResult := callOrderEndpoint(args, orderLookupPath, map[string]interface{}{
		"order_number": orderNumber,
		"verification": verification,
	})
	if errResult != nil {
		return errResult
	}
	order, _ := resp["order"].(map[string]interface{})
	if verified, _ := resp["verified"].(bool); !verified || order == nil {
		out := failureResult(resp)
		out["verified"] = false
		delete(out, "success")
		return jsonResult(out)
	}

	return jsonResult(map[string]interface{}{
		"verified":           true,
		"order":              order,
		"card":               orderStatusCard(order),
		"order_access_token": resp["order_access_token"],
		"token_expires_at":   resp["token_expires_at"],
	})
}

func handleMagentoPublicUpdateDelivery(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token := strArg(m, "order_access_token")
	if token == "" {
		return mcp.ToolResultError("order_access_token is required: verify the order with magento_external_public_lookup_order first")
	}
	street, district, province, note := strArg(m, "street"), strArg(m, "district"), strArg(m, "province"), strArg(m, "note")
	if street == "" && district == "" && province == "" && note == "" {
		return mcp.ToolResultError("give a new address (street, district, province) and/or a note")
	}

	resp, errResult := callOrderEndpoint(args, orderUpdateDeliveryPath, map[string]interface{}{
		"order_access_token": token,
		"street":             street,
		"district":           district,
		"province":           province,
		"note":               note,
	})
	if errResult != nil {
		return errResult
	}
	order, _ := resp["order"].(map[string]interface{})
	if ok, _ := resp["success"].(bool); !ok || order == nil {
		return jsonResult(failureResult(resp))
	}

	card := orderStatusCard(order)
	card["notice"] = resp["message"]
	return jsonResult(map[string]interface{}{
		"success": true,
		"changes": resp["changes"],
		"message": resp["message"],
		"card":    card,
	})
}

func argMap(args json.RawMessage) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal(args, &m); err != nil || m == nil {
		m = map[string]interface{}{}
	}
	return m
}

// orderStatusCard is the payload the chatbox renders for a ```order_status``` block.
// Keep it small: the agent copies it verbatim into its reply.
func orderStatusCard(order map[string]interface{}) map[string]interface{} {
	items := []interface{}{}
	if raw, ok := order["items"].([]interface{}); ok {
		for _, it := range raw {
			item, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			items = append(items, map[string]interface{}{
				"name":      item["name"],
				"sku":       item["sku"],
				"size":      item["size"],
				"color":     item["color"],
				"qty":       item["qty"],
				"price":     item["price"],
				"image_url": item["image_url"],
			})
		}
	}

	card := map[string]interface{}{
		"order_number":       order["order_number"],
		"status_label":       order["status_label"],
		"current_step":       order["current_step"],
		"is_canceled":        order["is_canceled"],
		"created_at":         order["created_at"],
		"items":              items,
		"total":              order["grand_total"],
		"currency":           order["currency"],
		"timeline":           order["timeline"],
		"estimated_delivery": order["estimated_delivery"],
		"tracking":           order["tracking"],
		"can_change_address": order["can_change_address"],
	}
	if reason := order["change_address_reason"]; reason != nil {
		card["change_address_reason"] = reason
	}
	if ship, ok := order["shipping"].(map[string]interface{}); ok {
		card["shipping"] = map[string]interface{}{
			"method":         ship["method"],
			"recipient_name": ship["recipient_name"],
			"phone":          ship["phone"],
			"address":        ship["full_address"],
		}
	}
	return card
}
