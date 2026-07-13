FROM python:3.12-slim

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

# Copy dependency files first for layer caching
COPY pyproject.toml uv.lock ./

# Install dependencies (no dev extras)
RUN uv sync --no-dev --frozen

# Copy source code (src/ layout)
COPY src/ ./src/

ENV PYTHONUNBUFFERED=1

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD cat /tmp/sinapsis_health || exit 1

ENTRYPOINT ["uv", "run", "python", "-m", "sinapsis_ai.main"]
