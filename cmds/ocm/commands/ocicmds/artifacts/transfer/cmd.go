// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"fmt"
	"path"

	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/open-component-model/ocm/v2/cmds/ocm/commands/ocicmds/common"
	"github.com/open-component-model/ocm/v2/cmds/ocm/commands/ocicmds/common/handlers/artifacthdlr"
	"github.com/open-component-model/ocm/v2/cmds/ocm/commands/ocicmds/common/options/repooption"
	"github.com/open-component-model/ocm/v2/cmds/ocm/commands/ocicmds/names"
	"github.com/open-component-model/ocm/v2/cmds/ocm/commands/verbs"
	"github.com/open-component-model/ocm/v2/cmds/ocm/pkg/utils"
	"github.com/open-component-model/ocm/v2/pkg/contexts/clictx"
	"github.com/open-component-model/ocm/v2/pkg/contexts/oci"
	"github.com/open-component-model/ocm/v2/pkg/contexts/oci/repositories/ctf"
	"github.com/open-component-model/ocm/v2/pkg/contexts/oci/transfer"
	"github.com/open-component-model/ocm/v2/pkg/errors"
	"github.com/open-component-model/ocm/v2/pkg/out"
)

var (
	Names = names.Artifacts
	Verb  = verbs.Transfer
)

type Command struct {
	utils.BaseCommand

	TransferRepo bool

	Refs   []string
	Target string
}

func NewCommand(ctx clictx.Context, names ...string) *cobra.Command {
	return utils.SetupCommand(&Command{BaseCommand: utils.NewBaseCommand(ctx, repooption.New())}, utils.Names(Names, names...)...)
}

func (o *Command) ForName(name string) *cobra.Command {
	return &cobra.Command{
		Use:   "[<options>] {<artifact-reference>} <target>",
		Args:  cobra.MinimumNArgs(1),
		Short: "transfer OCI artifacts",
		Long: `
Transfer OCI artifacts from one registry to another one.
Several transfer scenarios are supported:
- copy a set of artifacts (for the same repository) into another registry
- copy a set of artifacts (for the same repository) into another repository
- copy artifacts from multiple repositories into another registry
- copy artifacts from multiple repositories into another registry with a given repository prefix (option -R)

By default, the target is seen as a single repository if a repository is specified.
If a complete registry is specified as target, option -R is implied, but the source
must provide a repository. THis combination does not allow an artifact set as source, which
specifies no repository for the artifacts.

Sources may be specified as
- dedicated artifacts with repository and version or tag
- repository (without version), which is resolved to all available tags
- registry, if the specified registry implementation supports a namespace/repository lister,
  which is not the case for registries conforming to the OCI distribution specification.`,
		Example: `
$ ocm oci artifact transfer ghcr.io/mandelsoft/kubelink:v1.0.0 gcr.io
$ ocm oci artifact transfer ghcr.io/mandelsoft/kubelink gcr.io
$ ocm oci artifact transfer ghcr.io/mandelsoft/kubelink gcr.io/my-project
$ ocm oci artifact transfer /tmp/ctf gcr.io/my-project
`,
	}
}

func (o *Command) AddFlags(flags *pflag.FlagSet) {
	o.BaseCommand.AddFlags(flags)
	flags.BoolVarP(&o.TransferRepo, "repo-name", "R", false, "transfer repository name")
}

func (o *Command) Complete(args []string) error {
	if len(args) == 0 && repooption.From(o).Spec == "" {
		return fmt.Errorf("a repository or at least one argument that defines the reference is needed")
	}
	o.Target = args[len(args)-1]
	o.Refs = args[:len(args)-1]
	return nil
}

func (o *Command) Run() error {
	session := oci.NewSession(nil)
	defer session.Close()
	err := o.ProcessOnOptions(common.CompleteOptionsWithContext(o.Context, session))
	if err != nil {
		return err
	}
	a, err := NewAction(o.Context, session, o.Target, o.TransferRepo)
	if err != nil {
		return err
	}

	handler := artifacthdlr.NewTypeHandler(o.Context.OCI(), session, repooption.From(o).Repository)

	return utils.HandleOutput(a, handler, utils.StringElemSpecs(o.Refs...)...)
}

