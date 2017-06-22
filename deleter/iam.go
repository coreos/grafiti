package deleter

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/coreos/grafiti/arn"
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
	if len(iprs) == 0 {
		return nil
	}

	// Delete roles from instance profiles
	if err := rd.deleteIAMRolesFromInstanceProfiles(cfg, iprs); err != nil {
		return err
	}

	fmtStr := "Deleted IAM InstanceProfile"
	if cfg.DryRun {
		for _, n := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}

	var params *iam.DeleteInstanceProfileInput
	for _, ipr := range iprs {
		params = &iam.DeleteInstanceProfileInput{
			InstanceProfileName: ipr.InstanceProfileName,
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteInstanceProfileWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.IAMInstanceProfileRType, arn.ResourceName(*ipr.InstanceProfileName), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *ipr.InstanceProfileName)
	}

	return nil
}

func (rd *IAMInstanceProfileDeleter) deleteIAMRolesFromInstanceProfiles(cfg *DeleteConfig, iprs []*iam.InstanceProfile) error {
	if len(iprs) == 0 {
		return nil
	}

	if cfg.DryRun {
		for _, ipr := range iprs {
			fmt.Println(drStr, "Removed Role from IAM InstanceProfile", *ipr.InstanceProfileName)
		}
		return nil
	}

	var params *iam.RemoveRoleFromInstanceProfileInput
	for _, ipr := range iprs {
		for _, rl := range ipr.Roles {
			params = &iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: ipr.InstanceProfileName,
				RoleName:            rl.RoleName,
			}

			// Prevent throttling
			time.Sleep(cfg.BackoffTime)

			ctx := aws.BackgroundContext()
			_, err := rd.GetClient().RemoveRoleFromInstanceProfileWithContext(ctx, params)
			if err != nil {
				cfg.logDeleteError(arn.IAMRoleRType, arn.ResourceName(*rl.RoleName), err, logrus.Fields{
					"parent_resource_type": arn.IAMInstanceProfileRType,
					"parent_resource_name": *ipr.InstanceProfileName,
				})
				if cfg.IgnoreErrors {
					continue
				}
				return err
			}

			fmt.Printf("Removed Role %s from IAM InstanceProfile %s\n", *ipr.InstanceProfileName, *rl.RoleName)
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
	want := map[arn.ResourceName]struct{}{}
	for _, n := range rd.ResourceNames {
		if _, ok := want[n]; !ok {
			want[n] = struct{}{}
		}
	}

	params := &iam.ListInstanceProfilesInput{
		MaxItems: aws.Int64(100),
	}

	iprs := make([]*iam.InstanceProfile, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListInstanceProfilesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, rp := range resp.InstanceProfiles {
			if _, ok := want[arn.ResourceName(*rp.InstanceProfileName)]; ok {
				iprs = append(iprs, rp)
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.Marker = resp.Marker

	}
	return iprs, nil
}

// RequestIAMInstanceProfilesFromLaunchConfigurations retrieves instance profiles from
// launch configuration names
func (rd *IAMInstanceProfileDeleter) RequestIAMInstanceProfilesFromLaunchConfigurations(lcs []*autoscaling.LaunchConfiguration) ([]*iam.InstanceProfile, error) {
	if len(lcs) == 0 {
		return nil, nil
	}

	// We cannot request instance profiles by their ID's so we must search
	// iteratively with a map
	want := map[string]struct{}{}
	var iprName string
	for _, lc := range lcs {
		if lc.IamInstanceProfile == nil {
			continue
		}

		// The docs say that IAMInstanceProfile can be either an ARN or name; if an
		// ARN, parse out name
		iprName = *lc.IamInstanceProfile
		if strings.HasPrefix(*lc.IamInstanceProfile, "arn:") {
			iprSplit := strings.Split(*lc.IamInstanceProfile, "instance-profile/")
			if len(iprSplit) != 2 || iprSplit[1] == "" {
				continue
			}
			iprName = iprSplit[1]
		}
		if _, ok := want[iprName]; !ok {
			want[iprName] = struct{}{}
		}
	}

	iprs := make([]*iam.InstanceProfile, 0)
	params := &iam.ListInstanceProfilesInput{}

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListInstanceProfilesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, ipr := range resp.InstanceProfiles {
			if _, ok := want[*ipr.InstanceProfileName]; ok {
				iprs = append(iprs, ipr)
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.Marker = resp.Marker
	}

	return iprs, nil
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

	fmtStr := "Deleted IAM Role"
	if cfg.DryRun {
		for _, n := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}

	var (
		params *iam.DeleteRoleInput
		rpd    *IAMRolePolicyDeleter
	)
	for _, n := range rd.ResourceNames {
		// Delete role policies
		rpd = &IAMRolePolicyDeleter{RoleName: n}
		pls, rerr := rpd.RequestIAMRolePoliciesFromRoles()
		if rerr != nil || len(pls) == 0 {
			continue
		}
		rpd.PolicyNames = pls

		if err := rpd.DeleteResources(cfg); err != nil {
			continue
		}

		// Delete roles
		params = &iam.DeleteRoleInput{
			RoleName: n.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRoleWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.IAMRoleRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestIAMRoles requests IAM roles by name from the AWS API and returns IAM roles
func (rd *IAMRoleDeleter) RequestIAMRoles() ([]*iam.Role, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	// No filtering fields in ListRolesInput, must be done iteratively
	want := map[arn.ResourceName]struct{}{}
	for _, n := range rd.ResourceNames {
		if _, ok := want[n]; !ok {
			want[n] = struct{}{}
		}
	}

	var params *iam.ListRolesInput
	rls := make([]*iam.Role, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListRolesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, rl := range resp.Roles {
			if _, ok := want[arn.ResourceName(*rl.RoleName)]; ok {
				rls = append(rls, rl)
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
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
	if cfg.DryRun {
		for _, n := range rd.PolicyNames {
			fmt.Printf("%s %s %s for IAM Role %s\n", drStr, fmtStr, n, rd.RoleName)
		}
		return nil
	}

	var params *iam.DeleteRolePolicyInput
	for _, pn := range rd.PolicyNames {
		params = &iam.DeleteRolePolicyInput{
			RoleName:   rd.RoleName.AWSString(),
			PolicyName: pn.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRolePolicyWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.IAMPolicyRType, pn, err, logrus.Fields{
				"parent_resource_type": arn.IAMRoleRType,
				"parent_resource_name": rd.RoleName,
			})
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, pn)
	}

	return nil
}

// RequestIAMRolePoliciesFromRoles requests IAM role policies by role name from the AWS API and
// returns policy names
func (rd *IAMRolePolicyDeleter) RequestIAMRolePoliciesFromRoles() (arn.ResourceNames, error) {
	if rd.RoleName == "" {
		return nil, nil
	}

	params := &iam.ListRolePoliciesInput{
		MaxItems: aws.Int64(100),
		RoleName: rd.RoleName.AWSString(),
	}
	pls := make(arn.ResourceNames, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, lerr := rd.GetClient().ListRolePoliciesWithContext(ctx, params)
		if lerr != nil {
			return nil, lerr
		}

		for _, rp := range resp.PolicyNames {
			pls = append(pls, arn.ResourceName(*rp))
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.Marker = resp.Marker
	}

	return pls, nil
}
