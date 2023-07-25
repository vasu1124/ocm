// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"github.com/open-component-model/ocm/v2/pkg/contexts/credentials/internal"
)

const (
	ID_TYPE = internal.ID_TYPE

	ATTR_TYPE           = internal.ATTR_TYPE
	ATTR_USERNAME       = internal.ATTR_USERNAME
	ATTR_PASSWORD       = internal.ATTR_PASSWORD
	ATTR_CERTIFICATE    = internal.ATTR_CERTIFICATE // PEM encoded
	ATTR_PRIVATE_KEY    = internal.ATTR_PRIVATE_KEY // PEM encoded
	ATTR_SERVER_ADDRESS = internal.ATTR_SERVER_ADDRESS
	ATTR_IDENTITY_TOKEN = internal.ATTR_IDENTITY_TOKEN
	ATTR_REGISTRY_TOKEN = internal.ATTR_REGISTRY_TOKEN
	ATTR_TOKEN          = internal.ATTR_TOKEN
)