/////////////////////////////////////////////////////////////////////////////

type action struct {
	Context      clictx.Context
	Registry     oci.Repository
	Ref          oci.RefSpec
	TransferRepo bool

	srcs         []*artifacthdlr.Object
	repositories map[string]map[string]digest.Digest

	copied int
}

func NewAction(ctx clictx.Context, session oci.Session, target string, transferRepo bool) (*action, error) {
	ref, err := oci.ParseRef(target)
	if err != nil {
		return nil, err
	}
	if ref.Digest != nil {
		return nil, fmt.Errorf("copy to target digest not supported")
	}
	ref.CreateIfMissing = true
	ref.TypeHint = ctf.Type
	repo, err := session.DetermineRepositoryBySpec(ctx.OCIContext(), &ref.UniformRepositorySpec)
	if err != nil {
		return nil, err
	}

	if ref.IsVersion() && transferRepo {
		return nil, errors.Newf("repository names cannot be transferred for a given target version")
	}
	if ref.IsRegistry() {
		transferRepo = true
	}
	return &action{
		Context:      ctx,
		Ref:          ref,
		Registry:     repo,
		TransferRepo: transferRepo,
		repositories: map[string]map[string]digest.Digest{},
	}, nil
}

func (a *action) Add(e interface{}) error {
	src, ok := e.(*artifacthdlr.Object)
	if !ok {
		return fmt.Errorf("failed type assertion for type %T to artifacthtlr.Object", e)
	}

	ns := src.Namespace.GetNamespace()
	if ns == "" && a.Ref.IsRegistry() {
		return errors.Newf("target repository equired for repository-less artifact")
	}
	versions, ok := a.repositories[ns]
	if !ok {
		versions = map[string]digest.Digest{}
	}
	dig := src.Artifact.Digest()
	if src.Spec.IsTagged() {
		old, ok := versions[*src.Spec.Tag]
		if ok {
			if old != dig {
				return errors.Newf("duplicate tag %q with different digests: %s != %s", *src.Spec.Tag, dig, old)
			}
			return nil // skip second entry
		}
		versions[*src.Spec.Tag] = dig
	}
	_, ok = versions[dig.String()]
	if ok {
		return nil
	}
	versions[dig.String()] = dig
	a.repositories[ns] = versions
	a.srcs = append(a.srcs, src)
	return nil
}

func (a *action) Close() error {
	if len(a.repositories) > 1 && !a.TransferRepo {
		return fmt.Errorf("cannot copy artifacts from multiple OCI repositories to the same repository (%s) (use option -N to transfer repositories)", &a.Ref)
	}

	for _, src := range a.srcs {
		err := a.handleArtifact(src)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *action) Out() error {
	out.Outf(a.Context, "copied %d from %d artifact(s) and %d repositories\n", a.copied, len(a.srcs), len(a.repositories))
	return nil
}

func (a *action) handleArtifact(src *artifacthdlr.Object) error {
	repository, tag := a.Target(src)

	ns, err := a.Registry.LookupNamespace(repository)
	if err != nil {
		return err
	}
	defer ns.Close()
	tgt := a.Ref
	tgt.Repository = ns.GetNamespace()
	if tag != "" {
		tgt.Tag = &tag
	}
	out.Outf(a.Context, "copying %s to %s...\n", &src.Spec, &tgt)
	err = transfer.TransferArtifact(src.Artifact, ns, tag)
	if err == nil {
		a.copied++
	}
	return err
}

func (a *action) Target(obj *artifacthdlr.Object) (string, string) {
	repository := obj.Spec.Repository
	if a.TransferRepo {
		repository = path.Join(a.Ref.Repository, repository)
		if obj.Spec.Tag != nil {
			return repository, *obj.Spec.Tag
		}
		return repository, ""
	}
	if a.Ref.IsVersion() {
		return a.Ref.Repository, *a.Ref.Tag
	}
	if obj.Spec.Tag != nil {
		return a.Ref.Repository, *obj.Spec.Tag
	}
	return a.Ref.Repository, ""
}
