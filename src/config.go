package artifacts

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

type Specification struct {
	Domain    string
	PageSize  int      `default:"100""`
	Region    string   `default:"us-east-1""`
	DbFile    string   `default:".db.artifacts"`
	SkipRepos []string `default:"maven-central,maven-central-store"` // TODO remove before push
	Listen    string   `default:"localhost:3000"`
	Load      bool     `default:"false"`
	Templates string   `default:"src/templates/"`
}

// AwsPageSize returns the page size in *int64 so satisfy aws expectations :(
func (s *Specification) AwsPageSize() *int64 {
	i := int64(s.PageSize)
	return &i
}

func (s *Specification) Skip(name string) bool {
	for _, skip := range s.SkipRepos {
		if name == skip {
			return true
		}
	}
	return false
}

func LoadSpecification() (Specification, error) {
	var s Specification
	err := envconfig.Process("ARTIFACTS", &s)
	if err != nil {
		return s, err
	}
	log.Info().Msgf("Found config: %+v", s)
	return s, err
}
