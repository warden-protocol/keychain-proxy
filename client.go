package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/warden-protocol/wardenprotocol/keychain-sdk"
)

type Client struct {
	baseURL url.URL
	h       http.Client
}

func NewClient(baseURL url.URL, httpTimeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		h: http.Client{
			Timeout: httpTimeout,
		},
	}
}

type requestKeyResponse struct {
	Ok           bool   `json:"ok"`
	Key          []byte `json:"key"`
	RejectReason string `json:"reason"`
}

func (c *Client) requestKey(req *keychain.KeyRequest) (res requestKeyResponse, err error) {
	return res, c.postJson("/request_key", req, &res)
}

type requestSignatureResponse struct {
	Ok           bool   `json:"ok"`
	Signature    []byte `json:"signature"`
	RejectReason string `json:"reason"`
}

func (c *Client) requestSignature(req *keychain.SignRequest) (res requestSignatureResponse, err error) {
	return res, c.postJson("/request_signature", req, &res)
}

func (c *Client) postJson(path string, req any, response any) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding JSON payload: %w", err)
	}

	res, err := c.h.Post(c.url(path), "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("sending HTTP request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("unexpected status code: %s\nbody:\n%s", res.Status, string(body))
	}

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return fmt.Errorf("decoding HTTP response body: %w", err)
	}

	return nil
}

func (c *Client) url(elem ...string) string {
	return c.baseURL.JoinPath(elem...).String()
}
