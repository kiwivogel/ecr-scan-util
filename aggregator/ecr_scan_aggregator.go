package aggregator

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
	"github.com/kiwivogel/ecr-scan-util/helpers"
)

func EcrGetScanResults(image *ecr.Image, session *session.Session, l *logger.Logger) (result *ecr.DescribeImageScanFindingsOutput, err error) {
	helpers.Check(err, l)
	svc := ecr.New(session)
	input, err := createImageScanFindingsInput(image)
	if err != nil {
		fmt.Println(err.Error())
	}
	result, err = svc.DescribeImageScanFindings(input)
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
			case ecr.ErrCodeImageNotFoundException:
				l.Warningf("%s", err.Error())
			default:
				fmt.Println(err.Error())
			}
		} else {
			// Print the error, cast err to awserr. Error to get the Code and
			// Message from an error.
			l.Fatalf("%e is not a recognized Error", err.Error())
		}
		return &ecr.DescribeImageScanFindingsOutput{
			ImageId: input.ImageId,
			ImageScanStatus: &ecr.ImageScanStatus{
				Status:      aws.String("FAILED"),
				Description: aws.String(err.Error()),
			},
		}, err
	}
	return result, err
}

func createImageScanFindingsInput(image *ecr.Image) (input *ecr.DescribeImageScanFindingsInput, err error) {
	input = &ecr.DescribeImageScanFindingsInput{
		RepositoryName: image.RepositoryName,
		ImageId:        image.ImageId,
		MaxResults:     aws.Int64(1000), //to avoid paginated results with more than 100 but less than 1000 results.
	}
	if image.RegistryId != nil {
		input.RegistryId = image.RegistryId
	}
	return input, err
}
