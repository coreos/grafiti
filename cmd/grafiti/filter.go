package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	rgtaiface "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/coreos/grafiti/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ignoreFile string
	filterFile string
)

func init() {
	RootCmd.AddCommand(filterCmd)
	filterCmd.PersistentFlags().StringVarP(&ignoreFile, "ignore-file", "i", "", "File containing tags to ignore, in the format of the aws-sdk-go/service/resourcegrouptaggingapi.TagFilters struct.")
	filterCmd.PersistentFlags().StringVarP(&filterFile, "filter-file", "f", "", "File containing JSON objects of filterable resource ARN's and tag key/value pairs. Format is the output format of grafiti parse.")
}

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Filter AWS resources by tag",
	RunE:  runFilterCommand,
}

func runFilterCommand(cmd *cobra.Command, args []string) error {
	if ignoreFile == "" {
		fmt.Println("Required: --ignore-file <arg>.")
		os.Exit(1)
	}

	// Open ignoreFile
	iFile, ferr := os.OpenFile(ignoreFile, os.O_RDONLY, 0644)
	if ferr != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", ferr.Error())
		return nil
	}
	defer iFile.Close()

	svc := rgta.New(session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.region")),
		},
	)))

	if filterFile != "" {
		if ferr := filterFromFile(svc, iFile, filterFile); ferr != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", ferr.Error())
			os.Exit(1)
		}
		return nil
	}

	if serr := filterFromStdIn(svc, iFile); serr != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", serr.Error())
		os.Exit(1)
	}

	return nil
}

func filterFromFile(svc rgtaiface.ResourceGroupsTaggingAPIAPI, r io.Reader, fname string) error {
	// Open filterFile
	f, ferr := os.OpenFile(fname, os.O_RDONLY, 0644)
	if ferr != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", ferr.Error())
		return nil
	}
	defer f.Close()

	return filter(svc, r, f)
}

func filterFromStdIn(svc rgtaiface.ResourceGroupsTaggingAPIAPI, f io.Reader) error {
	return filter(svc, os.Stdin, f)
}

// Filter input by ignoreFile tags
func filter(svc rgtaiface.ResourceGroupsTaggingAPIAPI, r, f io.Reader) error {
	dec := json.NewDecoder(r)

	// Create a map that contains all ignorable ARN's
	itMap, ierr := initIgnoreTagMap(svc, f)
	if ierr != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", ierr.Error())
		return ierr
	}

	for {
		o, isEOF, err := decodeIntoOutput(dec)
		if err != nil {
			return err
		}
		if isEOF {
			break
		}
		if o == nil || o.TaggingMetadata == nil {
			continue
		}

		if _, ok := itMap[o.TaggingMetadata.ResourceARN]; !ok {
			forwardFilteredOutput(o)
		}
	}

	return nil
}

// Query relevant API's for resources with tags in the ignoreFile and return a
// map with all resource ARN's to ignore
func initIgnoreTagMap(svc rgtaiface.ResourceGroupsTaggingAPIAPI, r io.Reader) (map[arn.ResourceARN]struct{}, error) {
	// Tag file decoder
	dec := json.NewDecoder(r)
	// Collection of ARN's of resources to ignore
	arns := make(arn.ResourceARNs, 0)
	// Map of resources ARN's to ignore
	itMap := map[arn.ResourceARN]struct{}{}

	for {
		t, isEOF, err := decodeTagFileInput(dec)
		if err != nil {
			return nil, err
		}
		if isEOF {
			break
		}
		if t == nil {
			continue
		}

		arns = getARNsForResource(svc, t.TagFilters, arns)

		for rtk := range arn.RGTAUnsupportedResourceTypes {
			// Request all RGTA-unsupported resources with the same tags
			arns = getARNsForUnsupportedResource(rtk, t.TagFilters, arns)
		}
	}

	for _, arn := range arns {
		if _, ok := itMap[arn]; !ok {
			itMap[arn] = struct{}{}
		}
	}

	return itMap, nil
}

func forwardFilteredOutput(o *Output) {
	oj, err := json.Marshal(o)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
		return
	}

	fmt.Println(string(oj))
}

func decodeIntoOutput(decoder *json.Decoder) (*Output, bool, error) {
	var decoded Output
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return &decoded, true, nil
		}
		if ignoreErrors {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil, false, nil
		}
		return nil, false, err
	}
	return &decoded, false, nil
}
