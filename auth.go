package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"hash/crc32"
	"net/http"
	"os"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

func getAuth() (string, error) {

	secret, ok := os.LookupEnv("netapp_auth")
	if !ok {
		return "", fmt.Errorf("missing environment variable 'netapp_auth'")
	}
	payload, err := accessSecretVersion(secret)
	if err != nil {
		return "", err
	}

	return payload, nil
}

// https://cloud.google.com/secret-manager/docs/access-secret-version#secretmanager-access-secret-version-go
func accessSecretVersion(name string) (string, error) {

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	// Verify the data checksum.
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		return "", fmt.Errorf("data corruption detected")
	}

	return string(result.Payload.Data), nil
}

func loadEnv(envFile string) error {
	file, err := os.Open(envFile)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		ss := strings.SplitN(s, "=", 2)
		if len(ss) == 2 {
			k := cleanString(ss[0])
			v := cleanString(ss[1])
			if err := os.Setenv(k, v); err != nil {
				return err
			}
		} else {
			continue
		}
	}
	return nil
}

func cleanString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.Trim(s, "'")
	return s
}

func getHTTPClient(auth, url string) (*http.Response, error) {

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Basic "+auth)
	resp, err := client.Do(request)
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		} else {
			return nil, fmt.Errorf(resp.Status)
		}
	}
	return nil, err
}