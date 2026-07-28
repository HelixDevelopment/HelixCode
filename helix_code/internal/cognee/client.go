package cognee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/netutil"
)

// API path constants for the cognee 1.1.x versioned, auth-required surface.
//
// Verified against the live OpenAPI of the deployed cognee 1.1.2-local
// (GET <baseURL>/openapi.json — title "Cognee API", /health reports
// "version":"1.1.2-local"). The pre-1.1 unversioned paths (/api/cognify,
// /api/search, /api/memory) return 404 on this server and are GONE.
const (
	pathLogin    = "/api/v1/auth/login" // POST application/x-www-form-urlencoded {username,password} -> {access_token,token_type}
	pathAdd      = "/api/v1/add"         // POST multipart/form-data {data[], datasetName}
	pathCognify  = "/api/v1/cognify"     // POST application/json CognifyPayloadDTO {datasets,datasetIds,runInBackground,...}
	pathSearch   = "/api/v1/search"      // POST application/json SearchPayloadDTO {searchType,query,datasets,...}
	pathDatasets = "/api/v1/datasets"    // GET list / POST create {name} -> DatasetDTO
	pathDelete   = "/api/v1/delete"      // DELETE
	pathVisualize = "/api/v1/visualize"  // GET
)

// Client represents a Cognee API client
type Client struct {
	baseURL    string
	apiKey     string // X-Api-Key header value (ApiKeyAuth scheme); empty if using bearer login
	username   string // login username/email for BearerAuth scheme
	password   string // login password for BearerAuth scheme
	httpClient *http.Client
	logger     *logging.Logger
	mu         sync.RWMutex
	connected  bool
	lastCheck  time.Time

	// bearerToken caches the JWT obtained from pathLogin. Guarded by mu.
	bearerToken string
}

// NewClient creates a new Cognee API client.
//
// Auth credentials (CONST-042 / §11.4.10 — NEVER hardcoded) are sourced from
// the CogneeRemoteAPIConfig first, then fall back to the COGNEE_API_KEY /
// COGNEE_USERNAME / COGNEE_PASSWORD environment variables so deployment
// secrets stay out of committed config. If neither an API key nor a
// username/password pair is configured the client makes anonymous requests —
// against an auth-required server those requests surface the real 401, never
// a fabricated success.
func NewClient(cfg *config.CogneeConfig) *Client {
	// HXC-185: cfg.Host is operator-supplied, so it can be a bare IPv6
	// literal. An IPv6 literal is only valid in a URL authority when bracketed
	// (RFC 3986 §3.2.2); under this module's go1.26 language version net/url
	// rejects the unbracketed form with `invalid port "::1:8000" after host`,
	// so every request built from this base URL failed before dialling.
	baseURL := "http://" + netutil.JoinHostPort(cfg.Host, cfg.Port)
	if cfg.RemoteAPI != nil && cfg.RemoteAPI.ServiceEndpoint != "" && cfg.Mode == "cloud" {
		baseURL = cfg.RemoteAPI.ServiceEndpoint
	}

	timeout := 30 * time.Second
	if cfg.RemoteAPI != nil && cfg.RemoteAPI.Timeout > 0 {
		timeout = cfg.RemoteAPI.Timeout
	}

	apiKey, username, password := "", "", ""
	if cfg.RemoteAPI != nil {
		apiKey = cfg.RemoteAPI.APIKey
		username = cfg.RemoteAPI.Username
		password = cfg.RemoteAPI.Password
	}
	// Env fallback — never commit credentials into config files.
	if apiKey == "" {
		apiKey = os.Getenv("COGNEE_API_KEY")
	}
	if username == "" {
		username = os.Getenv("COGNEE_USERNAME")
	}
	if password == "" {
		password = os.Getenv("COGNEE_PASSWORD")
	}

	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		apiKey:   apiKey,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logging.NewLoggerWithName("cognee_client"),
	}
}

