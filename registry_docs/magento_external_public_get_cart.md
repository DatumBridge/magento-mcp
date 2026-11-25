# magento_external_public_get_cart

Retrieve guest cart contents and totals without authentication via Magento 2 GraphQL.

This tool is public and does not require authentication or API keys.

## Audience

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Call this tool when a guest shopper asks to view their cart.
- `cart_id` is required. Use the `cart_id` returned earlier by `magento_external_public_add_to_cart` or `magento_external_public_create_cart`.
- If the conversation has no `cart_id`, the shopper has no cart yet; do not call this tool.
- Does not require credentials_json or OAuth tokens.

## When to call

A customer asks: "Xem giỏ hàng của tôi", "Giỏ hàng có gì rồi?".

## Parameters

| Name | Required | Type | Sample | Meaning |
|---|---|---|---|---|
| `cart_id` | yes | string | "guest-cart-hash" | Guest cart ID |
| `store_url` | no | string | "https://store.com" | Optional override. When omitted, the tool uses the Magento store connected under Account → Integrations (`credentials_json.base_url`), then `MAGENTO_BASE_URL`. |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `cart_id` | "guest-cart-hash" | Guest cart ID |
| `cart` | {...} | Items, quantities, prices and grand total |
