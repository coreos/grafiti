package deleter

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/coreos/grafiti/arn"
	"github.com/sirupsen/logrus"
)

// IAMInstanceProfileDeleter represents an AWS instance profile
type IAMInstanceProfileDeleter struct {
	Client        iamiface.IAMAPI
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *IAMInstanceProfileDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *IAMInstanceProfileDeleter) GetClient() iamiface.IAMAPI {
	if rd.Client == nil {
		rd.Client = iam.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds instance profile names to Names
func (rd *IAMInstanceProfileDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes an instance profiles from AWS
// NOTE: must delete roles from instance profile before deleting roles. Must
// be done in this step because of only profiles contain role info, not visa versa.
func (rd *IAMInstanceProfileDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	iprs, err := rd.RequestIAMInstanceProfiles()
	if err != nil && !cfg.IgnoreErrors {
		return err
	}

	// Delete roles from instance profiles
	if err := rd.deleteIAMRolesFromInstanceProfiles(cfg, iprs); err != nil {
		return err
	}

	fmtStr := "Deleted IAM InstanceProfile"

	var params *iam.DeleteInstanceProfileInput
	for _, ipr := range iprs {
		nameStr := aws.StringValue(ipr.InstanceProfileName)

		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, nameStr)
			continue
		}

		params = &iam.DeleteInstanceProfileInput{
			InstanceProfileName: ipr.InstanceProfileName,
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteInstanceProfileWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.IAMInstanceProfileRType, nameStr, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.IAMInstanceProfileRType, nameStr)
		fmt.Println(fmtStr, nameStr)
	}

	return nil
}

func (rd *IAMInstanceProfileDeleter) deleteIAMRolesFromInstanceProfiles(cfg *DeleteConfig, iprs []*iam.InstanceProfile) error {
	if len(iprs) == 0 {
		return nil
	}

	var params *iam.RemoveRoleFromInstanceProfileInput
	for _, ipr := range iprs {
		iprNameStr := aws.StringValue(ipr.InstanceProfileName)

		for _, rl := range ipr.Roles {
			roleNameStr := aws.StringValue(rl.RoleName)

			if cfg.DryRun {
				fmt.Printf("%s Removed Role %s from IAM InstanceProfile %s\n", drStr, roleNameStr, iprNameStr)
				continue
			}

			params = &iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: ipr.InstanceProfileName,
				RoleName:            rl.RoleName,
			}

			ctx := aws.BackgroundContext()
			_, err := rd.GetClient().RemoveRoleFromInstanceProfileWithContext(ctx, params)
			if err != nil {
				cfg.logRequestError(arn.IAMRoleRType, roleNameStr, err, logrus.Fields{
					"parent_resource_type": arn.IAMInstanceProfileRType,
					"parent_resource_name": iprNameStr,
				})
				if cfg.IgnoreErrors {
					continue
				}
				return err
			}

			cfg.logRequestSuccess(arn.IAMRoleRType, roleNameStr, logrus.Fields{
				"parent_resource_type": arn.IAMInstanceProfileRType,
				"parent_resource_name": iprNameStr,
			})
			fmt.Printf("Removed Role %s from IAM InstanceProfile %s\n", iprNameStr, roleNameStr)
		}
	}

	return nil
}

// RequestIAMInstanceProfiles requests IAM instance profiles by name from the
// AWS API and IAM instance profiles
func (rd *IAMInstanceProfileDeleter) RequestIAMInstanceProfiles() ([]*iam.InstanceProfile, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	// We cannot request a filtered list of instance profiles, so we must
	// iterate through all returned profiles and select the ones we want.
	want, iprs := createResourceNameMapFromResourceNames(rd.ResourceNames), make([]*iam.InstanceProfile, 0)
	params := &iam.ListInstanceProfilesInput{
		MaxItems: aws.Int64(100),
	}
	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListInstanceProfilesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return iprs, err
		}

		for _, rp := range resp.InstanceProfiles {
			if _, ok := want[arn.ToResourceName(rp.InstanceProfileName)]; ok {
				iprs = append(iprs, rp)
			}
		}

		if !aws.BoolValue(resp.IsTruncated) {
			break
		}

		params.Marker = resp.Marker

	}
	return iprs, nil
}

func createResourceNameMapFromResourceNames(names arn.ResourceNames) map[arn.ResourceName]struct{} {
	want := map[arn.ResourceName]struct{}{}
	for _, n := range names {
		if _, ok := want[n]; !ok {
			want[n] = struct{}{}
		}
	}
	return want
}

