// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package download_test

import (
	"bytes"

	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/v2/pkg/common/accessio"
	metav1 "github.com/open-component-model/ocm/v2/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/grammar"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/repositories/comparch"
	"github.com/open-component-model/ocm/v2/pkg/mime"
)

const ARCH = "/tmp/ctf"
const PROVIDER = "mandelsoft"
const VERSION = "v1"
const COMPONENT = "github.com/mandelsoft/test"
const OUT = "/tmp/res"

var _ = Describe("Download Component Version", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = NewTestEnv()
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("download single component version from ctf file", func() {
		env.OCMCommonTransport(ARCH, accessio.FormatDirectory, func() {
			env.Component(COMPONENT, func() {
				env.Version(VERSION, func() {
					env.Provider(PROVIDER)
					env.Resource("testdata", "", "PlainText", metav1.LocalRelation, func() {
						env.BlobStringData(mime.MIME_TEXT, "testdata")
					})
				})
			})
		})

		buf := bytes.NewBuffer(nil)
		Expect(env.CatchOutput(buf).Execute("download", "component", "-O", OUT, "--repo", ARCH, COMPONENT+grammar.VersionSeparator+VERSION)).To(Succeed())
		Expect(buf.String()).To(StringEqualTrimmedWithContext(
			`
/tmp/res: downloaded
`))
		Expect(env.DirExists(OUT)).To(BeTrue())
		Expect(env.ReadFile(vfs.Join(env, OUT, comparch.BlobsDirectoryName, "sha256.810ff2fb242a5dee4220f2cb0e6a519891fb67f2f828a6cab4ef8894633b1f50"))).To(Equal([]byte("testdata")))

		cd := `component:
  componentReferences: []
  name: github.com/mandelsoft/test
  provider: mandelsoft
  repositoryContexts: []
  resources:
  - access:
      localReference: sha256.810ff2fb242a5dee4220f2cb0e6a519891fb67f2f828a6cab4ef8894633b1f50
      mediaType: text/plain
      type: localBlob
    name: testdata
    relation: local
    type: PlainText
    version: v1
  sources: []
  version: v1
meta:
  schemaVersion: v2
`
		Expect(env.ReadFile(vfs.Join(env, OUT, comparch.ComponentDescriptorFileName))).To(Equal([]byte(cd)))
	})
})
