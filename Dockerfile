# Copyright 2021 OpsMx, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#
# Install the latest versions of our mods. This is done as a separate step
# so it will pull from an image cache if possible, unless there are changes.
#

# ───────────────────────────────────────────────────────────────────────────────
# 1) Download Go modules in an isolated stage
# ───────────────────────────────────────────────────────────────────────────────
FROM --platform=${BUILDPLATFORM} golang:alpine AS build-mod
WORKDIR /src

# Copy from build context
COPY --from=gomodcache . /tmp/go-cache/

# Move and set permissions dynamically
RUN GOMOD_PATH=$(go env GOMODCACHE) && \
  mkdir -p "$GOMOD_PATH" && \
  cp -r /tmp/go-cache/* "$GOMOD_PATH"/ && \
  chmod -R a+r "$GOMOD_PATH" && \
  rm -rf /tmp/go-cache

# ───────────────────────────────────────────────────────────────────────────────
# 2) Build your static Go binary
# ───────────────────────────────────────────────────────────────────────────────
FROM build-mod AS build-binaries
WORKDIR /src
COPY . .
ARG GIT_BRANCH
ARG GIT_HASH
ARG BUILD_TYPE
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GIT_BRANCH=${GIT_BRANCH} \
    GIT_HASH=${GIT_HASH} \
    BUILD_TYPE=${BUILD_TYPE}
RUN go build -o /toolchain-service \
    -ldflags "\
      -X 'github.com/OpsMx/go-app-base/version.buildType=${BUILD_TYPE}' \
      -X 'github.com/OpsMx/go-app-base/version.gitHash=${GIT_HASH}' \
      -X 'github.com/OpsMx/go-app-base/version.gitBranch=${GIT_BRANCH}'" \
    .
# ───────────────────────────────────────────────────────────────────────────────
# 3) Final image: Debian slim + all tools, but cleaned up
# ───────────────────────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS final

# 3a) Install all apt packages in one go, no recommends, then clean up
RUN apt-get update && \
    apt-get -y dist-upgrade && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      gnupg \
      lsb-release \
      apt-transport-https \
      python3 \
      python3-venv \
      python3-dev \
      gcc \
      unzip \
      git && \
    rm -rf /var/lib/apt/lists/*

# 3b) Add Docker CE repo & install Docker CLI + containerd
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/debian/gpg \
      | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
    echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.gpg] \
      https://download.docker.com/linux/debian $(lsb_release -cs) stable" \
      > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends docker-ce-cli 'containerd.io=1.7.29-1~debian.12~bookworm' && \
    rm -rf /var/lib/apt/lists/*

# 3c) Set up Python venv and install Python-based tools, clean pip cache
# Pin ruamel.yaml<0.19.0 to avoid Zig toolchain requirement (0.19.0 switched to ruamel.yaml.clibz)
RUN python3 -m venv /venv && \
    /venv/bin/pip install --no-cache-dir --upgrade \
      "ruamel.yaml<0.19.0" \
      "azure-identity>=1.16.1" \
      semgrep \
      pipx && \
    rm -rf /root/.cache/pip

# Install ScoutSuite in isolation
RUN python3 -m venv /venv && \
    /venv/bin/pipx install "scoutsuite>=5.12.0"
# Set PATH manually (this is alterante to `pipx ensurepath`)
ENV PATH="/root/.local/bin:$PATH"

# Install Node.js and cdxgen
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    npm install -g @cyclonedx/cdxgen --unsafe-perm=false && \
    npm cache clean --force

# ─────────────────────────────────────────────────────────────
# Enable Java (Maven + Gradle) for cdxgen
# ─────────────────────────────────────────────────────────────
RUN curl -fsSL https://download.oracle.com/java/21/latest/jdk-21_linux-x64_bin.tar.gz -o /tmp/jdk.tar.gz && \
    mkdir -p /usr/lib/jvm && \
    tar -xzf /tmp/jdk.tar.gz -C /usr/lib/jvm && \
    mv /usr/lib/jvm/jdk-21* /usr/lib/jvm/java-21 && \
    rm /tmp/jdk.tar.gz

RUN apt-get update && \
    apt-get install -y --no-install-recommends maven gradle && \
    rm -rf /var/lib/apt/lists/*

ENV JAVA_HOME=/usr/lib/jvm/java-21
ENV PATH="${JAVA_HOME}/bin:${PATH}"
ENV PATH="/venv/bin:${PATH}"

# 3d) Install Helm CLI and remove installer
RUN curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 \
      -o /tmp/get_helm.sh && \
    chmod +x /tmp/get_helm.sh && \
    /tmp/get_helm.sh && \
    rm -f /tmp/get_helm.sh

# 3e) Install kubescape (static binary)
RUN curl -s -L -o install.sh https://raw.githubusercontent.com/kubescape/kubescape/v3.0.20/install.sh \
    && /bin/bash install.sh

# 3f) Install AWS CLI v2 and clean up
RUN curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" \
      -o /tmp/awscliv2.zip && \
    unzip /tmp/awscliv2.zip -d /tmp && \
    /tmp/aws/install && \
    rm -rf /tmp/aws /tmp/awscliv2.zip

# 3g) Install Trivy, Grype, Scorecard, Snyk, Syft, tfsec; remove archives
RUN set -eux; \
    for tool in \
      "https://github.com/aquasecurity/trivy/releases/download/v0.69.3/trivy_0.69.3_Linux-64bit.tar.gz" \
      "https://github.com/anchore/grype/releases/download/v0.102.0/grype_0.102.0_linux_amd64.tar.gz" \
    ; do \
      fname=${tool##*/}; \
      curl -fsSL -o /tmp/$fname $tool; \
      tar -xzf /tmp/$fname -C /usr/local/bin; \
      rm -f /tmp/$fname; \
    done && \
    # Scorecard
    curl -sfL https://github.com/ossf/scorecard/releases/download/v5.3.0/scorecard_5.3.0_linux_amd64.tar.gz | tar -xz scorecard && \
    install -m 755 scorecard /usr/local/bin/scorecard && \
    rm -f scorecard && \
    # Snyk
    curl -fsSL -o /usr/local/bin/snyk \
      https://github.com/snyk/cli/releases/download/v1.1300.2/snyk-linux && \
    chmod +x /usr/local/bin/snyk && \
    # Syft
    curl -fsSL "https://raw.githubusercontent.com/anchore/syft/v1.36.0/install.sh" \
      | sh -s -- -b /usr/local/bin v1.36.0 && \
    # tfsec
    curl -fsSL -o /usr/local/bin/tfsec \
      https://github.com/aquasecurity/tfsec/releases/download/v1.28.14/tfsec-linux-amd64 && \
    chmod +x /usr/local/bin/tfsec && \
    # Opengrep
    curl -fsSL -o /usr/local/bin/opengrep \
      https://github.com/opengrep/opengrep/releases/download/v1.0.0-alpha.15/opengrep_manylinux_x86 && \
    chmod +x /usr/local/bin/opengrep && \
    # Codacy
    curl -fsSL -o /tmp/codacy.tar.gz https://github.com/codacy/codacy-analysis-cli/archive/master.tar.gz && \
    tar -xzf /tmp/codacy.tar.gz -C /tmp && \
    cp /tmp/codacy-analysis-cli-*/bin/codacy-analysis-cli.sh /usr/local/bin/codacy-analysis-cli && \
    chmod +x /usr/local/bin/codacy-analysis-cli && \
    rm -rf /tmp/codacy*

    # Install Go
ENV GO_VERSION=1.25.7
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz \
    # Remove unnecessary files
    rm -rf /usr/local/go/pkg/*/race \
           /usr/local/go/pkg/*/msan \
           /usr/local/go/src \
           /usr/local/go/doc \
           /usr/local/go/test \
           /usr/local/go/blog \
           /usr/local/go/misc
ENV PATH="/usr/local/go/bin:${PATH}"

# Clean up build-only packages AFTER
RUN apt-get purge -y curl && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# 3h) Copy your entrypoint & Go binary
COPY docker/run.sh /tools/run.sh
COPY --from=build-binaries /toolchain-service /tools/toolchain-service
RUN chmod +x /tools/run.sh /tools/toolchain-service
WORKDIR /tools
ENTRYPOINT ["/bin/sh", "/tools/run.sh"]
EXPOSE 8090
CMD ["/tools/toolchain-service"]