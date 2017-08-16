# AWS resource deletion order

AWS resources have (potentially many) dependencies that must be explicitly detached/removed/deleted before deleting a top-level resource (ex. a VPC). Therefore a deletion order must be enforced. This order is universal for all AWS resources and is not use-case-specific, because deletion actions will only run if a resource with a specific tag, or one of it's dependencies, is detected.

## Order

The following order is not fixed and subject to change as more resource types are supported by grafiti. Sublists of resources are children that are implicitly deleted, i.e. deleted only when deleting their parent resource.

1. S3 Bucket
    1. S3 Object
1. Route53 HostedZone
    1. Route53 RecordSet
1. EC2 RouteTableAssociation
1. EC2 Instance
1. AutoScaling Group
1. AutoScaling LaunchConfiguration
1. ElasticLoadBalancer
1. EC2 NAT Gateway
1. ElasticIPAssociation
1. ElasticIP (Allocation)
1. IAM InstanceProfile
    1. IAM Role Association
1. IAM Role
1. IAM User
1. EC2 InternetGateway
    1. EC2 InternetGatewayAttachment
1. EC2 NetworkInterface
1. EC2 NetworkACL
    1. EC2 NetworkACL Entry
1. EC2 VPN Connection
    1. EC2 VPN Connection Route
1. EC2 CustomerGateway
1. EBS Volume
1. EC2 Subnet
1. EC2 RouteTable
    1. EC2 RouteTable Route
1. EC2 SecurityGroup
    1. EC2 SecurityGroup Ingress Rule
    1. EC2 SecurityGroup Egress Rule
1. EC2 VPN Gateway
    1. EC2 VPN Gateway Attachment
1. EC2 VPC
    1. EC2 VPC CIDRBlock
