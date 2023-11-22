package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func init() {
	setupLogger(true)
}

func TestGetConfigMap(t *testing.T) {
	// Test when ConfigMap does not exist
	t.Run("ConfigMap does not exist", func(t *testing.T) {

		fakeClientSet := fake.NewSimpleClientset()

		_, err := getConfigMap(fakeClientSet, "NOT_EXISTING_CONFIGMAP", "NOT_EXISTING_NAMESPASCE")
		if !errors.IsNotFound(err) {
			t.Errorf("Got unexpected error: %s, was expecting to get NotFound", err)
		}
	})

	// Test when ConfigMap exist
	t.Run("ConfigMap exist", func(t *testing.T) {

		want := &v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "NOT_EXISTING_CONFIGMAP",
				Namespace:   "NOT_EXISTING_NAMESPASCE",
				Annotations: map[string]string{},
			},
		}

		fakeClientSet := fake.NewSimpleClientset(want)

		got, err := getConfigMap(fakeClientSet, want.Name, want.Namespace)

		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("getConfigMap() returned unexpected object: %+v, want %+v", got, want)
		}
	})
}

func TestSetConfigMap(t *testing.T) {

	// Test when ConfigMap does no exist
	t.Run("ConfigMap does not exist", func(t *testing.T) {

		// Define a fake namespace
		ns := &v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "TEST_NAMESPACE",
				Annotations: map[string]string{},
			},
		}

		// Create a fake client
		fakeClientSet := fake.NewSimpleClientset(ns)

		// Define data for configMap
		cmdata := map[string]string{
			"mapAccounts": "[]\n",
			"mapUsers":    "[]\n",
			"mapRoles":    "- rolearn: arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef\n  username: devops:{{SessionName}}\n  groups:\n    - system:masters\n",
		}

		// Update configMap which does not exist (should create new configMap)
		err := setConfigMap(fakeClientSet, "NOT_EXISTING_CONFIGMAP", ns.Name, cmdata)
		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		// Check if configMap was created
		cm, err := fakeClientSet.CoreV1().ConfigMaps(ns.Name).Get(context.TODO(), "NOT_EXISTING_CONFIGMAP", metav1.GetOptions{})
		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		// Check if configMap created has data we expect
		if !reflect.DeepEqual(cm.Data, cmdata) {
			t.Errorf("setConfigMap() created unexpected object: %+v, want %+v", cm.Data, cmdata)
		}

	})

	// Test when namespace does exist
	t.Run("ConfigMap and Namespace does not exist", func(t *testing.T) {

		var fakeClientSet *fake.Clientset = fake.NewSimpleClientset()

		cmdata := map[string]string{
			"mapAccounts": "[]\n",
			"mapUsers":    "[]\n",
			"mapRoles":    "- rolearn: arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef\n  username: devops:{{SessionName}}\n  groups:\n    - system:masters\n",
		}

		err := setConfigMap(fakeClientSet, "NOT_EXISTING_CONFIGMAP", "NOT_EXISTING_NAMESPACE", cmdata)
		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		// Check if configMap was created
		cm, err := fakeClientSet.CoreV1().ConfigMaps("NOT_EXISTING_NAMESPACE").Get(context.TODO(), "NOT_EXISTING_CONFIGMAP", metav1.GetOptions{})
		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		// Check if configMap created has data we expect
		if !reflect.DeepEqual(cm.Data, cmdata) {
			t.Errorf("setConfigMap() created unexpected object: %+v, want %+v", cm.Data, cmdata)
		}
	})

	// Test when ConfigMap does not exist
	t.Run("ConfigMap does exist", func(t *testing.T) {

		// Define a fake namespace
		ns := &v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "TEST_NAMESPACE",
				Annotations: map[string]string{},
			},
		}

		// Define a fake configMap
		cm := &v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "TEST_CONFIGMAP",
				Namespace:   ns.Name,
				Annotations: map[string]string{},
			},
			Data: map[string]string{
				"mapAccounts": "[]\n",
				"mapUsers":    "[]\n",
				"mapRoles":    "- rolearn: arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef\n  username: devops:{{SessionName}}\n  groups:\n    - system:masters\n",
			},
		}

		// Create a fake client
		fakeClientSet := fake.NewSimpleClientset(ns, cm)

		cmdata := map[string]string{
			"mapAccounts": "[]\n",
			"mapUsers":    "[]\n",
			"mapRoles":    "- rolearn: arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef\n  username: devops:{{SessionName}}\n  groups:\n    - system:masters\n",
		}

		err := setConfigMap(fakeClientSet, cm.Name, ns.Name, cmdata)
		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		// Check if configMap was created
		updatedConfigMap, err := fakeClientSet.CoreV1().ConfigMaps(ns.Name).Get(context.TODO(), cm.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Got unexpected error: %s, was expecting to get nil", err)
		}

		// Check if configMap created has data we expect
		if !reflect.DeepEqual(updatedConfigMap.Data, cmdata) {
			t.Errorf("setConfigMap() created unexpected object: %+v, want %+v", cm.Data, cmdata)
		}

	})
}

