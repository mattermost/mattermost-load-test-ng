package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/mattermost/mattermost/server/public/model"
)

const (
	// Minimum permissions needed for the policy to be effective
	logsPermissionPutLogEventsBatch = "logs:PutLogEventsBatch"
	logsPermissionPutLogEvents      = "logs:PutLogEvents"
	logsPermissionCreateLogStream   = "logs:CreateLogStream"
)

var (
	// Minimum policy needed
	minPolicyDocument = policyDocument{
		Version: "2012-10-17",
		Statement: []policyDocumentStmt{
			{
				Effect: "Allow",
				Principal: policyDocumentStmtPrincipal{
					Service: "es.amazonaws.com",
				},
				Action: []string{
					logsPermissionPutLogEventsBatch,
					logsPermissionPutLogEvents,
					logsPermissionCreateLogStream,
				},
				Resource: "arn:aws:logs:*",
			},
		},
	}

	// Not found error
	ErrNotFound = fmt.Errorf("policy not found")
)

type policyDocument struct {
	Version   string               `json:"Version"`
	Statement []policyDocumentStmt `json:"Statement"`
}
type policyDocumentStmt struct {
	Effect    string                      `json:"Effect"`
	Principal policyDocumentStmtPrincipal `json:"Principal"`
	Action    []string                    `json:"Action"`
	Resource  string                      `json:"Resource"`
}
type policyDocumentStmtPrincipal struct {
	Service string `json:"Service"`
}

// checkCloudWatchLogsPolicy checks whether the AWS account has a
// CloudWatchLogs resource-based policy attached to the OpenSearch service
// that allows the logs:PutLogEventsBatch, logs:PutLogEvents and
// logs:CreateLogStream actions over the OpenSearch service.
// In particular, this is the minimum we need:
//
//	{
//		Version: "2012-10-17",
//		Statement: []{
//			{
//				Effect: "Allow",
//				Principal: polDocStmtPrincipal{
//					Service: "es.amazonaws.com",
//				},
//				Action: []string{
//					"logs:PutLogEventsBatch",
//					"logs:PutLogEvents",
//					"logs:CreateLogStream",
//				},
//				Resource: "arn:aws:logs:*",
//			},
//		},
//	}
func (t *Terraform) checkCloudWatchLogsPolicy() error {
	// Create CloudWatchLogs client
	cfg, err := t.GetAWSConfig()
	if err != nil {
		return err
	}
	cwclient := cloudwatchlogs.NewFromConfig(cfg)

	// Iterate through all available policies and check if one of them passes
	// the verification check
	var nextToken *string
	limit := int32(10)
	hasMore := true
	for hasMore {
		input := cloudwatchlogs.DescribeResourcePoliciesInput{
			Limit:     &limit,
			NextToken: nextToken,
		}
		output, err := cwclient.DescribeResourcePolicies(context.Background(), &input)
		if err != nil {
			return err
		}
		nextToken = output.NextToken
		hasMore = nextToken != nil

		for _, policy := range output.ResourcePolicies {
			var document policyDocument
			err := json.Unmarshal([]byte(*policy.PolicyDocument), &document)
			if err != nil {
				return fmt.Errorf("unable to unmarshal policy")
			}

			if verifyPolicy(document) {
				return nil
			}
		}
	}

	return ErrNotFound
}

// createCloudWatchLogsPolicy creates a resource-based policy to grant
// permissions to the AWS OpenSearch service to publish logs to CloudWatch
func (t *Terraform) createCloudWatchLogsPolicy() error {
	// Create CloudWatchLogs client
	cfg, err := t.GetAWSConfig()
	if err != nil {
		return err
	}
	cwclient := cloudwatchlogs.NewFromConfig(cfg)

	docJsonBytes, err := json.Marshal(minPolicyDocument)
	if err != nil {
		return err
	}
	docJsonStr := string(docJsonBytes)

	input := cloudwatchlogs.PutResourcePolicyInput{
		PolicyName:     model.NewPointer("lt-cloudwatch-log-policy"),
		PolicyDocument: model.NewPointer(docJsonStr),
	}
	if _, err := cwclient.PutResourcePolicy(context.Background(), &input); err != nil {
		return fmt.Errorf("failed to create CloudWatchLogs policy; it can be manually created by running `aws logs put-resource-policy --policy-name lt-cloudwatch-log-policy --policy-document %q`; the next `deployment create` should work when such a policy is present in the AWS account; original error: %w", docJsonStr, err)
	}

	return nil
}

// verifyPolicy verifies that the passed policy document has the minimum
// permissions needed for AWS OpenSearch to publish logs to CloudWatch.
func verifyPolicy(document policyDocument) bool {
	if len(document.Statement) == 0 {
		return false
	}

	for _, statement := range document.Statement {
		if statement.Effect != "Allow" {
			continue
		}

		if statement.Principal.Service != "es.amazonaws.com" {
			continue
		}

		if !slices.Contains(statement.Action, logsPermissionPutLogEventsBatch) {
			continue
		}

		if !slices.Contains(statement.Action, logsPermissionPutLogEvents) {
			continue
		}

		if !slices.Contains(statement.Action, logsPermissionCreateLogStream) {
			continue
		}

		if statement.Resource != "arn:aws:logs:*" {
			continue
		}

		return true
	}

	return false
}
