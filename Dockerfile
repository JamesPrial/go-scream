FROM golang:1.24-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    libopus-dev \
    pkg-config \
    g++ cmake libssl-dev git curl zip unzip tar \
    && rm -rf /var/lib/apt/lists/*

# Clone and build dave-go-bindings (libdave C++ library for DAVE E2EE).
RUN git clone --recurse-submodules https://github.com/JamesPrial/dave-go-bindings.git /dave-go-bindings
WORKDIR /dave-go-bindings
RUN make
# Symlink the vcpkg triplet so the hardcoded arm64-osx CGO LDFLAGS path resolves
# on Linux. This works around the path mismatch until dave-go-bindings is fixed.
RUN set -e; \
    triplet_dir=$(ls -d build/vcpkg_installed/*/lib 2>/dev/null | grep -v vcpkg | head -1); \
    if [ -n "$triplet_dir" ] && [ ! -d build/vcpkg_installed/arm64-osx ]; then \
        ln -s "$(dirname "$triplet_dir")" build/vcpkg_installed/arm64-osx; \
    fi

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -o /scream ./cmd/scream

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libopus0 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /scream /usr/local/bin/scream

ENTRYPOINT ["scream"]