func TestTransformRoleMappings(t *testing.T) {
	// Test when IAM role does not exist for provided PermissionSet
	t.Run("IAM role does not exist for provided permission set name", func(t *testing.T) {
		mappings := []SSORoleMapping{
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
			{
				RoleARN:       "",
				PermissionSet: "sre",
				Username:      "",
				Groups:        []string{},
			},
		}

		roles := []types.Role{
			{
				RoleName: aws.String("AWSReservedSSO_devops_0123456789abcdef"),
				Path:     aws.String("/aws-reserved/sso.amazonaws.com/eu-west-1/"),
				Arn:      aws.String("arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/eu-west-1/AWSReservedSSO_devops_0123456789abcdef"),
			},
		}

		want := []SSORoleMapping{
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
		}

		got := transformRoleMappings(mappings, roles, "")

		if !reflect.DeepEqual(got, want) {
			t.Errorf("TransformRoleMappings() returned unexpected object: %+v, want %+v", got, want)
		}
	})

	// Test when []SSORoleMapping does not need translation (no permissionSet names are provided)
	t.Run("Provided input does not need transformation", func(t *testing.T) {
		mappings := []SSORoleMapping{
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_sre_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
		}

		roles := []types.Role{
			{
				RoleName: aws.String("AWSReservedSSO_devops_0123456789abcdef"),
				Path:     aws.String("/aws-reserved/sso.amazonaws.com/eu-west-1/"),
				Arn:      aws.String("arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/eu-west-1/AWSReservedSSO_devops_0123456789abcdef"),
			},
		}

		want := []SSORoleMapping{
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_sre_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
		}

		got := transformRoleMappings(mappings, roles, "")

		if !reflect.DeepEqual(got, want) {
			t.Errorf("TransformRoleMappings() returned unexpected object: %+v, want %+v", got, want)
		}
	})

	// Test when PermissionSet name was provided and corresponding AWS IAM role exists
	t.Run("Translate permissionSet name to role ARN", func(t *testing.T) {
		mappings := []SSORoleMapping{
			{
				RoleARN:       "",
				PermissionSet: "devops",
				Username:      "",
				Groups:        []string{},
			},
			{
				RoleARN:       "",
				PermissionSet: "sre",
				Username:      "",
				Groups:        []string{},
			},
		}

		roles := []types.Role{
			{
				RoleName: aws.String("AWSReservedSSO_devops_0123456789abcdef"),
				Path:     aws.String("/aws-reserved/sso.amazonaws.com/eu-west-1/"),
				Arn:      aws.String("arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/eu-west-1/AWSReservedSSO_devops_0123456789abcdef"),
			},
			{
				RoleName: aws.String("AWSReservedSSO_sre_0123456789abcdef"),
				Path:     aws.String("/aws-reserved/sso.amazonaws.com/eu-west-1/"),
				Arn:      aws.String("arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/eu-west-1/AWSReservedSSO_sre_0123456789abcdef"),
			},
		}

		want := []SSORoleMapping{
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_devops_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
			{
				RoleARN:       "arn:aws:iam::123456789012:role/AWSReservedSSO_sre_0123456789abcdef",
				PermissionSet: "",
				Username:      "",
				Groups:        []string{},
			},
		}

		got := transformRoleMappings(mappings, roles, "")

		if !reflect.DeepEqual(got, want) {
			t.Errorf("TransformRoleMappings() returned unexpected object: %+v, want %+v", got, want)
		}
	})
}
