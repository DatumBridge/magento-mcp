package tools

import (
	"encoding/json"

	"github.com/datumbridge/magento-mcp/internal/mcp"
)

const (
	ticketOptionsPath = "/weaver/ticket/options"
	ticketCreatePath  = "/weaver/ticket/create"
	ticketStatusPath  = "/weaver/ticket/status"
)

func registerPublicTicketTools(add toolAdder) {
	add(
		"magento_external_public_get_ticket_options",
		"Complaint step 1, for an order the shopper verified (magento_external_public_lookup_order or magento_external_public_my_order_detail). "+
			"Returns the complaint categories (Sản phẩm lỗi/hư hỏng, Thiếu hàng, Giao hàng chậm, Sai sản phẩm, Hoàn tiền, Khác), the order's products and existing tickets. "+
			"Show the form by copying `card` verbatim inside a ```ticket_options ... ``` block.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
		}),
		[]string{"order_access_token"},
		handleMagentoPublicGetTicketOptions,
	)

	add(
		"magento_external_public_create_ticket",
		"Complaint step 2: create a support ticket (#TK000123) for a verified order. "+
			"Categories defective, missing_item and wrong_item need the product (`item`). "+
			"Show the result by copying `card` verbatim inside a ```support_ticket ... ``` block: the chatbox then lets the shopper upload up to 5 photos straight to the store. "+
			"Tell the shopper the ticket number and that staff will follow up.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "REQUIRED. defective (Sản phẩm lỗi/hư hỏng), missing_item (Thiếu hàng), late_delivery (Giao hàng chậm), wrong_item (Sai sản phẩm), refund (Hoàn tiền), other (Khác).",
				"enum":        []interface{}{"defective", "missing_item", "late_delivery", "wrong_item", "refund", "other"},
			},
			"item":        map[string]interface{}{"type": "string", "description": "SKU, order item ID or part of the product name. Required for defective, missing_item and wrong_item when the order has several products."},
			"description": map[string]interface{}{"type": "string", "description": "REQUIRED. The problem in the shopper's words, e.g. \"Áo bị bung chỉ ở tay trái\"."},
		}),
		[]string{"order_access_token", "category", "description"},
		handleMagentoPublicCreateTicket,
	)

	add(
		"magento_external_public_get_ticket_status",
		"List the support tickets of a verified order with their progress (Đã tiếp nhận → Đang xử lý → Chờ phản hồi khách → Đã giải quyết) and number of photos. "+
			"Show the relevant one by copying its `cards` entry verbatim inside a ```support_ticket ... ``` block.",
		publicProps(map[string]interface{}{
			"order_access_token": map[string]interface{}{"type": "string", "description": tokenParamDesc},
		}),
		[]string{"order_access_token"},
		handleMagentoPublicGetTicketStatus,
	)
}

func requireOrderToken(m map[string]interface{}) (string, map[string]interface{}) {
	token := strArg(m, "order_access_token")
	if token == "" {
		return "", mcp.ToolResultError("order_access_token is required: verify the order with magento_external_public_lookup_order (or magento_external_public_my_order_detail) first")
	}
	return token, nil
}

func handleMagentoPublicGetTicketOptions(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token, errResult := requireOrderToken(m)
	if errResult != nil {
		return errResult
	}
	resp, errResult := callOrderEndpoint(args, ticketOptionsPath, map[string]interface{}{"order_access_token": token})
	if errResult != nil {
		return errResult
	}
	if ok, _ := resp["success"].(bool); !ok {
		return jsonResult(failureResult(resp))
	}
	card := map[string]interface{}{
		"order_number": resp["order_number"],
		"categories":   resp["categories"],
		"items":        resp["items"],
	}
	return jsonResult(map[string]interface{}{
		"success":          true,
		"existing_tickets": resp["tickets"],
		"card":             card,
	})
}

func handleMagentoPublicCreateTicket(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token, errResult := requireOrderToken(m)
	if errResult != nil {
		return errResult
	}
	category, description := strArg(m, "category"), strArg(m, "description")
	if category == "" || description == "" {
		return mcp.ToolResultError("category and description are required")
	}
	resp, errResult := callOrderEndpoint(args, ticketCreatePath, map[string]interface{}{
		"order_access_token": token,
		"category":           category,
		"item":               firstStrArg(m, "item", "sku"),
		"description":        description,
	})
	if errResult != nil {
		return errResult
	}
	ticket, _ := resp["ticket"].(map[string]interface{})
	if ok, _ := resp["success"].(bool); !ok || ticket == nil {
		return jsonResult(failureResult(resp))
	}

	// The card carries the per-ticket upload URL and token for the chatbox only.
	card := map[string]interface{}{}
	for k, v := range ticket {
		card[k] = v
	}
	card["upload"] = resp["upload"]
	return jsonResult(map[string]interface{}{
		"success": true,
		"message": resp["message"],
		"ticket":  ticket,
		"card":    card,
	})
}

func handleMagentoPublicGetTicketStatus(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token, errResult := requireOrderToken(m)
	if errResult != nil {
		return errResult
	}
	resp, errResult := callOrderEndpoint(args, ticketStatusPath, map[string]interface{}{"order_access_token": token})
	if errResult != nil {
		return errResult
	}
	if ok, _ := resp["success"].(bool); !ok {
		return jsonResult(failureResult(resp))
	}
	tickets, _ := resp["tickets"].([]interface{})
	return jsonResult(map[string]interface{}{
		"success":      true,
		"order_number": resp["order_number"],
		"count":        len(tickets),
		"cards":        tickets,
	})
}
