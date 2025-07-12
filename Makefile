GOMOD = go mod
GOBUILD = go build -trimpath -v
GOTEST = go test -cover -race

ROOT = $(shell git rev-parse --show-toplevel)
BIN = dist/objdiff
CMD = "./cmd/objdiff"

.PHONY: $(BIN)
$(BIN):
	./bin/build.sh -o $@ $(CMD)

.PHONY: test
test:
	$(GOTEST) ./...

.PHONY: init
init:
	$(GOMOD) tidy -v

.PHONY: lint
lint: vet vuln golangci-lint

.PHONY: vuln
vuln:
	go tool govulncheck ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: golangci-lint
golangci-lint:
	go tool golangci-lint config verify -v
	go tool golangci-lint run

.PHONY: golden
golden:
	./bin/golden.sh

# .PHONY: generate
# generate:
# 	go generate ./...

# .PHONY: clean-generated
# clean-generated:
# 	find . -name "*_generated.go" -type f -delete
