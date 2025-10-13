GOMOD = go mod
GOBUILD = go build -trimpath -v
GOTEST = go test -cover -race

ROOT = $(shell git rev-parse --show-toplevel)
BIN = dist/objdiff
CMD = "./cmd/objdiff"

THIRD_PARTY_LICENSES = NOTICE

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
lint: check-licenses vet vuln golangci-lint

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

.PHONY: bench
bench:
	cd config ; go test -bench . -count=6 | go tool benchstat -

.PHONY: $(THIRD_PARTY_LICENSES)
$(THIRD_PARTY_LICENSES):
	./bin/license.sh report > $@

.PHONY: check-licenses-diff
check-licenses-diff: $(THIRD_PARTY_LICENSES)
	git diff --exit-code $(THIRD_PARTY_LICENSES)

.PHONY: check-licenses
check-licenses: check-licenses-diff
	./bin/license.sh check

# .PHONY: generate
# generate:
# 	go generate ./...

# .PHONY: clean-generated
# clean-generated:
# 	find . -name "*_generated.go" -type f -delete
