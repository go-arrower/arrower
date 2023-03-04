.PHONE: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'




.PHONY: static-check
static-check: ## Run static code checks
	golangci-lint run

.PHONY: generate
generate: ## Generate all code to run the service
	go generate ./...

.PHONY: test
test: static-check generate test-unit ## Run all tests
	go tool cover -html=cover.out -o cover.html; xdg-open cover.html
	go tool cover -func cover.out | grep total:


.PHONY: test-unit
test-unit:
	go test -race ./... -coverprofile cover.out




.PHONY: dev-tools
dev-tools: ## Initialise this machine with development dependencies
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b $(go env GOPATH)/bin v1.51.1

.PHONY: dev-run
dev-run:
	docker-compose pull
	docker-compose up -d postgres pgadmin
	xdg-open http://localhost:8081 # open pgadmin in the browser

.PHONY: dev-db
dev-db:
	PGPASSWORD=secret psql -U arrower -d arrower -h localhost