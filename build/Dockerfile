FROM golang:alpine AS build

RUN apk update && apk add --no-cache git
WORKDIR /workspace
COPY ./ .
RUN --mount=type=secret,id=GIT_TOKEN \
    git config --global url."https://$(cat /run/secrets/GIT_TOKEN)@github.com/OpsMx".insteadOf "https://github.com/OpsMx"
RUN go env -w GOPRIVATE=github.com/OpsMx
RUN go mod download
RUN go build -o app cmd/main.go
    
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
WORKDIR /
    RUN apt update && \
        apt install -y python3 python3-pip git wget && \
        rm -rf /var/lib/apt/lists/*
ENV SEMGREP_DISABLE_IO_URING=1
RUN pip install --no-cache-dir semgrep==1.63.0 && \
    semgrep --version

COPY config.yaml .
COPY --from=build /workspace/app .

ENTRYPOINT ["./app"]