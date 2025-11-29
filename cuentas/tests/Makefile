# Makefile for Inventory Legal Compliance Testing and Reporting
# Usage: make seed, make reports, make all

# Configuration
BASE_URL := http://localhost:8080

# Dynamic date calculation - current month
START_DATE := $(shell date +%Y-%m-01)
END_DATE := $(shell date -d "+1 day" +%Y-%m-%d 2>/dev/null || date -v +1d +%Y-%m-%d)

MONTH_NAME := $(shell date +%B_%Y)
OUTPUT_DIR := ./accounting_reports

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color


COMPANY_ID := $(shell cat .company_id 2>/dev/null || echo "")

# Initialize: Register company THEN capture ID
init:
	@echo "$(BLUE)🚀 Step 1: Registering company...$(NC)"
	@bash register_david_company.sh
	@echo ""
	@echo "$(BLUE)🔍 Step 2: Fetching company ID...$(NC)"
	@FETCHED_ID=$$(./test_list_companies.sh | jq -r '.companies[0].id'); \
	if [ -z "$$FETCHED_ID" ] || [ "$$FETCHED_ID" = "null" ]; then \
		echo "$(RED)❌ Failed to fetch company ID from API$(NC)"; \
		echo "$(YELLOW)Debug: Running test_list_companies.sh manually:$(NC)"; \
		./test_list_companies.sh; \
		exit 1; \
	else \
		echo "$$FETCHED_ID" > .company_id; \
		echo "$(GREEN)✓ Company ID saved to .company_id$(NC)"; \
		echo "Company ID: $$FETCHED_ID"; \
	fi
	@echo ""
	@echo "$(YELLOW)💡 You can now run other make targets:$(NC)"
	@echo "   make seed-purchases"
	@echo "   make seed-invoices"
	@echo "   make reports"
	@echo ""
	@echo "$(BLUE)🔍 Step 3: Creating CCF clients...$(NC)"
	@SAVED_ID=$$(cat .company_id); \
	./test_create_ccf_clients.py $$SAVED_ID
	@echo "configuring establishments"
	@SAVED_ID=$$(cat .company_id);./test_establishments.sh $$SAVED_ID
	@echo "configuring inventory"
	@SAVED_ID=$$(cat .company_id); ./test_inventory.sh $$SAVED_ID


# Show current company ID
show-company:
	@if [ -z "$(COMPANY_ID)" ]; then \
		echo "$(RED)❌ No company ID set. Run 'make init' first.$(NC)"; \
		exit 1; \
	else \
		echo "Current Company ID: $(COMPANY_ID)"; \
	fi

# Validate company ID is set (helper target)
check-company:
	@if [ -z "$(COMPANY_ID)" ]; then \
		echo "$(RED)❌ No company ID set. Run 'make init' first.$(NC)"; \
		exit 1; \
	fi

.PHONY: help clean seed test reports all seed-only reports-only init show-company check-company seed-invoices test_notas_debito test_notas_credito

# Default target
help:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)  Inventory Legal Compliance - Make Commands$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(GREEN)Available commands:$(NC)"
	@echo "  $(YELLOW)make clean$(NC)        - Delete old test data from database"
	@echo "  $(YELLOW)make seed$(NC)         - Seed inventory with random purchases"
	@echo "  $(YELLOW)make test$(NC)         - Run legal compliance validation tests"
	@echo "  $(YELLOW)make reports$(NC)      - Generate audit reports (legal registers)"
	@echo "  $(YELLOW)make all$(NC)          - Clean → Seed → Test → Reports (full workflow)"
	@echo ""
	@echo "$(GREEN)Quick start:$(NC)"
	@echo "  $(YELLOW)make all$(NC)          - Run complete workflow"
	@echo ""
	@echo "$(GREEN)Current Configuration:$(NC)"
	@echo "  Company ID:  $(COMPANY_ID)"
	@echo "  Date Range:  $(START_DATE) to $(END_DATE) (Current Month)"
	@echo "  Output Dir:  $(OUTPUT_DIR)"
	@echo ""
	@echo "$(GREEN)Custom dates:$(NC)"
	@echo "  $(YELLOW)make all START_DATE=2025-01-01 END_DATE=2025-01-31$(NC)"
	@echo ""

