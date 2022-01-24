// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"github.com/gardener/ocm/pkg/config/cpi"
	"github.com/gardener/ocm/pkg/runtime"
)

const (
	CredentialsConfigType   = "github.com/mandelsoft/ocm/pkg/credentials"
	CredentialsConfigTypeV1 = CredentialsConfigType + "/v1"
)

func init() {
	cpi.RegisterConfigType(CredentialsConfigType, cpi.NewConfigType(CredentialsConfigType, &ConfigSpec{}))
	cpi.RegisterConfigType(CredentialsConfigTypeV1, cpi.NewConfigType(CredentialsConfigTypeV1, &ConfigSpec{}))
}

// ConfigSpec describes a memory based repository interface.
type ConfigSpec struct {
	runtime.ObjectTypeVersion `json:",inline"`
	Test                      string `json:"test"`
}

// NewConfigSpec creates a new memory ConfigSpec
func NewConfigSpec(name string) *ConfigSpec {
	return &ConfigSpec{
		ObjectTypeVersion: runtime.NewObjectTypeVersion(CredentialsConfigType),
		Test:              name,
	}
}

func (a *ConfigSpec) GetType() string {
	return CredentialsConfigType
}

func (a *ConfigSpec) ApplyTo(ctx cpi.Context, target interface{}) error {
	return nil
}
