FROM python:3.13-slim
ARG GO_VERSION=1.24.6
ARG TARGETARCH=amd64

ENV PYTHONUNBUFFERED=1

RUN set -eux; \
    apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y --no-install-recommends \
        curl \
        ca-certificates \
        git \
        procps; \
    curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz -o go.tar.gz; \
    tar -C /usr/local -xzf go.tar.gz; \
    rm go.tar.gz; \
    apt-get purge -y curl; \
    apt-get autoremove -y; \
    rm -rf /var/lib/apt/lists/*


ENV PATH=$PATH:/usr/local/go/bin

RUN pip install --no-cache-dir uv

# Create a non-root user before setting up the application
RUN groupadd -r appuser && useradd -r -g appuser -m appuser

COPY --chown=appuser:appuser .claude/ /home/appuser/.claude/

WORKDIR /app

COPY pyproject.toml uv.lock ./
COPY src/ ./src/
COPY knowledge_base/ ./knowledge_base/
COPY README.md .

# Change ownership before creating venv and installing packages
RUN chown -R appuser:appuser /app

USER appuser

# Create venv and install packages as the non-root user
RUN uv venv && uv pip install -r pyproject.toml

CMD ["uv", "run", "uvicorn", "ai_guardian_remediation.main:app", "--host", "0.0.0.0", "--port", "8588"]