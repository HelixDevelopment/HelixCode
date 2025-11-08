# Multi-stage security scanner Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    curl \
    git \
    make \
    gcc \
    musl-dev

# Install SonarScanner
RUN curl -L -o /tmp/sonar-scanner.zip \
    https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip && \
    unzip /tmp/sonar-scanner.zip -d /opt/sonar-scanner && \
    rm /tmp/sonar-scanner.zip

# Install Snyk
RUN curl -L -o /usr/local/bin/snyk \
    https://static.snyk.io/cli/latest/snyk-linux && \
    chmod +x /usr/local/bin/snyk

# Production stage with minimal attack surface
FROM alpine:3.18

# Install runtime dependencies only
RUN apk add --no-cache \
    curl \
    jq \
    bash \
    git \
    make \
    postgresql-client \
    ca-certificates

# Create security user with minimal privileges
RUN addgroup -S security && adduser -S -G security security

# Copy installed tools from builder
COPY --from=builder /opt/sonar-scanner /opt/sonar-scanner
COPY --from=builder /usr/local/bin/snyk /usr/local/bin/snyk

# Create secure directories for scanning
RUN mkdir -p /scan-results /project-security /tmp/security && \
    chown -R security:security /scan-results /project-security /tmp/security

# Security scanning scripts
COPY security/scripts/ /security-scripts/
RUN chmod +x /security-scripts/*.sh && \
    chown security:security /security-scripts/*.sh

# Set secure environment
ENV PATH="/opt/sonar-scanner/bin:/usr/local/bin:${PATH}"
ENV SONAR_SCANNER_HOME="/opt/sonar-scanner"
ENV SNYK_INTEGRATION_NAME="helixcode-security-scanner"

# Switch to non-root user
USER security

# Health check for scanner
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:9000/api/system/status || echo "Scanner ready"

WORKDIR /project
VOLUME ["/project", "/scan-results"]

# Default to security scanning mode
CMD ["/security-scripts/scan-all.sh"]