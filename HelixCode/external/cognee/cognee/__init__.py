"""
Cognee.ai - Advanced Knowledge Graph and Data Orchestration Platform

Core library for building, managing, and analyzing knowledge graphs
with AI-powered insights and real-time processing capabilities.
"""

__version__ = "2.0.0"
__author__ = "Cognee Team"
__email__ = "team@cognee.ai"
__description__ = "Advanced AI-powered knowledge graph and data orchestration platform"

# Core components
from .core import CogneeCore, KnowledgeGraph
from .api import CogneeAPI, CogneeServer
from .config import CogneeConfig, ConfigManager
from .utils import logger, setup_logging

# Knowledge Graph operations
from .graph import GraphNode, GraphEdge, GraphQuery, GraphAnalytics
from .embedding import EmbeddingManager, VectorStore
from .search import SemanticSearch, KnowledgeSearch
from .processing import DataProcessor, Orchestrator

# Configuration and management
from .optimization import PerformanceOptimizer, HostOptimizer
from .cache import CacheManager, RedisCache, MemoryCache
from .monitoring import MetricsCollector, HealthChecker

__all__ = [
    # Core
    "CogneeCore",
    "KnowledgeGraph",
    "CogneeAPI",
    "CogneeServer",
    "CogneeConfig",
    "ConfigManager",
    "logger",
    "setup_logging",
    
    # Knowledge Graph
    "GraphNode",
    "GraphEdge",
    "GraphQuery",
    "GraphAnalytics",
    "EmbeddingManager",
    "VectorStore",
    "SemanticSearch",
    "KnowledgeSearch",
    "DataProcessor",
    "Orchestrator",
    
    # Management
    "PerformanceOptimizer",
    "HostOptimizer",
    "CacheManager",
    "RedisCache",
    "MemoryCache",
    "MetricsCollector",
    "HealthChecker",
]

# Version info
def get_version():
    """Get Cognee version information."""
    return {
        "version": __version__,
        "author": __author__,
        "email": __email__,
        "description": __description__,
    }

# Quick start helper
def quick_start(config_path=None, host_aware=True):
    """
    Quick start Cognee with optimal configuration.
    
    Args:
        config_path: Path to configuration file
        host_aware: Enable host-aware optimization
        
    Returns:
        CogneeCore instance
    """
    config = CogneeConfig.from_file(config_path) if config_path else CogneeConfig.default()
    
    if host_aware:
        config.apply_host_optimization()
    
    return CogneeCore(config=config)

# Auto-initialize
def auto_initialize():
    """Auto-initialize Cognee with default settings."""
    return quick_start()

# Export for convenience
__version_info__ = tuple(int(i) for i in __version__.split('.'))