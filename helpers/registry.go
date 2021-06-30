package helpers

import (
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
)

// Contains functions that handle (meta)data from the ECR api's we use.

// GetLatestTag queries the ecr.Repository for the lastest tag. takes a filter string to filter out particular tags.
// We use this filtering to not scan 'experimental' or 'snapshot' containers that are only used for development but still get pushed to the
// Repository. Returns a containerTag string and an error.
func GetLatestTag(repository *ecr.Repository, filter *string, session *session.Session, l *logger.Logger) (containerTag *string, err error) {

	// Get all tags/identifiers
	imageIdentifiers, err := listImageIdentifiers(repository, session, l)
	if err != nil {
		l.Error("Failed to retrieve list of images")
		return nil, err
	}
	// Check if we need to filter the indentifiers and then do so if needed.
	if *filter != "" {
		imageIdentifiers, err = filterImageIdentifiers(imageIdentifiers, filter, l)
	}
	if err != nil {
		l.Errorf("Failed to filter list of images for %s", repository.RepositoryName)
		return nil, err
	}
	// Use returned and optionally filtered list of imageIdentifiers and query ECR for metadata
	imagesWithTimestamp, err := getImageDetails(repository, imageIdentifiers, session, l)
	if err != nil {
		l.Error("Failed to retieve list of image details")
		return nil, err
	}

	// define top level variables for logic to determine most recently pushed image
	var imageAges []time.Duration
	var minAge time.Duration

	if len(imagesWithTimestamp) > 0 {

		// iterate over list of *ecr.ImageDetail to generate a list of 'ages'
		for image := range imagesWithTimestamp {
			imageAges = append(imageAges, time.Since(*imagesWithTimestamp[image].ImagePushedAt))
		}
		// handle potential (but very unlikely) nil slice
		if len(imageAges) == 0 {
			return nil, errors.New("no imageage could be determined. Check metadata in console")
		}
		var index int

		// find index of lowest age (and as such most recently pushed image)
		for a := range imageAges {
			if imageAges[a] < minAge {
				minAge = imageAges[a]
				index = a
			}

		}
		// Return tag for most recently pushed image.
		return imagesWithTimestamp[index].ImageTags[0], nil //return imagesWithTimestamp[blaat].ImageTags[0], nil

		// handle case where only a single image is present
	} else if len(imagesWithTimestamp) == 1 {
		return imagesWithTimestamp[0].ImageTags[0], nil
		// handle all other cases returning nothing
	} else {
		return
	}
}

// filterImageIdentifiers is a helper function for GetLatestTag that we use to filter the returned image identifiers (tags specifically) to omit them from the results.
func filterImageIdentifiers(unfilteredIdentifiers []*ecr.ImageIdentifier, filterQuery *string, l *logger.Logger) (filteredImageIdentifiers []*ecr.ImageIdentifier, err error) {

	// iterate over all identifiers
	for i := range unfilteredIdentifiers {
		l.Infof("Checking tag  %s versus filter %s \n", *unfilteredIdentifiers[i].ImageTag, *filterQuery)
		// if filter does not match append to list of returned identifiers
		if strings.Contains(*unfilteredIdentifiers[i].ImageTag, *filterQuery) == false {
			filteredImageIdentifiers = append(filteredImageIdentifiers, unfilteredIdentifiers[i])
		}
	}
	// output how many results are being omited based on what filter query
	l.Infof("%v tags matched filter %s.\n", len(unfilteredIdentifiers)-len(filteredImageIdentifiers), *filterQuery)

	// handle case where filter would filter out all identifiers
	if len(filteredImageIdentifiers) == 0 {
		return nil, errors.New("All tags match filter. check filter/available tags.")
	}
	return filteredImageIdentifiers, nil
}

// getImageDetails queries ECR for details of a given image for it's identifier
func getImageDetails(repository *ecr.Repository, identifiers []*ecr.ImageIdentifier, session *session.Session, l *logger.Logger) ([]*ecr.ImageDetail, error) {
	l.Infof("Getting details for tagged images in %s", *repository.RepositoryName)

	describeImagesInput := createDescribeImagesInput(repository, identifiers)

	svc := ecr.New(session)

	describeImagesOutput, err := svc.DescribeImages(describeImagesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				l.Error(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeImageNotFoundException:
				l.Error(ecr.ErrCodeImageNotFoundException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				l.Error(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				l.Error(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				l.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			l.Error(err.Error())
		}
	}
	if describeImagesOutput != nil && describeImagesOutput.ImageDetails != nil && len(describeImagesOutput.ImageDetails) > 0 {
		return describeImagesOutput.ImageDetails, nil
	} else {
		return nil, errors.New("Could not retrieve image details")
	}
}

// listImageIdentifiers retreives ImageIdentifiers (tags and or hashes) from a given ECR repository.
func listImageIdentifiers(repository *ecr.Repository, session *session.Session, l *logger.Logger) (imageIdentifiers []*ecr.ImageIdentifier, err error) {
	l.Infof("Grabbing list of Tags for Repository %s", *repository.RepositoryName)

	listImagesInput := createListImagesInput(repository)

	svc := ecr.New(session)

	listImageOutput, err := svc.ListImages(listImagesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				l.Error(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				l.Error(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				l.Error(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				l.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			l.Error(err.Error())
		}
		return nil, err
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
				l.Error(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				l.Error(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				l.Error(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				l.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			l.Error(err.Error())
		}
		return nil, err
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
