# magento_external_public_create_ticket

Complaint, step 2. Creates a support ticket (`#TK000123`) stored in the Magento table `datumbridge_support_ticket` and adds a staff-only note to the order. The returned card includes a per-ticket upload URL and token, so the chatbox can send up to 5 photos (JPG/PNG/WEBP, 5MB each) straight to Magento. Photos are kept in `var/datumbridge/tickets` and are visible only to staff, in **Sales → AI Support Tickets**. Calls `POST {store}/weaver/ticket/create`.

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Verify the order first and reuse its `order_access_token`.
- `defective`, `missing_item` and `wrong_item` need the product: pass `item` when the order has several products.
- Copy `card` verbatim inside a ```` ```support_ticket ```` block. Never retype or shorten `card.upload`.
- Tell the shopper the ticket number and invite them to add photos with the card's upload button.
- Order cancellation and refunds are decided by staff; do not promise them.

## When to call

The shopper chose the category and described the problem (from the ```` ```ticket_options ```` form or in chat).

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |
| `category` | yes | string | "defective" | magento_external_public_get_ticket_options | Complaint type |
| `item` | sometimes | string | "BLZ-M-BE" | magento_external_public_get_ticket_options | Product concerned |
| `description` | yes | string | "Áo bị bung chỉ ở tay trái" | — | Problem description (≥ 5 chars) |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `ticket.ticket_number` | "TK000123" | Ticket number for the shopper |
| `ticket.timeline` | [{"key":"received","status":"current"}] | Progress |
| `card.upload` | {"url":"https://store/weaver/ticket/upload","token":"dbu1…","remaining":5} | Photo upload for the chatbox |

## Error codes

| `error_code` | Meaning |
|---|---|
| `invalid_input` | Unknown category, description too short, or product missing |
| `rate_limited` | Too many tickets today for this order |
| `token_invalid` | Look the order up again |

## Cases

1. Defective blazer → `TK000123` created → the shopper uploads 2 photos from the card.
2. Late delivery → no product needed → ticket created.
