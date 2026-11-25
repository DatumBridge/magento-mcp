package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/datumbridge/magento-mcp/internal/magento"
	"github.com/datumbridge/magento-mcp/internal/mcp"
)

func registerPublicCatalogTools(add toolAdder) {
	add(
		"magento_external_public_search_products",
		"Search and list products for guest shoppers without authentication via Magento GraphQL. Supports keyword search, category filter, price range, and returns product attributes with website stock status (IN_STOCK / OUT_OF_STOCK). Always pass `search` with the product keyword from the shopper's message (e.g. {\"search\": \"blazer\"}); `search` may be empty only when category_id or a price filter is given. To view one product whose SKU is known, call magento_external_public_get_product instead.",
		publicProps(map[string]interface{}{
			"search":       map[string]interface{}{"type": "string", "description": "REQUIRED. Product keyword taken from the shopper's message, e.g. \"blazer\", \"chân váy\", \"áo sơ mi trắng\". Never omit this field. Pass \"\" only when filtering purely by category_id or price."},
			"category_id":  map[string]interface{}{"type": "string", "description": "Filter by Magento category ID"},
			"price_from":   map[string]interface{}{"type": "number", "description": "Minimum price filter"},
			"price_to":     map[string]interface{}{"type": "number", "description": "Maximum price filter"},
			"page_size":    map[string]interface{}{"type": "integer", "description": "Number of products per page (default 20, max 50)"},
			"current_page": map[string]interface{}{"type": "integer", "description": "Current page number (default 1)"},
		}),
		[]string{"search"},
		handleMagentoPublicSearchProducts,
	)

	add(
		"magento_external_public_get_product",
		"Retrieve full product details by SKU for guest shoppers without authentication via Magento GraphQL. Includes description, image gallery, available sizes, colors, and stock status. Use this whenever the shopper's message contains a SKU, e.g. \"Xem chi tiết ... (SKU: WF-TP-004)\".",
		publicProps(map[string]interface{}{
			"sku": map[string]interface{}{"type": "string", "description": "Product SKU copied exactly from the message, e.g. \"WF-TP-004\""},
		}),
		[]string{"sku"},
		handleMagentoPublicGetProduct,
	)

	add(
		"magento_external_public_get_outfit",
		"Retrieve outfit styling recommendations (cross-sell items), similar alternatives (up-sell items), and accessories (related items) for a product SKU without authentication.",
		publicProps(map[string]interface{}{
			"sku": map[string]interface{}{"type": "string", "description": "Product SKU to find styling recommendations for"},
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Recommendation type: 'crosssell' (outfit pairing), 'upsell' (similar items), 'related' (accessories), or 'all' (default)",
				"enum":        []interface{}{"crosssell", "upsell", "related", "all"},
			},
		}),
		[]string{"sku"},
		handleMagentoPublicGetOutfit,
	)

	add(
		"magento_external_public_list_categories",
		"Retrieve category hierarchy for guest shoppers to explore styles and catalog tree without authentication.",
		publicProps(map[string]interface{}{
			"category_id": map[string]interface{}{"type": "string", "description": "Parent category ID (optional, defaults to root catalog)"},
		}),
		nil,
		handleMagentoPublicListCategories,
	)

	add(
		"magento_external_public_create_cart",
		"Create a new empty guest cart for shoppers without authentication via Magento GraphQL. Returns cart_id. Usually not needed: magento_external_public_add_to_cart creates a cart automatically when cart_id is omitted.",
		publicProps(map[string]interface{}{}),
		nil,
		handleMagentoPublicCreateCart,
	)

	add(
		"magento_external_public_add_to_cart",
		"Add a product to a guest cart without authentication via Magento GraphQL. Always pass the product `sku`. Pass `cart_id` to add to an existing cart; when omitted a new guest cart is created and its cart_id is returned.",
		publicProps(map[string]interface{}{
			"sku":     map[string]interface{}{"type": "string", "description": "REQUIRED. Product SKU to add, copied exactly from the product, e.g. \"WF-SK-013\""},
			"cart_id": map[string]interface{}{"type": "string", "description": "Existing guest cart ID. Omit to create a new cart automatically."},
			"qty":     map[string]interface{}{"type": "number", "description": "Quantity to add (default 1)"},
		}),
		[]string{"sku"},
		handleMagentoPublicAddToCart,
	)

	add(
		"magento_external_public_get_cart",
		"Retrieve guest cart contents and totals without authentication via Magento GraphQL. Always pass `cart_id` returned by magento_external_public_add_to_cart or magento_external_public_create_cart.",
		publicProps(map[string]interface{}{
			"cart_id": map[string]interface{}{"type": "string", "description": "REQUIRED. Guest cart ID returned when the cart was created"},
		}),
		[]string{"cart_id"},
		handleMagentoPublicGetCart,
	)
}

