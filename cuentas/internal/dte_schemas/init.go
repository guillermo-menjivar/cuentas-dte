package dte_schemas

import "log"

func init() {
	// Initialize validator on package load
	if err := InitGlobalValidator(); err != nil {
		log.Printf("WARNING: Failed to initialize DTE validator: %v", err)
		log.Printf("DTE validation will be skipped!")
	}
}
