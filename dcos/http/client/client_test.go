package client

import (
	"crypto/x509"
	"net/http"
	"reflect"
	"testing"
)

const (
	CACertPath     = "fixtures/root_ca_cert.pem"
	ServiceAccount = "fixtures/test_service_account.json"
)

func TestLoadCAPool(t *testing.T) {
	caPool, err := loadCAPool(CACertPath)

	if err != nil {
		t.Error("Expected no errors loading CA cert fixutre, got", err)
	}

	if reflect.TypeOf(caPool) != reflect.TypeOf(&x509.CertPool{}) {
		t.Errorf("loadCAPool() returned invalid type, got %T", caPool)
	}

	_, defErr := loadCAPool("fake/cert/path")
	if defErr == nil {
		t.Error("Expected error with bad path, got", defErr)
	}
}

func TestGetTransport(t *testing.T) {
	transport, err := getTransport(CACertPath)

	if err != nil {
		t.Error("Expected nil error getting transport, got", err.Error())
	}

	if reflect.TypeOf(transport) != reflect.TypeOf(&http.Transport{}) {
		t.Errorf("Expected &http.Transport type, got %T", transport)
	}

	_, defErr := getTransport("fake/cert/path")
	if defErr == nil {
		t.Error("Expected error with bad path, got", defErr)
	}
}

func TestNewDCOSClient(t *testing.T) {
	// Passing empty strings ensures the JWT lib does not try
	// to query Bouncer service for JWT token auth.
	// TODO(malnick) use test server for fake bouncer response
	client, err := NewDCOSClient("", "")

	if err != nil {
		t.Error("Expected nil error, got", err.Error())
	}

	if reflect.TypeOf(client) != reflect.TypeOf(&http.Client{}) {
		t.Errorf("Expected *http.Client, got %T", client)
	}

	if !client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected no cert verification")
	}

	caClient, err := NewDCOSClient(CACertPath, "")
	if err != nil {
		t.Error("Expected nil error, got", err.Error())
	}

	if reflect.TypeOf(caClient) != reflect.TypeOf(&http.Client{}) {
		t.Errorf("Expected *http.Client, got %T", client)
	}

}