func handleMagentoPublicSearchProducts(args json.RawMessage) map[string]interface{} {
	client, m, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	search := firstStrArg(m, "search", "keyword", "query", "q", "name")
	categoryID := strArg(m, "category_id")
	priceFrom := floatArg(m, "price_from", -1)
	priceTo := floatArg(m, "price_to", -1)
	pageSize := intArg(m, "page_size", 20)
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	currentPage := intArg(m, "current_page", 1)
	if currentPage <= 0 {
		currentPage = 1
	}

	if search == "" && categoryID == "" && priceFrom < 0 && priceTo < 0 {
		return mcp.ToolResultError("search is required: retry with {\"search\": \"<product keyword from the shopper's message>\"} (or pass category_id / price_from / price_to). To view a product by SKU, call magento_external_public_get_product with {\"sku\": \"...\"}")
	}

	variables := map[string]interface{}{
		"pageSize":    pageSize,
		"currentPage": currentPage,
	}
	if search != "" {
		variables["search"] = search
	}

	filter := map[string]interface{}{}
	if categoryID != "" {
		filter["category_id"] = map[string]interface{}{"eq": categoryID}
	}
	if priceFrom >= 0 || priceTo >= 0 {
		priceFilter := map[string]interface{}{}
		if priceFrom >= 0 {
			priceFilter["from"] = fmt.Sprintf("%.2f", priceFrom)
		}
		if priceTo >= 0 {
			priceFilter["to"] = fmt.Sprintf("%.2f", priceTo)
		}
		filter["price"] = priceFilter
	}
	if len(filter) > 0 {
		variables["filter"] = filter
	}

	query := `query SearchProducts($search: String, $filter: ProductAttributeFilterInput, $pageSize: Int, $currentPage: Int) {
		products(search: $search, filter: $filter, pageSize: $pageSize, currentPage: $currentPage) {
			total_count
			page_info {
				current_page
				total_pages
			}
			items {
				id
				sku
				name
				stock_status
				only_x_left_in_stock
				url_key
				small_image {
					url
					label
				}
				price_range {
					minimum_price {
						regular_price {
							value
							currency
						}
						final_price {
							value
							currency
						}
						discount {
							amount_off
							percent_off
						}
					}
				}
			}
		}
	}`

	c, cancel := ctx()
	defer cancel()

	data, err := client.Query(c, query, variables)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err == nil {
		enrichGraphQLProductsResponse(resp, client.BaseURL())
		return jsonResult(resp)
	}
	return rawResult(data)
}

