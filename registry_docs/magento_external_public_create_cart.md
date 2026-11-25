# magento_external_public_create_cart

Create a new empty guest cart without authentication via Magento 2 GraphQL.

This tool is public and does not require authentication or API keys.

## Audience

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Call this tool only when a guest shopper explicitly asks for a new, empty cart.
- To add a product, call `magento_external_public_add_to_cart` directly; it creates a cart on its own when `cart_id` is omitted.
- Does not require credentials_json or OAuth tokens.

## When to call

A customer asks: "Tạo giỏ hàng mới".

## Parameters

| Name | Required | Type | Sample | Meaning |
|---|---|---|---|---|
| `store_url` | no | string | "https://store.com" | Optional override. When omitted, the tool uses the Magento store connected under Account → Integrations (`credentials_json.base_url`), then `MAGENTO_BASE_URL`. |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `cart_id` | "guest-cart-hash" | ID of the new guest cart |
