// Package appsigningclient implements the two official enterprise self-hosted
// Cloud KMS endpoints that are present in Android Publisher discovery but not
// yet present in the latest tagged generated Go client.
package appsigningclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func New(httpClient *http.Client, baseURL string) *Client {
	return &Client{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/") + "/"}
}

type CloudKMSKey struct {
	CryptoKeyVersionResource string `json:"cryptoKeyVersionResource,omitempty"`
}

type CloudKMSKeyAndCert struct {
	CloudKMSKey    *CloudKMSKey `json:"cloudKmsKey,omitempty"`
	PEMCertificate string       `json:"pemCertificate,omitempty"`
}

type EnrollNewApp struct {
	CloudKMSKeyAndCert *CloudKMSKeyAndCert `json:"cloudKmsKeyAndCert,omitempty"`
}

type EnrollExistingApp struct {
	CloudKMSKey *CloudKMSKey `json:"cloudKmsKey,omitempty"`
}

type EnrollAppRequest struct {
	PEMUploadCertificate string             `json:"pemUploadCertificate,omitempty"`
	EnrollNewApp         *EnrollNewApp      `json:"enrollNewApp,omitempty"`
	EnrollExistingApp    *EnrollExistingApp `json:"enrollExistingApp,omitempty"`
}

type CertificateHashes struct {
	CertificateHashMD5    string `json:"certificateHashMd5,omitempty"`
	CertificateHashSHA1   string `json:"certificateHashSha1,omitempty"`
	CertificateHashSHA256 string `json:"certificateHashSha256,omitempty"`
}

type EnrollAppResponse struct {
	UploadCertificate  *CertificateHashes `json:"uploadCertificate,omitempty"`
	SigningCertificate *CertificateHashes `json:"signingCertificate,omitempty"`
}

type RotatedCloudKMSKey struct {
	CloudKMSKeyAndCert        *CloudKMSKeyAndCert `json:"cloudKmsKeyAndCert,omitempty"`
	SigningCertificateLineage string              `json:"signingCertificateLineage,omitempty"`
}

type RotateAppSigningKeyRequest struct {
	KeyRotationReason  string              `json:"keyRotationReason,omitempty"`
	RotatedCloudKMSKey *RotatedCloudKMSKey `json:"rotatedCloudKmsKey,omitempty"`
}

type RotateAppSigningKeyResponse struct {
	RotatedKeyCertificate *CertificateHashes `json:"rotatedKeyCertificate,omitempty"`
}

func (c *Client) EnrollApp(ctx context.Context, packageName string, request *EnrollAppRequest) (*EnrollAppResponse, error) {
	var response EnrollAppResponse
	path := "androidpublisher/v3/applications/" + url.PathEscape(packageName) + "/appSigning:enrollApp"
	if err := c.post(ctx, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) RotateAppSigningKey(ctx context.Context, packageName string, request *RotateAppSigningKeyRequest) (*RotateAppSigningKeyResponse, error) {
	var response RotateAppSigningKeyResponse
	path := "androidpublisher/v3/applications/" + url.PathEscape(packageName) + "/appSigning:rotateAppSigningKey"
	if err := c.post(ctx, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) post(ctx context.Context, path string, request, response any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("androidpublisher request failed: %s: %s", res.Status, strings.TrimSpace(string(responseBody)))
	}
	if len(bytes.TrimSpace(responseBody)) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