func handleMagentoPublicGetProduct(args json.RawMessage) map[string]interface{} {
	client, m, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	sku := strArg(m, "sku")
	if sku == "" {
		return mcp.ToolResultError("sku is required")
	}

	query := `query GetProduct($sku: String!) {
		products(filter: { sku: { eq: $sku } }) {
			items {
				id
				sku
				name
				stock_status
				only_x_left_in_stock
				url_key
				description {
					html
				}
				short_description {
					html
				}
				price_range {
					minimum_price {
						regular_price {
							value
							currency
						}
						final_price {
							value
							currency
						}
						discount {
							amount_off
							percent_off
						}
					}
				}
				media_gallery {
					url
					label
					disabled
				}
				... on ConfigurableProduct {
					configurable_options {
						id
						attribute_id
						label
						attribute_code
						values {
							value_index
							label
							swatch_data {
								value
							}
						}
					}
					variants {
						attributes {
							label
							code
							value_index
						}
						product {
							id
							sku
							name
							stock_status
							url_key
							image {
								url
								label
							}
							small_image {
								url
								label
							}
							media_gallery {
								url
								label
								disabled
							}
							price_range {
								minimum_price {
									final_price {
										value
										currency
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	c, cancel := ctx()
	defer cancel()

	data, err := client.Query(c, query, map[string]interface{}{"sku": sku})
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err == nil {
		enrichGraphQLProductsResponse(resp, client.BaseURL())
		return jsonResult(resp)
	}
	return rawResult(data)
}

func handleMagentoPublicGetOutfit(args json.RawMessage) map[string]interface{} {
	client, m, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	sku := strArg(m, "sku")
	if sku == "" {
		return mcp.ToolResultError("sku is required")
	}

	linkType := strings.ToLower(strArg(m, "type"))
	if linkType == "" {
		linkType = "all"
	}

	query := `query GetOutfit($sku: String!) {
		products(filter: { sku: { eq: $sku } }) {
			items {
				sku
				name
				crosssell_products {
					id
					sku
					name
					stock_status
					url_key
					small_image {
						url
						label
					}
					price_range {
						minimum_price {
							regular_price {
								value
								currency
							}
							final_price {
								value
								currency
							}
						}
					}
				}
				related_products {
					id
					sku
					name
					stock_status
					url_key
					small_image {
						url
						label
					}
					price_range {
						minimum_price {
							regular_price {
								value
								currency
							}
							final_price {
								value
								currency
							}
						}
					}
				}
				upsell_products {
					id
					sku
					name
					stock_status
					url_key
					small_image {
						url
						label
					}
					price_range {
						minimum_price {
							regular_price {
								value
								currency
							}
							final_price {
								value
								currency
							}
						}
					}
				}
			}
		}
	}`

	c, cancel := ctx()
	defer cancel()

	data, err := client.Query(c, query, map[string]interface{}{"sku": sku})
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	var resp struct {
		Products struct {
			Items []struct {
				SKU               string                   `json:"sku"`
				Name              string                   `json:"name"`
				CrosssellProducts []map[string]interface{} `json:"crosssell_products"`
				RelatedProducts   []map[string]interface{} `json:"related_products"`
				UpsellProducts    []map[string]interface{} `json:"upsell_products"`
			} `json:"items"`
		} `json:"products"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return rawResult(data)
	}

	if len(resp.Products.Items) == 0 {
		return jsonResult(map[string]interface{}{
			"sku":     sku,
			"message": "Product not found",
			"items":   []interface{}{},
		})
	}

	item := resp.Products.Items[0]
	for _, it := range item.CrosssellProducts {
		enrichGraphQLProductItem(it, client.BaseURL())
	}
	for _, it := range item.RelatedProducts {
		enrichGraphQLProductItem(it, client.BaseURL())
	}
	for _, it := range item.UpsellProducts {
		enrichGraphQLProductItem(it, client.BaseURL())
	}

	result := map[string]interface{}{
		"source_sku":  item.SKU,
		"source_name": item.Name,
	}

	switch linkType {
	case "crosssell":
		result["crosssell_outfit"] = item.CrosssellProducts
		result["total"] = len(item.CrosssellProducts)
	case "upsell":
		result["upsell_alternatives"] = item.UpsellProducts
		result["total"] = len(item.UpsellProducts)
	case "related":
		result["related_accessories"] = item.RelatedProducts
		result["total"] = len(item.RelatedProducts)
	default:
		result["crosssell_outfit"] = item.CrosssellProducts
		result["upsell_alternatives"] = item.UpsellProducts
		result["related_accessories"] = item.RelatedProducts
		result["total"] = len(item.CrosssellProducts) + len(item.UpsellProducts) + len(item.RelatedProducts)
	}

	return jsonResult(result)
}

func handleMagentoPublicListCategories(args json.RawMessage) map[string]interface{} {
	client, m, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	catID := strArg(m, "category_id")
	variables := map[string]interface{}{}
	query := `query GetCategories {
		categoryList {
			id
			name
			url_path
			product_count
			children {
				id
				name
				url_path
				product_count
				children {
					id
					name
					url_path
					product_count
				}
			}
		}
	}`

	if catID != "" {
		variables["id"] = catID
		query = `query GetCategories($id: String) {
			categoryList(filters: { ids: { eq: $id } }) {
				id
				name
				url_path
				product_count
				children {
					id
					name
					url_path
					product_count
					children {
						id
						name
						url_path
						product_count
					}
				}
			}
		}`
	}

	c, cancel := ctx()
	defer cancel()

	data, err := client.Query(c, query, variables)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}
	return rawResult(data)
}

