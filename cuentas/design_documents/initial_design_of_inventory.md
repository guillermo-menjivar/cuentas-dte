Inventory Management System - Phase 1 Design Document
Overview
Build a simplified inventory management system that provides a product catalog with tax configuration, stock tracking, and complete transaction history for audit purposes. The system will support DTE (electronic invoice) generation and scale to corporate-level transactions.
Database Schema
Already implemented via migrations 0006 through 0010:

inventory_items - Product/service catalog
inventory_item_taxes - Tax associations per item
inventory_transactions - Complete audit trail of stock movements
client_pricing - Client-specific pricing (future use)
inventory_alerts - View for low stock/oversold items

Phase 1 Scope
1. Data Models (internal/models/inventory.go)
InventoryItem
go- ID, CompanyID, TipoItem (1=Bienes, 2=Servicios, 3=Ambos, 4=Otros)
- SKU, CodigoBarras (optional)
- Name, Description, Manufacturer, ImageURL
- GoodsPercentage, ServicesPercentage (for tipo 3)
- CostPrice (nullable), UnitPrice, UnitOfMeasure
- Color (variant)
- TrackInventory (bool), CurrentStock, MinimumStock
- Active, CreatedAt, UpdatedAt
InventoryItemTax
go- ID, ItemID, TributoCode
- AppliesToGoods, AppliesToServices (for tipo 3 items)
InventoryTransaction
go- ID, CompanyID, ItemID
- TransactionType (purchase, sale, adjustment, return, transfer)
- Quantity (positive/negative)
- ReferenceType, ReferenceID
- Notes, TransactionDate, CreatedBy
2. Validation Rules (models/inventory.go Validate methods)
SKU Format

Pattern: ^[A-Z0-9][A-Z0-9-]{1,48}[A-Z0-9]$
3-50 characters, alphanumeric with dashes only
Enforce uppercase, must start/end with alphanumeric

Unit of Measure

Whitelist: unidad, servicio, hora, kilogramo, gramo, litro, mililitro, metro, centimetro, caja
Enforce lowercase convention

Business Rules

Services (tipo 2) cannot have track_inventory = true
Tipo 3 (Ambos) requires goods_percentage + services_percentage = 100
Validate tributo_code exists in pkg/codigos/Tributos map
For tipo 3 items, tax must specify applies_to_goods/applies_to_services explicitly

Transaction Validation

Purchase: quantity must be positive
Sale: quantity must be negative
Adjustment: any non-zero quantity
Return: non-zero quantity

3. Service Layer (internal/services/inventory_service.go)
InventoryService methods:
CreateItem(ctx, companyID, request) - Create item with taxes in single request
GetItemByID(ctx, companyID, itemID) - Retrieve single item
ListItems(ctx, companyID, filters) - List with filters (active, tipo_item, search)
UpdateItem(ctx, companyID, itemID, request) - Update item details
DeleteItem(ctx, companyID, itemID) - Soft delete (set active=false)

AddItemTax(ctx, companyID, itemID, taxRequest) - Associate tax with item
ListItemTaxes(ctx, companyID, itemID) - Get all taxes for item
RemoveItemTax(ctx, companyID, itemID, tributoCode) - Remove tax association

RecordTransaction(ctx, companyID, transaction) - Record stock movement
GetItemTransactions(ctx, companyID, itemID, filters) - Transaction history
Key Service Logic

Convert formatted values (SKU uppercase, unit_of_measure lowercase)
Calculate current_stock via database triggers (no manual updates)
For tipo 3 items, validate tax configuration completeness
Track all stock changes through transactions table only

4. HTTP Handlers (internal/handlers/inventory.go)
Endpoints
POST   /v1/inventory/items                    - Create item
GET    /v1/inventory/items/:id                - Get item details
GET    /v1/inventory/items                    - List items
PUT    /v1/inventory/items/:id                - Update item
DELETE /v1/inventory/items/:id                - Soft delete

POST   /v1/inventory/items/:id/taxes          - Add tax to item
GET    /v1/inventory/items/:id/taxes          - List item taxes
DELETE /v1/inventory/items/:id/taxes/:code    - Remove tax

POST   /v1/inventory/transactions             - Record transaction (purchase/adjustment)
GET    /v1/inventory/items/:id/transactions   - Get transaction history
Handler Responsibilities

Extract company_id from context (via middleware)
Parse request body and validate JSON format
Call validation methods
Invoke service layer
Map errors to appropriate HTTP status codes
Return formatted JSON responses

Error Handling

400: Validation errors, malformed JSON
404: Item not found
409: Duplicate SKU within company
500: Database or internal errors

5. Implementation Order
Step 1: Models & Validation

Define Go structs for all entities
Implement validation methods with business rules
Add helper functions for SKU/unit formatting

Step 2: Basic CRUD Service

InventoryService constructor
CreateItem with tax associations
GetItemByID
ListItems with filtering

Step 3: Basic CRUD Handlers

CreateItemHandler
GetItemHandler
ListItemsHandler
Register routes in cmd/serve.go

Step 4: Tax Management

AddItemTax service method
ListItemTaxes service method
RemoveItemTax service method
Corresponding handlers

Step 5: Transaction Recording

RecordTransaction service (purchase, adjustment types only)
GetItemTransactions service with pagination
Transaction handlers
Validate stock calculations work via triggers

Step 6: Testing

Create test script similar to test_clients.sh
Test all CRUD operations
Test transaction recording and stock updates
Verify validation rules

Deferred to Later Phases
Not in Phase 1:

Client-specific pricing (table exists, no endpoints)
Sale transactions (added when invoice system built)
Return/transfer transactions
Inventory alerts endpoint
Barcode validation (optional field, no format enforcement)
Stock reservation/allocation
Multi-location warehouse support
Bulk import/export
Item categories/hierarchies
Cost tracking analytics

Design Principles

Tax-exclusive pricing - All prices stored without taxes, calculated at invoice time
Transaction-only stock updates - Never directly modify current_stock, always via transactions
Complete audit trail - Every stock change recorded in inventory_transactions
Flexible tax application - Default taxes on item, can override at invoice time
Simple now, scalable later - Build minimum for DTE generation, extend as needed
Company isolation - All operations scoped to company_id from auth context

Success Criteria
Phase 1 complete when:

Can create items with prices and tax configuration
Can list and retrieve items for invoice generation
Can record stock adjustments and purchases
Can view complete transaction history for any item
Stock levels automatically maintained via database triggers
All validation rules enforced at application layer
Full audit trail for compliance
Retry
