# magento_external_public_my_order_detail

Details of one order belonging to the logged-in shopper, without asking for phone or email. Same result as `magento_external_public_lookup_order`, including `order_access_token` for follow-up actions. Calls `POST {store}/weaver/customer/order` (module `DatumBridge_Chatbot`).

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Use only when the conversation has a `customer_access_token` context line. Copy the token exactly.
- Copy `card` verbatim inside a ```` ```order_status ```` block.
- Keep `order_access_token` for `magento_external_public_update_delivery` and the exchange tools; never show it.
- An order of another customer answers `not_found`, the same as a missing one.

## When to call

A logged-in shopper picks an order from the list or names one: "Xem đơn 000000123", "Đơn mới nhất của tôi đang ở đâu?".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `customer_access_token` | yes | string | "dbc1...." | — | Signed customer token from the conversation context |
| `order_number` | yes | string | "000000123" | magento_external_public_my_orders | Order number |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `order` | {...} | Status, items, timeline, ETA, address, exchanges |
| `card` | {...} | ```` ```order_status ```` card |
| `order_access_token` | "dbo1...." | For delivery changes and size exchange |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Token missing, forged, expired or revoked by logout |
| `not_found` | No such order in this account |
| `invalid_input` | No order number |

## Cases

1. "Đơn mới nhất của tôi đâu?" → `magento_external_public_my_orders` → this tool with the first order number.
2. Shopper asks for an order of another account → `not_found`.
