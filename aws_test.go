package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func init() {
	setupLogger(true)
}

func TestRemovePathFromRoleARN(t *testing.T) {

	// Tests if a valid path is removed from role ARN
	t.Run("Remove valid path from role ARN", func(t *testing.T) {
		arn := "arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/eu-west-1/roleName"
		path := "/aws-reserved/sso.amazonaws.com/eu-west-1/"

		want := "arn:aws:iam::123456789012:role/roleName"
		got := removePathFromRoleARN(arn, path)

		if got != want {
			t.Errorf("removePathFromRoleARN(%s, %s) = %s, want %s", arn, path, got, want)
		}
	})

	// Tests if a invalid path is not removed from role ARN
	t.Run("Remove not valitpath from role ARN", func(t *testing.T) {
		arn := "arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/eu-west-1/roleName"
		path := "/path/"

		want := arn
		got := removePathFromRoleARN(arn, path)

		if got != want {
			t.Errorf("removePathFromRoleARN(%s, %s) = %s, want %s", arn, path, got, want)
		}
	})
}

func TestTranslatePermissionSetNameToARN(t *testing.T) {
	mapping := SSORoleMapping{
		RoleARN:       "",
		PermissionSet: "devops",
		Username:      "",
		Groups:        []string{},
		UserID:        "",
	}

	// Test when permission set does not exist
	t.Run("Role exist", func(t *testing.T) {

		iamRoles := []types.Role{{
			RoleName: aws.String("AWSReservedSSO_devops_0123456789abcdef"),
			Path:     aws.String("/path/"),
			Arn:      aws.String("arn:aws:iam::123456789012:role/path/AWSReservedSSO_devops_0123456789abcdef"),
		}}

		want := SSORoleMapping{
			RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef",
			PermissionSet: "",
			Username:      "",
			Groups:        []string{},
			UserID:        "",
		}

		got, _ := translatePermissionSetNameToARN(mapping, iamRoles)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("TranslatePermissionSetNameToARNFound() returned unexpected object: %+v, want %+v", got, want)
		}
	})

	// Test when permission set does exist
	t.Run("Role does not exist", func(t *testing.T) {
		iamRoles := []types.Role{{
			RoleName: aws.String("AWSReservedSSO_sre_0123456789abcdef"),
			Path:     aws.String("/path/"),
			Arn:      aws.String("arn:aws:iam::123456789012:role/path/AWSReservedSSO_sre_0123456789abcdef"),
		}}

		want := fmt.Sprintf("permission set %s not found in AWS IAM service", mapping.PermissionSet)
		_, got := translatePermissionSetNameToARN(mapping, iamRoles)

		if got.Error() != want {
			t.Errorf("Got unexpected error: %s, was expecting to get: %s", got, want)
		}
	})

}
