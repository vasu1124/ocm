// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rsakeypair_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/open-component-model/ocm/v2/cmds/ocm/testhelper"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/v2/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/v2/pkg/signing/handlers/rsa"
)

const ISSUER = "mandelsoft"

var _ = Describe("Test Environment", func() {
	var env *TestEnv
	var defaultContext credentials.Context

	BeforeEach(func() {
		env = NewTestEnv()
		defaultContext = credentials.New()
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("create key pair", func() {

		buf := bytes.NewBuffer(nil)
		Expect(env.CatchOutput(buf).Execute("create", "rsakeypair", "key.priv")).To(Succeed())
		Expect(buf.String()).To(StringEqualTrimmedWithContext(`
created rsa key pair key.priv[key.pub]
`))
		priv, err := env.ReadFile("key.priv")
		Expect(err).To(Succeed())
		pub, err := env.ReadFile("key.pub")
		Expect(err).To(Succeed())

		d := digest.FromBytes([]byte("digest"))
		sig, err := rsa.Handler{}.Sign(defaultContext, d.Hex(), 0, ISSUER, priv)
		Expect(err).To(Succeed())
		Expect(sig.Algorithm).To(Equal(rsa.Algorithm))
		Expect(sig.MediaType).To(Equal(rsa.MediaType))
		Expect(sig.Issuer).To(Equal(ISSUER))

		err = rsa.Handler{}.Verify(d.Hex(), 0, sig, pub)
		Expect(err).To(Succeed())
	})

	It("create self-signed key pair", func() {

		buf := bytes.NewBuffer(nil)
		Expect(env.CatchOutput(buf).Execute("create", "rsakeypair", "key.priv", "CN=mandelsoft")).To(Succeed())
		Expect(buf.String()).To(StringEqualTrimmedWithContext(`
created rsa key pair key.priv[key.cert]
`))
		priv, err := env.ReadFile("key.priv")
		Expect(err).To(Succeed())
		pub, err := env.ReadFile("key.cert")
		Expect(err).To(Succeed())

		d := digest.FromBytes([]byte("digest"))
		sig, err := rsa.Handler{}.Sign(defaultContext, d.Hex(), 0, "mandelsoft", priv)
		Expect(err).To(Succeed())
		Expect(sig.Algorithm).To(Equal(rsa.Algorithm))
		Expect(sig.MediaType).To(Equal(rsa.MediaType))
		Expect(sig.Issuer).To(Equal(ISSUER))

		err = rsa.Handler{}.Verify(d.Hex(), 0, sig, pub)
		Expect(err).To(Succeed())
	})
})
