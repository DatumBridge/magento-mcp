package tools

import (
	"encoding/json"

	"github.com/datumbridge/magento-mcp/internal/mcp"
)

const (
	customerOrdersPath = "/weaver/customer/orders"
	customerOrderPath  = "/weaver/customer/order"
)

const customerTokenParamDesc = "REQUIRED. `customer_access_token` (starts with \"dbc1.\") from the conversation context line \"[Khách đã đăng nhập … customer_access_token: …]\". Copy it exactly; never invent one. Without it, use magento_external_public_lookup_order instead."

func registerPublicCustomerTools(add toolAdder) {
	add(
		"magento_external_public_my_orders",
		"Order history of the LOGGED-IN shopper, newest first, without asking for order number or phone. "+
			"Only usable when the conversation contains a customer_access_token (the shopper is logged in on the website). "+
			"Show the result by copying `card` verbatim inside a ```order_list ... ``` block. To open one order call magento_external_public_my_order_detail.",
		publicProps(map[string]interface{}{
			"customer_access_token": map[string]interface{}{"type": "string", "description": customerTokenParamDesc},
			"page":                  map[string]interface{}{"type": "integer", "description": "Page number (default 1)."},
			"page_size":             map[string]interface{}{"type": "integer", "description": "Orders per page (default 5, max 10)."},
		}),
		[]string{"customer_access_token"},
		handleMagentoPublicMyOrders,
	)

	add(
		"magento_external_public_my_order_detail",
		"Details of one order of the LOGGED-IN shopper (identified by customer_access_token), without asking for phone or email. "+
			"Returns the same result as magento_external_public_lookup_order, including `order_access_token` for changing the delivery address or exchanging a size. "+
			"Show it by copying `card` verbatim inside a ```order_status ... ``` block.",
		publicProps(map[string]interface{}{
			"customer_access_token": map[string]interface{}{"type": "string", "description": customerTokenParamDesc},
			"order_number":          map[string]interface{}{"type": "string", "description": "REQUIRED. Order number from the shopper or from magento_external_public_my_orders, e.g. \"000000123\"."},
		}),
		[]string{"customer_access_token", "order_number"},
		handleMagentoPublicMyOrderDetail,
	)
}

func customerTokenFailure(resp map[string]interface{}) map[string]interface{} {
	out := failureResult(resp)
	if resp["error_code"] == "token_invalid" {
		out["next_step"] = "The login session expired. Ask the shopper to log in again on the website, or look the order up with magento_external_public_lookup_order (order number + email/phone)."
	}
	return out
}

func handleMagentoPublicMyOrders(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token := strArg(m, "customer_access_token")
	if token == "" {
		return mcp.ToolResultError("customer_access_token is required; the shopper is not logged in, use magento_external_public_lookup_order instead")
	}
	resp, errResult := callOrderEndpoint(args, customerOrdersPath, map[string]interface{}{
		"customer_access_token": token,
		"page":                  intArg(m, "page", 1),
		"page_size":             intArg(m, "page_size", 5),
	})
	if errResult != nil {
		return errResult
	}
	if ok, _ := resp["success"].(bool); !ok {
		return jsonResult(customerTokenFailure(resp))
	}
	card := map[string]interface{}{
		"total_count": resp["total_count"],
		"page":        resp["page"],
		"page_size":   resp["page_size"],
		"orders":      resp["orders"],
	}
	return jsonResult(map[string]interface{}{
		"success":     true,
		"total_count": resp["total_count"],
		"orders":      resp["orders"],
		"card":        card,
	})
}

func handleMagentoPublicMyOrderDetail(args json.RawMessage) map[string]interface{} {
	m := argMap(args)
	token := strArg(m, "customer_access_token")
	orderNumber := strArg(m, "order_number")
	if token == "" {
		return mcp.ToolResultError("customer_access_token is required; the shopper is not logged in, use magento_external_public_lookup_order instead")
	}
	if orderNumber == "" {
		return mcp.ToolResultError("order_number is required; call magento_external_public_my_orders to list the shopper's orders")
	}
	resp, errResult := callOrderEndpoint(args, customerOrderPath, map[string]interface{}{
		"customer_access_token": token,
		"order_number":          orderNumber,
	})
	if errResult != nil {
		return errResult
	}
	order, _ := resp["order"].(map[string]interface{})
	if ok, _ := resp["success"].(bool); !ok || order == nil {
		return jsonResult(customerTokenFailure(resp))
	}
	return jsonResult(map[string]interface{}{
		"verified":           true,
		"order":              order,
		"card":               orderStatusCard(order),
		"order_access_token": resp["order_access_token"],
		"token_expires_at":   resp["token_expires_at"],
	})
}
