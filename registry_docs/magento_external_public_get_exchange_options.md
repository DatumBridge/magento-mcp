# magento_external_public_get_exchange_options

Size exchange, step 1. For a verified order line, returns the sizes of the same product and colour with stock, the exchange methods and the brand's exchange conditions evaluated for this item. Calls `POST {store}/weaver/exchange/options` (module `DatumBridge_Chatbot`).

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Verify the order first with `magento_external_public_lookup_order` and reuse its `order_access_token`.
- If the result has `need_item: true`, ask which product to exchange and call again with `item`.
- Copy `card` verbatim inside a ```` ```exchange_options ```` block. Explain any condition with `status: "fail"`.
- The condition "còn nguyên tem, chưa qua sử dụng" has `status: "confirm"`: the shopper must confirm it before `magento_external_public_create_exchange`.
- Method `at_delivery` is available only before the parcel is delivered.

## When to call

"Tôi muốn đổi size", "Áo này có size L không, đổi được không?".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |
| `item` | no | string | "BLZ-M-BE" | magento_external_public_lookup_order | SKU, order item ID or part of the product name |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `eligible` | true | Every checkable condition passes and other sizes exist |
| `card.sizes` | [{"label":"L","sku":"BLZ-L-BE","in_stock":true,"current":false}] | Sizes in the same colour |
| `card.methods` | [{"value":"at_delivery","available":true}] | Exchange methods |
| `card.conditions` | [{"label":"Trong vòng 7 ngày kể từ ngày nhận hàng","status":"pass"}] | Policy checklist: pass, fail or confirm |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Look the order up again |
| `not_allowed` | Order canceled |
| `not_found` | No such product in the order |

## Cases

1. One-product order → card with sizes S/M/L/XL, current size marked, out-of-stock sizes disabled.
2. Delivered 10 days ago with a 7-day window → `eligible: false`, window condition fails.
3. Two-product order without `item` → `need_item: true` → ask the shopper.
