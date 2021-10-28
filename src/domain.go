package artifacts

import (
	"github.com/aws/aws-sdk-go/service/codeartifact"
	"time"
)

type ArtifactId struct {
	Namespace string
	Package   string
	Version   string
}

type Artifact struct {
	ArtifactId
	Repository string
	Revision   string
	DomainName string
	Format     string
	Error      error    `json:"-"`
	Problems   []string `json:",omitempty"`
	Status     Status
	CreateTime time.Time
}

// Status represents package version status. See: https://docs.aws.amazon.com/codeartifact/latest/ug/packages-overview.html#package-version-status
type Status string

const (
	Published  Status = "Published"
	Unfinished Status = "Unfinished"
	Unlisted   Status = "Unlisted"
	Archived   Status = "Archived"
	Disposed   Status = "Disposed"
	Deleted    Status = "Deleted"
)

var AllStatuses = []Status{
	Published,
	Unfinished,
	Unlisted,
	Archived,
	Disposed,
	Deleted,
}

type Package struct {
	*codeartifact.RepositorySummary
	*codeartifact.PackageSummary
	Error error
}
