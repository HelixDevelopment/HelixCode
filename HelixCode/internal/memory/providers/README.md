# Vector Database Providers

This package contains implementations for 24 different vector database and AI memory providers for the HelixCode project.

## Provider List

### Production-Ready Vector Databases
1. **Pinecone** (`pinecone_provider.go`) - Fully managed vector database
2. **Milvus** (`milvus_provider.go`) - Open-source vector database
3. **Weaviate** (`weaviate_provider.go`) - Knowledge graph with vector search
4. **Qdrant** (`qdrant_provider.go`) - Vector similarity search engine
5. **Redis** (`redis_provider.go`) - In-memory data structure store with RediSearch

### AI Memory & Character Systems
6. **MemGPT** (`memgpt_provider.go`) - AI memory management system
7. **CrewAI** (`crewai_provider.go`) - Multi-agent coordination platform
8. **Character.AI** (`character_ai_provider.go`) - Character creation and interaction
9. **Replika** (`replika_provider.go`) - AI companion platform
10. **Anima** (`anima_provider.go`) - Avatar and activity tracking

### Large Language Models
11. **Gemma** (`gemma_provider.go`) - Google's lightweight LLM
12. **LlamaIndex** (`llama_index_provider.go`) - Data framework for LLM applications
13. **OpenAI** (`openai_provider.go`) - GPT models and embeddings
14. **Anthropic** (`anthropic_provider.go`) - Claude models
15. **HuggingFace** (`huggingface_provider.go`) - Open model hub
16. **Cohere** (`cohere_provider.go`) - Enterprise AI models
17. **Mistral** (`mistral_provider.go`) - European AI models

### Vector Search & Indexing
18. **FAISS** (`faiss_provider.go`) - Facebook AI similarity search
19. **Chroma** (`chroma_provider.go`) - Open-source embedding database
20. **Vertex AI** (`vertex_ai_provider.go`) - Google's ML platform

### Alternative Search & Storage
21. **ClickHouse** (`clickhouse_provider.go`) - OLAP database with vector functions
22. **Supabase** (`supabase_provider.go`) - Open source Firebase alternative
23. **DeepLake** (`deeplake_provider.go`) - Vector database for ML
24. **Provider Agnostic** (`provider_agnostic_provider.go`) - Abstract interface

## Architecture

All providers implement the `VectorProvider` interface defined in `types.go`:

```go
type VectorProvider interface {
    Initialize(ctx context.Context, config interface{}) error
    Start(ctx context.Context) error
    Store(ctx context.Context, vectors []*VectorData) error
    Retrieve(ctx context.Context, ids []string) ([]*VectorData, error)
    Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error)
    FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error)
    // ... other methods
}
```

## Key Features

### Common Capabilities
- Vector storage and retrieval
- Similarity search with multiple metrics (cosine, euclidean, dot product)
- Metadata filtering
- Batch operations
- Collection management
- Index management
- Health monitoring
- Backup and restore
- Performance optimization

### Provider-Specific Features

#### Vector Databases
- **High-performance search**
- **Scalable storage**
- **Advanced indexing**
- **Real-time updates**
- **Multi-tenant support**

#### AI Memory Systems
- **Personality management**
- **Conversation memory**
- **Emotional state tracking**
- **Relationship mapping**
- **Long-term memory**

#### Large Language Models
- **Text generation**
- **Embedding creation**
- **Fine-tuning support**
- **Context management**
- **Token optimization**

## Configuration

Each provider has its own configuration struct with provider-specific settings:

```go
// Example: Pinecone configuration
type PineconeConfig struct {
    APIKey      string `json:"api_key"`
    Environment string `json:"environment"`
    ProjectID   string `json:"project_id"`
    IndexName   string `json:"index_name"`
    // ... other fields
}
```

## Usage Example

```go
// Initialize provider
config := map[string]interface{}{
    "api_key": "your-api-key",
    "environment": "us-west1-gcp",
    "project_id": "your-project-id",
    "index_name": "vectors",
}

provider, err := NewProvider("pinecone", config)
if err != nil {
    return err
}

// Initialize and start
ctx := context.Background()
if err := provider.Initialize(ctx, nil); err != nil {
    return err
}
if err := provider.Start(ctx); err != nil {
    return err
}

// Store vectors
vectors := []*VectorData{
    {
        ID:         "doc1",
        Vector:     []float64{0.1, 0.2, 0.3, ...},
        Metadata:   map[string]interface{}{"title": "Document 1"},
        Collection: "documents",
        Timestamp:  time.Now(),
    },
}

if err := provider.Store(ctx, vectors); err != nil {
    return err
}

// Search vectors
query := &VectorQuery{
    Vector:     []float64{0.15, 0.25, 0.35, ...},
    Collection: "documents",
    TopK:       10,
    Threshold:  0.7,
}

results, err := provider.Search(ctx, query)
if err != nil {
    return err
}
```

## Provider Selection

Choose a provider based on your specific needs:

### For High-Performance Vector Search
- **Pinecone** - Managed, scalable, production-ready
- **Milvus** - Open-source, self-hosted, flexible
- **Qdrant** - Rust-based, fast, efficient

### For AI/ML Applications
- **OpenAI** - GPT models, embeddings, fine-tuning
- **Anthropic** - Claude models, safety-focused
- **HuggingFace** - Open models, community-driven

### For Memory & Character Systems
- **MemGPT** - Advanced AI memory management
- **Character.AI** - Character creation and interaction
- **CrewAI** - Multi-agent coordination

### For Cost-Effective Solutions
- **Chroma** - Open-source, easy to use
- **FAISS** - High-performance, local processing
- **Supabase** - Full-featured, open-source

## Performance Considerations

### Vector Dimensions
- Most providers support 1,000-40,000 dimensions
- Consider dimensionality reduction for large datasets
- Test performance with your specific dimension size

### Batch Operations
- Use batch inserts for better performance
- Typical batch sizes: 100-1000 vectors
- Adjust based on provider limits and network conditions

### Indexing Strategy
- Choose appropriate metric (cosine, euclidean, dot product)
- Consider materialized indexes for frequent queries
- Monitor index size and query performance

## Monitoring and Observability

All providers support:
- Health checks
- Performance metrics
- Error tracking
- Usage statistics
- Cost monitoring

## Security

- API key management
- Encryption at rest and in transit
- Access control and permissions
- Data residency compliance
- Audit logging

## Cost Management

Each provider implements `GetCostInfo()` to help with:
- Usage tracking
- Cost estimation
- Free tier monitoring
- Billing period awareness

## Integration

The provider system is designed to work with:
- REST APIs
- gRPC interfaces
- Database connections
- Message queues
- Event streaming

## Future Enhancements

Planned additions include:
- Additional provider implementations
- Performance optimizations
- Enhanced monitoring
- Better cost tracking
- Automated failover
- Cross-provider replication

## Support

For provider-specific issues:
1. Check provider documentation
2. Review configuration settings
3. Monitor health endpoints
4. Check rate limits and quotas
5. Verify network connectivity

## Contributing

To add a new provider:
1. Implement the `VectorProvider` interface
2. Add provider to registry
3. Include comprehensive tests
4. Document configuration options
5. Add cost tracking support
6. Update this README

## License

This package is part of the HelixCode project and follows the same licensing terms.