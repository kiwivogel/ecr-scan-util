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

func BatchGetScanResults(repositories map[interface{}]interface{}) {
	for c, v := range repositories {
		ecrGetTagScanResults(strings.Join([]string{"zorgdomein/", strings.Replace(strings.Replace(c.(string), "_version", "", 1), "_", "-", -1)}, ""), v.(string))
	}
}

func ecrGetTagScanResults(repositoryName string, imageTag string) {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	svc := ecr.New(s)
	input := &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &ecr.ImageIdentifier{
			ImageTag: aws.String(imageTag),
		},
	}
	result, err := svc.DescribeImageScanFindings(input)
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

	fmt.Println(result)
}
