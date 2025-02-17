package maven

import (
	"crypto"
	"fmt"
	"mime"
	"path"
	"path/filepath"
	"strings"

	. "github.com/mandelsoft/goutils/regexutils"

	"github.com/mandelsoft/goutils/generics"
	"github.com/mandelsoft/goutils/optionutils"

	ocmmime "github.com/open-component-model/ocm/pkg/mime"
)

type CoordinateOption = optionutils.Option[*Coordinates]

type WithClassifier string

func WithOptionalClassifier(c *string) CoordinateOption {
	if c != nil {
		return WithClassifier(*c)
	}
	return nil
}

func (o WithClassifier) ApplyTo(c *Coordinates) {
	c.Classifier = optionutils.PointerTo(string(o))
}

type WithExtension string

func WithOptionalExtension(e *string) CoordinateOption {
	if e != nil {
		return WithExtension(*e)
	}
	return nil
}

func (o WithExtension) ApplyTo(c *Coordinates) {
	c.Extension = optionutils.PointerTo(string(o))
}

// Coordinates holds the typical Maven coordinates groupId, artifactId, version. Optional also classifier and extension.
// https://maven.apache.org/ref/3.9.6/maven-core/artifact-handlers.html
type Coordinates struct {
	// GroupId of the Maven artifact.
	GroupId string `json:"groupId"`
	// ArtifactId of the Maven artifact.
	ArtifactId string `json:"artifactId"`
	// Version of the Maven artifact.
	Version string `json:"version"`
	// Classifier of the Maven artifact.
	Classifier *string `json:"classifier,omitempty"`
	// Extension of the Maven artifact.
	Extension *string `json:"extension,omitempty"`
}

func NewCoordinates(groupId, artifactId, version string, opts ...CoordinateOption) *Coordinates {
	c := &Coordinates{
		GroupId:    groupId,
		ArtifactId: artifactId,
		Version:    version,
	}
	optionutils.ApplyOptions(c, opts...)
	return c
}

// GAV returns the GAV coordinates of the Maven Coordinates.
func (c *Coordinates) GAV() string {
	return c.GroupId + ":" + c.ArtifactId + ":" + c.Version
}

// String returns the Coordinates as a string (GroupId:ArtifactId:Version:WithClassifier:WithExtension).
func (c *Coordinates) String() string {
	return c.GroupId + ":" + c.ArtifactId + ":" + c.Version + ":" + optionutils.AsValue(c.Classifier) + ":" + optionutils.AsValue(c.Extension)
}

// GavPath returns the Maven repository path.
func (c *Coordinates) GavPath() string {
	return c.GroupPath() + "/" + c.ArtifactId + "/" + c.Version
}

func (c *Coordinates) GavLocation(repo *Repository) *Location {
	return repo.AddPath(c.GavPath())
}

func (c *Coordinates) FileName() string {
	file := c.FileNamePrefix()
	if optionutils.AsValue(c.Classifier) != "" {
		file += "-" + *c.Classifier
	}
	if optionutils.AsValue(c.Extension) != "" {
		file += "." + *c.Extension
	} else {
		file += ".jar"
	}
	return file
}

// FilePath returns the Maven Coordinates's GAV-name with classifier and extension.
// Which is equal to the URL-path of the artifact in the repository.
// Default extension is jar.
func (c *Coordinates) FilePath() string {
	return c.GavPath() + "/" + c.FileName()
}

func (c *Coordinates) Location(repo *Repository) *Location {
	return repo.AddPath(c.FilePath())
}

// GroupPath returns GroupId with `/` instead of `.`.
func (c *Coordinates) GroupPath() string {
	return strings.ReplaceAll(c.GroupId, ".", "/")
}

func (c *Coordinates) FileNamePrefix() string {
	return c.ArtifactId + "-" + c.Version
}

// Purl returns the Package URL of the Maven Coordinates.
func (c *Coordinates) Purl() string {
	return "pkg:maven/" + c.GroupId + "/" + c.ArtifactId + "@" + c.Version
}

// SetClassifierExtensionBy extracts the classifier and extension from the filename (without any path prefix).
func (c *Coordinates) SetClassifierExtensionBy(filename string) error {
	s := strings.TrimPrefix(path.Base(filename), c.FileNamePrefix())
	if strings.HasPrefix(s, "-") {
		s = strings.TrimPrefix(s, "-")
		i := strings.Index(s, ".")
		if i < 0 {
			return fmt.Errorf("no extension after classifier found in filename: %s", filename)
		}
		c.Classifier = optionutils.PointerTo(s[:i])
		s = strings.TrimPrefix(s, optionutils.AsValue(c.Classifier))
	} else {
		c.Classifier = optionutils.PointerTo("")
	}
	c.Extension = optionutils.PointerTo(strings.TrimPrefix(s, "."))
	return nil
}

// MimeType returns the MIME type of the Maven Coordinates based on the file extension.
// Default is application/x-tgz.
func (c *Coordinates) MimeType() string {
	if c.Extension != nil && c.Classifier != nil {
		m := mime.TypeByExtension("." + optionutils.AsValue(c.Extension))
		if m != "" {
			return m
		}
		return ocmmime.MIME_OCTET
	}
	return ocmmime.MIME_TGZ
}

// Copy creates a new Coordinates with the same values.
func (c *Coordinates) Copy() *Coordinates {
	return generics.Pointer(*c)
}

func (c *Coordinates) FilterFileMap(fileMap map[string]crypto.Hash) map[string]crypto.Hash {
	if c.Classifier == nil && c.Extension == nil {
		return fileMap
	}
	exp := Literal(c.ArtifactId + "-" + c.Version)
	if optionutils.AsValue(c.Classifier) != "" {
		exp = Sequence(exp, Literal("-"+*c.Classifier))
	}
	if optionutils.AsValue(c.Extension) != "" {
		if c.Classifier == nil {
			exp = Sequence(exp, Optional(Literal("-"), Match(".+")))
		}
		exp = Sequence(exp, Literal("."+*c.Extension))
	} else {
		exp = Sequence(exp, Literal("."), Match(".*"))
	}
	exp = Anchored(exp)
	for file := range fileMap {
		if !exp.MatchString(file) {
			delete(fileMap, file)
		}
	}
	return fileMap
}

// Parse creates a Coordinates from it's serialized form (see Coordinates.String).
func Parse(serializedArtifact string) (*Coordinates, error) {
	parts := strings.Split(serializedArtifact, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid coordination string: %s", serializedArtifact)
	}
	coords := &Coordinates{
		GroupId:    parts[0],
		ArtifactId: parts[1],
		Version:    parts[2],
	}
	if len(parts) >= 4 {
		coords.Classifier = optionutils.PointerTo(parts[3])
	}
	if len(parts) >= 5 {
		coords.Extension = optionutils.PointerTo(parts[4])
	}
	return coords, nil
}

// IsResource returns true if the filename is not a checksum or signature file.
func IsResource(fileName string) bool {
	switch filepath.Ext(fileName) {
	case ".asc", ".md5", ".sha1", ".sha256", ".sha512":
		return false
	default:
		return true
	}
}
