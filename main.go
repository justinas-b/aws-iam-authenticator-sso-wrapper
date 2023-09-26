package main

import (
	"flag"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var logger *zap.Logger
var sourceConfigMapName string
var sourceNamespaceName string
var destinationConfigMapName string
var destinationNamespaceName string
var defaultAWSRegion string
var debug bool

func init() {

	// Parce cli arguments
	flag.StringVar(&sourceConfigMapName, "src-configmap", "aws-auth", "Name of the source Kubernetes ConfigMap to read data from and perform transformation upon")
	flag.StringVar(&sourceNamespaceName, "src-namespace", "", "Kubernetes namespace from which to read ConfigMap which containes mapRoles with permissionset names. If not defined, current namespace of pod will be used")
	flag.StringVar(&destinationConfigMapName, "dst-configmap", "aws-auth", "Name of the destination Kubernets ConfigMap which will be updated after transformation")
	flag.StringVar(&destinationNamespaceName, "dst-namespace", "kube-system", "Name of the destination Kubernetes Namespace where new ConfigMap will be updated")
	flag.StringVar(&defaultAWSRegion, "aws-region", "us-east-1", "AWS region to use when interacting with IAM service")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse() // Enable command-line parsing

	// Setup logger
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	logger = zap.Must(zap.NewProduction())
	if debug {
		logger = zap.Must(zap.NewDevelopment())
	}
}

func main() {

	defer logger.Sync() // nolint:errcheck

	logger.Info("Starting process...")

	// Get name of kubernetes namespace pod is running
	if sourceNamespaceName == "" {
		var err error
		sourceNamespaceName, err = getCurrentNamespace()
		if err != nil {
			logger.Panic("Failed to get current namespace", zap.Error(err))
		}
	}

	// Read configMap template from current namespace which will be transformed
	configMap, err := getConfigMap(sourceConfigMapName, sourceNamespaceName)
	if err != nil {
		logger.Panic(fmt.Sprintf("Failed to get configMap %s from namespace %s", sourceConfigMapName, sourceNamespaceName), zap.Error(err))
	}

	// Unmarshal RoleMappings from configMap
	// _, roleMappings, _, _ := aiacm.ParseMap(configMap.Data)
	roleMappings := []SSORoleMapping{}
	err = yaml.Unmarshal([]byte(configMap.Data["mapRoles"]), &roleMappings)
	if err != nil {
		logger.Error("Failed to unmarshal RoleMappings from configMap", zap.Error(err))
	}

	// Read all SSO roles from AWS IAM
	awsIAMRoles, err := listSSORoles()
	if err != nil {
		logger.Panic("Error occurred while retrieving SSO Roles for AWS IAM service", zap.Error(err))
	}

	// Replace PermissionSet name with Role ARN, if permission set is not found - remove it from configMap
	roleMappingsUpdated := transformRoleMappings(roleMappings, awsIAMRoles)

	// Marshal new role mappings into string format and update configMap on destination namespace
	data, err := yaml.Marshal(roleMappingsUpdated) // Marshal new role mappings into string format
	if err != nil {
		logger.Panic("Failed to marshal RoleMappings", zap.Error(err))
	}

	cmdata := configMap.Data // Read Data from existing configMap and replates "mapRoles" with new data
	cmdata["mapRoles"] = string(data)

	err = setConfigMap(destinationConfigMapName, destinationNamespaceName, cmdata) // Update configMap
	if err != nil {
		logger.Panic("Failed to set configMap", zap.Error(err))
	}
}
