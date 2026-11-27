package magento

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// GraphQLClient executes public, unauthenticated GraphQL queries against
// Magento 2's /graphql endpoint (for guest storefront interactions).
type GraphQLClient struct {
	endpoint   string
	httpClient tls_client.HttpClient
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// NewGraphQLClient initializes a client for a given Magento store base URL.
// It normalizes the base URL and sets up browser-grade TLS fingerprinting.
func NewGraphQLClient(baseURL string) (*GraphQLClient, error) {
	norm := NormalizeBaseURL(baseURL)
	if norm == "" {
		norm = NormalizeBaseURL(os.Getenv("MAGENTO_BASE_URL"))
	}
	if norm == "" {
		return nil, fmt.Errorf("store_url is required (connect the store under Account → Integrations, or configure MAGENTO_BASE_URL)")
	}

	timeoutSec := 30
	if v := strings.TrimSpace(os.Getenv("MAGENTO_HTTP_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}

	insecure := os.Getenv("MAGENTO_INSECURE_SKIP_VERIFY") == "true" || os.Getenv("MAGENTO_INSECURE_SKIP_VERIFY") == "1"
	if u, err := url.Parse(norm); err == nil {
		host := u.Hostname()
		if strings.HasSuffix(host, ".test") || host == "localhost" || host == "127.0.0.1" {
			insecure = true
		}
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(timeoutSec),
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithNotFollowRedirects(),
	}
	if insecure {
		options = append(options, tls_client.WithInsecureSkipVerify())
	}

	httpClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		httpClient, _ = tls_client.NewHttpClient(tls_client.NewNoopLogger())
	}

	return &GraphQLClient{
		endpoint:   norm + "/graphql",
		httpClient: httpClient,
	}, nil
}

// Endpoint returns the target GraphQL endpoint URL.
func (c *GraphQLClient) Endpoint() string {
	if c == nil {
		return ""
	}
	return c.endpoint
}

// BaseURL returns the normalized store base URL without the /graphql suffix.
func (c *GraphQLClient) BaseURL() string {
	if c == nil || c.endpoint == "" {
		return ""
	}
	return strings.TrimSuffix(c.endpoint, "/graphql")
}

// Query sends a GraphQL query or mutation and returns the raw `data` JSON payload.
func (c *GraphQLClient) Query(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	if c == nil || c.endpoint == "" {
		return nil, fmt.Errorf("graphql client not configured")
	}

	reqBody, err := json.Marshal(graphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := fhttp.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	setBrowserHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute graphql request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("graphql http %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(bodyBytes, &gqlResp); err != nil {
		return nil, fmt.Errorf("unmarshal graphql response: %w", err)
	}

	// Magento answers resolver failures as {"data":{"products":null},"errors":[...]},
	// so a data object whose fields are all null is treated as no data.
	if len(gqlResp.Errors) > 0 && !hasNonNullData(gqlResp.Data) {
		var errMsgs []string
		for _, e := range gqlResp.Errors {
			errMsgs = append(errMsgs, e.Message)
		}
		return nil, fmt.Errorf("graphql error: %s", strings.Join(errMsgs, "; "))
	}

	return gqlResp.Data, nil
}

// hasNonNullData reports whether a GraphQL `data` payload carries at least one non-null field.
func hasNonNullData(data json.RawMessage) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return true
	}
	for _, v := range fields {
		if !bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return true
		}
	}
	return false
}

// setBrowserHeaders makes storefront calls look like the browser fingerprint the
// TLS profile already claims, so WAFs such as Cloudflare let them through.
func setBrowserHeaders(req *fhttp.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
}

// PostStorefrontJSON POSTs a JSON body to a public storefront path (e.g.
// "/weaver/order/lookup", served by the DatumBridge_Chatbot module) and returns
// the HTTP status and raw body. 4xx answers are returned, not turned into
// errors, because the module explains them in the JSON body.
func (c *GraphQLClient) PostStorefrontJSON(ctx context.Context, path string, body interface{}) (int, []byte, error) {
	if c == nil || c.endpoint == "" {
		return 0, nil, fmt.Errorf("storefront client not configured")
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := c.BaseURL() + "/" + strings.TrimLeft(path, "/")
	req, err := fhttp.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return 0, nil, fmt.Errorf("create http request: %w", err)
	}
	setBrowserHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response body: %w", err)
	}
	return resp.StatusCode, data, nil
}
