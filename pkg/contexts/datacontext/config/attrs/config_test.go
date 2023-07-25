// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package attrs_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-component-model/ocm/v2/pkg/contexts/config"
	"github.com/open-component-model/ocm/v2/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/v2/pkg/contexts/datacontext"
	local "github.com/open-component-model/ocm/v2/pkg/contexts/datacontext/config/attrs"
	"github.com/open-component-model/ocm/v2/pkg/runtime"
)

const ATTR_KEY = "test"

func init() {
	datacontext.RegisterAttributeType(ATTR_KEY, AttributeType{})
}

type AttributeType struct {
}

func (a AttributeType) Name() string {
	return ATTR_KEY
}

func (a AttributeType) Description() string {
	return `
A Test attribute.
`
}

type Attribute struct {
	Value string `json:"value"`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(*Attribute); !ok {
		return nil, fmt.Errorf("boolean required")
	}
	return marshaller.Marshal(v)
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value Attribute
	err := unmarshaller.Unmarshal(data, &value)
	return &value, err
}

////////////////////////////////////////////////////////////////////////////////

var _ = Describe("generic attributes", func() {
	attribute := &Attribute{"TEST"}
	var ctx config.Context

	BeforeEach(func() {
		ctx = config.WithSharedAttributes(datacontext.New(nil)).New()
	})

	Context("applies", func() {

		It("applies later attribute config", func() {

			sub := credentials.WithConfigs(ctx).New()
			spec := local.New()
			Expect(spec.AddAttribute(ATTR_KEY, attribute)).To(Succeed())
			Expect(ctx.ApplyConfig(spec, "test")).To(Succeed())

			Expect(sub.GetAttributes().GetAttribute(ATTR_KEY, nil)).To(Equal(attribute))
		})

		It("applies earlier attribute config", func() {

			spec := local.New()
			Expect(spec.AddAttribute(ATTR_KEY, attribute)).To(Succeed())
			Expect(ctx.ApplyConfig(spec, "test")).To(Succeed())

			sub := credentials.WithConfigs(ctx).New()
			Expect(sub.GetAttributes().GetAttribute(ATTR_KEY, nil)).To(Equal(attribute))
		})
	})
})
