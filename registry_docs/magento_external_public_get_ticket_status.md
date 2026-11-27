# magento_external_public_get_ticket_status

List the support tickets of a verified order with their progress and photo count. Calls `POST {store}/weaver/ticket/status` (module `DatumBridge_Chatbot`).

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Verify the order first; its `order.tickets` already lists tickets, so call this tool to refresh them.
- Copy the relevant `cards` entry verbatim inside a ```` ```support_ticket ```` block.
- If `count` is 0, tell the shopper the order has no complaint yet.

## When to call

"Khiếu nại của tôi tới đâu rồi?", "Kiểm tra ticket TK000123".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `count` | 1 | Number of tickets |
| `cards` | [{"ticket_number":"TK000123","status_label":"Đang xử lý","image_count":2,"timeline":[...]}] | Newest first |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Look the order up again |

## Cases

1. Ticket in progress → card with "Đang xử lý" as the current step.
2. No ticket → `count: 0`.
