"""
Cognee.ai Core Engine

Central orchestrator for knowledge graph management,
data processing, and AI-powered insights.
"""

import asyncio
import logging
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass, field
from pathlib import Path
import json
import time

from .config import CogneeConfig, HostProfile
from .graph import KnowledgeGraph, GraphNode, GraphEdge, GraphQuery
from .api import CogneeAPI, CogneeServer
from .embedding import EmbeddingManager, VectorStore
from .search import SemanticSearch, KnowledgeSearch
from .processing import DataProcessor, Orchestrator
from .optimization import PerformanceOptimizer, HostOptimizer
from .cache import CacheManager
from .monitoring import MetricsCollector, HealthChecker
from .utils import logger, setup_logging


@dataclass
class CogneeMetrics:
    """Cognee performance and operational metrics."""
    
    # Knowledge Graph Metrics
    total_nodes: int = 0
    total_edges: int = 0
    graph_complexity: float = 0.0
    
    # Processing Metrics
    processed_documents: int = 0
    processing_time: float = 0.0
    embeddings_generated: int = 0
    
    # Search Metrics
    search_queries: int = 0
    average_response_time: float = 0.0
    cache_hit_rate: float = 0.0
    
    # Performance Metrics
    memory_usage: float = 0.0
    cpu_usage: float = 0.0
    gpu_usage: float = 0.0
    
    # Integration Metrics
    provider_connections: int = 0
    model_integrations: int = 0
    api_requests: int = 0


