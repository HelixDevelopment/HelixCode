"""
Cognee.ai Setup Script
Advanced Knowledge Graph and Data Orchestration Platform
"""

from setuptools import setup, find_packages
import os

# Read requirements
def read_requirements(filename):
    with open(filename, 'r') as f:
        return [line.strip() for line in f if line.strip() and not line.startswith('#')]

# Read README
def read_readme():
    with open('README.md', 'r') as f:
        return f.read()

setup(
    name="cognee-ai",
    version="2.0.0",
    author="Cognee Team",
    author_email="team@cognee.ai",
    description="Advanced AI-powered knowledge graph and data orchestration platform",
    long_description=read_readme(),
    long_description_content_type="text/markdown",
    url="https://github.com/cognee-ai/cognee",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 5 - Production/Stable",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
        "Topic :: Software Development :: Libraries :: Python Modules",
    ],
    python_requires=">=3.8",
    install_requires=read_requirements('requirements.txt'),
    extras_require={
        "dev": read_requirements('requirements-dev.txt'),
        "gpu": read_requirements('requirements-gpu.txt'),
        "web": read_requirements('requirements-web.txt'),
    },
    entry_points={
        "console_scripts": [
            "cognee=cognee.cli:main",
            "cognee-server=cognee.server:main",
            "cognee-web=cognee.web:main",
        ],
    },
    include_package_data=True,
    package_data={
        "cognee": [
            "config/*.yaml",
            "templates/*.html",
            "static/**/*",
        ],
    },
    zip_safe=False,
)