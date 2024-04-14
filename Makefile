
.PHONY: generate
generate: generate-buf generate-go-jet ## Run all the generate commands

.PHONY: generate-buf
generate-buf: ## Generate the protobuf files
	buf generate

.PHONY: generate-go-jet
generate-go-jet: ## Generate the go-jet files
	cd backend && go run tools/jet/main.go

.PHONY: verify
verify: ## Verify all staged files
	pre-commit run --all-files

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
