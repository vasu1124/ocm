// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifacthdlr

import (
	"fmt"
	"os"

	"github.com/open-component-model/ocm/v2/cmds/ocm/pkg/output"
	"github.com/open-component-model/ocm/v2/cmds/ocm/pkg/tree"
	"github.com/open-component-model/ocm/v2/cmds/ocm/pkg/utils"
	"github.com/open-component-model/ocm/v2/pkg/common"
	"github.com/open-component-model/ocm/v2/pkg/contexts/clictx"
	"github.com/open-component-model/ocm/v2/pkg/contexts/oci"
	"github.com/open-component-model/ocm/v2/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/v2/pkg/errors"
)

func Elem(e interface{}) oci.ArtifactAccess {
	return e.(*Object).Artifact
}

////////////////////////////////////////////////////////////////////////////////

type Object struct {
	History    common.History
	Key        common.NameVersion
	Spec       oci.RefSpec
	AttachKind string
	Namespace  oci.NamespaceAccess
	Artifact   oci.ArtifactAccess
}

var (
	_ common.HistoryElement = (*Object)(nil)
	_ tree.Object           = (*Object)(nil)
	_ tree.Typed            = (*Object)(nil)
)

func (o *Object) GetHistory() common.History {
	return o.History
}

func (o *Object) GetKey() common.NameVersion {
	return o.Key
}

func (o *Object) GetKind() string {
	return o.AttachKind
}

func (o *Object) IsNode() *common.NameVersion {
	blob, _ := o.Artifact.Blob()
	nv := common.NewNameVersion("", blob.Digest().String())
	return &nv
}

func (o *Object) AsManifest() interface{} {
	var digest string
	b, err := o.Artifact.Blob()
	if err == nil {
		digest = b.Digest().String()
	} else {
		digest = err.Error()
	}
	return &Manifest{
		Spec:     o.Spec,
		Digest:   digest,
		Manifest: o.Artifact.GetDescriptor(),
	}
}

func (o *Object) String() string {
	blob, _ := o.Artifact.Blob()
	dig := blob.Digest()
	tag := "-"
	if o.Spec.Tag != nil {
		tag = *o.Spec.Tag
	}
	return fmt.Sprintf("%s [%s]: %s", dig, tag, o.History)
}

type Manifest struct {
	Spec     oci.RefSpec
	Digest   string
	Manifest *artdesc.Artifact
}

////////////////////////////////////////////////////////////////////////////////

func Key(a oci.ArtifactAccess) common.NameVersion {
	blob, _ := a.Blob()
	return common.NewNameVersion("", blob.Digest().String())
}

type TypeHandler struct {
	octx     clictx.OCI
	session  oci.Session
	repobase oci.Repository
}

func NewTypeHandler(octx clictx.OCI, session oci.Session, repobase oci.Repository) utils.TypeHandler {
	return &TypeHandler{
		octx:     octx,
		session:  session,
		repobase: repobase,
	}
}

func (h *TypeHandler) Close() error {
	return h.session.Close()
}

func (h *TypeHandler) All() ([]output.Object, error) {
	return h.all(h.repobase)
}

func (h *TypeHandler) all(repo oci.Repository) ([]output.Object, error) {
	if repo == nil {
		return nil, nil
	}
	lister := repo.NamespaceLister()
	if lister == nil {
		return nil, nil
	}
	list, err := lister.GetNamespaces("", true)
	if err != nil {
		return nil, err
	}
	var result []output.Object
	for _, l := range list {
		part, err := h.get(repo, utils.StringSpec(l))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
		}
		result = append(result, part...)
	}
	output.Print(result, "all")
	return result, nil
}

func (h *TypeHandler) Get(elemspec utils.ElemSpec) ([]output.Object, error) {
	result, err := h.get(h.repobase, elemspec)
	output.Print(result, "get %s", elemspec)
	return result, err
}

func (h *TypeHandler) get(repo oci.Repository, elemspec utils.ElemSpec) ([]output.Object, error) {
	var namespace oci.NamespaceAccess
	var result []output.Object
	var err error

	name := elemspec.String()
	spec := oci.RefSpec{}
	if repo == nil {
		evaluated, err := h.session.EvaluateRef(h.octx.Context(), name)
		if err != nil {
			return nil, errors.Wrapf(err, "repository %q", name)
		}
		if evaluated.Namespace == nil {
			return h.all(evaluated.Repository)
		}
		spec = evaluated.Ref
		namespace = evaluated.Namespace
		if evaluated.Artifact != nil {
			obj := &Object{
				Key:       Key(evaluated.Artifact),
				Spec:      spec,
				Namespace: namespace,
				Artifact:  evaluated.Artifact,
			}
			result = append(result, obj)
			return result, nil
		}
	} else {
		art := oci.ArtSpec{Repository: ""}
		if name != "" {
			art, err = oci.ParseArt(name)
			if err != nil {
				return nil, errors.Wrapf(err, "artifact reference %q", name)
			}
		}
		namespace, err = h.session.LookupNamespace(repo, art.Repository)
		if err != nil {
			return nil, errors.Wrapf(err, "reference %q", name)
		}
		spec.UniformRepositorySpec = *repo.GetSpecification().UniformRepositorySpec()
		spec.Repository = art.Repository
		spec.Tag = art.Tag
		spec.Digest = art.Digest
	}

	if spec.IsVersion() {
		a, err := namespace.GetArtifact(spec.Version())
		if err != nil {
			return nil, err
		}
		h.session.AddCloser(a)
		obj := &Object{
			Key:       Key(a),
			Spec:      spec,
			Namespace: namespace,
			Artifact:  a,
		}
		result = append(result, obj)
	} else {
		tags, err := namespace.ListTags()
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			a, err := namespace.GetArtifact(tag)
			if err != nil {
				return nil, err
			}
			h.session.AddCloser(a)
			t := tag
			s := spec
			s.Tag = &t
			result = append(result, &Object{
				Key:       Key(a),
				Spec:      s,
				Namespace: namespace,
				Artifact:  a,
			})
		}
	}
	return result, nil
}
