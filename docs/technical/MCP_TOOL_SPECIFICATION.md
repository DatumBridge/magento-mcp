# MCP Tool Specification — magento-mcp

| Field | Value |
|---|---|
| Folder | `magento-mcp` |
| mcpServer id | `magento-mcp` |
| Transport | Streamable HTTP `POST /mcp` |
| Protocol | `2024-11-05` |
| Health | `GET /health` |
| Default port | `8012` |
| Upstream | Magento 2 REST API (`/rest/V1/...`) |

## Tool visibility

Full scope by default (27 tools). `MAGENTO_TOOLS=tool1,tool2` narrows to an explicit allowlist.

## Tools (27)

| Name | Group | Method | Endpoint |
|---|---|---|---|
| `magento_external_list_products` | products | GET | `/products` |
| `magento_external_get_product` | products | GET | `/products/{sku}` |
| `magento_external_create_product` | products | POST | `/products` |
| `magento_external_update_product` | products | PUT | `/products/{sku}` |
| `magento_external_delete_product` | products | DELETE | `/products/{sku}` |
| `magento_external_list_product_types` | products | GET | `/products/types` |
| `magento_external_list_orders` | orders | GET | `/orders` |
| `magento_external_get_order` | orders | GET | `/orders/{id}` |
| `magento_external_add_order_comment` | orders | POST | `/orders/{id}/comments` |
| `magento_external_cancel_order` | orders | POST | `/orders/{id}/cancel` |
| `magento_external_hold_order` | orders | POST | `/orders/{id}/hold` (or `unhold`) |
| `magento_external_list_customers` | customers | GET | `/customers/search` |
| `magento_external_get_customer` | customers | GET | `/customers/{id}` |
| `magento_external_create_customer` | customers | POST | `/customers` |
| `magento_external_update_customer` | customers | PUT | `/customers/{id}` |
| `magento_external_get_stock_item` | inventory | GET | `/stockItems/{sku}` |
| `magento_external_update_stock_item` | inventory | PUT | `/products/{sku}/stockItems/{itemId}` |
| `magento_external_list_inventory_sources` | inventory | GET | `/inventory/sources` |
| `magento_external_list_categories` | categories | GET | `/categories` |
| `magento_external_get_category` | categories | GET | `/categories/{id}` |
| `magento_external_list_category_products` | categories | GET | `/categories/{id}/products` |
| `magento_external_assign_category_product` | categories | POST | `/categories/{id}/products` |
| `magento_external_list_invoices` | invoices_shipments | GET | `/invoices` |
| `magento_external_get_invoice` | invoices_shipments | GET | `/invoices/{id}` |
| `magento_external_create_shipment` | invoices_shipments | POST | `/order/{orderId}/ship` |
| `magento_external_list_store_views` | store | GET | `/store/storeViews` |
| `magento_external_list_store_configs` | store | GET | `/store/storeConfigs` |
