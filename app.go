package main

import (
	"artifacts/src"
	"fmt"
	"github.com/aws/aws-sdk-go/service/codeartifact"
	"github.com/rs/zerolog/log"
	"time"
)

func main() {
	//log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	//zerolog.SetGlobalLevel(zerolog.DebugLevel)

	s, err := artifacts.LoadSpecification()
	if err != nil {
		log.Fatal().Msgf("Failed to load config %v\n", err)
	}

	session, err := artifacts.InitDbConnection(s)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to db %v\n", err)
	}

	aux := artifacts.NewCodeArtifactAux(s)

	if s.Load {
		log.Info().Fields(s).Msg("Importing artifact lists from AWS codeartifact")
		go loadArtifacts(err, aux, s, session)
	}
	server := artifacts.NewServer(s.Listen, session)
	artifacts.StartServer(server)
}

func loadArtifacts(err error, aux artifacts.CodeArtifactWrapper, s artifacts.Specification, session *artifacts.RethinkStorage) {
	repos, err := aux.AllRepos()
	if err != nil {
		log.Fatal().Msgf("Failed to list repos %v\n", err)
	}

	for _, repo := range repos.Repositories {
		if s.Skip(*repo.Name) {
			log.Printf("Skipping %v\n", *repo.Name)
			continue
		}
		log.Printf("Extracting %v", repo)

		// a channel of packages for this repo
		ps := make(chan artifacts.Package)
		go packages(repo, aux, ps)

		for p := range ps {
			if p.Error != nil {
				log.Fatal().Stack().Err(p.Error)
			}

			// a channel of artifacts
			as := make(chan artifacts.Artifact)
			go versions(p, aux, as)

			for a := range as {
				if a.Error != nil {
					log.Fatal().Stack().Err(a.Error)
				}

				go func() {
					errors := session.Insert(a)
					var fatal = false
					for _, err := range errors {
						if err != nil {
							fatal = true
							log.Err(err)
						}

					}
					if fatal {
						log.Fatal().Msg("Error on insert; exiting")
					}
				}()

			}
		}
	}
}

func packages(repository *codeartifact.RepositorySummary, aux artifacts.CodeArtifactWrapper, ps chan artifacts.Package) {
	packages, err := aux.AllPackagesInRepo(repository)
	if err != nil {
		ps <- artifacts.Package{Error: err}
		close(ps)
		return
	}

	log.Info().Msgf("Found %d packages in %s", len(packages.Packages), *repository.Name)
	start := time.Now()

	for _, pack := range packages.Packages {
		ps <- artifacts.Package{
			RepositorySummary: repository,
			PackageSummary:    pack,
		}
	}
	log.Info().Msgf("Finished repository %s in %s", *repository.Name, time.Since(start))
	close(ps)
}

func versions(p artifacts.Package, aux artifacts.CodeArtifactWrapper, vers chan artifacts.Artifact) {
	response, err := aux.AllPackageVersions(p.PackageSummary, p.RepositorySummary)
	if err != nil {
		vers <- artifacts.Artifact{Error: err}
		close(vers)
		return
	}

	for _, version := range response.Versions {
		packageId := *p.Package
		namespace := *p.Namespace
		v := *version.Version
		status := artifacts.Status(*version.Status)

		vers <- artifacts.Artifact{
			Id:         fmt.Sprintf("%s:%s:%s", namespace, packageId, v),
			Repository: *p.Name,
			Package:    packageId,
			Namespace:  namespace,
			Version:    v,
			Revision:   *version.Revision,
			DomainName: *p.DomainName,
			Format:     *p.Format,
			Status:     status,
			CreateTime: time.Now(),
		}
	}
	close(vers)
}
