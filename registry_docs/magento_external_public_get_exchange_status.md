# magento_external_public_get_exchange_status

List the size exchange requests of a verified order with their progress. Calls `POST {store}/weaver/exchange/status` (module `DatumBridge_Chatbot`).

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Verify the order first with `magento_external_public_lookup_order`; its `order.exchanges` already lists the requests, so call this tool only to refresh them.
- Copy the relevant `cards` entry verbatim inside a ```` ```exchange_request ```` block.
- If `count` is 0, tell the shopper the order has no exchange request yet.

## When to call

"Yêu cầu đổi size của tôi tới đâu rồi?", "Kiểm tra yêu cầu EX000123".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `count` | 1 | Number of requests |
| `cards` | [{"request_number":"EX000123","status_label":"Đang kiểm tra","timeline":[...]}] | One ```` ```exchange_request ```` card per request, newest first |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Look the order up again |

## Cases

1. A request in review → card with "Đang kiểm tra" as the current step.
2. No request → `count: 0`.
