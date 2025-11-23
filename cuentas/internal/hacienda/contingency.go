package hacienda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
)

type ContingencyEventResponse struct {
	Estado        string   `json:"estado"`
	SelloRecibido string   `json:"selloRecibido"`
	FechaHora     string   `json:"fechaHora"`
	Mensaje       string   `json:"mensaje"`
	Observaciones []string `json:"observaciones"`
}

// SubmitContingencyEvent submits Evento de Contingencia to Hacienda
func (c *Client) SubmitContingencyEvent(
	ctx context.Context,
	token string,
	nit string,
	signedEvent string,
) (*ContingencyEventResponse, error) {

	url := fmt.Sprintf("%s/fesv/contingencia", c.baseURL)

	payload := map[string]interface{}{
		"nit":       nit,
		"documento": signedEvent,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use retryablehttp.NewRequestWithContext
	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit contingency event: %w", err)
	}
	defer resp.Body.Close()

	var response ContingencyEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != 200 {
		return &response, fmt.Errorf("contingency event failed: %s", response.Mensaje)
	}

	return &response, nil
}
