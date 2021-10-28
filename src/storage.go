package artifacts

import (
	asn1 "encoding/asn1"
	log "github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"strings"
	"time"
)

func (a *Artifact) data() ArtifactData {
	return ArtifactData{
		Repository: a.Repository,
		Revision:   a.Revision,
		DomainName: a.DomainName,
		Format:     a.Format,
		Status:     a.Status,
		CreateTime: a.CreateTime,
	}
}

type ArtifactData struct {
	Repository string
	Revision   string
	DomainName string
	Format     string
	Status     Status
	CreateTime time.Time
}

func (a *Artifact) populateProblems() {
	if a.Error != nil && len(a.Problems) == 0 {
		a.Problems = []string{a.Error.Error()}
	}
}

func (i ArtifactId) Marshal() ([]byte, error) {
	return asn1.Marshal(i)
}

func UnmarshalArtifactId(i []byte) (ArtifactId, error) {
	id := ArtifactId{}
	_, err := asn1.Unmarshal(i, &id)
	return id, err
}

func (a *ArtifactData) Bucket() []byte {
	return []byte(a.Status)
}

func (a *ArtifactId) Key() ([]byte, error) {
	return a.Marshal()
}

type BoltStorage struct {
	db *bolt.DB
}

func NewStorage(s Specification) (*BoltStorage, error) {
	db, err := bolt.Open(s.DbFile, 0666, nil)

	if err != nil {
		return nil, err
	}

	storage := BoltStorage{
		db,
	}
	return &storage, err
}

type ValidationError struct {
	Problems []string
}

func (v ValidationError) Error() string {
	return "Validaiton problems"
}

func (rs *BoltStorage) Insert(artifacts ...Artifact) ([]Artifact, error) {
	err := rs.db.Update(func(tx *bolt.Tx) error {
		for _, artifact := range artifacts {
			if len(artifact.Problems) > 0 {
				continue
			}

			problems := make([]string, 0)
			if artifact.ArtifactId.Version == "" {
				problems = append(problems, "Must have a non-blank version")
			}
			if artifact.ArtifactId.Package == "" {
				problems = append(problems, "Must have a non-blank package")
			}
			if artifact.ArtifactId.Namespace == "" {
				problems = append(problems, "Must have a non-blank namespace")
			}
			if len(problems) > 0 {
				artifact.Error = &ValidationError{
					Problems: problems,
				}
				artifact.Problems = problems
				continue
			}

			id := artifact.ArtifactId
			data := artifact.data()

			value, err := asn1.Marshal(data)

			if err != nil {
				return err
			}

			bucket := tx.Bucket(data.Bucket())
			if bucket == nil {
				log.Debug().Interface("bucket", artifact.Status).Interface("id", artifact.ArtifactId).Msg("Created bucket")
				bucket, err = tx.CreateBucket(data.Bucket())
				if err != nil {
					artifact.Error = err
					continue
				}
			}

			key, err := id.Key()

			if err != nil {
				return err
			}

			err = bucket.Put(key, value)

			if err != nil {
				artifact.Error = err
				continue
			}

			if artifact.Error != nil {
				log.Err(err).Interface("artifact", artifact).Msg("Error inserting artifact")
			}

		}
		return nil
	})

	log.Info().Err(err).Interface("artifacts", len(artifacts)).Msgf("Finished inset")

	return artifacts, err
}

func substringMatch(needle, haystack string) bool {
	log.Debug().Str("needle", needle).Str("haystack", haystack).Msg("Checking contains")
	return needle == "" || strings.Contains(needle, haystack)
}

func (rs *BoltStorage) List(status []Status, namespaceSubstring, packageIdSubstring string) ([]Artifact, error) {
	log.Info().
		Interface("status", status).
		Str("namespaceSubstring", namespaceSubstring).
		Str("packageIdSubstring", packageIdSubstring).
		Msg("List query")

	results := make([]Artifact, 0)
	err := rs.db.View(func(tx *bolt.Tx) error {
		for _, s := range status {
			bucket := tx.Bucket([]byte(s))
			if bucket == nil {
				log.Debug().Str("bucket", string(s)).Msg("no such bucket")
				continue
			}
			err := bucket.ForEach(func(k, v []byte) error {
				id, err := UnmarshalArtifactId(k)
				if err != nil {
					return err
				}

				namespaceMatch := substringMatch(id.Namespace, namespaceSubstring)
				packageMatch := substringMatch(id.Package, packageIdSubstring)
				log.Debug().
					Bool("namespaceSubstring", namespaceMatch).
					Bool("packageIdSubstring", packageMatch).
					Interface("candidate", id).
					Msg("Evaluation")
				if packageMatch && namespaceMatch {
					data := ArtifactData{}
					_, err := asn1.Unmarshal(v, &data)
					if err != nil {
						return err
					}
					results = append(results, Artifact{
						ArtifactId: id,
						Repository: data.Repository,
						Revision:   data.Revision,
						DomainName: data.DomainName,
						Format:     data.Format,
						Error:      nil,
						Problems:   nil,
						Status:     data.Status,
						CreateTime: data.CreateTime,
					})
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return results, err
}

type Storage interface {
	Insert(artifacts ...Artifact) ([]Artifact, error)
	List(status []Status, namespaceSubstring, packageIdSubstring string) ([]Artifact, error)
}
