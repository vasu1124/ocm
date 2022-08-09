// Copyright 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package s3_test

import (
	"fmt"
	"os"
	"path/filepath"

	awscreds "github.com/aws/aws-sdk-go/aws/credentials"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/core"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/s3"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

type mockDownloader struct {
	expected []byte
	err      error
}

func (m *mockDownloader) Download(region, bucket, key, version string, creds *awscreds.Credentials) ([]byte, error) {
	return m.expected, m.err
}

var _ = Describe("Method", func() {
	var (
		accessSpec      *s3.AccessSpec
		downloader      s3.Downloader
		expectedContent []byte
		err             error
		mcc             ocm.Context
	)
	BeforeEach(func() {
		expectedContent, err = os.ReadFile(filepath.Join("testdata", "repo.tar.gz"))
		Expect(err).ToNot(HaveOccurred())
		downloader = &mockDownloader{
			expected: expectedContent,
		}
		accessSpec = s3.New(
			"region",
			"bucket",
			"key",
			"version",
			downloader,
		)
		mcc = &mockCredContext{
			Context: ocm.New(),
			creds: &mockCredSource{
				cred: &mockCredentials{
					value: map[string]string{
						"accessKeyID":  "accessKeyID",
						"accessSecret": "accessSecret",
					},
				},
			},
		}
	})

	/*
		FIt("downloads s3 objects", func() {
			accessSpec := s3.New("", "gardenlinux", "objects/fb65cf72b53dccb24ce387becfbc5aed48755a6d", "", nil)
			m, err := accessSpec.AccessMethod(&mockComponentVersionAccess{credContext: mcc})
			Expect(err).ToNot(HaveOccurred())
			r, err := m.Reader()
			Expect(err).ToNot(HaveOccurred())
			defer r.Close()
			dr:=accessio.NewDefaultDigestReader(r)
			var buf [8096]byte
			size:=0
			for {
				s, err := dr.Read(buf[:])
				if s >=0 {
					size+=s
				}
				if err == io.EOF {
					break
				}
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(size).To(Equal(dr.Size()))
		})
	*/

	It("downloads s3 objects", func() {
		m, err := accessSpec.AccessMethod(&mockComponentVersionAccess{credContext: mcc})
		Expect(err).ToNot(HaveOccurred())
		blob, err := m.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(blob).To(Equal(expectedContent))
	})

	When("the downloader fails to download the bucket object", func() {
		BeforeEach(func() {
			downloader = &mockDownloader{
				err: fmt.Errorf("object not found"),
			}
			accessSpec = s3.New(
				"region",
				"bucket",
				"key",
				"version",
				downloader,
			)
		})
		It("errors", func() {
			m, err := accessSpec.AccessMethod(&mockComponentVersionAccess{credContext: mcc})
			Expect(err).ToNot(HaveOccurred())
			_, err = m.Get()
			Expect(err).To(MatchError(ContainSubstring("object not found")))
		})
	})
	When("no credentials are provided", func() {
		It("returns an error stating that credentials have to be provided", func() {
			_, err := accessSpec.AccessMethod(&cpi.DummyComponentVersionAccess{Context: ocm.New()})
			Expect(err).To(MatchError(ContainSubstring("failed to return any credentials; they MUST be provided for s3 access")))
		})
	})
})

type mockComponentVersionAccess struct {
	ocm.ComponentVersionAccess
	credContext ocm.Context
}

func (m *mockComponentVersionAccess) GetContext() ocm.Context {
	return m.credContext
}

type mockCredContext struct {
	ocm.Context
	creds credentials.Context
}

func (m *mockCredContext) CredentialsContext() credentials.Context {
	return m.creds
}

type mockCredSource struct {
	credentials.Context
	cred credentials.Credentials
	err  error
}

func (m *mockCredSource) GetCredentialsForConsumer(credentials.ConsumerIdentity, ...credentials.IdentityMatcher) (credentials.CredentialsSource, error) {
	return m, m.err
}

func (m *mockCredSource) Credentials(credentials.Context, ...credentials.CredentialsSource) (credentials.Credentials, error) {
	return m.cred, nil
}

type mockCredentials struct {
	value map[string]string
}

func (m *mockCredentials) Credentials(context core.Context, source ...core.CredentialsSource) (core.Credentials, error) {
	panic("implement me")
}

func (m *mockCredentials) ExistsProperty(name string) bool {
	panic("implement me")
}

func (m *mockCredentials) PropertyNames() sets.String {
	panic("implement me")
}

func (m *mockCredentials) Properties() common.Properties {
	panic("implement me")
}

func (m *mockCredentials) GetProperty(name string) string {
	return m.value[name]
}
