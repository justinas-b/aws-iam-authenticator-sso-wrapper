package main

import (
  "flag"
  "fmt"
  "gopkg.in/yaml.v2"
  "os"
  "os/signal"
  "time"

  "go.uber.org/zap"
)

var (
	logger                   *zap.Logger
	sourceConfigMapName      string
	sourceNamespaceName      string
	destinationConfigMapName string
	destinationNamespaceName string
	defaultAWSRegion         string
	debug                    bool
	interval                 int
)

// init is a special function in Go that is automatically called before the main function.
func init() {
}

// main is the entry point of the program.
//
// It initializes a scheduler to periodically execute the updateRoleMappings function.
// The scheduler runs every interval seconds.
//
// No parameters are required.
// No return types.
func main() {
	parseCliArgs()
	setupLogger(debug)
	done := scheduler(updateRoleMappings, time.Duration(interval)*time.Second)
	defer close(done)
}

// parseCliArgs parses the command-line arguments and sets the corresponding variables.
//
// No parameters.
// No return type.
func parseCliArgs() {
	// Parse cli arguments
	flag.StringVar(&sourceConfigMapName, "src-configmap", "aws-auth", "Name of the source Kubernetes ConfigMap to read data from and perform transformation upon")
	flag.StringVar(&sourceNamespaceName, "src-namespace", "", "Kubernetes namespace from which to read ConfigMap which contains mapRoles with permissionset names. If not defined, current namespace of pod will be used")
	flag.StringVar(&destinationConfigMapName, "dst-configmap", "aws-auth", "Name of the destination Kubernetes ConfigMap which will be updated after transformation")
	flag.StringVar(&destinationNamespaceName, "dst-namespace", "kube-system", "Name of the destination Kubernetes Namespace where new ConfigMap will be updated")
	flag.StringVar(&defaultAWSRegion, "aws-region", "us-east-1", "AWS region to use when interacting with IAM service")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.IntVar(&interval, "interval", 1800, "Interval in seconds on which application will check for updates")
	flag.Parse() // Enable command-line parsing
}

// setupLogger sets up the logger based on the debug flag.
//
// It takes a boolean parameter, debug, which specifies whether or not to enable debug logs.
// There is no return value.
func setupLogger(debug bool) {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	logger = zap.Must(zap.NewProduction())
	if debug {
		logger = zap.Must(zap.NewDevelopment())
	}
	defer logger.Sync() // nolint:errcheck
}

// scheduler schedules the execution of a given function at a specified time interval.
//
// Parameters:
// - f: The function to be executed.
// - timeInterval: The time interval between function executions.
//
// Returns:
// - done: A channel that can be used to signal the completion of the scheduler.
func scheduler(f func(), timeInterval time.Duration) chan bool {

	logger.Info(fmt.Sprintf("Starting scheduler to run every %d seconds", timeInterval))

	tick := time.NewTicker(timeInterval)
	defer tick.Stop()

	done := make(chan bool)
	sigs := make(chan os.Signal, 1)

	// signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		defer logger.Info("Quitting application due to SIGTERM/SIGINT signal")
		for {
			f()
			select {
			case <-tick.C:
				continue
			case <-done:
				return
			}
		}
	}()

	<-sigs
	done <- true

	return done
}

// updateRoleMappings updates the role mappings in the configMap.
//
// This function retrieves the current namespace where the pod is running and
// reads the configMap template from that namespace. It then unmarshal the
// RoleMappings from the configMap and reads all the SSO roles from AWS IAM.
// The function replaces the PermissionSet name with the Role ARN and removes
// the permission set from the configMap if it is not found. It then marshals
// the new role mappings into a string format and updates the configMap in the
// destination namespace.
func updateRoleMappings() {

	logger.Info("Starting process...")

	// Creates Kubernetes clientset to authenticate and interact with API
	clientset, err := getKubernetesClientSet()
	if err != nil {
		logger.Panic("Failed to create Kubernetes clientset", zap.Error(err))
	}

	// Get name of kubernetes namespace pod is running
	if sourceNamespaceName == "" {
		var err error
		sourceNamespaceName, err = getCurrentNamespace()
		if err != nil {
			logger.Panic("Failed to get current namespace", zap.Error(err))
		}
	}

	// Read configMap template from current namespace which will be transformed
	configMap, err := getConfigMap(clientset, sourceConfigMapName, sourceNamespaceName)
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

	accountId, err := getAccountId()
	if err != nil {
		logger.Panic("Failed to read AWS Account ID", zap.Error(err))
	}

	// Replace PermissionSet name with Role ARN, if permission set is not found - remove it from configMap
	roleMappingsUpdated := transformRoleMappings(roleMappings, awsIAMRoles, accountId)

	// Marshal new role mappings into string format and update configMap on destination namespace
	data, err := yaml.Marshal(roleMappingsUpdated) // Marshal new role mappings into string format
	if err != nil {
		logger.Panic("Failed to marshal RoleMappings", zap.Error(err))
	}

	cmdata := configMap.Data // Read Data from existing configMap and replaces "mapRoles" with new data
	cmdata["mapRoles"] = string(data)

	err = setConfigMap(clientset, destinationConfigMapName, destinationNamespaceName, cmdata) // Update configMap
	if err != nil {
		logger.Panic("Failed to set configMap", zap.Error(err))
	}

	logger.Info("Finished processing configMaps")
}
