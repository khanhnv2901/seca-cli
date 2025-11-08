# SECA-CLI Compliance Makefile
# Automates evidence verification, signing, and retention management

# Default engagement ID (override with: make verify ENGAGEMENT_ID=123)
ENGAGEMENT_ID ?=
RESULTS_DIR ?= ./results
RETENTION_DAYS ?= 90

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help verify verify-all sign sign-all purge-raw package clean

help: ## Show this help message
	@echo "SECA-CLI Compliance Makefile"
	@echo ""
	@echo "Usage: make [target] ENGAGEMENT_ID=<id>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make verify ENGAGEMENT_ID=1762627948156627663"
	@echo "  make sign ENGAGEMENT_ID=1762627948156627663"
	@echo "  make purge-raw ENGAGEMENT_ID=1762627948156627663 RETENTION_DAYS=90"
	@echo "  make package ENGAGEMENT_ID=1762627948156627663"

verify: ## Verify SHA256 hashes for a specific engagement
	@if [ -z "$(ENGAGEMENT_ID)" ]; then \
		echo "$(RED)Error: ENGAGEMENT_ID is required$(NC)"; \
		echo "Usage: make verify ENGAGEMENT_ID=<id>"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Verifying evidence integrity for engagement $(ENGAGEMENT_ID)...$(NC)"
	@cd $(RESULTS_DIR)/$(ENGAGEMENT_ID) && \
		if sha256sum -c *.sha256 2>/dev/null; then \
			echo "$(GREEN)✓ All hashes verified successfully$(NC)"; \
		else \
			echo "$(RED)✗ Hash verification failed$(NC)"; \
			exit 1; \
		fi

verify-all: ## Verify SHA256 hashes for all engagements
	@echo "$(YELLOW)Verifying all engagements...$(NC)"
	@for dir in $(RESULTS_DIR)/*/; do \
		if [ -d "$$dir" ]; then \
			eng_id=$$(basename "$$dir"); \
			echo "Checking $$eng_id..."; \
			cd "$$dir" && sha256sum -c *.sha256 2>/dev/null || echo "$(RED)Failed: $$eng_id$(NC)"; \
			cd - > /dev/null; \
		fi \
	done
	@echo "$(GREEN)Verification complete$(NC)"

sign: ## Sign audit and results files with GPG for a specific engagement
	@if [ -z "$(ENGAGEMENT_ID)" ]; then \
		echo "$(RED)Error: ENGAGEMENT_ID is required$(NC)"; \
		echo "Usage: make sign ENGAGEMENT_ID=<id>"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Signing evidence files for engagement $(ENGAGEMENT_ID)...$(NC)"
	@cd $(RESULTS_DIR)/$(ENGAGEMENT_ID) && \
		gpg --detach-sign --armor audit.csv && \
		gpg --detach-sign --armor results.json && \
		echo "$(GREEN)✓ Files signed successfully$(NC)" && \
		echo "  - audit.csv.asc" && \
		echo "  - results.json.asc"

sign-all: ## Sign audit and results files for all engagements
	@echo "$(YELLOW)Signing all engagements...$(NC)"
	@for dir in $(RESULTS_DIR)/*/; do \
		if [ -d "$$dir" ]; then \
			eng_id=$$(basename "$$dir"); \
			echo "Signing $$eng_id..."; \
			cd "$$dir" && \
			gpg --detach-sign --armor audit.csv 2>/dev/null && \
			gpg --detach-sign --armor results.json 2>/dev/null && \
			cd - > /dev/null; \
		fi \
	done
	@echo "$(GREEN)Signing complete$(NC)"

