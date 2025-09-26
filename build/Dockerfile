# Base Go build
FROM golang:alpine AS build

WORKDIR /workspace

COPY ./ .

RUN go mod download

RUN go build -o app cmd/main.go


# Final image with dependencies pre-installed
FROM alpine
    
WORKDIR /

# Copy binaries and configs
COPY config.yaml .
COPY --from=build /workspace/app .

ENTRYPOINT ["./app"]