package dte

import (
	"time"
)

// DTEDocument is the interface that both Invoice and Nota implement
type DTEDocument interface {
	GetID() string
	GetCompanyID() string
	GetClientID() string
	GetEstablishmentID() string
	GetPointOfSaleID() string
	GetDteNumeroControl() string
	GetFinalizedAt() *time.Time
	GetPaymentTerms() string
	GetNotes() string
	GetLineItems() DTELineItems
	GetRelatedDocuments() DTERelatedDocuments
}

// DTELineItems represents line items for any DTE document
type DTELineItems interface {
	Count() int
	Get(index int) DTELineItem
}

// DTELineItem represents a single line item
type DTELineItem interface {
	GetLineNumber() int
	GetItemType() int
	GetItemSku() string
	GetItemName() string
	GetQuantity() float64
	GetUnitOfMeasure() int
	GetUnitPrice() float64
	GetDiscountAmount() float64
	GetRelatedDocumentRef() string
}

// DTERelatedDocuments represents related documents collection
type DTERelatedDocuments interface {
	Count() int
	Get(index int) DTERelatedDocument
}

// DTERelatedDocument represents a single related document
type DTERelatedDocument interface {
	GetDocumentType() string
	GetGenerationType() int
	GetDocumentNumber() string
	GetDocumentDate() time.Time
}
