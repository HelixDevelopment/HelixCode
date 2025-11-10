package providers

import (
	"context"
	"time"
)

// ProviderType represents the type of vector provider
type ProviderType string

// LoadBalanceType represents the type of load balancing
type LoadBalanceType string

// Load balancing type constants
const (
	LoadBalanceRoundRobin LoadBalanceType = "round_robin"
	LoadBalanceWeighted   LoadBalanceType = "weighted"
	LoadBalanceRandom     LoadBalanceType = "random"
)

// Provider type constants
const (
	ProviderTypePinecone    ProviderType = "pinecone"
	ProviderTypeMilvus      ProviderType = "milvus"
	ProviderTypeWeaviate    ProviderType = "weaviate"
	ProviderTypeQdrant      ProviderType = "qdrant"
	ProviderTypeRedis       ProviderType = "redis"
	ProviderTypeChroma      ProviderType = "chroma"
	ProviderTypeOpenAI      ProviderType = "openai"
	ProviderTypeAnthropic   ProviderType = "anthropic"
	ProviderTypeCohere      ProviderType = "cohere"
	ProviderTypeHuggingFace ProviderType = "huggingface"
	ProviderTypeMistral     ProviderType = "mistral"
	ProviderTypeGemini      ProviderType = "gemini"
	ProviderTypeVertexAI    ProviderType = "vertexai"
	ProviderTypeClickHouse  ProviderType = "clickhouse"
	ProviderTypeSupabase    ProviderType = "supabase"
	ProviderTypeDeepLake    ProviderType = "deeplake"
	ProviderTypeFAISS       ProviderType = "faiss"
	ProviderTypeLlamaIndex  ProviderType = "llamaindex"
	ProviderTypeMemGPT      ProviderType = "memgpt"
	ProviderTypeCrewAI      ProviderType = "crewai"
	ProviderTypeCharacterAI ProviderType = "characterai"
	ProviderTypeReplika     ProviderType = "replika"
	ProviderTypeAgnostic    ProviderType = "agnostic"
	ProviderTypeAnima       ProviderType = "anima"
	ProviderTypeGemma       ProviderType = "gemma"
	ProviderTypeMem0        ProviderType = "mem0"
	ProviderTypeZep         ProviderType = "zep"
	ProviderTypeMemonto     ProviderType = "memonto"
	ProviderTypeBaseAI      ProviderType = "baseai"
)

// VectorProvider defines interface for vector database providers
type VectorProvider interface {
	// Core operations
	Store(ctx context.Context, vectors []*VectorData) error
	Retrieve(ctx context.Context, ids []string) ([]*VectorData, error)
	Update(ctx context.Context, id string, vector *VectorData) error
	Delete(ctx context.Context, ids []string) error

	// Search operations
	Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error)
	FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error)
	BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error)

	// Collection management
	CreateCollection(ctx context.Context, name string, config *CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]*CollectionInfo, error)
	GetCollection(ctx context.Context, name string) (*CollectionInfo, error)

	// Index management
	CreateIndex(ctx context.Context, collection string, config *IndexConfig) error
	DeleteIndex(ctx context.Context, collection, name string) error
	ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error)

	// Metadata operations
	AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error)
	DeleteMetadata(ctx context.Context, ids []string, keys []string) error

	// Management
	GetStats(ctx context.Context) (*ProviderStats, error)
	Optimize(ctx context.Context) error
	Backup(ctx context.Context, path string) error
	Restore(ctx context.Context, path string) error

	// Lifecycle
	Initialize(ctx context.Context, config interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) (*HealthStatus, error)

	// Metadata
	GetName() string
	GetType() string
	GetCapabilities() []string
	GetConfiguration() interface{}
	IsCloud() bool
	GetCostInfo() *CostInfo
}

// VectorData represents a vector data entry
type VectorData struct {
	ID         string                 `json:"id"`
	Vector     []float64              `json:"vector"`
	Metadata   map[string]interface{} `json:"metadata"`
	Collection string                 `json:"collection"`
	Timestamp  time.Time              `json:"timestamp"`
	TTL        *time.Duration         `json:"ttl,omitempty"`
	Namespace  string                 `json:"namespace,omitempty"`
}

// VectorQuery represents a vector search query
type VectorQuery struct {
	Vector        []float64              `json:"vector,omitempty"`
	Text          string                 `json:"text,omitempty"`
	Collection    string                 `json:"collection,omitempty"`
	Namespace     string                 `json:"namespace,omitempty"`
	TopK          int                    `json:"top_k,omitempty"`
	Threshold     float64                `json:"threshold,omitempty"`
	IncludeVector bool                   `json:"include_vector,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// VectorSearchResult represents the result of a vector search
type VectorSearchResult struct {
	Results   []*VectorSearchResultItem `json:"results"`
	Total     int                       `json:"total"`
	Query     *VectorQuery              `json:"query,omitempty"`
	Duration  time.Duration             `json:"duration"`
	Namespace string                    `json:"namespace,omitempty"`
}

// VectorSearchResultItem represents a single search result item
type VectorSearchResultItem struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance"`
}