verify-signature: ## Verify GPG signatures for a specific engagement
	@if [ -z "$(ENGAGEMENT_ID)" ]; then \
		echo "$(RED)Error: ENGAGEMENT_ID is required$(NC)"; \
		echo "Usage: make verify-signature ENGAGEMENT_ID=<id>"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Verifying GPG signatures for engagement $(ENGAGEMENT_ID)...$(NC)"
	@cd $(RESULTS_DIR)/$(ENGAGEMENT_ID) && \
		gpg --verify audit.csv.asc audit.csv && \
		gpg --verify results.json.asc results.json && \
		echo "$(GREEN)✓ Signatures verified successfully$(NC)"

purge-raw: ## Delete raw captures older than RETENTION_DAYS for a specific engagement
	@if [ -z "$(ENGAGEMENT_ID)" ]; then \
		echo "$(RED)Error: ENGAGEMENT_ID is required$(NC)"; \
		echo "Usage: make purge-raw ENGAGEMENT_ID=<id> RETENTION_DAYS=90"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Purging raw captures older than $(RETENTION_DAYS) days for engagement $(ENGAGEMENT_ID)...$(NC)"
	@count=$$(find $(RESULTS_DIR)/$(ENGAGEMENT_ID) -name "raw_*.txt" -mtime +$(RETENTION_DAYS) 2>/dev/null | wc -l); \
	if [ $$count -gt 0 ]; then \
		find $(RESULTS_DIR)/$(ENGAGEMENT_ID) -name "raw_*.txt" -mtime +$(RETENTION_DAYS) -delete; \
		echo "$(GREEN)✓ Deleted $$count raw capture file(s)$(NC)"; \
	else \
		echo "$(YELLOW)No raw captures older than $(RETENTION_DAYS) days found$(NC)"; \
	fi

purge-raw-all: ## Delete raw captures older than RETENTION_DAYS for all engagements
	@echo "$(YELLOW)Purging raw captures older than $(RETENTION_DAYS) days from all engagements...$(NC)"
	@total=0; \
	for dir in $(RESULTS_DIR)/*/; do \
		if [ -d "$$dir" ]; then \
			count=$$(find "$$dir" -name "raw_*.txt" -mtime +$(RETENTION_DAYS) 2>/dev/null | wc -l); \
			if [ $$count -gt 0 ]; then \
				find "$$dir" -name "raw_*.txt" -mtime +$(RETENTION_DAYS) -delete; \
				total=$$((total + count)); \
			fi \
		fi \
	done; \
	echo "$(GREEN)✓ Deleted $$total raw capture file(s) total$(NC)"

package: ## Create signed evidence package for delivery
	@if [ -z "$(ENGAGEMENT_ID)" ]; then \
		echo "$(RED)Error: ENGAGEMENT_ID is required$(NC)"; \
		echo "Usage: make package ENGAGEMENT_ID=<id>"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Creating evidence package for engagement $(ENGAGEMENT_ID)...$(NC)"
	@cd $(RESULTS_DIR) && \
		tar -czf ../evidence-$(ENGAGEMENT_ID).tar.gz $(ENGAGEMENT_ID)/ && \
		cd .. && \
		gpg --detach-sign --armor evidence-$(ENGAGEMENT_ID).tar.gz && \
		sha256sum evidence-$(ENGAGEMENT_ID).tar.gz > evidence-$(ENGAGEMENT_ID).tar.gz.sha256 && \
		echo "$(GREEN)✓ Evidence package created:$(NC)" && \
		echo "  - evidence-$(ENGAGEMENT_ID).tar.gz" && \
		echo "  - evidence-$(ENGAGEMENT_ID).tar.gz.asc (GPG signature)" && \
		echo "  - evidence-$(ENGAGEMENT_ID).tar.gz.sha256 (SHA256 hash)"

list-engagements: ## List all engagement directories
	@echo "$(YELLOW)Available engagements:$(NC)"
	@for dir in $(RESULTS_DIR)/*/; do \
		if [ -d "$$dir" ]; then \
			eng_id=$$(basename "$$dir"); \
			file_count=$$(find "$$dir" -type f | wc -l); \
			size=$$(du -sh "$$dir" | cut -f1); \
			echo "  $(GREEN)$$eng_id$(NC) - $$file_count files ($$size)"; \
		fi \
	done

