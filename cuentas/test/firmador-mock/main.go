package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// ============================================
// RESPONSE TYPES (matching real firmador)
// ============================================

type ResponseBody struct {
	Status string      `json:"status"`
	Body   interface{} `json:"body"`
}

type ErrorBody struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

// ============================================
// MOCK STATE
// ============================================

type MockState struct {
	mu           sync.RWMutex
	mode         string // "fail" or "success"
	requestCount int
	realFirmador string
}

var state = &MockState{
	mode:         "success", // Default to success (proxy mode)
	realFirmador: "http://167.172.230.154:8113",
}

// ============================================
// HANDLERS
// ============================================

// POST /firmardocumento/ - Main signing endpoint
func handleFirmarDocumento(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state.mu.Lock()
	state.requestCount++
	currentMode := state.mode
	state.mu.Unlock()

	log.Printf("[Mock] /firmardocumento/ called - Mode: %s, Request #%d", currentMode, state.requestCount)

	w.Header().Set("Content-Type", "application/json")

	if currentMode == "fail" {
		// Return error response
		response := ResponseBody{
			Status: "ERROR",
			Body: ErrorBody{
				Codigo:  "COD_803_ERROR_LLAVE_PRUBLICA",
				Mensaje: "Mock: Firmador unavailable (test mode)",
			},
		}
		log.Printf("[Mock] Returning ERROR response")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Mode is "success" - proxy to real firmador
	log.Printf("[Mock] Proxying to real firmador: %s", state.realFirmador)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Mock] Error reading request body: %v", err)
		response := ResponseBody{
			Status: "ERROR",
			Body: ErrorBody{
				Codigo:  "MOCK_ERROR",
				Mensaje: fmt.Sprintf("Failed to read request: %v", err),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Forward to real firmador
	proxyURL := state.realFirmador + "/firmardocumento/"
	proxyReq, err := http.NewRequest("POST", proxyURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[Mock] Error creating proxy request: %v", err)
		response := ResponseBody{
			Status: "ERROR",
			Body: ErrorBody{
				Codigo:  "MOCK_ERROR",
				Mensaje: fmt.Sprintf("Failed to create proxy request: %v", err),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Copy headers
	proxyReq.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("[Mock] Error calling real firmador: %v", err)
		response := ResponseBody{
			Status: "ERROR",
			Body: ErrorBody{
				Codigo:  "MOCK_PROXY_ERROR",
				Mensaje: fmt.Sprintf("Failed to reach real firmador: %v", err),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer resp.Body.Close()

	// Copy response from real firmador
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Mock] Error reading firmador response: %v", err)
		response := ResponseBody{
			Status: "ERROR",
			Body: ErrorBody{
				Codigo:  "MOCK_ERROR",
				Mensaje: fmt.Sprintf("Failed to read firmador response: %v", err),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("[Mock] Real firmador responded with status: %d", resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// POST /control/mode - Set mock mode
func handleSetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Mode string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Mode != "fail" && req.Mode != "success" {
		http.Error(w, "Mode must be 'fail' or 'success'", http.StatusBadRequest)
		return
	}

	state.mu.Lock()
	oldMode := state.mode
	state.mode = req.Mode
	state.mu.Unlock()

	log.Printf("[Mock] Mode changed: %s -> %s", oldMode, req.Mode)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"mode":     req.Mode,
		"previous": oldMode,
	})
}

// GET /control/status - Get current status
func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state.mu.RLock()
	response := map[string]interface{}{
		"mode":          state.mode,
		"request_count": state.requestCount,
		"real_firmador": state.realFirmador,
	}
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /control/reset - Reset request counter
func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state.mu.Lock()
	state.requestCount = 0
	state.mode = "success"
	state.mu.Unlock()

	log.Printf("[Mock] State reset - mode: success, count: 0")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Mock state reset",
	})
}

// ============================================
// MAIN
// ============================================

func main() {
	port := "8113"

	// Routes
	http.HandleFunc("/firmardocumento/", handleFirmarDocumento)
	http.HandleFunc("/control/mode", handleSetMode)
	http.HandleFunc("/control/status", handleGetStatus)
	http.HandleFunc("/control/reset", handleReset)

	// Health check
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Mock Firmador is running...!!"))
	})

	log.Printf("============================================")
	log.Printf("  FIRMADOR MOCK SERVICE")
	log.Printf("============================================")
	log.Printf("Port:          %s", port)
	log.Printf("Real Firmador: %s", state.realFirmador)
	log.Printf("Default Mode:  %s", state.mode)
	log.Printf("============================================")
	log.Printf("Endpoints:")
	log.Printf("  POST /firmardocumento/  - Sign DTE (or fail)")
	log.Printf("  POST /control/mode      - Set mode (fail/success)")
	log.Printf("  GET  /control/status    - Get current status")
	log.Printf("  POST /control/reset     - Reset state")
	log.Printf("  GET  /status            - Health check")
	log.Printf("============================================")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
