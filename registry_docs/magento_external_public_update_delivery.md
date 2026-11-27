# magento_external_public_update_delivery

Change the delivery address and/or add a delivery note for an order the shopper already verified. Calls `POST {store}/weaver/order/updateDelivery` (module `DatumBridge_Chatbot`). No store token is used: the `order_access_token` from `magento_external_public_lookup_order` proves the shopper owns the order.

## Audience

The MCP server decides this row. Weaver reads it on Setup and does not let an admin change it.

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Verify the order first with `magento_external_public_lookup_order` and reuse its `order_access_token`.
- The address can change only before packing (`can_change_address` from the lookup). Otherwise explain `change_address_reason`; a note is still allowed.
- Read the full new address back and get a yes before calling, unless the shopper sent it from the chat form ("Đơn #...: tôi muốn đổi địa chỉ giao hàng sang: ...").
- Order cancellation is not supported. Do not offer it.
- On success, copy `card` verbatim inside a ```` ```order_status ```` block.
- On `token_invalid`, run the lookup again with the details the shopper already gave, then retry.

## When to call

"Đổi địa chỉ giao hàng đơn #123 sang 456 Lê Lợi, Quận 1, TP.HCM", "Nhắn shipper gọi trước khi giao".

## Parameters

| Name | Required | Type | Sample | Related tool | Meaning |
|---|---|---|---|---|---|
| `order_access_token` | yes | string | "dbo1...." | magento_external_public_lookup_order | Proof of verification |
| `street` | no | string | "456 Lê Lợi, P. Bến Nghé" | — | House number, street, ward |
| `district` | with street | string | "Quận 1" | — | District |
| `province` | with street | string | "TP.HCM" | — | Province / city |
| `note` | no | string | "Giao giờ hành chính" | — | Note for the shipper (≤ 500 chars) |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `changes` | ["address","note"] | What was updated |
| `message` | "Đã cập nhật địa chỉ giao hàng." | Text for the shopper |
| `card` | {...} | Updated ```` ```order_status ```` card with a `notice` |

## Error codes

| `error_code` | Meaning |
|---|---|
| `token_invalid` | Token missing, forged or expired (~30 min): look the order up again |
| `not_allowed` | Order already packed/shipped/canceled; `message` says why |
| `invalid_input` | Address incomplete (needs street, district, province) or nothing to change |
| `rate_limited` | Address already changed 3 times today for this order |

## Cases

1. Order not packed: the shopper submits the form → address updated, the card shows the new address and a notice.
2. Order shipped: `not_allowed` → explain; offer to add a note instead.
3. Token expired: call the lookup again, then retry.
