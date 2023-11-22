package main

import (
	"context"
	"fmt"
	"io"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

// getAWSClientConfig returns an aws.Config to be used on clients.
//
// It initializes the AWS SDK and creates an Amazon client configuration.
// It takes no parameters and returns a aws.Config and an error.
func getAWSClientConfig() (aws.Config, error) {
	// Initialize AWS SDK
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(defaultAWSRegion))
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

// listSSORoles retrieves a list of IAM roles that are used by AWS SSO service.
//
// This function does not take any parameters.
// It returns a slice of types.Role and an error.
func listSSORoles() ([]types.Role, error) {

	var pathPrefix string = "/aws-reserved/sso.amazonaws.com/"
	var pageSize int32 = 10

	logger.Info("Retrieving SSO roles from AWS IAM...")

	cfg, err := getAWSClientConfig()
	if err != nil {
		logger.Fatal("Unable to load SDK config, %v", zap.Error(err))
	}
	client := iam.NewFromConfig(cfg)

	// Create a list roles request
	params := &iam.ListRolesInput{
		MaxItems:   aws.Int32(10),
		PathPrefix: aws.String(pathPrefix),
	}

	// Create paginator for listing roles
	paginator := iam.NewListRolesPaginator(
		client,
		params,
		func(o *iam.ListRolesPaginatorOptions) { o.Limit = pageSize },
	)

	// Paginate through IAM Roles
	pageNum := 0
	var roles []types.Role
	for paginator.HasMorePages() {
		logger.Debug(fmt.Sprintf("Paginating through IAM Roles (page %d)...", (pageNum + 1)))
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			logger.Error("Error occurred while paginating through roles", zap.Error(err))
			return roles, err
		}
		roles = append(roles, output.Roles...)
		pageNum++
	}
	logger.Info(fmt.Sprintf("%d SSO roles retrieved from AWS IAM", len(roles)))
	return roles, nil
}

// translatePermissionSetNameToARN translates a given permission set name to an ARN in the SSORoleMapping struct.
//
// The function takes a SSORoleMapping struct and a slice of IAM roles as input parameters.
// It returns the updated SSORoleMapping struct with the RoleARN field populated and the PermissionSet field cleared, or an error if the permission set name is not found in the IAM roles.
func translatePermissionSetNameToARN(mapping SSORoleMapping, iamRoles []types.Role) (SSORoleMapping, error) {

	logger.Debug(fmt.Sprintf("Translating %s permission set to ARN", mapping.PermissionSet))

	// Create a regex matchet to find a role by permission set name ("AWSReservedSSO_devops_07572db8b73986b8")
	r, err := regexp.Compile(fmt.Sprintf("^AWSReservedSSO_%s_[[:alnum:]]{16}$", mapping.PermissionSet))
	if err != nil {
		panic(err)
	}

	// Get index of IAM role matching permission set name. If permission set name is not found - return error
	idx := slices.IndexFunc(iamRoles, func(role types.Role) bool { return r.Match([]byte(*role.RoleName)) })
	if idx == -1 {
		return mapping, fmt.Errorf("permission set %s not found in AWS IAM service", mapping.PermissionSet)
	}

	logger.Debug(fmt.Sprintf("Found IAM role %s with ARN %s which matches %s permission set", *iamRoles[idx].RoleName, *iamRoles[idx].Arn, mapping.PermissionSet))
	mapping.RoleARN = removePathFromRoleARN(*iamRoles[idx].Arn, *iamRoles[idx].Path) // Populate RoleARN field with retrieved value and path removed
	mapping.PermissionSet = ""                                                       // Clear PermissionSet field with empty string

	return mapping, nil
}

// removePathFromRoleARN removes the specified path from the given role ARN.
//
// It takes in two parameters:
// - arn (string): The role ARN to remove the path from.
// - path (string): The path to be removed from the role ARN.
//
// It returns a string representing the modified role ARN.
func removePathFromRoleARN(arn string, path string) string {
	r, err := regexp.Compile(path)
	if err != nil {
		panic(err)
	}

	return r.ReplaceAllString(arn, "/")
}

// Get AWS account ID
func getAccountId() (string, error) {
	logger.Debug("Reading AWS Account ID...")

	cfg, err := getAWSClientConfig()
	if err != nil {
		logger.Fatal("Unable to load SDK config, %v", zap.Error(err))
	}

	client := sts.NewFromConfig(cfg)
	if err != nil {
		logger.Fatal("Unable to load SDK config, %v", zap.Error(err))
	}
	input := &sts.GetCallerIdentityInput{}

	req, err := client.GetCallerIdentity(context.TODO(), input)
	if err != nil {
		return "", err
	}

	logger.Debug(fmt.Sprintf("Retrievied %s as AWS Account ID", *req.Account))

	return *req.Account, nil
}

func getInstanceRole() string {
	cfg, err := getAWSClientConfig()
	if err != nil {
		logger.Fatal("Unable to load SDK config, %v", zap.Error(err))
	}

	client := imds.NewFromConfig(cfg)
	response, err := client.GetMetadata(context.TODO(), &imds.GetMetadataInput{Path: "iam/security-credentials"})
	if err != nil {
		logger.Fatal("Unable to retrieve the private IP address from the EC2 instance: %v", zap.Error(err))
	}

	role, err := io.ReadAll(response.Content)
	if err != nil {
		logger.Fatal("Unable to read role name from response: %v", zap.Error(err))
	}

	return string(role)
}
