package main

// SSORoleMapping struct defines a roleMapping used in aws-auth configMap
type SSORoleMapping struct {
	// RoleARN is the AWS Resource Name of the role. (e.g., "arn:aws:iam::000000000000:role/Foo").
	RoleARN string `json:"rolearn,omitempty" yaml:"rolearn,omitempty"`

	// RoleARN is the AWS Resource Name of the role. (e.g., "arn:aws:iam::000000000000:role/Foo").
	PermissionSet string `json:"permissionSet,omitempty" yaml:"permissionSet,omitempty"`

	// Username is the username pattern that this instances assuming this
	// role will have in Kubernetes.
	Username string `json:"username"`

	// Groups is a list of Kubernetes groups this role will authenticate
	// as (e.g., `system:masters`). Each group name can include placeholders.
	Groups []string `json:"groups" yaml:"groups"`

	// UserID is the AWS PrincipalId of the role. (e.g., "ABCXSOTJDDV").
	UserID string `json:"userid,omitempty" yaml:"userid,omitempty"`
}
