# magento_external_public_add_to_cart

Add a product to a guest cart without authentication via Magento 2 GraphQL. Creates a new guest cart automatically when `cart_id` is omitted.

This tool is public and does not require authentication or API keys.

## Audience

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Call this tool when a guest shopper asks to add a product to the cart.
- `sku` is required. Copy it exactly from the product the shopper chose, e.g. "Thêm mẫu WF-SK-013 vào giỏ" → `{"sku": "WF-SK-013"}`.
- Pass `cart_id` when the conversation already has one; otherwise omit it and reuse the `cart_id` returned for later calls.
- Does not require credentials_json or OAuth tokens.

## When to call

A customer asks: "Thêm mẫu này vào giỏ hàng", "Cho mình 2 cái size M".

## Parameters

| Name | Required | Type | Sample | Meaning |
|---|---|---|---|---|
| `sku` | yes | string | "WF-SK-013" | Product SKU to add |
| `cart_id` | no | string | "guest-cart-hash" | Existing guest cart ID; omitted → a new cart is created |
| `qty` | no | number | 1 | Quantity to add (default 1) |
| `store_url` | no | string | "https://store.com" | Optional override. When omitted, the tool uses the Magento store connected under Account → Integrations (`credentials_json.base_url`), then `MAGENTO_BASE_URL`. |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `cart_id` | "guest-cart-hash" | Cart the item was added to |
| `cart_created` | true | Whether a new cart was created by this call |
| `cart` | {...} | Cart contents after adding |
| `user_errors` | [{"message": "..."}] | Magento errors, e.g. out of stock (only when present) |
