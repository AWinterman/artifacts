package artifacts

import (
	"github.com/aws/aws-sdk-go/service/codeartifact"
	"github.com/rs/zerolog/log"
	"time"
)

func LoadArtifacts(err error, aux CodeArtifactWrapper, s Specification, session *BoltStorage) {
	repos, err := aux.AllRepos()
	if err != nil {
		log.Fatal().Msgf("Failed to list repos %v\n", err)
	}

	for _, repo := range repos.Repositories {
		if s.Skip(*repo.Name) {
			log.Printf("Skipping %v\n", *repo.Name)
			continue
		}
		log.Printf("Extracting REpo %v", repo)

		// a channel of packages for this repo
		ps := make(chan Package)
		go Packages(repo, aux, ps)

		for p := range ps {
			if p.Error != nil {
				log.Fatal().Stack().Err(p.Error)
			}
			log.Debug().
				Interface("package", p).
				Msg("Extracting package")

			// a channel of artifacts
			as := make(chan Artifact, s.PageSize)

			go Versions(p, aux, as)

			//for artifact := range as {
			//	insertBatch([]Artifact{artifact}, p, session)
			//}
			batchArtifacts := BatchArtifacts(s.PageSize, as)

			for batch := range batchArtifacts {
				insertBatch(batch, p, session)
			}

		}
	}
}

func insertBatch(batch []Artifact, p Package, session *BoltStorage) {
	for _, a := range batch {
		if a.Error != nil {
			log.Fatal().Err(a.Error).Msgf("Failed retrieving artifact from aws for package %v", p)
		}
	}

	arts, err := session.Insert(batch...)

	for _, a := range arts {
		if a.Error != nil {
			log.Err(a.Error)
		}
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Error on insert; exiting")
	}
}

func BatchArtifacts(batchSize int, inChan chan Artifact) chan []Artifact {
	outChan := make(chan []Artifact)
	batch := make([]Artifact, 0)
	go func() {
		defer close(outChan)
		for {
			select {
			case event, ok := <-inChan:
				if !ok {
					return
				}

				batch = append(batch, event)

				if len(batch) >= batchSize {
					log.Debug().Interface("batch", batch).Int("size", batchSize).Msg("Emitting batch")
					outChan <- batch
					// reset for next batch
					batch = make([]Artifact, 0)
				}
			case <-time.After(5 * time.Second):
				log.Info().Int("size", len(batch)).Msg("Sending partial batch")
				// process whatever we have seen so far if the batch size isn't filled in 5 secs
				if len(batch) > 0 {
					outChan <- batch
				}
			}
		}
	}()
	return outChan
}

func Packages(repository *codeartifact.RepositorySummary, aux CodeArtifactWrapper, ps chan Package) {
	packages, err := aux.AllPackagesInRepo(repository)
	if err != nil {
		ps <- Package{Error: err}
		close(ps)
		return
	}

	log.Info().Msgf("Found %d packages in %s", len(packages.Packages), *repository.Name)
	start := time.Now()

	for _, pack := range packages.Packages {
		ps <- Package{
			RepositorySummary: repository,
			PackageSummary:    pack,
		}
	}
	log.Info().Msgf("Finished repository %s in %s", *repository.Name, time.Since(start))
	close(ps)
}

func Versions(p Package, aux CodeArtifactWrapper, vers chan Artifact) {
	response, err := aux.AllPackageVersions(p.PackageSummary, p.RepositorySummary)
	if err != nil {
		vers <- Artifact{Error: err}
		log.Info().Err(err).Interface("package", p.PackageSummary).Msg("Error extracting versions for package")
		close(vers)
		return
	}
	log.Info().Int("versions", len(response.Versions)).Interface("package", response.Package).Msg("Retrieving package")

	for _, version := range response.Versions {
		packageId := *p.Package
		namespace := *p.Namespace
		v := *version.Version
		status := Status(*version.Status)

		artifact := Artifact{
			Repository: *p.Name,
			ArtifactId: ArtifactId{
				Package:   packageId,
				Namespace: namespace,
				Version:   v,
			},
			Revision:   *version.Revision,
			DomainName: *p.DomainName,
			Format:     *p.Format,
			Status:     status,
			CreateTime: time.Now(),
		}
		vers <- artifact
	}
	close(vers)
}
