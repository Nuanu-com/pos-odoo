test:
	@ginkgo -r -cover --coverprofile=coverage.out
	@go tool cover -html=coverage.out

init-test:
	@if [ -z "$(name)" ]; then echo "Error: Please provide a directory name, e.g., make test-init name=views"; exit 1; fi
	@cd $(name) && ginkgo bootstrap