// IsConnected returns whether the client is connected to Cognee
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetBaseURL returns the base URL of the Cognee API. The read is guarded by
// c.mu because SetBaseURL mutates baseURL under the same lock — an unguarded
// read here races a concurrent SetBaseURL.
func (c *Client) GetBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// getBaseURL is the lock-guarded internal accessor every request builder MUST
// use to read baseURL. baseURL is mutable (SetBaseURL writes it under c.mu), so
// reading c.baseURL directly while a request is in flight is a data race.
func (c *Client) getBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// getAPIKey is the lock-guarded internal accessor for apiKey. apiKey is mutable
// (SetAPIKey writes it under c.mu), so reading c.apiKey directly in setHeaders
// while SetAPIKey runs concurrently is a data race.
func (c *Client) getAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey
}

// TestConnection tests the connection to Cognee
func (c *Client) TestConnection(ctx context.Context) bool {
	url := fmt.Sprintf("%s/health", c.getBaseURL())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.logger.Debug("Failed to create health check request: %v", err)
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Debug("Health check failed: %v", err)
		c.mu.Lock()
		c.connected = false
		c.lastCheck = time.Now()
		c.mu.Unlock()
		return false
	}
	defer resp.Body.Close()

	connected := resp.StatusCode == http.StatusOK
	c.mu.Lock()
	c.connected = connected
	c.lastCheck = time.Now()
	c.mu.Unlock()

	return connected
}