// VectorSimilarityResult represents a similarity search result
type VectorSimilarityResult struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance,omitempty"`
}

// CollectionConfig represents configuration for a vector collection
type CollectionConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Dimension   int                    `json:"dimension"`
	Metric      string                 `json:"metric"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Replicas    int                    `json:"replicas,omitempty"`
	Shards      int                    `json:"shards,omitempty"`
}

// CollectionInfo represents information about a vector collection
type CollectionInfo struct {
	Name        string                 `json:"name"`
	Dimension   int                    `json:"dimension"`
	Metric      string                 `json:"metric"`
	Size        int64                  `json:"size"` // VectorsCount mapped to Size
	VectorCount int64                  `json:"vector_count"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	Config      *CollectionConfig      `json:"config,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// IndexConfig represents configuration for a vector index
type IndexConfig struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Metric     string                 `json:"metric,omitempty"`
}

// IndexInfo represents information about a vector index
type IndexInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	State     string                 `json:"state"` // Status mapped to State
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at,omitempty"`
	Config    *IndexConfig           `json:"config,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderStats represents statistics for a vector provider
type ProviderStats struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Status           string                 `json:"status"`
	TotalOperations  int64                  `json:"total_operations"`
	SuccessfulOps    int64                  `json:"successful_ops"`
	FailedOps        int64                  `json:"failed_ops"`
	AverageLatency   time.Duration          `json:"average_latency"`
	TotalVectors     int64                  `json:"total_vectors"`
	TotalCollections int64                  `json:"total_collections"`
	TotalSize        int64                  `json:"total_size"` // StorageSize mapped to TotalSize
	LastHealthCheck  time.Time              `json:"last_health_check"`
	LastOperation    time.Time              `json:"last_operation"`
	ErrorCount       int64                  `json:"error_count"`
	Uptime           time.Duration          `json:"uptime"`
	CostInfo         *CostInfo              `json:"cost_info,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// CostInfo represents cost information for a provider
type CostInfo struct {
	Currency      string             `json:"currency"`
	ComputeCost   float64            `json:"compute_cost"`
	TransferCost  float64            `json:"transfer_cost"`
	StorageCost   float64            `json:"storage_cost"`
	TotalCost     float64            `json:"total_cost"`
	BillingPeriod string             `json:"billing_period"`
	FreeTierUsed  float64            `json:"free_tier_used"`
	FreeTierLimit float64            `json:"free_tier_limit"`
	Details       map[string]float64 `json:"details,omitempty"`
}

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	Timestamp    time.Time              `json:"timestamp"`
	LastCheck    time.Time              `json:"last_check"`
	ResponseTime time.Duration          `json:"response_time"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Dependencies map[string]string      `json:"dependencies,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// Model represents a machine learning model
type Model struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description,omitempty"`
	Architecture    string                 `json:"architecture,omitempty"`
	Parameters      int64                  `json:"parameters,omitempty"`
	IsActive        bool                   `json:"is_active"`
	CPUOptimization bool                   `json:"cpu_optimization"`
	GPUEnabled      bool                   `json:"gpu_enabled"`
	Quantization    bool                   `json:"quantization"`
	Caching         bool                   `json:"caching"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Embedding represents a text embedding
type Embedding struct {
	ID        string                 `json:"id"`
	ModelID   string                 `json:"model_id"`
	Text      string                 `json:"text"`
	Values    []float64              `json:"values"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GenerationOptions represents options for text generation
type GenerationOptions struct {
	MaxTokens        int      `json:"max_tokens,omitempty"`
	Temperature      float64  `json:"temperature,omitempty"`
	TopP             float64  `json:"top_p,omitempty"`
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	Stream           bool     `json:"stream,omitempty"`
}

// ModelPerformance represents model performance metrics
type ModelPerformance struct {
	ID                string                 `json:"id"`
	ModelID           string                 `json:"model_id"`
	ResponseTime      time.Duration          `json:"response_time"`
	Throughput        float64                `json:"throughput,omitempty"`
	CPUUtilization    float64                `json:"cpu_utilization,omitempty"`
	GPUUtilization    float64                `json:"gpu_utilization,omitempty"`
	MemoryUsage       int64                  `json:"memory_usage,omitempty"`
	Accuracy          float64                `json:"accuracy,omitempty"`
	ErrorRate         float64                `json:"error_rate,omitempty"`
	RequestsPerSecond float64                `json:"requests_per_second,omitempty"`
	LastUpdated       time.Time              `json:"last_updated"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}