func createGuestCart(client *magento.GraphQLClient) (string, error) {
	c, cancel := ctx()
	defer cancel()
	data, err := client.Query(c, `mutation CreateCart { createEmptyCart }`, nil)
	if err != nil {
		return "", err
	}
	var resp struct {
		CreateEmptyCart string `json:"createEmptyCart"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || resp.CreateEmptyCart == "" {
		return "", fmt.Errorf("could not create guest cart: %s", string(data))
	}
	return resp.CreateEmptyCart, nil
}

func handleMagentoPublicCreateCart(args json.RawMessage) map[string]interface{} {
	client, _, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}
	cartID, err := createGuestCart(client)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}
	return jsonResult(map[string]interface{}{
		"cart_id": cartID,
		"message": "Guest cart created successfully",
	})
}

func handleMagentoPublicAddToCart(args json.RawMessage) map[string]interface{} {
	client, m, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	sku := strArg(m, "sku")
	if sku == "" {
		return mcp.ToolResultError("sku is required: retry with {\"sku\": \"<product SKU>\"}")
	}
	qty := floatArg(m, "qty", 1)
	if qty <= 0 {
		qty = 1
	}

	cartID := strArg(m, "cart_id")
	created := false
	if cartID == "" {
		cartID, err = createGuestCart(client)
		if err != nil {
			return mcp.ToolResultError(err.Error())
		}
		created = true
	}

	mutation := `mutation AddToCart($cartId: String!, $cartItems: [CartItemInput!]!) {
		addProductsToCart(cartId: $cartId, cartItems: $cartItems) {
			cart {
				id
				total_quantity
				items {
					id
					quantity
					product {
						sku
						name
					}
					prices {
						price {
							value
							currency
						}
					}
				}
			}
			user_errors {
				message
			}
		}
	}`

	variables := map[string]interface{}{
		"cartId": cartID,
		"cartItems": []map[string]interface{}{
			{"sku": sku, "quantity": qty},
		},
	}

	c, cancel := ctx()
	defer cancel()

	data, err := client.Query(c, mutation, variables)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	var resp struct {
		AddProductsToCart struct {
			Cart       map[string]interface{}   `json:"cart"`
			UserErrors []map[string]interface{} `json:"user_errors"`
		} `json:"addProductsToCart"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return rawResult(data)
	}
	result := map[string]interface{}{
		"cart_id":      cartID,
		"cart_created": created,
		"cart":         resp.AddProductsToCart.Cart,
	}
	if len(resp.AddProductsToCart.UserErrors) > 0 {
		result["user_errors"] = resp.AddProductsToCart.UserErrors
	}
	return jsonResult(result)
}

