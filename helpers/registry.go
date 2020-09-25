package helpers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
)

func GetLatestTag(repository *ecr.Repository, filter *string, session *session.Session, l *logger.Logger) (containerTag *string, err error) {
	//We need to grab a list of tags/hashes to use as input for createGetLifecyclePolicyPreviewInput because the
	//getLifecyclePolicyPreviewOutput is what actually contains the tag metadata (because reasons).
	imageIdentifiers, err := listImageIdentifiers(repository, session, l)
	if *filter != "" {
		imageIdentifiers, err = filterImageIdentifiers(imageIdentifiers, filter, l)
	}
	imagesWithTimestamp, err := getImageDetails(repository, imageIdentifiers, session, l)
	//We're iterating over the Results of getImageDetails

	var imageAges []time.Duration
	var minAge time.Duration

	var latestTag string
	if len(imagesWithTimestamp) > 0 {

		for image := range imagesWithTimestamp {
			imageAges = append(imageAges, time.Since(*imagesWithTimestamp[image].ImagePushedAt))
		}
		if len(imageAges) > 0 {
			minAge = imageAges[0]
		}
		for a := range imageAges {
			if imageAges[a] < minAge {
				minAge = imageAges[a]
				latestTag = *imagesWithTimestamp[a].ImageTags[0]
			}
		}
		return &latestTag, nil
	} else if len(imagesWithTimestamp) == 1 {
		return imagesWithTimestamp[0].ImageTags[0], nil
	} else {
		return
	}
}

func filterImageIdentifiers(unfilteredIdentifiers []*ecr.ImageIdentifier, filterQuery *string, l *logger.Logger) (filteredImageIdentifiers []*ecr.ImageIdentifier, err error) {

	for i := range unfilteredIdentifiers {
		l.Infof("Checking tag  %s versus filter %s \n", *unfilteredIdentifiers[i].ImageTag, *filterQuery)
		if strings.Contains(*unfilteredIdentifiers[i].ImageTag, *filterQuery) == false {
			filteredImageIdentifiers = append(filteredImageIdentifiers, unfilteredIdentifiers[i])
		}
	}
	l.Infof("%v tags matched filter %s.\n", len(unfilteredIdentifiers)-len(filteredImageIdentifiers), *filterQuery)
	if len(filteredImageIdentifiers) == 0 {
		return nil, errors.New("All tags match filter. check filter/available tags.")
	}
	return filteredImageIdentifiers, nil
}

func getImageDetails(repository *ecr.Repository, identifiers []*ecr.ImageIdentifier, session *session.Session, l *logger.Logger) ([]*ecr.ImageDetail, error) {
	l.Infof("Getting details for tagged images in %s", *repository.RepositoryName)

	describeImagesInput := createDescribeImagesInput(repository, identifiers)

	svc := ecr.New(session)

	describeImagesOutput, err := svc.DescribeImages(describeImagesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeImageNotFoundException:
				fmt.Println(ecr.ErrCodeImageNotFoundException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				fmt.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}
	if describeImagesOutput != nil && describeImagesOutput.ImageDetails != nil && len(describeImagesOutput.ImageDetails) > 0 {
		return describeImagesOutput.ImageDetails, nil
	} else {
		return nil, errors.New("Could not retrieve image details")
	}
}

func listImageIdentifiers(repository *ecr.Repository, session *session.Session, l *logger.Logger) (imageIdentifiers []*ecr.ImageIdentifier, err error) {
	l.Infof("Grabbing list of Tags for Repository %s", *repository.RepositoryName)

	listImagesInput := createListImagesInput(repository)

	svc := ecr.New(session)

	listImageOutput, err := svc.ListImages(listImagesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				fmt.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	if len(listImageOutput.ImageIds) > 0 {
		return listImageOutput.ImageIds, nil
	} else {
		l.Warningf("No Tags found for repository %s", repository.RepositoryName)
		return
	}
}

func GetEcrRepositories(registryID *string, session *session.Session, l logger.Logger) (repositoryList []*ecr.Repository, err error) {
	if registryID != nil {
		l.Infof("Getting list of ECR repostitories for registry %s", *registryID)
	} else {
		l.Infof("Getting list of ECR repostories for default registry")
	}
	input, _ := createDescribeRepositoriesInput(registryID) //Use default registry for now

	svc := ecr.New(session)

	result, err := svc.DescribeRepositories(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				fmt.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	return getRepositoryList(result)
}

func getRepositoryList(I *ecr.DescribeRepositoriesOutput) ([]*ecr.Repository, error) {
	if len(I.Repositories) > 0 {
		return I.Repositories, nil
	} else {
		return nil, errors.New("No repositories found")
	}

}

func createDescribeRepositoriesInput(registryId *string) (input *ecr.DescribeRepositoriesInput, err error) {
	input = &ecr.DescribeRepositoriesInput{
		MaxResults:      aws.Int64(1000), //Avoid dealing with paginated results.
		NextToken:       nil,
		RepositoryNames: nil,
	}
	if registryId != nil {
		input.RegistryId = registryId
	}
	return input, err
}

func createDescribeImagesInput(repository *ecr.Repository, identifiers []*ecr.ImageIdentifier) (input *ecr.DescribeImagesInput) {
	input = &ecr.DescribeImagesInput{
		ImageIds:       identifiers,
		RepositoryName: repository.RepositoryName,
	}
	if repository.RegistryId != nil {
		input.RegistryId = repository.RegistryId
	}
	return input
}

func createListImagesInput(repository *ecr.Repository) (input *ecr.ListImagesInput) {
	input = &ecr.ListImagesInput{
		Filter:         &ecr.ListImagesFilter{TagStatus: aws.String("TAGGED")},
		MaxResults:     aws.Int64(100),
		NextToken:      nil,
		RepositoryName: repository.RepositoryName,
	}
	if repository.RegistryId != nil {
		input.RegistryId = repository.RegistryId
	}
	return input
}

func createStartImageScanInput(image *ecr.Image) *ecr.StartImageScanInput {
	return &ecr.StartImageScanInput{
		ImageId:        image.ImageId,
		RegistryId:     image.RegistryId,
		RepositoryName: image.RepositoryName,
	}
}