// IAMRoleDeleter represents an AWS IAM role
type IAMRoleDeleter struct {
	Client        iamiface.IAMAPI
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *IAMRoleDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *IAMRoleDeleter) GetClient() iamiface.IAMAPI {
	if rd.Client == nil {
		rd.Client = iam.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds IAM role names to Names
func (rd *IAMRoleDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes IAM roles from AWS
func (rd *IAMRoleDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	rls, rerr := rd.RequestIAMRoles()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted IAM Role"

	var (
		params *iam.DeleteRoleInput
		rpd    *IAMRolePolicyDeleter
	)
	for _, rl := range rls {
		nameStr := aws.StringValue(rl.RoleName)

		// Delete role policies
		rpd = &IAMRolePolicyDeleter{RoleName: arn.ResourceName(nameStr)}
		policyNames, rerr := rpd.RequestIAMRolePoliciesFromRoles()
		if rerr != nil && !cfg.IgnoreErrors {
			continue
		}
		rpd.PolicyNames = policyNames

		if err := rpd.DeleteResources(cfg); err != nil {
			continue
		}

		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, nameStr)
			continue
		}

		// Delete roles
		params = &iam.DeleteRoleInput{
			RoleName: rl.RoleName,
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRoleWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.IAMRoleRType, nameStr, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.IAMRoleRType, nameStr)
		fmt.Println(fmtStr, nameStr)
	}

	return nil
}

// RequestIAMRoles requests IAM roles by name from the AWS API and returns IAM roles
func (rd *IAMRoleDeleter) RequestIAMRoles() ([]*iam.Role, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	// No filtering fields in ListRolesInput, must be done iteratively
	want, rls := createResourceNameMapFromResourceNames(rd.ResourceNames), make([]*iam.Role, 0)
	params := new(iam.ListRolesInput)
	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListRolesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return rls, err
		}

		for _, rl := range resp.Roles {
			if _, ok := want[arn.ToResourceName(rl.RoleName)]; ok {
				rls = append(rls, rl)
			}
		}

		if !aws.BoolValue(resp.IsTruncated) {
			break
		}

		params.Marker = resp.Marker
	}

	return rls, nil
}

// IAMRolePolicyDeleter represents an AWS IAM role policy
type IAMRolePolicyDeleter struct {
	Client       iamiface.IAMAPI
	ResourceType arn.ResourceType
	RoleName     arn.ResourceName
	PolicyNames  arn.ResourceNames
}

func (rd *IAMRolePolicyDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "RoleName": %s, "PolicyNames": %v}`, rd.ResourceType, rd.RoleName, rd.PolicyNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *IAMRolePolicyDeleter) GetClient() iamiface.IAMAPI {
	if rd.Client == nil {
		rd.Client = iam.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds IAM role policy names to Names
func (rd *IAMRolePolicyDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.PolicyNames = append(rd.PolicyNames, ns...)
}

// DeleteResources deletes IAM role policies from AWS by role name
func (rd *IAMRolePolicyDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.PolicyNames) == 0 {
		return nil
	}

	fmtStr := "Deleted IAM RolePolicy"

	var params *iam.DeleteRolePolicyInput
	for _, pn := range rd.PolicyNames {
		if cfg.DryRun {
			fmt.Printf("%s %s %s from IAM Role %s\n", drStr, fmtStr, pn, rd.RoleName)
			continue
		}

		params = &iam.DeleteRolePolicyInput{
			RoleName:   rd.RoleName.AWSString(),
			PolicyName: pn.AWSString(),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRolePolicyWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.IAMPolicyRType, pn, err, logrus.Fields{
				"parent_resource_type": arn.IAMRoleRType,
				"parent_resource_name": rd.RoleName,
			})
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.IAMPolicyRType, pn, logrus.Fields{
			"parent_resource_type": arn.IAMRoleRType,
			"parent_resource_name": rd.RoleName,
		})
		fmt.Printf("%s %s from IAM Role %s\n", fmtStr, pn, rd.RoleName)
	}

	return nil
}

// RequestIAMRolePoliciesFromRoles requests IAM role policies by role name from the AWS API and
// returns policy names
func (rd *IAMRolePolicyDeleter) RequestIAMRolePoliciesFromRoles() (arn.ResourceNames, error) {
	if rd.RoleName == "" {
		return nil, nil
	}

	policyNames := make(arn.ResourceNames, 0)
	params := &iam.ListRolePoliciesInput{
		MaxItems: aws.Int64(100),
		RoleName: rd.RoleName.AWSString(),
	}

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListRolePoliciesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return policyNames, err
		}

		for _, policyName := range resp.PolicyNames {
			policyNames = append(policyNames, arn.ToResourceName(policyName))
		}

		if !aws.BoolValue(resp.IsTruncated) {
			break
		}

		params.Marker = resp.Marker
	}

	return policyNames, nil
}
