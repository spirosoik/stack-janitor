package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var cfg config

func main() {
	err := LoadConfig(log.New())
	if err != nil {
		log.WithError(err).Error("Unable to load config")
		os.Exit(1)
	}
	if cfg.Debug {
		handler(context.Background(), events.CloudWatchEvent{})
		return
	}
	lambda.Start(handler)
}

func handler(ctx context.Context, cloudWatchEvent events.CloudWatchEvent) error {
	stacks, err := fetchStacks()
	if err != nil {
		return err
	}

	filteredStacks, err := filterStacks(stacks, cfg.TagKey, cfg.TagValue, cfg.MaxExpirationHours)
	if err != nil {
		return err
	}

	err = forceDelete(filteredStacks)
	if err != nil {
		return err
	}
	return nil
}

// fetchStacks fetches the cloudformation stacks
func fetchStacks() ([]string, error) {
	log.Info("Collecting cloudformation stacks...")
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return []string{}, errors.Wrap(err, "session.NewSession")
	}
	svc := cloudformation.New(sess)
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
func filterStacks(stacks []string, tagKey string, tagValue string, maxTime *int) ([]string, error) {
	log.Info("Filtering cloudformation stacks...")
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return []string{}, errors.Wrap(err, "session.NewSession")
	}
	var filteredNames []string
	svc := cloudformation.New(sess)
	for _, n := range stacks {
		result, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(n),
		})
		if err != nil {
			return []string{}, errors.Wrap(err, "svc.DescribeStacks")
		}
		for _, s := range result.Stacks {
			elapsedHours := time.Now().Sub(*s.CreationTime).Hours()
			if ok := find(s.Tags, tagKey, tagValue); ok && elapsedHours >= float64(*maxTime) {
				filteredNames = append(filteredNames, *s.StackName)
				log.Info(*s.StackName)
			}
		}
	}
	log.Infof("Found %d stacks with the provided tag: %s:%s", len(filteredNames), tagKey, tagValue)
	return filteredNames, nil
}

func forceDelete(stacks []string) error {
	log.Info("Deleting cloudformation stacks...")
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return errors.Wrap(err, "session.NewSession")
	}

	deletedCounter := 0
	svc := cloudformation.New(sess)
	for _, n := range stacks {
		_, err = svc.DeleteStack(&cloudformation.DeleteStackInput{
			StackName: aws.String(n),
		})
		if err != nil {
			log.WithError(err).Error("Unable to delete stack with name: %s", n)
		}
		deletedCounter++
	}
	log.Infof("Deleted %d stacks", deletedCounter)
	return nil
}

// find searches in a slice if the value exists
func find(slice []*cloudformation.Tag, tagKey string, tagValue string) bool {
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
