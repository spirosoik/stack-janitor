package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/getsentry/sentry-go"
	"github.com/makasim/sentryhook"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// cfg global configuration across the whole
// services
var cfg config

// GitSHA is set during build
var GitSHA = "<not set>"

// logger
var logger *logrus.Logger

func main() {
	logger = logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.TextFormatter{
		DisableColors: !isatty.IsTerminal(os.Stdout.Fd()),
		FullTimestamp: true,
	}
	logger.WithField("git_sha", GitSHA).Info("Current version")

	// load config
	if err := LoadConfig(logger); err != nil {
		logger.WithError(err).Error("Unable to load config")
		os.Exit(1)
	}

	// sentry setup
	if cfg.Sentry.Enabled {
		logger.AddHook(sentryhook.New([]logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel}))
		defer sentry.Flush(time.Second * 5)
		defer sentry.Recover()
		err := sentry.Init(sentry.ClientOptions{
			Release:          fmt.Sprintf("stack-janitor@%s", GitSHA),
			Dsn:              cfg.Sentry.DSN,
			AttachStacktrace: true,
			Environment:      cfg.Environment,
			Debug:            cfg.Debug,
		})
		if err != nil {
			logger.WithError(err).Error("Sentry.Init failed")
			os.Exit(1)
		}
	}

	// if it's debug run locally
	if cfg.Debug {
		handler(context.Background(), events.CloudWatchEvent{})
		return
	}

	// start handler
	lambda.Start(handler)
}

func handler(ctx context.Context, cloudWatchEvent events.CloudWatchEvent) {
	sess, err := session.NewSession()
	if err != nil {
		logger.WithError(errors.Wrap(err, "session.NewSession")).Error()
		return
	}
	svc := cloudformation.New(sess)

	stacks, err := fetchStacks(svc)
	if err != nil {
		logger.WithError(err).Error()
		return
	}
	filteredStacks, err := filterStacks(svc, stacks, cfg)
	if err != nil {
		logger.WithError(err).Error()
		return
	}

	err = forceDelete(svc, filteredStacks)
	if err != nil {
		logger.WithError(err).Error()
	}
}

// fetchStacks fetches the cloudformation stacks
func fetchStacks(svc *cloudformation.CloudFormation) ([]string, error) {
	logger.Info("Collecting cloudformation stacks...")
	result, err := svc.ListStacks(&cloudformation.ListStacksInput{
		StackStatusFilter: []*string{
			aws.String(cloudformation.ResourceStatusCreateComplete),
			aws.String(cloudformation.ResourceStatusCreateFailed),
			aws.String(cloudformation.ResourceStatusUpdateComplete),
			aws.String(cloudformation.ResourceStatusDeleteFailed),
		},
	})
	if err != nil {
		return []string{}, errors.Wrap(err, "svc.ListStacks")
	}
	var names []string
	for _, s := range result.StackSummaries {
		names = append(names, *s.StackName)
	}
	return names, nil
}

// filterStacks filter tasks based on rule we give in map string
func filterStacks(svc *cloudformation.CloudFormation, stacks []string, cfg config) ([]string, error) {
	logger.Info("Filtering cloudformation stacks...")
	var filteredNames []string
	for _, n := range stacks {
		result, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(n),
		})
		if err != nil {
			return nil, errors.Wrap(err, "svc.DescribeStacks")
		}
		for _, s := range result.Stacks {
			if !hasTag(s.Tags, cfg.TagKey, cfg.TagValue) {
				continue
			}
			elapsedHours := time.Since(*s.CreationTime).Hours()
			if elapsedHours < cfg.MaxExpirationHours.Hours() {
				continue
			}
			filteredNames = append(filteredNames, aws.StringValue(s.StackName))
		}
	}
	logger.Infof("Found %d stacks with the provided tag: %s:%s", len(filteredNames), cfg.TagKey, cfg.TagValue)
	return filteredNames, nil
}

func forceDelete(svc *cloudformation.CloudFormation, stacks []string) error {
	logger.Info("Deleting cloudformation stacks...")
	deletedCounter := 0
	for _, n := range stacks {
		_, err := svc.DeleteStack(&cloudformation.DeleteStackInput{
			StackName: aws.String(n),
		})
		if err != nil {
			logger.WithError(err).Errorf("Unable to delete stack with name: %s", n)
			continue
		}
		deletedCounter++
	}
	logger.Infof("Deleted %d stacks", deletedCounter)
	return nil
}

// find searches in a slice if the value exists
func hasTag(slice []*cloudformation.Tag, tagKey string, tagValue string) bool {
	for _, item := range slice {
		if *item.Key != tagKey {
			continue
		}
		if tagValue == *item.Value {
			return true
		}
	}
	return false
}
