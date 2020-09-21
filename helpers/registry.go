package helpers

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
)

func GetLatestImage(registryID *string, repository *ecr.Repository) (container *ecr.Image, err error) {
	input := createListImagesInput(*repository)
	svc := ecr.New(session.New())
	output, err := svc.ListImagesRequest(input)

}

func GetEcrRepositories(registryID *string) (repositoryList *ecr.DescribeRepositoriesOutput, err error) {

	input, _ := createDescribeRepositoriesInput(registryID) //Use default registry for now

	svc := ecr.New(session.New())

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

	return result, err
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

func createListImagesInput(repository ecr.Repository) (input *ecr.ListImagesInput) {
	input = &ecr.ListImagesInput{
		Filter:         &ecr.ListImagesFilter{TagStatus: aws.String("TAGGED")},
		MaxResults:     aws.Int64(1000),
		NextToken:      nil,
		RepositoryName: repository.RepositoryName,
	}
	if repository.RegistryId != nil {
		input.RegistryId = repository.RegistryId
	}
	return input
}
