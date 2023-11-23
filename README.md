# aws-iam-authenticator-sso-wrapper

[![CI](https://github.com/justinas-b/aws-iam-authenticator-sso-wrapper/actions/workflows/app-ci.yaml/badge.svg?branch=main)](https://github.com/justinas-b/aws-iam-authenticator-sso-wrapper/actions/workflows/app-ci.yaml?event=schedule)
![CodeQL](https://github.com/justinas-b/aws-iam-authenticator-sso-wrapper/workflows/CodeQL/badge.svg)
[![codecov](https://codecov.io/gh/justinas-b/aws-iam-authenticator-sso-wrapper/branch/main/graph/badge.svg)](https://codecov.io/gh/justinas-b/aws-iam-authenticator-sso-wrapper)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/justinas-b/aws-iam-authenticator-sso-wrapper?sort=semver)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/justinas-b/aws-iam-authenticator-sso-wrapper)
![GitHub](https://img.shields.io/github/license/justinas-b/aws-iam-authenticator-sso-wrapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/justinas-b/aws-iam-authenticator-sso-wrapper)](https://goreportcard.com/report/github.com/justinas-b/aws-iam-authenticator-sso-wrapper)
![GitHub last commit (branch)](https://img.shields.io/github/last-commit/justinas-b/aws-iam-authenticator-sso-wrapper/main)
![Docker Image Version (latest semver)](https://img.shields.io/docker/v/justinasb/aws-iam-authenticator-sso-wrapper?logo=docker)
![Docker Image Size (tag)](https://img.shields.io/docker/image-size/justinasb/aws-iam-authenticator-sso-wrapper/latest?logo=docker)

## Purpose

This tool addressess an issue when you use AWS SSO (AWS IAM Identity Center) roles to authenticate against your AWS EKS clusters. AWS natively supports authentication to AWS EKS when using [AWS IAM Roles](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html), however you need to provide a role ARN and there is no way to provide AWS SSO PermissionSet name.

IAM roles that are created from PermissionSets contain random suffixes, that can change whenever you would update PermissionSet's configuration locking you out from access to EKS. This becomes especially a headache when you have multiple EKS clusters that are spread across multiple AWS accounts.

By default, on every EKS cluster you would have to provide `aws-auth` ConfigMap in `kube-system` namespace with corresponding role's ARN (excluding role's path):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapAccounts: |
    []
  mapRoles: |
    - "groups":
      - "system:masters"
        "rolearn": "arn:aws:iam::000000000000:role/AWSReservedSSO_AdminRole_0123456789abcdef"
        "username": "AdminRole:{{SessionName}}"
  mapUsers: |
    []
```

While using this tool, it enables you to deploy `aws-auth` ConfigMap to tool's namespace and provide PermissionSet's name instead of role ARN under `mapRoles` key:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: aws-iam-authenticator-sso-wrapper
data:
  mapAccounts: |
    []
  mapRoles: |
    - "groups":
      - "system:masters"
        "permissionset": AdminRole"
        "username": "AdminRole:{{SessionName}}"
  mapUsers: |
    []
```

The Tool also allows a placeholder `$ACCOUNTID` in the role ARN, to give a generic role. This is for a specific use case where multiple clusters in multiple accounts use a role of the same name, but the ARN changes because of account ID.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: aws-iam-authenticator-sso-wrapper
data:
  mapAccounts: |
    []
  mapRoles: |
    - "groups":
      - "system:masters"
        "rolearn": "arn:aws:iam::$ACCOUNTID:role/generic-role"
        "username": "AdminRole:{{SessionName}}"
  mapUsers: |
    []
```
The above config map retrieves the AWS account ID and replaces `$ACCOUNTID`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapAccounts: |
    []
  mapRoles: |
    - "groups":
      - "system:masters"
        "rolearn": "arn:aws:iam::123456789:role/generic-role"
        "username": "AdminRole:{{SessionName}}"
  mapUsers: |
    []
```

To make sure EKS cluster may continue to function, if below map is not provided explicitly for kubernetes worker nodes,
tool will read the ARN of IAM Role which is used by worker nodes from Instance Metadata Service (IMDS) and will inject
it automatically:
```yaml
- groups:
  - system:bootstrappers
  - system:nodes
  rolearn: arn:aws:iam::111122223333:role/my-node-role
  username: system:node:{{EC2PrivateDNSName}}
```

The tool will process `aws-auth` ConfigMap from it's local kubernetes namespace and transform it to the format AWS EKS cluster expects. After processing ConfigMap, it's output is saved `kube-system` namespace where PermissionSet's name is translated to corresponding role ARN, meaning `"permissionset": AdminRole"` line will become `"rolearn": "arn:aws:iam::000000000000:role/AWSReservedSSO_AdminRole_0123456789abcdef"`

More details on this problem can found on below issues:

- <https://github.com/aws/containers-roadmap/issues/185>
- <https://github.com/aws/containers-roadmap/issues/474>
- <https://github.com/aws/containers-roadmap/issues/1837>
- <https://github.com/kubernetes-sigs/aws-iam-authenticator/pull/416>
- <https://github.com/kubernetes-sigs/aws-iam-authenticator/issues/333>

## Usage

```text
❯ aws-iam-authenticator-sso-wrapper -h
Usage of aws-iam-authenticator-sso-wrapper:
  -aws-region string
        AWS region to use when interacting with IAM service (default "us-east-1")
  -debug
        Enable debug logging
  -dst-configmap string
        Name of the destination Kubernets ConfigMap which will be updated after transformation (default "aws-auth")
  -dst-namespace string
        Name of the destination Kubernetes Namespace where new ConfigMap will be updated (default "kube-system")
  -interval int
        Interval in seconds on which application will check for updates (default 1800)
  -src-configmap string
        Name of the source Kubernetes ConfigMap to read data from and perform transformation upon (default "aws-auth")
  -src-namespace string
        Kubernetes namespace from which to read ConfigMap which containes mapRoles with permissionset names. If not defined, current namespace of pod will be used
```

## Deployment

Docker image can be obtained from [justinasb/aws-iam-authenticator-sso-wrapper](https://hub.docker.com/r/justinasb/aws-iam-authenticator-sso-wrapper). As this application needs to list AWS IAM Roles, it needs to authenticate against AWS. To do so, you need to create new IAM role with below privileges:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
              "iam:ListRoles",
              "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
```

For the role trust policy, please enable your AWS EKS cluster to use that role as described in [AWS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) document. Your trust policy should look something like:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::[AWS-ACCOUNT-ID]:oidc-provider/oidc.eks.[EKS-CLUSTER-REGION].amazonaws.com/id/[EKS-CLUSTER-ID]"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "oidc.eks.[EKS-CLUSTER-REGION].amazonaws.com/id/[EKS-CLUSTER-ID]:sub": "system:serviceaccount:aws-iam-authenticator-sso-wrapper:aws-iam-authenticator-sso-wrapper"
                }
            }
        }
    ]
}
```

Once you have created new role, dont forget to set `serviceaccount.annotations.eks.amazonaws.com/role-arn` value on Helm chart to actual role ARN.

### Helm chart

Add helm repo locally:

```txt
> helm repo add aws-iam-authenticator-sso-wrapper https://justinas-b.github.io/aws-iam-authenticator-sso-wrapper/

"aws-iam-authenticator-sso-wrapper" has been added to your repositories
```

Update helm repo:

```txt
> helm repo update

Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "aws-iam-authenticator-sso-wrapper" chart repository
```

Deploy application:

```txt
> helm install \
  aws-iam-authenticator-sso-wrapper justinas-b/aws-iam-authenticator-sso-wrapper \
  --namespace aws-iam-authenticator-sso-wrapper \
  --create-namespace \
  --set serviceaccount.annotations."eks\.amazonaws\.com/role-arn"=arn:aws:iam::123456789012:role/my-iam-role

NAME: aws-iam-authenticator-sso-wrapper
LAST DEPLOYED: Mon Oct  2 15:23:36 2023
NAMESPACE: aws-iam-authenticator-sso-wrapper
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

### Authentication

For this tool to be able to authenticate with AWS (required when translating PermissionSet name to role ARN) it is recommended to use [AWS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html), however any authentication methos it supported (you can also add `~/.aws/config` or `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.
