module github.com/opsmx/ai-gyardian-api

go 1.24.0

toolchain go1.24.7

require (
	github.com/antonlindstrom/pgstore v0.0.0-20220421113606-e3a6e3fed12a
	github.com/google/uuid v1.6.0
	github.com/gorilla/securecookie v1.1.2
	github.com/gorilla/sessions v1.4.0
	github.com/lib/pq v1.10.9
	github.com/rbcervilla/redisstore/v9 v9.0.0
	github.com/redis/go-redis/v9 v9.14.0
	github.com/rs/zerolog v1.34.0
	go.yaml.in/yaml/v2 v2.4.2
	golang.org/x/oauth2 v0.31.0
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

require (
	github.com/OpsMx/go-app-base v0.0.24
	github.com/gorilla/mux v1.8.1
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.17.0 // indirect
)
