# magento_external_public_create_exchange

Size exchange, step 2. Creates an exchange request (`#EX000123`) for a verified order line, stored in the Magento table `datumbridge_exchange_request`, and logs it on the order for staff. Calls `POST {store}/weaver/exchange/create` (module `DatumBridge_Chatbot`).

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Call `magento_external_public_get_exchange_options` first, so the shopper picks from real sizes and methods.
- Pass `confirm_unused: true` only after the shopper confirmed the item is unused with tags on.
- Copy `card` verbatim inside a ```` ```exchange_request ```` block and give the request number.
- Each order line can be exchanged once. Staff move the request through Đang kiểm tra → Chờ nhận hàng → Đổi size thành công in Magento admin (Sales → AI Size Exchanges).

## When to call

The shopper picked the new size and method: "Đổi sang size L, đổi khi nhận hàng giúp tôi".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |
| `item` | no | string | "BLZ-M-BE" | magento_external_public_get_exchange_options | Product to exchange (omit for single-product orders) |
| `new_size` | yes | string | "L" | magento_external_public_get_exchange_options | New size label |
| `method` | yes | string | "at_delivery" | magento_external_public_get_exchange_options | at_delivery or return_ship |
| `confirm_unused` | yes | boolean | true | — | Shopper confirmed unused, tags on |
| `note` | no | string | "Áo hơi rộng" | — | Shopper note |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `request.request_number` | "EX000123" | Request number for the shopper |
| `request.timeline` | [{"key":"created","status":"current"}] | Progress steps |
| `card` | {...} | ```` ```exchange_request ```` card |

## Error codes

| `error_code` | Meaning |
|---|---|
| `confirmation_required` | Ask the shopper to confirm the item is unused with tags on |
| `not_allowed` | Condition failed (window, discounted, already exchanged), size out of stock or method unavailable |
| `invalid_input` | Unknown size, same size, or several products without `item` |
| `token_invalid` | Look the order up again |

## Cases

1. M → L, at delivery, before shipping → `EX000123` created with status "Đã tạo yêu cầu".
2. Second request for the same line → `not_allowed` ("Sản phẩm đã có yêu cầu đổi #EX...").
3. Size L out of stock → `not_allowed`; suggest the sizes still in stock.
