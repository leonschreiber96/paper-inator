# paper-inator build helpers. CGO is disabled so the result is a single static
# binary that cross-compiles trivially (pure-Go SQLite driver).

BINARY := paper-inator
PKG    := .

export CGO_ENABLED := 0

.PHONY: build run test vet tidy clean cross

build: ## Build the single binary for the host platform
	go build -o $(BINARY) $(PKG)

run: ## Run directly from source
	go run $(PKG)

test: ## Run the test suite
	go test ./...

vet: ## Static analysis
	go vet ./...

tidy: ## Sync go.mod/go.sum
	go mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64

# Example cross-compile targets for common Linux servers.
cross: ## Build Linux amd64 + arm64 binaries
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 $(PKG)
	GOOS=linux GOARCH=arm64 go build -o $(BINARY)-linux-arm64 $(PKG)