func handleMagentoPublicGetCart(args json.RawMessage) map[string]interface{} {
	client, m, err := graphqlClientFrom(args)
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}

	cartID := strArg(m, "cart_id")
	if cartID == "" {
		return mcp.ToolResultError("cart_id is required: retry with {\"cart_id\": \"...\"} from magento_external_public_add_to_cart or magento_external_public_create_cart")
	}

	query := `query GetCart($cartId: String!) {
		cart(cart_id: $cartId) {
			id
			is_virtual
			total_quantity
			items {
				id
				quantity
				product {
					sku
					name
					small_image {
						url
					}
				}
				prices {
					price {
						value
						currency
					}
				}
			}
			prices {
				grand_total {
					value
					currency
				}
			}
		}
	}`

	c, cancel := ctx()
	defer cancel()

	data, err := client.Query(c, query, map[string]interface{}{"cartId": cartID})
	if err != nil {
		return mcp.ToolResultError(err.Error())
	}
	var resp struct {
		Cart map[string]interface{} `json:"cart"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return rawResult(data)
	}
	return jsonResult(map[string]interface{}{"cart_id": cartID, "cart": resp.Cart})
}

func enrichGraphQLProductItem(item map[string]interface{}, baseURL string) {
	if item == nil {
		return
	}
	urlKey, _ := item["url_key"].(string)
	if urlKey != "" {
		item["url"] = formatProductPageURL(baseURL, urlKey)
		item["url_path"] = formatProductPageURL("", urlKey)
	}
	// Enrich top-level image_url from small_image.url if available
	if sm, ok := item["small_image"].(map[string]interface{}); ok {
		if u, ok := sm["url"].(string); ok && u != "" {
			item["image_url"] = u
		}
	}
	if _, hasImg := item["image_url"]; !hasImg {
		if im, ok := item["image"].(map[string]interface{}); ok {
			if u, ok := im["url"].(string); ok && u != "" {
				item["image_url"] = u
			}
		}
	}
	// Enrich variants if configurable
	if variants, ok := item["variants"].([]interface{}); ok {
		for _, v := range variants {
			if vm, ok := v.(map[string]interface{}); ok {
				if vprod, ok := vm["product"].(map[string]interface{}); ok {
					enrichGraphQLProductItem(vprod, baseURL)
				}
			}
		}
	}

	enrichGraphQLConfigurableOptions(item)
}

type gqlColorCollector struct {
	label     string
	vIdx      string
	rawIdx    interface{}
	hexCode   string
	mainImage string
	gallery   []string
	seen      map[string]bool
	vMap      map[string]interface{}
}

func enrichGraphQLConfigurableOptions(item map[string]interface{}) {
	if item == nil {
		return
	}
	rawOpts, ok := item["configurable_options"].([]interface{})
	if !ok || len(rawOpts) == 0 {
		return
	}

	var colorCollectors []*gqlColorCollector
	collectorsByVIdx := make(map[string]*gqlColorCollector)
	collectorsByLabel := make(map[string]*gqlColorCollector)
	var availableColors []string
	var availableSizes []string

	for _, optItem := range rawOpts {
		optMap, ok := optItem.(map[string]interface{})
		if !ok {
			continue
		}
		attrCode := strings.ToLower(fmt.Sprintf("%v", optMap["attribute_code"]))
		attrLabel := strings.ToLower(fmt.Sprintf("%v", optMap["label"]))

		// Size option
		if strings.Contains(attrCode, "size") || strings.Contains(attrLabel, "size") || strings.Contains(attrLabel, "kích thước") {
			if vals, ok := optMap["values"].([]interface{}); ok {
				for _, valItem := range vals {
					if vm, ok := valItem.(map[string]interface{}); ok {
						valLabel := strings.TrimSpace(fmt.Sprintf("%v", vm["label"]))
						if valLabel != "" {
							availableSizes = append(availableSizes, valLabel)
						}
					}
				}
			}
		}

		// Color option
		if strings.Contains(attrCode, "color") || strings.Contains(attrLabel, "color") || strings.Contains(attrLabel, "màu") {
			if vals, ok := optMap["values"].([]interface{}); ok {
				for _, valItem := range vals {
					vm, ok := valItem.(map[string]interface{})
					if !ok {
						continue
					}
					valLabel := strings.TrimSpace(fmt.Sprintf("%v", vm["label"]))
					if valLabel == "" {
						continue
					}
					vIdxStr := fmt.Sprintf("%v", vm["value_index"])

					var swatchHex string
					if sw, ok := vm["swatch_data"].(map[string]interface{}); ok {
						if val, ok := sw["value"].(string); ok && strings.HasPrefix(strings.TrimSpace(val), "#") {
							swatchHex = strings.TrimSpace(val)
						}
					}
					if swatchHex == "" {
						swatchHex = resolveColorHex(valLabel)
					}

					col := &gqlColorCollector{
						label:   valLabel,
						vIdx:    vIdxStr,
						rawIdx:  vm["value_index"],
						hexCode: swatchHex,
						seen:    make(map[string]bool),
						vMap:    vm,
					}
					colorCollectors = append(colorCollectors, col)
					collectorsByVIdx[vIdxStr] = col
					collectorsByLabel[strings.ToLower(valLabel)] = col
					availableColors = append(availableColors, valLabel)
				}
			}
		}
	}

	if len(colorCollectors) == 0 {
		return
	}

	addImage := func(col *gqlColorCollector, url string) {
		clean := strings.TrimSpace(url)
		if clean == "" || col.seen[clean] {
			return
		}
		col.seen[clean] = true
		col.gallery = append(col.gallery, clean)
		if col.mainImage == "" {
			col.mainImage = clean
		}
	}

	// 1. Collect images from variants
	if rawVariants, ok := item["variants"].([]interface{}); ok {
		for _, varItem := range rawVariants {
			varMap, ok := varItem.(map[string]interface{})
			if !ok {
				continue
			}
			var matchedCol *gqlColorCollector
			if attrs, ok := varMap["attributes"].([]interface{}); ok {
				for _, aItem := range attrs {
					if am, ok := aItem.(map[string]interface{}); ok {
						aCode := strings.ToLower(fmt.Sprintf("%v", am["code"]))
						aLabel := strings.ToLower(fmt.Sprintf("%v", am["label"]))
						vIdx := fmt.Sprintf("%v", am["value_index"])
						if col, exists := collectorsByVIdx[vIdx]; exists {
							matchedCol = col
							break
						}
						if strings.Contains(aCode, "color") || strings.Contains(aLabel, "color") || strings.Contains(aLabel, "màu") {
							if col, exists := collectorsByLabel[aLabel]; exists {
								matchedCol = col
								break
							}
						}
					}
				}
			}

			if matchedCol != nil {
				if prodMap, ok := varMap["product"].(map[string]interface{}); ok {
					// small_image
					if sm, ok := prodMap["small_image"].(map[string]interface{}); ok {
						if u, _ := sm["url"].(string); u != "" {
							addImage(matchedCol, u)
						}
					}
					// image
					if im, ok := prodMap["image"].(map[string]interface{}); ok {
						if u, _ := im["url"].(string); u != "" {
							addImage(matchedCol, u)
						}
					}
					// image_url
					if u, _ := prodMap["image_url"].(string); u != "" {
						addImage(matchedCol, u)
					}
					// media_gallery
					if mg, ok := prodMap["media_gallery"].([]interface{}); ok {
						for _, mgItem := range mg {
							if mm, ok := mgItem.(map[string]interface{}); ok {
								if dis, _ := mm["disabled"].(bool); !dis {
									if u, _ := mm["url"].(string); u != "" {
										addImage(matchedCol, u)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Collect images from parent media_gallery with matching label or url
	if mg, ok := item["media_gallery"].([]interface{}); ok {
		for _, mgItem := range mg {
			mm, ok := mgItem.(map[string]interface{})
			if !ok {
				continue
			}
			if dis, _ := mm["disabled"].(bool); dis {
				continue
			}
			u, _ := mm["url"].(string)
			if u == "" {
				continue
			}
			lbl := strings.ToLower(fmt.Sprintf("%v", mm["label"]))
			uLower := strings.ToLower(u)
			for _, col := range colorCollectors {
				cLower := strings.ToLower(col.label)
				if (lbl != "" && strings.Contains(lbl, cLower)) || (cLower != "" && strings.Contains(uLower, cLower)) {
					addImage(col, u)
				}
			}
		}
	}

	// 3. Fallback: If only 1 color or a color matches parent context, fallback to item["image_url"]
	parentImg, _ := item["image_url"].(string)
	if parentImg != "" {
		for _, col := range colorCollectors {
			if col.mainImage == "" {
				parentContext := strings.ToLower(fmt.Sprintf("%v %v", item["url_key"], item["name"]))
				if strings.Contains(parentContext, strings.ToLower(col.label)) || len(colorCollectors) == 1 {
					addImage(col, parentImg)
				}
			}
		}
	}

	// 4. Build output arrays
	var colorImages []map[string]interface{}
	var rootColors []map[string]interface{}
	for _, col := range colorCollectors {
		colorImages = append(colorImages, map[string]interface{}{
			"color":       col.label,
			"color_code":  col.hexCode,
			"value_index": col.rawIdx,
			"image_url":   col.mainImage,
			"gallery":     col.gallery,
		})
		rootColors = append(rootColors, map[string]interface{}{
			"name":      col.label,
			"code":      col.hexCode,
			"imageUrl":  col.mainImage,
			"image_url": col.mainImage,
		})
		if col.vMap != nil {
			col.vMap["color_code"] = col.hexCode
			col.vMap["image_url"] = col.mainImage
			col.vMap["imageUrl"] = col.mainImage
			col.vMap["gallery"] = col.gallery
		}
	}

	if len(colorImages) > 0 {
		item["color_images"] = colorImages
	}
	if len(rootColors) > 0 {
		item["colors"] = rootColors
	}
	if len(availableColors) > 0 {
		item["available_colors"] = availableColors
	}
	if len(availableSizes) > 0 {
		item["available_sizes"] = availableSizes
	}
	item["configurable_product_options"] = rawOpts
}

func enrichGraphQLProductsResponse(resp map[string]interface{}, baseURL string) {
	if resp == nil {
		return
	}
	prods, ok := resp["products"].(map[string]interface{})
	if !ok {
		if data, ok := resp["data"].(map[string]interface{}); ok {
			prods, _ = data["products"].(map[string]interface{})
		}
	}
	if prods == nil {
		return
	}
	items, ok := prods["items"].([]interface{})
	if !ok {
		return
	}
	for _, it := range items {
		if itemMap, ok := it.(map[string]interface{}); ok {
			enrichGraphQLProductItem(itemMap, baseURL)
		}
	}
}
