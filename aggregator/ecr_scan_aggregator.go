package aggregator

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
	"strings"
)

var region = "eu-west-1"

func BatchGetScanResultsByTag(repositories map[string]string, registryId string, prefix string, l logger.Logger) (map[string]*ecr.DescribeImageScanFindingsOutput, error) {
	results := make(map[string]*ecr.DescribeImageScanFindingsOutput)
	var err error
	for c, v := range repositories {
		result, err := EcrGetScanResultsByTag(strings.Join([]string{prefix, c}, "/"), v, registryId, l)
		if err == nil {
			fmt.Printf("appending result for %s:%s \n", c, v)
			results[c] = result
		}
	}
	return results, err
}

func EcrGetScanResultsByTag(repositoryName string, imageTag string, registryId string, l logger.Logger) (findings *ecr.DescribeImageScanFindingsOutput, err error) {
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
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case ecr.ErrCodeServerException:
				l.Warningf("%s", err.Error())
			case ecr.ErrCodeInvalidParameterException:
				l.Warningf("%s", err.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				l.Warningf("%s", err.Error())
			case ecr.ErrCodeScanNotFoundException:
				l.Warningf("%s", err.Error())
			default:
				fmt.Println(err.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			l.Fatalf("%e is no a recognized Error", err.Error())
		}
		return &ecr.DescribeImageScanFindingsOutput{
			ImageId: input.ImageId,
			ImageScanStatus: &ecr.ImageScanStatus{
				Status:      aws.String("failed"),
				Description: aws.String(err.Error()),
			},
		}, err
	}
	return result, err
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