// AddMemory adds a memory entry to Cognee via POST /api/v1/add.
//
// The v1 /add endpoint takes multipart/form-data: a "data" file part (the
// content) plus a "datasetName" field. The request's Content text becomes the
// uploaded file body. The server returns an arbitrary JSON object; AddMemory
// maps the conventional fields onto AddMemoryResponse and preserves the raw
// payload in GraphNodes so callers see the real response.
func (c *Client) AddMemory(ctx context.Context, req *AddMemoryRequest) (*AddMemoryResponse, error) {
	endpoint := c.getBaseURL() + pathAdd

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	filename := "memory.txt"
	if req.ContentType != "" && !strings.EqualFold(req.ContentType, "text") {
		filename = "memory." + sanitizeExt(req.ContentType)
	}
	fw, err := mw.CreateFormFile("data", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart data part: %w", err)
	}
	if _, err := fw.Write([]byte(req.Content)); err != nil {
		return nil, fmt.Errorf("failed to write multipart content: %w", err)
	}
	if req.DatasetName != "" {
		if err := mw.WriteField("datasetName", req.DatasetName); err != nil {
			return nil, fmt.Errorf("failed to write datasetName field: %w", err)
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	response := &AddMemoryResponse{GraphNodes: raw}
	if v, ok := raw["id"].(string); ok {
		response.ID = v
	}
	if v, ok := raw["message"].(string); ok {
		response.Message = v
	}
	return response, nil
}

// sanitizeExt derives a safe filename extension from a content-type hint.
func sanitizeExt(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexAny(ct, "/;"); i >= 0 {
		ct = ct[i+1:]
	}
	ct = strings.TrimSpace(ct)
	if ct == "" {
		return "txt"
	}
	// keep only alnum
	var b strings.Builder
	for _, r := range ct {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "txt"
	}
	return b.String()
}

// SearchMemory searches for memories in Cognee via POST /api/v1/search.
//
// The v1 /search endpoint takes a SearchPayloadDTO {searchType, query,
// datasets, ...} and returns an array of SearchResult {search_result,
// dataset_id, dataset_name}. The default searchType is GRAPH_COMPLETION; the
// request's SearchType overrides it when set.
func (c *Client) SearchMemory(ctx context.Context, req *SearchMemoryRequest) (*SearchMemoryResponse, error) {
	startTime := time.Now()
	results, err := c.doSearch(ctx, req.Query, req.SearchType, mergeDatasets(req.DatasetName, req.Datasets))
	if err != nil {
		return nil, err
	}
	return &SearchMemoryResponse{
		Results:    results,
		TotalCount: len(results),
		Query:      req.Query,
		Duration:   time.Since(startTime),
	}, nil
}

// mergeDatasets folds a single DatasetName into the Datasets slice without
// duplicating it, producing the v1 `datasets` array.
func mergeDatasets(name string, datasets []string) []string {
	out := append([]string{}, datasets...)
	if name != "" {
		found := false
		for _, d := range out {
			if d == name {
				found = true
				break
			}
		}
		if !found {
			out = append(out, name)
		}
	}
	return out
}

// doSearch issues the real POST /api/v1/search call and maps the v1
// SearchResult array onto the package's MemorySource results. searchType
// defaults to GRAPH_COMPLETION (the server default) when empty.
func (c *Client) doSearch(ctx context.Context, query, searchType string, datasets []string) ([]MemorySource, error) {
	payload := map[string]interface{}{
		"query": query,
	}
	if searchType != "" {
		payload["searchType"] = searchType
	}
	if len(datasets) > 0 {
		payload["datasets"] = datasets
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.getBaseURL()+pathSearch, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	// v1 returns an array of SearchResult objects.
	var raw []struct {
		SearchResult interface{} `json:"search_result"`
		DatasetID    string      `json:"dataset_id"`
		DatasetName  string      `json:"dataset_name"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]MemorySource, 0, len(raw))
	for _, r := range raw {
		content := ""
		switch v := r.SearchResult.(type) {
		case string:
			content = v
		default:
			if b, mErr := json.Marshal(v); mErr == nil {
				content = string(b)
			}
		}
		results = append(results, MemorySource{
			Content:     content,
			DatasetName: r.DatasetName,
		})
	}
	return results, nil
}

// Cognify processes data into knowledge graphs via POST /api/v1/cognify.
//
// The v1 /cognify endpoint takes a CognifyPayloadDTO {datasets, datasetIds,
// runInBackground, ...} and returns an arbitrary JSON object describing the
// run. The dataset names map onto the `datasets` field.
func (c *Client) Cognify(ctx context.Context, req *CognifyRequest) (*CognifyResponse, error) {
	payload := map[string]interface{}{
		"runInBackground": false,
	}
	if len(req.Datasets) > 0 {
		payload["datasets"] = req.Datasets
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.getBaseURL()+pathCognify, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(body, &raw)
	response := &CognifyResponse{Status: "completed", StartedAt: time.Now()}
	if v, ok := raw["status"].(string); ok && v != "" {
		response.Status = v
	}
	if v, ok := raw["message"].(string); ok {
		response.Message = v
	}
	return response, nil
}

// SearchInsights performs insight-based search using graph reasoning via
// POST /api/v1/search with searchType=INSIGHTS.
func (c *Client) SearchInsights(ctx context.Context, req *InsightsRequest) (*InsightsResponse, error) {
	startTime := time.Now()
	results, err := c.doSearch(ctx, req.Query, "INSIGHTS", req.Datasets)
	if err != nil {
		return nil, err
	}
	insights := make([]Insight, 0, len(results))
	for _, r := range results {
		insights = append(insights, Insight{Content: r.Content})
	}
	return &InsightsResponse{
		Insights: insights,
		Query:    req.Query,
		Duration: time.Since(startTime),
	}, nil
}

// SearchGraphCompletion performs LLM-powered graph completion search via
// POST /api/v1/search with searchType=GRAPH_COMPLETION.
func (c *Client) SearchGraphCompletion(ctx context.Context, query string, datasets []string, limit int) (*SearchMemoryResponse, error) {
	startTime := time.Now()
	results, err := c.doSearch(ctx, query, "GRAPH_COMPLETION", datasets)
	if err != nil {
		return nil, err
	}
	return &SearchMemoryResponse{
		Results:    results,
		TotalCount: len(results),
		Query:      query,
		Duration:   time.Since(startTime),
	}, nil
}

// ErrUnsupportedEndpoint is returned by client methods whose pre-1.1
// unversioned endpoint has no equivalent on the cognee 1.1.x versioned
// surface. Surfacing this honestly (§11.4.6) is preferable to issuing a
// request against a fabricated path and decoding a 404 as if it were data.
var ErrUnsupportedEndpoint = fmt.Errorf("cognee: endpoint not available on the cognee 1.1.x versioned API")

// ProcessCodePipeline processes code through Cognee's code-understanding
// pipeline.
//
// The pre-1.1 /api/code-pipeline/index endpoint has NO equivalent on the
// verified cognee 1.1.x surface (its OpenAPI exposes no code-pipeline path).
// Code is instead added as ordinary content via the real /api/v1/add endpoint
// (then cognified), which is the supported 1.1.x ingestion path.
func (c *Client) ProcessCodePipeline(ctx context.Context, req *CodePipelineRequest) (*CodePipelineResponse, error) {
	if _, err := c.AddMemory(ctx, &AddMemoryRequest{
		Content:     req.Code,
		DatasetName: req.DatasetName,
		ContentType: "code",
		UserID:      req.UserID,
		ProjectID:   req.ProjectID,
	}); err != nil {
		return nil, err
	}
	return &CodePipelineResponse{
		Processed: true,
		Message:   "code added to dataset via /api/v1/add (run Cognify to build the graph)",
	}, nil
}

// datasetDTO mirrors the v1 DatasetDTO {id, name, createdAt, updatedAt,
// ownerId} returned by /api/v1/datasets.
type datasetDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	OwnerID   string `json:"ownerId"`
}

func (d datasetDTO) toDataset() Dataset {
	ds := Dataset{ID: d.ID, Name: d.Name, UserID: d.OwnerID}
	if t, err := time.Parse(time.RFC3339, d.CreatedAt); err == nil {
		ds.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, d.UpdatedAt); err == nil {
		ds.UpdatedAt = t
	}
	return ds
}

// CreateDataset creates a new dataset via POST /api/v1/datasets.
//
// The v1 endpoint takes a DatasetCreationPayload {name} (name only — the
// description/metadata fields of CreateDatasetRequest are not part of the v1
// contract) and returns a flat DatasetDTO.
func (c *Client) CreateDataset(ctx context.Context, req *CreateDatasetRequest) (*DatasetResponse, error) {
	data, err := json.Marshal(map[string]interface{}{"name": req.Name})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.getBaseURL()+pathDatasets, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	var dto datasetDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	ds := dto.toDataset()
	return &DatasetResponse{Dataset: &ds}, nil
}

// ListDatasets retrieves all datasets via GET /api/v1/datasets.
//
// The v1 endpoint returns a flat array of DatasetDTO (not a {datasets:[...]}
// envelope).
func (c *Client) ListDatasets(ctx context.Context) (*DatasetsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.getBaseURL()+pathDatasets, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	var dtos []datasetDTO
	if err := json.Unmarshal(body, &dtos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	out := &DatasetsResponse{Datasets: make([]Dataset, 0, len(dtos)), Total: len(dtos)}
	for _, d := range dtos {
		out.Datasets = append(out.Datasets, d.toDataset())
	}
	return out, nil
}

// GetDataset retrieves a specific dataset by name.
//
// The v1 API addresses datasets by UUID (/api/v1/datasets/{dataset_id}), so a
// by-name lookup is resolved by listing datasets and matching the name. Returns
// (nil, nil) when no dataset with that name exists.
func (c *Client) GetDataset(ctx context.Context, name string) (*Dataset, error) {
	list, err := c.ListDatasets(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list.Datasets {
		if list.Datasets[i].Name == name {
			ds := list.Datasets[i]
			return &ds, nil
		}
	}
	return nil, nil
}

// DeleteDataset deletes a dataset by name.
//
// The v1 DELETE endpoint is keyed by UUID (/api/v1/datasets/{dataset_id}), so
// the name is first resolved to its id via ListDatasets. A non-existent
// dataset is treated as already-deleted (no error).
func (c *Client) DeleteDataset(ctx context.Context, name string) error {
	ds, err := c.GetDataset(ctx, name)
	if err != nil {
		return err
	}
	if ds == nil {
		return nil // nothing to delete
	}

	endpoint := fmt.Sprintf("%s%s/%s", c.getBaseURL(), pathDatasets, ds.ID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}
	return nil
}

// VisualizeGraph retrieves graph visualization data via GET /api/v1/visualize.
//
// The v1 /visualize endpoint is a GET that returns the graph rendering. The
// raw response is preserved; the typed graph fields are populated when the
// server returns the expected shape.
func (c *Client) VisualizeGraph(ctx context.Context, req *GraphVisualizationRequest) (*GraphVisualizationResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.getBaseURL()+pathVisualize, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	var response GraphVisualizationResponse
	// /visualize may return HTML or JSON depending on the server; decode
	// best-effort and surface the format.
	if err := json.Unmarshal(body, &response); err != nil {
		response = GraphVisualizationResponse{Format: "raw"}
	}
	return &response, nil
}

// DeleteData removes data from a dataset via DELETE /api/v1/delete.
func (c *Client) DeleteData(ctx context.Context, req *DeleteDataRequest) (*DeleteDataResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", c.getBaseURL()+pathDelete, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)
	if err := c.attachAuth(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognee API error: %s, body: %s", resp.Status, string(body))
	}

	response := &DeleteDataResponse{}
	_ = json.Unmarshal(body, response)
	return response, nil
}

// GetHealth retrieves health status
func (c *Client) GetHealth(ctx context.Context) (*HealthStatus, error) {
	url := fmt.Sprintf("%s/health", c.getBaseURL())

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &HealthStatus{
			Status:       "unhealthy",
			Timestamp:    time.Now(),
			LastCheck:    time.Now(),
			ResponseTime: time.Since(startTime),
			Components:   map[string]string{"connection": "failed"},
		}, nil
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime)

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		health = HealthStatus{
			Status:    "unknown",
			Timestamp: time.Now(),
		}
	}

	health.LastCheck = time.Now()
	health.ResponseTime = responseTime

	if resp.StatusCode == http.StatusOK {
		health.Status = "healthy"
		c.mu.Lock()
		c.connected = true
		c.lastCheck = time.Now()
		c.mu.Unlock()
	} else {
		health.Status = "unhealthy"
		c.mu.Lock()
		c.connected = false
		c.lastCheck = time.Now()
		c.mu.Unlock()
	}

	return &health, nil
}

// GetStatistics retrieves Cognee statistics.
//
// The pre-1.1 /api/stats endpoint has NO equivalent on the verified cognee
// 1.1.x surface. Rather than query a fabricated path, this returns
// ErrUnsupportedEndpoint; CogneeService.GetStatistics tolerates this and falls
// back to locally-tracked counters.
func (c *Client) GetStatistics(ctx context.Context) (*CogneeStatistics, error) {
	return nil, ErrUnsupportedEndpoint
}

// AddBatchMemory adds multiple memories in batch.
//
// The pre-1.1 /api/memory/batch endpoint has NO equivalent on the cognee 1.1.x
// surface; the batch is implemented by issuing one real /api/v1/add call per
// item. Any item failure aborts and is surfaced.
func (c *Client) AddBatchMemory(ctx context.Context, req *BatchMemoryRequest) (*BatchMemoryResponse, error) {
	response := &BatchMemoryResponse{IDs: make([]string, 0, len(req.Memories))}
	for i := range req.Memories {
		mem := req.Memories[i]
		addResp, err := c.AddMemory(ctx, &mem)
		if err != nil {
			return nil, fmt.Errorf("batch add failed at item %d: %w", i, err)
		}
		response.Processed++
		if addResp != nil {
			response.IDs = append(response.IDs, addResp.ID)
		}
	}
	return response, nil
}

// SubmitFeedback submits feedback on search results.
//
// The pre-1.1 /api/feedback endpoint has NO equivalent on the verified cognee
// 1.1.x surface; this returns ErrUnsupportedEndpoint rather than POST to a
// fabricated path.
func (c *Client) SubmitFeedback(ctx context.Context, req *FeedbackRequest) (*FeedbackResponse, error) {
	return nil, ErrUnsupportedEndpoint
}

// AutoContainerize starts Cognee in a container if not running
func (c *Client) AutoContainerize(ctx context.Context) error {
	if c.TestConnection(ctx) {
		c.logger.Info("Cognee is already running")
		return nil
	}

	c.logger.Info("Cognee not running, attempting to start container...")

	return c.startCogneeContainer(ctx)
}

// startCogneeContainer starts Cognee using Docker or Podman
func (c *Client) startCogneeContainer(ctx context.Context) error {
	var containerRuntime string
	var composeCmd []string

	if _, err := exec.LookPath("docker"); err == nil {
		containerRuntime = "docker"
		composeCmd = []string{"compose", "up", "-d", "cognee"}
	} else if _, err := exec.LookPath("podman"); err == nil {
		containerRuntime = "podman"
		composeCmd = []string{"compose", "up", "-d", "cognee"}
	} else {
		return fmt.Errorf("neither docker nor podman found in PATH")
	}

	c.logger.Info("Starting Cognee using %s...", containerRuntime)

	cmd := exec.CommandContext(ctx, containerRuntime, composeCmd...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start Cognee container: %w, output: %s", err, string(output))
	}

	c.logger.Info("Cognee container start initiated, waiting for service to be ready...")

	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			if c.TestConnection(ctx) {
				c.logger.Info("Cognee service is now ready")
				return nil
			}
			c.logger.Debug("Waiting for Cognee service... attempt %d/30", i+1)
		}
	}

	return fmt.Errorf("cognee container started but service is not responding after 60 seconds")
}

// setHeaders sets common headers for API requests. It does NOT set
// Content-Type (callers set it per content type: application/json for the
// JSON DTOs, multipart/form-data for /add, application/x-www-form-urlencoded
// for login). Authentication is attached separately via attachAuth so that
// multipart and JSON requests share the same auth path.
func (c *Client) setHeaders(req *http.Request) {
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
}

// attachAuth attaches the appropriate credential to an authenticated request.
//
// cognee 1.1.x exposes two schemes (see CogneeRemoteAPIConfig):
//   - ApiKeyAuth: a pre-issued API key sent as the "X-Api-Key" header.
//   - BearerAuth: a JWT obtained from POST /api/v1/auth/login, sent as
//     "Authorization: Bearer <jwt>".
//
// If an API key is configured it is preferred; otherwise, when a
// username/password is configured, attachAuth lazily logs in (caching the JWT)
// and attaches the bearer token. With neither configured the request goes out
// unauthenticated and the server's real 401 is surfaced — never masked.
func (c *Client) attachAuth(ctx context.Context, req *http.Request) error {
	if key := c.getAPIKey(); key != "" {
		req.Header.Set("X-Api-Key", key)
		return nil
	}

	c.mu.RLock()
	user, pass, tok := c.username, c.password, c.bearerToken
	c.mu.RUnlock()

	if user == "" || pass == "" {
		// No credentials — anonymous request. The server's 401 (if it
		// requires auth) is the honest, surfaced outcome.
		return nil
	}

	if tok == "" {
		var err error
		tok, err = c.login(ctx, user, pass)
		if err != nil {
			return fmt.Errorf("cognee login failed: %w", err)
		}
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	return nil
}

// login performs the form-urlencoded login against pathLogin and caches the
// returned JWT. It is a real HTTP call to the real cognee auth endpoint.
func (c *Client) login(ctx context.Context, username, password string) (string, error) {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set("grant_type", "password")

	loginURL := c.getBaseURL() + pathLogin
	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cognee login error: %s, body: %s", resp.Status, string(body))
	}

	var lr struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &lr); err != nil {
		return "", fmt.Errorf("failed to decode login response: %w", err)
	}
	if lr.AccessToken == "" {
		return "", fmt.Errorf("cognee login returned empty access_token")
	}

	c.mu.Lock()
	c.bearerToken = lr.AccessToken
	c.mu.Unlock()
	return lr.AccessToken, nil
}

// Close closes the client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
	return nil
}

// SetAPIKey updates the API key
func (c *Client) SetAPIKey(apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKey = apiKey
}

// SetBaseURL updates the base URL
func (c *Client) SetBaseURL(baseURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimSuffix(baseURL, "/")
}

// SetTimeout updates the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient.Timeout = timeout
}
