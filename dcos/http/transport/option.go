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

import "errors"

// OptionFunc type sets optional configurations for the
// DC/OS HTTP client.
type OptionFunc func(*options) error

// options struct contains configurable parameters
// for the DC/OS HTTP client.
type options struct {
	CaCertificatePath string
	IAMConfigPath     string
}

func errorOnEmpty(arg string) error {
	if len(arg) == 0 {
		return errors.New("Must pass non-empty string to this option")
	}
	return nil
}

// OptionCaCertificatePath sets the CA certificate path option.
func OptionCaCertificatePath(caCertificatePath string) OptionFunc {
	return func(o *options) error {
		err := errorOnEmpty(caCertificatePath)
		if err == nil {
			o.CaCertificatePath = caCertificatePath
		}
		return err
	}
}

// OptionIAMConfigPath sets the IAM configuration path option.
func OptionIAMConfigPath(iamConfigPath string) OptionFunc {
	return func(o *options) error {
		err := errorOnEmpty(iamConfigPath)
		if err == nil {
			o.IAMConfigPath = iamConfigPath
		}
		return err
	}
}
