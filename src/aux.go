package artifacts

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codeartifact"
)

type CodeArtifactWrapper struct {
	Specification
	Client *codeartifact.CodeArtifact
}

func NewCodeArtifactAux(s Specification) CodeArtifactWrapper {
	whatIsASessionLol := session.Must(session.NewSession())
	client := codeartifact.New(whatIsASessionLol, aws.NewConfig().WithRegion(s.Region).WithCredentialsChainVerboseErrors(true))

	return CodeArtifactWrapper{
		Specification: s,
		Client:        client,
	}
}

func (s *CodeArtifactWrapper) AllPackageVersions(pack *codeartifact.PackageSummary, repository *codeartifact.RepositorySummary) (codeartifact.ListPackageVersionsOutput, error) {
	var output = codeartifact.ListPackageVersionsOutput{
		DefaultDisplayVersion: nil,
		Format:                nil,
		Namespace:             nil,
		NextToken:             nil,
		Package:               nil,
		Versions:              make([]*codeartifact.PackageVersionSummary, 0),
	}

	for {

		input := codeartifact.ListPackageVersionsInput{
			Domain:     &s.Domain,
			MaxResults: s.AwsPageSize(),
			Namespace:  pack.Namespace,
			NextToken:  output.NextToken,
			Package:    pack.Package,
			Repository: repository.Name,
			Format:     pack.Format,
		}

		var response, err = s.Client.ListPackageVersions(&input)
		if err != nil {
			return output, err
		}

		output = codeartifact.ListPackageVersionsOutput{
			DefaultDisplayVersion: response.DefaultDisplayVersion,
			Format:                response.Format,
			Namespace:             response.Namespace,
			NextToken:             response.NextToken,
			Package:               response.Package,
			Versions:              append(output.Versions, response.Versions...),

		}

		if response.NextToken == nil {
			return output, nil
		}
	}

}

func (s *CodeArtifactWrapper) AllPackagesInRepo(repository *codeartifact.RepositorySummary) (codeartifact.ListPackagesOutput, error) {
	output := codeartifact.ListPackagesOutput{
		NextToken: nil,
		Packages:  make([]*codeartifact.PackageSummary, 0),
	}

	for {
		var response, err = s.Client.ListPackages(&codeartifact.ListPackagesInput{
			Domain:        &s.Domain,
			MaxResults:    s.AwsPageSize(),
			NextToken:     output.NextToken,
			PackagePrefix: nil,
			Repository:    repository.Name,
		})

		if err != nil {
			return output, err
		}

		output = codeartifact.ListPackagesOutput{
			NextToken: response.NextToken,
			Packages:  append(output.Packages, response.Packages...),
		}

		if response.NextToken == nil {
			return output, nil
		}
	}

}

// AllRepos Lists the repositories available to the creds: Equivalent to aws codeartifact list-repositories.
func (s *CodeArtifactWrapper) AllRepos() (codeartifact.ListRepositoriesOutput, error) {
	output := codeartifact.ListRepositoriesOutput{
		NextToken:    nil,
		Repositories: make([]*codeartifact.RepositorySummary, 0),
	}

	for {
		input := codeartifact.ListRepositoriesInput{
			MaxResults:       s.AwsPageSize(),
			NextToken:        output.NextToken,
			RepositoryPrefix: nil,
		}
		response, err := s.Client.ListRepositories(&input)
		if err != nil {
			return output, err
		}

		output := codeartifact.ListRepositoriesOutput{
			NextToken:    response.NextToken,
			Repositories: append(output.Repositories, response.Repositories...),
		}

		if response.NextToken == nil {
			return output, nil
		}
	}
}
