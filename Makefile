.PHONY: help cluster images up e2e drills verify fresh lint test portal clean

help: ## list targets
	@grep -E '^[a-z0-9-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

cluster: ## create the local kind cluster
	. hack/versions.env && kind create cluster --config hack/kind.yaml --image $$KIND_NODE_IMAGE --wait 120s

images: ## build gantry-cloud + provider images and load them into kind
	./hack/images.sh

up: ## install the platform on the current context
	./hack/up.sh

e2e: ## apply every request and run the e2e checks
	./tests/e2e/run.sh

drills: ## install observability and run the failure drills
	./hack/observability.sh && ./hack/drills.sh

verify: ## bootstrap + e2e + snapshot (what CI runs)
	./hack/verify.sh snapshot.json

fresh: ## recreate the kind cluster and verify from scratch
	./hack/fresh-local.sh snapshot.json

lint: ## lint requests with gantry-lint
	cd tools/lint && go run ./cmd/gantry-lint --root ../..

test: ## unit tests: Go modules + portal + alert rules
	cd tools/lint && go test ./...
	cd cloud && go test ./...
	cd provider && go test ./...
	./hack/test-rules.sh
	cd portal && npm test

portal: ## run the portal locally
	cd portal && npm run dev

clean: ## delete the local cluster
	kind delete cluster --name gantry
