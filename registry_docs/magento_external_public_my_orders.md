# magento_external_public_my_orders

Order history of the shopper who is logged in on the store website, newest first. No order number, phone or store token is needed: the signed `customer_access_token` identifies the customer. Calls `POST {store}/weaver/customer/orders` (module `DatumBridge_Chatbot`).

The token is minted by Magento at login (cookie `weaver_customer_token`), forwarded by the chatbox, and added to the conversation by cms-backend as a context line: `[Khách đã đăng nhập — customer_access_token: dbc1…]`. Logout revokes it.

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Use only when the conversation has a `customer_access_token` context line. Copy the token exactly; never invent or modify it.
- No token: the shopper is a guest. Use `magento_external_public_lookup_order` (order number + email/phone).
- Copy `card` verbatim inside a ```` ```order_list ```` block. To open an order, call `magento_external_public_my_order_detail`.
- On `token_invalid`, ask the shopper to log in again, or fall back to the guest lookup.

## When to call

"Đơn hàng của tôi", "Tôi đã đặt những đơn nào?", "Đơn gần nhất của tôi đâu rồi?" from a logged-in shopper.

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `customer_access_token` | yes | string | "dbc1...." | — | Signed customer token from the conversation context |
| `page` | no | integer | 1 | — | Page number |
| `page_size` | no | integer | 5 | — | Orders per page (max 10) |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `total_count` | 7 | Orders in the account |
| `orders` | [{"order_number":"000000123","status_label":"Đang giao","total":1790000,"item_count":1,"first_item":{...},"estimated_delivery":"18/10/2026"}] | Newest first |
| `card` | {...} | ```` ```order_list ```` card |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Token missing, forged, expired or revoked by logout |

## Cases

1. Logged-in shopper: "đơn hàng của tôi" → list of 5 newest orders.
2. The shopper logged out → `token_invalid` → ask for order number + email/phone.
