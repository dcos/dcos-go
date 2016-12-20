// Copyright 2016 Mesosphere, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/dcos/dcos-go/jwt/transport"
)

// loadCAPool will load a valid x509 cert.
func loadCAPool(path string) (*x509.CertPool, error) {
	caPool := x509.NewCertPool()
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if !caPool.AppendCertsFromPEM(b) {
		return nil, errors.New("CACertFile parsing failed")
	}

	return caPool, nil
}

// newTransport will return transport for http.Client
func createTransport(caCertificatePath string) (*http.Transport, error) {
	tr := &http.Transport{}
	// if user provided CA cert we must use it, otherwise use InsecureSkipVerify: true for all HTTPS requests.
	if caCertificatePath != "" {
		caPool, err := loadCAPool(caCertificatePath)
		if err != nil {
			return tr, err
		}

		tr.TLSClientConfig = &tls.Config{
			RootCAs: caPool,
		}
	} else {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return tr, nil
}

// NewTransport returns a DC/OS transport implementation by leveraging a roundtripper for
// IAM configuration if passed with a pre-configured TLS configuration.
func NewTransport(clientOptionFuncs ...OptionFunc) (http.RoundTripper, error) {
	transportOptions := options{}
	for _, opt := range clientOptionFuncs {
		if err := opt(&transportOptions); err != nil {
			return nil, err
		}
	}

	tr, err := createTransport(transportOptions.CaCertificatePath)
	if err != nil {
		return nil, err
	}

	if len(transportOptions.IAMConfigPath) != 0 {
		withIAM, err := transport.NewRoundTripper(
			tr,
			transport.OptionReadIAMConfig(transportOptions.IAMConfigPath))
		if err != nil {
			return nil, err
		}
		return withIAM, nil
	}

	return tr, nil
}
