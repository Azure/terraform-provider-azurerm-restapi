package services_test

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Azure/terraform-provider-azapi/internal/acceptance"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/zclconf/go-cty/cty"
)

type ExampleTestcase struct {
	Path              string
	Config            string
	ExternalProviders map[string]resource.ExternalProvider
}

func knownExternalProvidersAzurerm() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"azurerm": {
			VersionConstraint: "4.20.0",
			Source:            "hashicorp/azurerm",
		},
	}
}

// TestAccExamples_Selected runs acceptance tests for selected examples
// based on the ARM_TEST_EXAMPLES environment variable.
// The environment variable should be a comma-separated list of example directories.
// For example, ARM_TEST_EXAMPLES=Microsoft.AlertsManagement_actionRules@2021-08-08,Microsoft.ApiManagement_service_groups@2021-08-01
func TestAccExamples_Selected(t *testing.T) {
	if os.Getenv("ARM_TEST_EXAMPLES") == "" {
		t.Skip("Skipping TestAccExamples_Selected because ARM_TEST_EXAMPLES is not set")
	}
	resourceTypes := strings.Split(os.Getenv("ARM_TEST_EXAMPLES"), ",")
	exampleDir := path.Join("..", "..", "examples")
	testcases := make([]ExampleTestcase, 0)
	for _, resourceType := range resourceTypes {
		if strings.Trim(resourceType, " ") == "" {
			continue
		}

		resourceTypeDir := path.Join(exampleDir, resourceType)
		testcases = append(testcases, ListTestcases(resourceTypeDir, true)...)
	}

	t.Logf("Found %d testcases", len(testcases))
	for _, tc := range testcases {
		t.Run(tc.Path, func(t *testing.T) {
			r := GenericResource{}
			data := acceptance.BuildTestData(nil, "azapi_resource", "test")
			data.ResourceTest(t, r, []resource.TestStep{
				{
					Config:            tc.Config,
					ExternalProviders: tc.ExternalProviders,
				},
			})
		})
	}
}

// TestAccExamples_All runs acceptance tests for all examples
// in the examples directory except for data-sources, ephemeral-resources, functions, and resources.
// The testcases are generated by listing all main.tf files in the examples directory and its subdirectories.
func TestAccExamples_All(t *testing.T) {
	if os.Getenv("ARM_TEST_EXAMPLES_ALL") == "" {
		t.Skip("Skipping TestAccExamples_All because ARM_TEST_EXAMPLES_ALL is not set")
	}
	if os.Getenv("ARM_TEST_EXAMPLES") != "" {
		t.Skip("Skipping TestAccExamples_All because ARM_TEST_EXAMPLES is set, use TestAccExamples_Selected instead")
	}
	exampleDir := path.Join("..", "..", "examples")
	resourceTypes, err := os.ReadDir(exampleDir)
	if err != nil {
		t.Fatalf("Error reading examples directory: %v", err)
	}
	testcases := make([]ExampleTestcase, 0)
	for _, resourceType := range resourceTypes {
		if !resourceType.IsDir() {
			continue
		}
		if resourceType.Name() == "data-sources" || resourceType.Name() == "ephemeral-resources" || resourceType.Name() == "functions" || resourceType.Name() == "resources" {
			continue
		}

		resourceTypeDir := path.Join(exampleDir, resourceType.Name())
		testcases = append(testcases, ListTestcases(resourceTypeDir, false)...)
	}

	t.Logf("Found %d testcases", len(testcases))
	for _, tc := range testcases {
		t.Run(tc.Path, func(t *testing.T) {
			r := GenericResource{}
			data := acceptance.BuildTestData(nil, "azapi_resource", "test")
			data.ResourceTest(t, r, []resource.TestStep{
				{
					Config: tc.Config,
				},
			})
		})
	}
}

func ListTestcases(resourceTypeDir string, includingRootTestcase bool) []ExampleTestcase {
	testcases := make([]ExampleTestcase, 0)

	if includingRootTestcase {
		rootConfig := path.Join(resourceTypeDir, "main.tf")
		if tc, err := LoadTestcase(rootConfig); err == nil {
			testcases = append(testcases, *tc)
		} else {
			fmt.Printf("Error loading config %s: %v\n", rootConfig, err)
		}
	}

	subDirs, err := os.ReadDir(resourceTypeDir)
	if err != nil {
		return testcases
	}
	for _, subDir := range subDirs {
		if !subDir.IsDir() {
			continue
		}

		subConfig := path.Join(resourceTypeDir, subDir.Name(), "main.tf")
		if tc, err := LoadTestcase(subConfig); err == nil {
			testcases = append(testcases, *tc)
		} else {
			fmt.Printf("Error loading config %s: %v\n", subConfig, err)
		}
	}

	return testcases
}

func LoadTestcase(configPath string) (*ExampleTestcase, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s: %v", configPath, err)
	}

	hclFile, diags := hclwrite.ParseConfig(content, configPath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("unable to parse HCL file %s: %v", configPath, diags)
	}

	data := acceptance.BuildTestData(nil, "azapi_resource", "test")
	externalProviders := make(map[string]resource.ExternalProvider)
	for _, block := range hclFile.Body().Blocks() {
		if block.Type() == "variable" && len(block.Labels()) != 0 {
			if block.Labels()[0] == "resource_name" {
				block.Body().SetAttributeValue("default", cty.StringVal(fmt.Sprintf("acctest%s", data.RandomString)))
			}
			if block.Labels()[0] == "subscription_id" {
				block.Body().SetAttributeValue("default", cty.StringVal(os.Getenv("ARM_SUBSCRIPTION_ID")))
			}
		}
		if block.Type() == "terraform" {
			for _, nestedBlock := range block.Body().Blocks() {
				if nestedBlock.Type() == "required_providers" {
					nestedBlock.Body().RemoveAttribute("azapi")
				}

				for providerName := range nestedBlock.Body().Attributes() {
					externalProviders[providerName] = knownExternalProvidersAzurerm()[providerName]
				}
			}
		}
	}
	return &ExampleTestcase{
		Path:              configPath,
		Config:            string(hclFile.Bytes()),
		ExternalProviders: externalProviders,
	}, nil
}
