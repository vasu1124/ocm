// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/open-component-model/ocm/v2/pkg/testutils"
)

var _ = Describe("registry", func() {
	var reg Registry

	BeforeEach(func() {
		reg = New()
	})

	It("sets and retrieves type", func() {
		reg.RegisterValueType("string", NewStringOptionType, "string")

		t := reg.GetValueType("string")
		Expect(t).NotTo(BeNil())

		o := Must(reg.CreateOptionType("string", "test", "some test"))
		Expect(o.GetName()).To(Equal("test"))
		Expect(o.GetDescription()).To(Equal("[*string*] some test"))
	})

	It("sets and retrieves option", func() {
		reg.RegisterOptionType(HostnameOption)

		t := reg.GetOptionType(HostnameOption.GetName())
		Expect(t).NotTo(BeNil())
	})

	It("creates merges a new type", func() {
		reg.RegisterValueType("string", NewStringOptionType, "string")
		reg.RegisterOptionType(HostnameOption)

		o := Must(reg.CreateOptionType("string", HostnameOption.GetName(), "some test"))
		Expect(o).To(BeIdenticalTo(HostnameOption))
	})

	It("fails creating existing", func() {
		reg.RegisterValueType("string", NewStringOptionType, "string")
		reg.RegisterValueType("int", NewIntOptionType, "int")
		reg.RegisterOptionType(HostnameOption)

		_, err := reg.CreateOptionType("int", HostnameOption.GetName(), "some test")
		MustFailWithMessage(err, "option \"accessHostname\" already exists")
	})
})
