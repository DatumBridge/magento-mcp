# magento_external_public_search_products

Search and list products for guest shoppers without authentication via Magento 2 GraphQL. Supports keyword search, category filter, price range, and returns product attributes with website stock status (IN_STOCK / OUT_OF_STOCK).

This tool is public and does not require authentication or API keys.

## Audience

| Who | Token |
|---|---|
| anonymous | none |

## Agent rules

- Call this tool when an unauthenticated guest shopper asks to search, browse, or filter products by name, keyword, category, or price range.
- Does not require credentials_json or OAuth tokens.
- `search` is required. Always fill it with the product-type keyword from the shopper's message, e.g. "Tìm áo blazer" → `{"search": "blazer"}`, "Có chân váy nào không?" → `{"search": "chân váy"}`.
- Magento requires every word of `search` to match in the same field, so put adjectives, occasions and colors in `style`: "set chân váy duyên dáng" → `{"search": "set chân váy", "style": "duyên dáng"}`.
- If nothing matches every word, the tool retries with the product-type keywords and returns `search_meta.relaxed: true` with `search_used`. Tell the shopper these are the closest matches. Items with `matches_style: true` also match the descriptive words in their name or description; show them first.
- Pass `search: ""` only when filtering purely by `category_id` or price; a call with no filter at all returns an error.
- Do not use this tool to view one product whose SKU is known (e.g. "Xem chi tiết ... (SKU: WF-TP-004)"); call `magento_external_public_get_product` with `sku` instead.

## When to call

A customer asks to find or explore products ("Tìm áo blazer", "Có váy nào dưới 1 triệu không?", "Xem danh mục đồ công sở").

## Parameters

| Name | Required | Type | Sample | Meaning |
|---|---|---|---|---|
| `search` | yes | string | "blazer" | Search keyword matching product name or description |
| `style` | no | string | "duyên dáng" | Descriptive words (style, occasion, color, material) matched against name and description; matching items come first with `matches_style: true` |
| `category_id` | no | string | "20" | Filter by category ID |
| `price_from` | no | number | 200000 | Minimum price filter |
| `price_to` | no | number | 1500000 | Maximum price filter |
| `page_size` | no | integer | 20 | Number of products per page (default 20, max 50) |
| `current_page` | no | integer | 1 | Page number |
| `store_url` | no | string | "https://store.com" | Optional override. When omitted, the tool uses the Magento store connected under Account → Integrations (`credentials_json.base_url`), then `MAGENTO_BASE_URL`. |

## Success fields

| Name | Sample | Meaning |
|---|---|---|
| `products` | {"total_count": 10, "items": [...]} | Search results matching criteria |
| `search_meta` | {"requested_search": "set chân váy duyên dáng", "search_used": "set chân váy", "relaxed": true, "style_terms": ["duyên", "dáng"], "style_matched": 0} | How the query was relaxed and ranked |