show-stats: ## Show statistics for a specific engagement
	@if [ -z "$(ENGAGEMENT_ID)" ]; then \
		echo "$(RED)Error: ENGAGEMENT_ID is required$(NC)"; \
		echo "Usage: make show-stats ENGAGEMENT_ID=<id>"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Statistics for engagement $(ENGAGEMENT_ID):$(NC)"
	@dir=$(RESULTS_DIR)/$(ENGAGEMENT_ID); \
	if [ -d "$$dir" ]; then \
		echo "  Total files: $$(find "$$dir" -type f | wc -l)"; \
		echo "  Size: $$(du -sh "$$dir" | cut -f1)"; \
		if [ -f "$$dir/audit.csv" ]; then \
			audit_lines=$$(wc -l < "$$dir/audit.csv"); \
			echo "  Audit entries: $$((audit_lines - 1))"; \
		fi; \
		if [ -f "$$dir/results.json" ]; then \
			echo "  Results file: ✓"; \
		fi; \
		raw_count=$$(find "$$dir" -name "raw_*.txt" 2>/dev/null | wc -l); \
		if [ $$raw_count -gt 0 ]; then \
			echo "  Raw captures: $$raw_count"; \
		fi; \
		if [ -f "$$dir/audit.csv.asc" ]; then \
			echo "  GPG signed: ✓"; \
		else \
			echo "  GPG signed: ✗"; \
		fi; \
	else \
		echo "$(RED)Engagement directory not found$(NC)"; \
		exit 1; \
	fi

clean: ## Remove evidence packages (does NOT remove results directory)
	@echo "$(YELLOW)Removing evidence packages...$(NC)"
	@rm -f evidence-*.tar.gz evidence-*.tar.gz.asc evidence-*.tar.gz.sha256
	@echo "$(GREEN)✓ Evidence packages removed$(NC)"

clean-all: ## Remove all generated files, test data, and build artifacts
	@./scripts/cleanup.sh --force

build: ## Build the SECA-CLI binary
	@echo "$(YELLOW)Building SECA-CLI...$(NC)"
	@go build -o seca main.go
	@echo "$(GREEN)✓ Build complete: ./seca$(NC)"

test: ## Run unit tests
	@echo "$(YELLOW)Running unit tests...$(NC)"
	@go test ./cmd/... -v

test-coverage: ## Run tests with coverage report
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@go test ./cmd/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out
	@echo "$(GREEN)Coverage report: coverage.out$(NC)"
	@echo "Generate HTML: go tool cover -html=coverage.out -o coverage.html"

test-integration: build ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@./tests/integration_test.sh

test-all: test test-integration ## Run all tests (unit + integration)
	@echo "$(GREEN)✓ All tests completed$(NC)"

test-clean: ## Clean test cache and artifacts
	@echo "$(YELLOW)Cleaning test artifacts...$(NC)"
	@go clean -testcache
	@rm -f coverage.out coverage.html
	@rm -rf test_results_integration
	@rm -f test_engagements_integration.json
	@echo "$(GREEN)✓ Test artifacts cleaned$(NC)"

install: build ## Install SECA-CLI to /usr/local/bin
	@echo "$(YELLOW)Installing SECA-CLI...$(NC)"
	@sudo cp seca /usr/local/bin/
	@sudo chmod +x /usr/local/bin/seca
	@echo "$(GREEN)✓ Installed to /usr/local/bin/seca$(NC)"

build-all: ## Build binaries for all platforms
	@./scripts/build.sh

release: ## Create a new release (usage: make release VERSION=1.0.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Error: VERSION is required$(NC)"; \
		echo "Usage: make release VERSION=1.0.0"; \
		exit 1; \
	fi
	@./scripts/release.sh $(VERSION)
