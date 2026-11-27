# magento_external_public_lookup_order

Look up a shopper's order without any store token. The shopper proves ownership with the order number plus the email or phone number used on the order.

The tool calls `POST {store}/weaver/order/lookup`, served by the `DatumBridge_Chatbot` Magento module. The store must have that module installed with **Stores → Configuration → DatumBridge → AI Customer Service - Order Support → Enable Guest Order Lookup** set to Yes. No `credentials_json` is needed; when the gateway passes one, only its `base_url` is read.

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Call this tool when a shopper asks where their order is, when it arrives, or what is in it.
- Both `order_number` and `verification` are required. If the shopper gave only one, ask for the other first. Never guess or reuse a phone/email from another order.
- Show the result by copying `card` verbatim, as JSON, inside a ```` ```order_status ```` block, then add one or two short sentences (status and estimated delivery).
- Keep `order_access_token` for follow-up actions on the same order (change delivery address, size exchange). Never show it to the shopper.
- If `verified` is false, relay `message` politely. Do not say whether the order number exists.
- If `can_change_address` is false and the shopper wants a new address, explain `change_address_reason`. Order cancellation is not offered.

## When to call

"Đơn hàng #000000123 của tôi đang ở đâu?", "Khi nào tôi nhận được hàng?", "Kiểm tra đơn 123, sđt 0901234567".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_number` | yes | string | "#000000123" | — | Order number as the shopper typed it; `#`, spaces and missing leading zeros are accepted |
| `verification` | yes | string | "0901234567" | — | Email or phone used on the order; `+84`/`84` prefixes are accepted |
| `store_url` | no | string | "https://store.com" | — | Optional override. When omitted, the tool uses the connected store (`credentials_json.base_url`), then `MAGENTO_BASE_URL` |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `verified` | true | Ownership check passed |
| `order` | {...} | Order number, status, items (size/color/image), totals, masked recipient, address, tracking, timeline |
| `card` | {...} | Compact payload for the chatbox ```` ```order_status ```` block |
| `order.timeline` | [{"key":"shipping","label":"Đang giao","date":"16/10","status":"current"}] | Steps confirmed → packing → shipping → delivered |
| `order.estimated_delivery` | "18/10/2026" | Shipment date + configured days, or order date + configured days before shipping |
| `order.can_change_address` | false | True only before packing starts |
| `order_access_token` | "dbo1...." | Proof of verification for follow-up actions, valid ~30 minutes |

## Error codes

| `error_code` | Meaning | What to tell the shopper |
|---|---|---|
| `invalid_input` | Missing order number, or the email/phone is malformed | Ask them to re-check the details |
| `not_found_or_mismatch` | No order matches both values | Ask them to re-check the order number and the email/phone on the order |
| `rate_limited` | 5 wrong attempts for this order or this email/phone within 30 minutes | Ask them to try again later or contact support |
| tool error "not available on this store" | Module missing or disabled | Apologise; order lookup is unavailable |

## Cases

1. Shopper: "Đơn #000000123 của tôi đâu rồi?" → ask for the phone or email used on the order → call with both → reply with the ```` ```order_status ```` block.
2. Shopper gives a wrong phone → `verified: false` → relay `message`; do not hint that the order exists.
3. Shopper asks to change the address of a shipped order → `can_change_address: false` → explain `change_address_reason`.
