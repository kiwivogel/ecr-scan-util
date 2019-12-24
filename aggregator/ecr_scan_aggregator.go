package aggregator

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"strings"
)

var region = "eu-west-1"

func BatchGetScanResultsByTag(repositories map[string]string, registryId string, prefix string) (map[string]*ecr.DescribeImageScanFindingsOutput, error) {
	result := make(map[string]*ecr.DescribeImageScanFindingsOutput)
	var err error
	for c, v := range repositories {

		result[c], err = EcrGetScanResultsByTag(strings.Join([]string{prefix, c}, "/"), v, registryId)
		if err != nil {
			return nil, err
		}
	}
	return result, err
}

func EcrGetScanResultsByTag(repositoryName string, imageTag string, registryId string) (findings *ecr.DescribeImageScanFindingsOutput, err error) {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	svc := ecr.New(s)
	input, err := createImageScanFindingsInput(repositoryName, imageTag, registryId)
	if err != nil {
		fmt.Println(err.Error())
	}
	result, err := svc.DescribeImageScanFindings(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			//TODO: handle missing repository or test result more gracefully (warn and skip if in batchmode)
			case ecr.ErrCodeRepositoryNotFoundException:
				fmt.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			case ecr.ErrCodeScanNotFoundException:
				fmt.Println(ecr.ErrCodeScanNotFoundException, aerr.Error())
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
	return result, nil
}

func createImageScanFindingsInput(repositoryName string, imageTag string, registryId string) (input *ecr.DescribeImageScanFindingsInput, err error) {
	input = &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &ecr.ImageIdentifier{
			ImageTag: aws.String(imageTag),
		},
		MaxResults: aws.Int64(1000), //to avoid paginated results with more than 100 but less than 1000 results.
	}
	if registryId != "" {
		input.RegistryId = aws.String(registryId)
	}
	return input, err
}
