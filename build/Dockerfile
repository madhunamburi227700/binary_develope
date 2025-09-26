# Base Go build
FROM golang:alpine AS build

RUN apk update && apk add --no-cache git

WORKDIR /workspace

COPY ./ .

RUN --mount=type=secret,id=GIT_TOKEN \
    git config --global url."https://$(cat /run/secrets/GIT_TOKEN)@github.com/OpsMx".insteadOf "https://github.com/OpsMx"

RUN go env -w GOPRIVATE=github.com/OpsMx && \
    go mod download

RUN go mod download

RUN go build -o app cmd/main.go


# Final image with dependencies pre-installed
FROM alpine
    
WORKDIR /

# Copy binaries and configs
COPY config.yaml .
COPY --from=build /workspace/app .

ENTRYPOINT ["./app"]