# Clean old test data
clean:
	@echo "Cleanind accounting reports directory"
	rm -rf $(OUTPUT_DIR)/*
	@echo "$(BLUE)🧹 Cleaning old test data...$(NC)"
	@echo "⚠️  This will delete ALL inventory events and state for company $(COMPANY_ID)"
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -i cuentas-postgres psql -U cuentas_user -d cuentas -c \
		"DELETE FROM inventory_events WHERE company_id = '$(COMPANY_ID)'; \
		 DELETE FROM inventory_state WHERE company_id = '$(COMPANY_ID)';"; \
		echo "$(GREEN)✅ Database cleaned$(NC)"; \
	else \
		echo "$(YELLOW)⏭️  Skipped cleaning$(NC)"; \
	fi
	@echo ""
	rm -rf .company_id

# Seed inventory with random purchases
seed:
	@echo "$(BLUE)🌱 Seeding inventory with random purchases...$(NC)"
	@./seed_inventory_purchases.py $(COMPANY_ID) \
		--base-url $(BASE_URL) \
		--start-date $(START_DATE) \
		--end-date $(END_DATE) \
		--min 3 \
		--max 10
	@echo ""

# Run compliance tests
test:
	@echo "$(BLUE)🧪 Running legal compliance validation tests...$(NC)"
	@./test_inventory_legal_compliance.py $(COMPANY_ID) \
		--base-url $(BASE_URL)
	@echo ""

# Generate audit reports
reports:
	@echo "$(BLUE)📊 Generating audit reports and legal registers...$(NC)"
	@./test_audit_inventory_reports.py $(COMPANY_ID) \
		--base-url $(BASE_URL) \
		--mode audit \
		--start-date $(START_DATE) \
		--end-date $(END_DATE) \
		--output-dir $(OUTPUT_DIR)
	@echo ""
	@echo "$(GREEN)✅ Reports saved to: $(OUTPUT_DIR)$(NC)"
	@echo ""

# Run everything
all: clean seed test seed-invoices reports
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)✅ COMPLETE WORKFLOW FINISHED$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(GREEN)Summary:$(NC)"
	@echo "  • Database cleaned"
	@echo "  • Inventory seeded with random purchases"
	@echo "  • Compliance tests passed"
	@echo "  • Audit reports generated"
	@echo ""
	@echo "$(GREEN)Next steps:$(NC)"
	@echo "  • Review reports in: $(OUTPUT_DIR)"
	@echo "  • Check legal registers (registro_legal_*.csv)"
	@echo "  • Verify all products have transactions"
	@echo ""

# Quick seed without cleaning (for adding more data)
seed-only:
	@echo "$(BLUE)🌱 Adding more random purchases (no cleaning)...$(NC)"
	@./test_seed_inventory_purchases.py $(COMPANY_ID) \
		--base-url $(BASE_URL) \
		--start-date $(START_DATE) \
		--end-date $(END_DATE) \
		--min 1 \
		--max 5
	@echo ""

seed-invoices:
	@echo "$(BLUE)🧾 Creating and finalizing invoices with sales...$(NC)"
	@./test_seed_invoices_with_sales.py $(COMPANY_ID) \
		--base-url $(BASE_URL) \
		--start-date $(START_DATE) \
		--end-date $(END_DATE) \
		--count 30
	@echo ""

.PHONY: test_export_invoices
test_export_invoices:
	@echo "Test export invoices"
	@./test_export_invoices.py $(COMPANY_ID) \
	--start-date $(START_DATE) \
	--end-date $(END_DATE) \
	--count 5

.PHONY: test_export_invoices_no_discount
test_export_invoices_no_discount:
	@echo "Test export invoices"
	@./test_export_invoices_without_discount.py $(COMPANY_ID) \
	--start-date $(START_DATE) \
	--end-date $(END_DATE) \
	--count 5

test_dte_reconciliation:
	@echo "Running reconciliation"
	@./test_dte_reconciliation.py $(COMPANY_ID)
# Generate reports only (no seeding/testing)
reports-only: reports

test_notas_debito:
	@echo "Testing notas de debito"
	./test_notas_debito.sh $(COMPANY_ID)

test_notas_credito:
	@echo "Testing notas de credito"
	./test_seed_notas_credito.py $(COMPANY_ID) --count 50

.PHONY: test_nota_remision
test_nota_remision:
	./test_nota_remision.py $(COMPANY_ID)

.PHONY: test_fse
test_fse:
	./test_fse.py $(COMPANY_ID)

# Set mock firmador to FAIL mode
firmador-fail:
	@echo "❌ Setting firmador to FAIL mode..."
	@curl -s -X POST http://localhost:8114/control/mode \
		-H "Content-Type: application/json" \
		-d '{"mode": "fail"}' | jq .

# Set mock firmador to SUCCESS mode (proxy to real)
firmador-success:
	@echo "✅ Setting firmador to SUCCESS mode..."
	@curl -s -X POST http://localhost:8114/control/mode \
		-H "Content-Type: application/json" \
		-d '{"mode": "success"}' | jq .

# Check mock firmador status
firmador-status:
	@echo "📊 Firmador mock status:"
	@curl -s http://localhost:8114/control/status | jq .

# Reset mock firmador state
firmador-reset:
	@echo "🔄 Resetting firmador mock..."
	@curl -s -X POST http://localhost:8114/control/reset | jq .

# Run contingency E2E test
test-contingency: firmador-status
	@echo "🧪 Running CCF contingency E2E test..."
	python tests/test_ccf_contingency.py $(COMPANY_ID)