class CogneeCore:
    """
    Core Cognee.ai engine for knowledge graph management.
    
    Provides centralized orchestration for:
    - Knowledge graph creation and management
    - Data processing and analysis
    - Semantic search and insights
    - Provider and model integration
    - Performance optimization
    """
    
    def __init__(self, config: Optional[CogneeConfig] = None):
        """Initialize Cognee core with configuration."""
        self.config = config or CogneeConfig.default()
        self.logger = setup_logging(self.config.logging)
        
        # Core components
        self.knowledge_graph = KnowledgeGraph(config=self.config.graph)
        self.data_processor = DataProcessor(config=self.config.processing)
        self.embedding_manager = EmbeddingManager(config=self.config.embedding)
        self.search_engine = SemanticSearch(config=self.config.search)
        self.orchestrator = Orchestrator(config=self.config.orchestration)
        
        # Management components
        self.performance_optimizer = PerformanceOptimizer(config=self.config.performance)
        self.cache_manager = CacheManager(config=self.config.cache)
        self.metrics_collector = MetricsCollector(config=self.config.metrics)
        self.health_checker = HealthChecker(config=self.config.health)
        
        # API components
        self.api = CogneeAPI(core=self, config=self.config.api)
        self.server = CogneeServer(core=self, config=self.config.server)
        
        # Runtime state
        self._initialized = False
        self._running = False
        self._metrics = CogneeMetrics()
        
        # Background tasks
        self._background_tasks = set()
        
        self.logger.info("Cognee Core initialized")
    
    async def initialize(self) -> bool:
        """Initialize all Cognee components."""
        if self._initialized:
            return True
        
        try:
            self.logger.info("Initializing Cognee components...")
            start_time = time.time()
            
            # Apply host-aware optimization
            if self.config.dynamic_config:
                await self._apply_host_optimization()
            
            # Initialize core components
            await self.knowledge_graph.initialize()
            await self.data_processor.initialize()
            await self.embedding_manager.initialize()
            await self.search_engine.initialize()
            await self.orchestrator.initialize()
            
            # Initialize management components
            await self.performance_optimizer.initialize()
            await self.cache_manager.initialize()
            await self.metrics_collector.initialize()
            await self.health_checker.initialize()
            
            # Initialize API components
            await self.api.initialize()
            await self.server.initialize()
            
            # Start background tasks
            await self._start_background_tasks()
            
            self._initialized = True
            initialization_time = time.time() - start_time
            
            self.logger.info(f"Cognee initialized successfully in {initialization_time:.2f}s")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to initialize Cognee: {e}")
            return False
    
    async def _apply_host_optimization(self) -> None:
        """Apply host-aware optimizations."""
        try:
            host_profile = HostProfile.detect()
            optimizer = HostOptimizer(profile=host_profile)
            
            # Optimize configuration based on host
            optimized_config = optimizer.optimize_config(self.config)
            self.config.update(optimized_config)
            
            self.logger.info(f"Applied host-aware optimization: {host_profile}")
            
        except Exception as e:
            self.logger.warning(f"Host optimization failed: {e}")
    
    async def _start_background_tasks(self) -> None:
        """Start background maintenance tasks."""
        # Metrics collection task
        metrics_task = asyncio.create_task(
            self._collect_metrics_loop(),
            name="metrics_collection"
        )
        self._background_tasks.add(metrics_task)
        
        # Health monitoring task
        health_task = asyncio.create_task(
            self._health_monitoring_loop(),
            name="health_monitoring"
        )
        self._background_tasks.add(health_task)
        
        # Performance optimization task
        perf_task = asyncio.create_task(
            self._performance_optimization_loop(),
            name="performance_optimization"
        )
        self._background_tasks.add(perf_task)
        
        self.logger.info("Background tasks started")
    
    async def _collect_metrics_loop(self) -> None:
        """Collect metrics in background."""
        while self._running:
            try:
                # Update knowledge graph metrics
                self._metrics.total_nodes = await self.knowledge_graph.node_count()
                self._metrics.total_edges = await self.knowledge_graph.edge_count()
                self._metrics.graph_complexity = await self.knowledge_graph.complexity_score()
                
                # Update processing metrics
                self._metrics.processing_time = await self.data_processor.average_processing_time()
                self._metrics.embeddings_generated = await self.embedding_manager.total_embeddings()
                
                # Update search metrics
                self._metrics.search_queries = await self.search_engine.total_queries()
                self._metrics.average_response_time = await self.search_engine.average_response_time()
                self._metrics.cache_hit_rate = await self.cache_manager.hit_rate()
                
                # Update performance metrics
                self._metrics.memory_usage = await self.performance_optimizer.memory_usage()
                self._metrics.cpu_usage = await self.performance_optimizer.cpu_usage()
                self._metrics.gpu_usage = await self.performance_optimizer.gpu_usage()
                
                # Update integration metrics
                self._metrics.provider_connections = await self._get_provider_connections()
                self._metrics.model_integrations = await self._get_model_integrations()
                self._metrics.api_requests = await self.api.total_requests()
                
                # Collect and store metrics
                await self.metrics_collector.collect(self._metrics)
                
            except Exception as e:
                self.logger.error(f"Metrics collection failed: {e}")
            
            await asyncio.sleep(self.config.metrics.collection_interval)
    
    async def _health_monitoring_loop(self) -> None:
        """Monitor system health in background."""
        while self._running:
            try:
                health_status = await self.health_checker.check_health()
                
                if not health_status.healthy:
                    self.logger.warning(f"Health check failed: {health_status}")
                    await self._handle_health_issue(health_status)
                
            except Exception as e:
                self.logger.error(f"Health monitoring failed: {e}")
            
            await asyncio.sleep(self.config.health.check_interval)
    
    async def _performance_optimization_loop(self) -> None:
        """Optimize performance in background."""
        while self._running:
            try:
                optimization_result = await self.performance_optimizer.optimize()
                
                if optimization_result.optimized:
                    self.logger.info(f"Performance optimization applied: {optimization_result}")
                    await self._apply_performance_optimizations(optimization_result)
                
            except Exception as e:
                self.logger.error(f"Performance optimization failed: {e}")
            
            await asyncio.sleep(self.config.performance.optimization_interval)
    
    # Knowledge Graph Operations
    
    async def add_knowledge(self, data: Union[str, Dict, List], 
                          metadata: Optional[Dict] = None) -> List[str]:
        """
        Add knowledge to the knowledge graph.
        
        Args:
            data: Data to add (text, dict, or list)
            metadata: Optional metadata for the knowledge
            
        Returns:
            List of created node IDs
        """
        if not self._initialized:
            raise RuntimeError("Cognee not initialized")
        
        try:
            # Process data
            processed_data = await self.data_processor.process(data, metadata)
            
            # Generate embeddings
            embeddings = await self.embedding_manager.generate_embeddings(processed_data)
            
            # Add to knowledge graph
            node_ids = await self.knowledge_graph.add_nodes(processed_data, embeddings)
            
            # Update cache
            await self.cache_manager.cache_knowledge(node_ids, processed_data)
            
            # Update metrics
            self._metrics.processed_documents += 1
            
            self.logger.info(f"Added {len(node_ids)} knowledge nodes")
            return node_ids
            
        except Exception as e:
            self.logger.error(f"Failed to add knowledge: {e}")
            raise
    
    async def query_knowledge(self, query: str, 
                            filters: Optional[Dict] = None,
                            limit: Optional[int] = None) -> List[Dict]:
        """
        Query the knowledge graph.
        
        Args:
            query: Search query
            filters: Optional filters
            limit: Optional result limit
            
        Returns:
            List of matching knowledge items
        """
        if not self._initialized:
            raise RuntimeError("Cognee not initialized")
        
        try:
            # Check cache first
            cache_key = f"query:{query}:{hash(str(filters))}:{limit}"
            cached_result = await self.cache_manager.get(cache_key)
            
            if cached_result:
                self.logger.debug(f"Cache hit for query: {query}")
                return cached_result
            
            # Perform semantic search
            results = await self.search_engine.semantic_search(
                query, 
                filters=filters,
                limit=limit
            )
            
            # Update cache
            await self.cache_manager.set(cache_key, results)
            
            # Update metrics
            self._metrics.search_queries += 1
            
            return results
            
        except Exception as e:
            self.logger.error(f"Failed to query knowledge: {e}")
            raise
    
    async def get_insights(self, analysis_type: str,
                         parameters: Optional[Dict] = None) -> Dict:
        """
        Get insights from the knowledge graph.
        
        Args:
            analysis_type: Type of analysis to perform
            parameters: Optional analysis parameters
            
        Returns:
            Analysis results and insights
        """
        if not self._initialized:
            raise RuntimeError("Cognee not initialized")
        
        try:
            # Perform graph analysis
            insights = await self.knowledge_graph.analyze(analysis_type, parameters)
            
            # Enhance with AI insights
            ai_insights = await self._generate_ai_insights(insights)
            
            # Combine insights
            combined_insights = {
                "graph_insights": insights,
                "ai_insights": ai_insights,
                "analysis_type": analysis_type,
                "parameters": parameters,
                "timestamp": time.time()
            }
            
            self.logger.info(f"Generated insights for {analysis_type}")
            return combined_insights
            
        except Exception as e:
            self.logger.error(f"Failed to generate insights: {e}")
            raise
    
    # Provider and Model Integration
    
    async def integrate_provider(self, provider_name: str,
                              provider_config: Dict) -> bool:
        """
        Integrate with a provider.
        
        Args:
            provider_name: Name of the provider
            provider_config: Provider configuration
            
        Returns:
            True if integration successful
        """
        try:
            # Initialize provider integration
            integration = await self.orchestrator.integrate_provider(
                provider_name,
                provider_config
            )
            
            # Configure for Cognee integration
            if integration:
                await self._configure_provider_integration(provider_name, integration)
                
                self.logger.info(f"Successfully integrated provider: {provider_name}")
                return True
            
            return False
            
        except Exception as e:
            self.logger.error(f"Failed to integrate provider {provider_name}: {e}")
            return False
    
    async def integrate_model(self, provider_name: str,
                          model_name: str,
                          model_config: Dict) -> bool:
        """
        Integrate with a specific model.
        
        Args:
            provider_name: Name of the provider
            model_name: Name of the model
            model_config: Model configuration
            
        Returns:
            True if integration successful
        """
        try:
            # Initialize model integration
            integration = await self.orchestrator.integrate_model(
                provider_name,
                model_name,
                model_config
            )
            
            # Configure for Cognee integration
            if integration:
                await self._configure_model_integration(provider_name, model_name, integration)
                
                self.logger.info(f"Successfully integrated model: {provider_name}/{model_name}")
                return True
            
            return False
            
        except Exception as e:
            self.logger.error(f"Failed to integrate model {provider_name}/{model_name}: {e}")
            return False
    
    # API Operations
    
    async def start_api(self, host: Optional[str] = None,
                       port: Optional[int] = None) -> None:
        """Start Cognee API server."""
        if not self._initialized:
            raise RuntimeError("Cognee not initialized")
        
        try:
            self._running = True
            
            # Start API server
            await self.server.start(host=host, port=port)
            
            self.logger.info(f"Cognee API started on {host}:{port}")
            
        except Exception as e:
            self.logger.error(f"Failed to start API: {e}")
            self._running = False
            raise
    
    async def stop_api(self) -> None:
        """Stop Cognee API server."""
        try:
            self._running = False
            
            # Stop background tasks
            for task in self._background_tasks:
                task.cancel()
            
            # Wait for tasks to complete
            await asyncio.gather(*self._background_tasks, return_exceptions=True)
            
            # Stop API server
            await self.server.stop()
            
            self.logger.info("Cognee API stopped")
            
        except Exception as e:
            self.logger.error(f"Failed to stop API: {e}")
    
    # Metrics and Status
    
    def get_metrics(self) -> CogneeMetrics:
        """Get current metrics."""
        return self._metrics
    
    async def get_status(self) -> Dict:
        """Get current system status."""
        status = {
            "initialized": self._initialized,
            "running": self._running,
            "metrics": self._metrics,
            "health": await self.health_checker.check_health(),
            "performance": await self.performance_optimizer.get_status(),
            "config": self.config.to_dict(),
            "uptime": time.time() - self._start_time if hasattr(self, '_start_time') else 0
        }
        
        return status
    
    # Private Helper Methods
    
    async def _generate_ai_insights(self, graph_insights: Dict) -> Dict:
        """Generate AI-powered insights from graph analysis."""
        # This would integrate with available AI models
        # For now, return placeholder insights
        return {
            "ai_analysis": "Advanced AI insights would be generated here",
            "recommendations": [],
            "confidence_scores": {},
            "related_concepts": []
        }
    
    async def _configure_provider_integration(self, provider_name: str, integration) -> None:
        """Configure provider for Cognee integration."""
        # Configure provider to work with Cognee knowledge graph
        # This would involve setting up callbacks, data flows, etc.
        pass
    
    async def _configure_model_integration(self, provider_name: str, 
                                       model_name: str, integration) -> None:
        """Configure model for Cognee integration."""
        # Configure model to work with Cognee search and analytics
        # This would involve setting up embeddings, search indexing, etc.
        pass
    
    async def _get_provider_connections(self) -> int:
        """Get number of active provider connections."""
        # Return count of active provider connections
        return self._metrics.provider_connections
    
    async def _get_model_integrations(self) -> int:
        """Get number of model integrations."""
        # Return count of integrated models
        return self._metrics.model_integrations
    
    async def _handle_health_issue(self, health_status) -> None:
        """Handle health monitoring issues."""
        # Implement automatic recovery or alerting
        pass
    
    async def _apply_performance_optimizations(self, optimization_result) -> None:
        """Apply performance optimizations."""
        # Apply the recommended optimizations
        pass
    
    # Context Manager
    
    async def __aenter__(self):
        """Async context manager entry."""
        await self.initialize()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        if self._running:
            await self.stop_api()