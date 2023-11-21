package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// getKubernetesClientSet returns a Kubernetes clientset and an error.
//
// This function initializes a Kubernetes in-cluster clientset. If the initialization fails,
// it falls back to a kubeconfig clientset. It returns the initialized clientset or an error.
//
// Return:
// - *kubernetes.Clientset: The initialized Kubernetes clientset.
// - error: An error if the initialization fails.
func getKubernetesClientSet() (*kubernetes.Clientset, error) {
	logger.Debug("Initialising Kubernetes in-cluster clientset")

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Debug("Failed to initialise in-cluster clientset, failing back to kubeconfig clientset")
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		logger.Debug("Successfully initialised kubeconfig clientset")
	} else {
		logger.Debug("Successfully initialised in-cluster clientset")
	}

	return kubernetes.NewForConfig(config)
}

// getCurrentNamespace returns the current namespace.
//
// It checks if the LOCAL_NAMESPACE environment variable is defined and uses it as the namespace name.
// If the environment variable is not defined, it gets the namespace from Kubernetes.
//
// Returns the current namespace as a string and any error encountered.
func getCurrentNamespace() (string, error) {
	logger.Info("Getting current namespace")

	// If LOCAL_NAMESPACE environment variable is not defined, get namespace from Kubernetes
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}

	logger.Debug(fmt.Sprintf("Current namespace: %s", string(namespace)))
	return string(namespace), nil
}

// getConfigMap retrieves a ConfigMap from a Kubernetes cluster.
//
// Parameters:
// - configMapName: the name of the ConfigMap to retrieve.
// - namespaceName: the name of the namespace where the ConfigMap is located.
//
// Returns:
// - *v1.ConfigMap: the retrieved ConfigMap.
// - error: an error if the retrieval fails.
func getConfigMap(clientset kubernetes.Interface, configMapName string, namespaceName string) (*v1.ConfigMap, error) {

	logger.Info(fmt.Sprintf("Retrieving ConfigMap %s from namespace %s", configMapName, namespaceName))

	configMap, err := clientset.CoreV1().ConfigMaps(namespaceName).Get(context.TODO(), configMapName, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		logger.Error(fmt.Sprintf("ConfigMap %s not found in namespace %s", configMapName, namespaceName), zap.Error(err))
		return nil, err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		logger.Error(fmt.Sprintf("Error getting %s config-map from namespace %s. %s", configMapName, namespaceName, statusError.ErrStatus.Message), zap.Error(err))
		return nil, err
	} else if err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("Successfully retrieved ConfigMap %s from namespace %s", configMapName, namespaceName))

	return configMap, nil
}

// setConfigMap creates or updates a ConfigMap in a Kubernetes cluster.
//
// Parameters:
//   - configMapName: The name of the ConfigMap.
//   - namespaceName: The namespace of the ConfigMap.
//   - data: The data to be stored in the ConfigMap.
//
// Returns:
//   - error: An error if the creation or update fails.
func setConfigMap(clientset kubernetes.Interface, configMapName string, namespaceName string, data map[string]string) error {

	logger.Info(fmt.Sprintf("Setting ConfigMap %s in namespace %s", configMapName, namespaceName))

	// Define ConfigMap's metadata
	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespaceName,
		},
		Data: data,
	}

	// Check if configMap already exists and if not, create it
	if _, err := clientset.CoreV1().ConfigMaps(namespaceName).Get(context.TODO(), configMapName, metav1.GetOptions{}); errors.IsNotFound(err) {
		_, err = clientset.CoreV1().ConfigMaps(namespaceName).Create(context.TODO(), &cm, metav1.CreateOptions{})
		if err != nil {
			return err
		}

	} else { // Otherwise update existing configMap
		_, err = clientset.CoreV1().ConfigMaps(namespaceName).Update(context.TODO(), &cm, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	logger.Info(fmt.Sprintf("Successfully set ConfigMap %s in namespace %s", configMapName, namespaceName))
	return nil
}

// transformRoleMappings replaces PermissionSet name with Role ARN in RoleMappings.
//
// It takes the following parameters:
// - roleMappings: a slice of SSORoleMapping structs
// - awsIAMRoles: a slice of types.Role structs
//
// It returns a slice of SSORoleMapping structs, where the PermissionSet name is replaced with Role ARN.
func transformRoleMappings(roleMappings []SSORoleMapping, awsIAMRoles []types.Role, accountId string) []SSORoleMapping {
	// Replace PermissionSet name with Role ARN, if permission
	// set is not found - remove it from configMap

	logger.Info("Translating permissionSets to RoleARNs in RoleMappings...")

	var roleMappingsUpdated []SSORoleMapping

	for _, roleMapping := range roleMappings {

		// Check if Role Mapping needs translation. If not,
		// skip this itteration and add object to updated list
		if (roleMapping.PermissionSet == "") || (roleMapping.RoleARN != "") {
			//Check if rolemapping requires fetching the accountid of the aws account
			if(strings.Contains(roleMapping.RoleARN, "$ACCOUNTID")) {
				logger.Info("Replacing $ACCOUNTID with Actual account ID")
				roleMapping.RoleARN = strings.Replace(roleMapping.RoleARN, "$ACCOUNTID", accountId, -1)
				roleMappingsUpdated = append(roleMappingsUpdated, roleMapping)
				continue
			} else {	
				logger.Debug("Role Mapping does not need to be translated", zap.Any("roleMapping", roleMapping))
				roleMappingsUpdated = append(roleMappingsUpdated, roleMapping)
				continue
			}
		}

		// Translate permission set name to ARN
		role, err := translatePermissionSetNameToARN(roleMapping, awsIAMRoles)
		if err != nil {
			logger.Warn(fmt.Sprintf("Role that would correspond to %s permission set not found. Removing mapping from the list", roleMapping.PermissionSet), zap.Error(err))
			continue
		}

		logger.Debug("Role Mapping successfully translated", zap.Any("roleMapping", roleMapping))
		roleMappingsUpdated = append(roleMappingsUpdated, role)

	}
	logger.Info("Translation finished successfully")
	return roleMappingsUpdated
}
