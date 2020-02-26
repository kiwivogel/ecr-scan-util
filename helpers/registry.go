package helpers

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

func GetEcrRepositories(registryID *string) (repositoryList []*ecr.Repository, err error) {

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

	return result.Repositories, err
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
