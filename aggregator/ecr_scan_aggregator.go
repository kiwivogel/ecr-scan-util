package aggregator

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
)

// EcrGetScanResults pulls results from the latest image scan for a given ecr.Image it uses a globally defined session.Session and outputs stdout messages to a logger.Logger.
// It returns a pointer to a ecr.DescribeImageScanFindingsOutput and an error.
func EcrGetScanResults(image *ecr.Image, session *session.Session, l *logger.Logger) (result *ecr.DescribeImageScanFindingsOutput, err error) {
	svc := ecr.New(session)
	// Create input parameter for api call
	input := createImageScanFindingsInput(image)

	// Retrieve results of scan
	result, err = svc.DescribeImageScanFindings(input)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			// Handle specific error types as defined by the aws SDK
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
				l.Error(err.Error())
			}
		} else {
			// Print the error, cast err to awserr. Error to get the Code and
			// Message from an error.
			l.Fatalf("%e is not a recognized Error", err.Error())
		}
		// Return an output struct with failed status and an error message when results cannot be retrieved.
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

// createImageScanFindingsInput takes an input ecr.Image and returns a ecr.DescribeImageScanFindingsInput that's used by
// the ECR client to retrieve scan results. We do not return an error since ecr.Image object has been validated upstream.
// Using a lazy hack to avoid implementing pagination.
func createImageScanFindingsInput(image *ecr.Image) (input *ecr.DescribeImageScanFindingsInput) {
	input = &ecr.DescribeImageScanFindingsInput{
		RepositoryName: image.RepositoryName,
		ImageId:        image.ImageId,
		MaxResults:     aws.Int64(1000), //to avoid paginated results with more than 100 but less than 1000 results.
	}
	if image.RegistryId != nil {
		input.RegistryId = image.RegistryId
	}
	return input
}
