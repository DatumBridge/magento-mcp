# magento_external_public_get_ticket_options

Complaint, step 1. For a verified order, returns the complaint categories, the order's products and the tickets already opened. Calls `POST {store}/weaver/ticket/options` (module `DatumBridge_Chatbot`).

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Verify the order first (`magento_external_public_lookup_order`, or `magento_external_public_my_order_detail` for logged-in shoppers) and reuse its `order_access_token`.
- Copy `card` verbatim inside a ```` ```ticket_options ```` block. The shopper picks a category, the product when needed, and describes the problem.
- If `existing_tickets` already has an open ticket for the same problem, show it instead of opening a new one.

## When to call

"Áo bị lỗi", "Tôi nhận thiếu hàng", "Giao sai sản phẩm", "Tôi muốn khiếu nại".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `card.categories` | [{"value":"defective","label":"Sản phẩm lỗi/hư hỏng","needs_item":true}] | Complaint types |
| `card.items` | [{"item_id":3,"name":"Áo blazer","sku":"BLZ-M-BE","size":"M","color":"Be"}] | Order products |
| `existing_tickets` | [{"ticket_number":"TK000001","status_label":"Đang xử lý"}] | Tickets already opened |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Look the order up again |

## Cases

1. "Áo bị bung chỉ" → verify the order → this tool → ```` ```ticket_options ```` form.
2. The order already has an open ticket for the same issue → show that ticket's status.
