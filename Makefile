.PHONY: build test testacc verify clean fmt-check

# SUPERMETAL_AGENT_BINARY must be set to the path of the supermetal agent binary
# Example: export SUPERMETAL_AGENT_BINARY=/path/to/supermetal

build:
	go build -o terraform-provider-supermetal

test:
	go test ./...

testacc:
	@if [ -z "$$SUPERMETAL_AGENT_BINARY" ]; then \
		echo "Error: SUPERMETAL_AGENT_BINARY must be set"; \
		exit 1; \
	fi
	TF_ACC=1 go test ./internal/provider/... -v -timeout 10m -count=1

fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Go files not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

verify: build fmt-check
	go vet ./...
	go test ./... -count=1
	@if [ -z "$$SUPERMETAL_AGENT_BINARY" ]; then \
		echo "Error: SUPERMETAL_AGENT_BINARY must be set for full verification"; \
		exit 1; \
	fi
	TF_ACC=1 go test ./internal/provider/... -v -timeout 10m -count=1 2>&1 | tee /tmp/verify-$$(date +%s).log
	@echo "All checks passed"

clean:
	rm -f terraform-provider-supermetal
