test:
	@ginkgo -r -cover --coverprofile=coverage.out
	@go tool cover -html=coverage.out

init-test:
	@if [ -z "$(name)" ]; then echo "Error: Please provide a directory name, e.g., make test-init name=views"; exit 1; fi
	@cd $(name) && ginkgo bootstrap

MOCKS_DIR = $(dir $(name))mocks/
mock:
	@if [ -z "$(name)" ]; then echo "Error: Please provide a module name, e.g., make mock name=app/api/users.go"; exit 1; fi
	@mockgen -source=$(name) -destination=$(MOCKS_DIR)$(notdir $(name))
	@echo "Mock generated" $(MOCKS_DIR)$(notdir $(name))
