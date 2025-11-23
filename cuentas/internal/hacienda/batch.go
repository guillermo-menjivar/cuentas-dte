package hacienda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type BatchSubmitRequest struct {
	Ambiente   string   `json:"ambiente"`
	IdEnvio    string   `json:"idEnvio"`
	Version    int      `json:"version"`
	NitEmisor  string   `json:"nitEmisor"`
	Documentos []string `json:"documentos"`
}

type BatchSubmitResponse struct {
	Version         int    `json:"version"`
	Ambiente        string `json:"ambiente"`
	VersionApp      int    `json:"versionApp"`
	Estado          string `json:"estado"`
	CodigoLote      string `json:"codigoLote"`
	IdEnvio         string `json:"idEnvio"`
	FhProcesamiento string `json:"fhProcesamiento"`
	CodigoMsg       string `json:"codigoMsg"`
	DescripcionMsg  string `json:"descripcionMsg"`
}

type BatchQueryResponse struct {
	Procesados []DTEResultado `json:"procesados"`
	Rechazados []DTEResultado `json:"rechazados"`
}

type DTEResultado struct {
	Version          int      `json:"version"`
	Ambiente         string   `json:"ambiente"`
	VersionApp       int      `json:"versionApp"`
	Estado           string   `json:"estado"`
	CodigoGeneracion string   `json:"codigoGeneracion"`
	SelloRecibido    string   `json:"selloRecibido,omitempty"`
	FhProcesamiento  string   `json:"fhProcesamiento"`
	ClasificacionMsg string   `json:"clasificacionMsg,omitempty"`
	CodigoMsg        string   `json:"codigoMsg"`
	DescripcionMsg   string   `json:"descripcionMsg"`
	Observaciones    []string `json:"observaciones,omitempty"`
}

// SubmitBatch submits a batch of DTEs to Hacienda
func (c *Client) SubmitBatch(
	ctx context.Context,
	token string,
	ambiente string,
	nitEmisor string,
	documentos []string,
) (*BatchSubmitResponse, error) {

	url := fmt.Sprintf("%s/fesv/recepcionlote", c.baseURL)

	payload := BatchSubmitRequest{
		Ambiente:   ambiente,
		IdEnvio:    generateUUID(),
		Version:    1,
		NitEmisor:  nitEmisor,
		Documentos: documentos,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit batch: %w", err)
	}
	defer resp.Body.Close()

	var response BatchSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &response, fmt.Errorf("batch submission failed: %s", response.DescripcionMsg)
	}

	return &response, nil
}

// QueryBatchStatus queries the status of a submitted batch
func (c *Client) QueryBatchStatus(
	ctx context.Context,
	token string,
	codigoLote string,
) (*BatchQueryResponse, error) {

	url := fmt.Sprintf("%s/fesv/recepcion/consultadtelote/%s", c.baseURL, codigoLote)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch: %w", err)
	}
	defer resp.Body.Close()

	var response BatchQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func generateUUID() string {
	// Use your UUID library
	return strings.ToUpper(uuid.New().String())
}
