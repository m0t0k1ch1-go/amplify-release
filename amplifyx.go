package amplifyx

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsamplify "github.com/aws/aws-sdk-go-v2/service/amplify"
	awsamplifytypes "github.com/aws/aws-sdk-go-v2/service/amplify/types"
	"github.com/samber/oops"
)

// CLI represents the command-line interface.
var CLI struct {
	Deploy  DeployArgs `cmd:""`
	Version struct{}   `cmd:""`
}

// Client is a client for AWS Amplify.
type Client struct {
	amplify *awsamplify.Client
}

// NewClient returns a new Client.
func NewClient(ctx context.Context) (*Client, error) {
	awsConf, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, oops.Wrapf(err, "failed to load aws config")
	}

	return &Client{
		amplify: awsamplify.NewFromConfig(awsConf),
	}, nil
}

// DeployArgs is the set of arguments for Deploy.
type DeployArgs struct {
	AppID               string        `required:""`
	BranchName          string        `default:"main"`
	ObservationTimeout  time.Duration `default:"5m"`
	ObservationInterval time.Duration `default:"5s"`
}

// Deploy starts a release job and waits for it to complete.
func (c *Client) Deploy(ctx context.Context, args DeployArgs) error {
	var jobSummary *awsamplifytypes.JobSummary

	{
		out, err := c.amplify.StartJob(ctx, &awsamplify.StartJobInput{
			AppId:      aws.String(args.AppID),
			BranchName: aws.String(args.BranchName),
			JobType:    awsamplifytypes.JobTypeRelease,
		})
		if err != nil {
			return oops.Wrapf(err, "failed to start job")
		}

		jobSummary = out.JobSummary
	}

	jobStatusCh := make(chan awsamplifytypes.JobStatus, 1)
	errCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(ctx, args.ObservationTimeout)
	defer cancel()

	go func() {
		for jobSummary.Status == awsamplifytypes.JobStatusPending || jobSummary.Status == awsamplifytypes.JobStatusRunning {
			time.Sleep(args.ObservationInterval)

			{
				out, err := c.amplify.GetJob(ctx, &awsamplify.GetJobInput{
					AppId:      aws.String(args.AppID),
					BranchName: aws.String(args.BranchName),
					JobId:      jobSummary.JobId,
				})
				if err != nil {
					errCh <- oops.Wrapf(err, "failed to get job")
					return
				}

				jobSummary = out.Job.Summary
			}
		}

		jobStatusCh <- jobSummary.Status
	}()

	select {
	case jobStatus := <-jobStatusCh:
		switch jobStatus {
		case awsamplifytypes.JobStatusSucceed:
			return nil
		case awsamplifytypes.JobStatusFailed:
			return oops.Errorf("job failed")
		default:
			return oops.Errorf("unexpected job status: %s", jobStatus)
		}
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Version returns the module version.
func (c *Client) Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	return info.Main.Version
}
