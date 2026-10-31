// Package registrar spricht mit dem Lab-DNS-Registrar (code/registrar).
package registrar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client pflegt DNS-Einträge per HTTP. Ein leerer BaseURL schaltet ihn ab.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New liefert einen Client mit Timeout – ein hängender Dienst darf den
// Reconcile-Worker nicht blockieren.
func New(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: &http.Client{Timeout: 5 * time.Second}}
}

// Upsert legt den Eintrag an oder aktualisiert ihn (PUT ist idempotent).
func (c *Client) Upsert(ctx context.Context, name, target string) error {
	body, err := json.Marshal(map[string]string{"target": target})
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPut, name, body, http.StatusOK, http.StatusCreated)
}

// Delete entfernt den Eintrag. 404 zählt als Erfolg: Das Ziel ist erreicht.
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, name, nil,
		http.StatusNoContent, http.StatusNotFound)
}

func (c *Client) do(ctx context.Context, method, name string, body []byte,
	ok ...int) error {
	url := fmt.Sprintf("%s/records/%s", c.BaseURL, name)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("registrar %s %s: %w", method, name, err)
	}
	defer resp.Body.Close()
	for _, code := range ok {
		if resp.StatusCode == code {
			return nil
		}
	}
	return fmt.Errorf("registrar %s %s: HTTP %d", method, name, resp.StatusCode)
}
