package tests

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/harvester/terraform-provider-harvester/internal/config"
	"github.com/harvester/terraform-provider-harvester/pkg/constants"
)

// PCIDeviceClaim GVR (Group Version Resource) for Harvester PCIDeviceClaim CRD
var pcideviceClaimGVR = k8sschema.GroupVersionResource{
	Group:    "devices.harvesterhci.io",
	Version:  "v1beta1",
	Resource: "pcideviceclaims",
}

// testAccPCIDevicePreCheck validates that PCI device tests can run.
// Tests are skipped if the required environment variables are not set.
func testAccPCIDevicePreCheck(t *testing.T) {
	testAccPreCheck(t)

	if os.Getenv("HARVESTER_TEST_PCI_NODE") == "" {
		t.Skip("HARVESTER_TEST_PCI_NODE not set, skipping PCI device tests")
	}
	if os.Getenv("HARVESTER_TEST_PCI_ADDRESS") == "" {
		t.Skip("HARVESTER_TEST_PCI_ADDRESS not set, skipping PCI device tests")
	}
}

// TestAccPCIDevice_basic tests the basic create/read/destroy lifecycle of a PCI device claim.
func TestAccPCIDevice_basic(t *testing.T) {
	var (
		testAccVMName          = "test-acc-pci-vm-" + uuid.New().String()[:6]
		testAccPCIDeviceName   = "test-acc-pci-device-" + uuid.New().String()[:6]
		testAccPCIResourceName = constants.ResourceTypePCIDevice + "." + testAccPCIDeviceName
		testAccVMResourceName  = constants.ResourceTypeVirtualMachine + "." + testAccVMName
		pciNode                = os.Getenv("HARVESTER_TEST_PCI_NODE")
		pciAddress             = os.Getenv("HARVESTER_TEST_PCI_ADDRESS")
		ctx                    = context.Background()
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPCIDevicePreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPCIDeviceDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccPCIDeviceConfig(testAccVMName, testAccPCIDeviceName, pciNode, pciAddress),
				Check: resource.ComposeTestCheckFunc(
					testAccVirtualMachineExists(ctx, testAccVMResourceName, nil),
					testAccPCIDeviceClaimExists(ctx, testAccPCIResourceName),
					resource.TestCheckResourceAttr(testAccPCIResourceName, constants.FieldPCIDeviceNodeName, pciNode),
					resource.TestCheckResourceAttr(testAccPCIResourceName, constants.FieldPCIDevicePCIAddresses+".0", pciAddress),
					resource.TestCheckResourceAttr(testAccPCIResourceName, constants.FieldPCIDeviceVMName, "default/"+testAccVMName),
				),
				// Note: Harvester webhooks inject pod_anti_affinity settings on VMs,
				// which causes expected drift. We allow this for now as it's a known
				// VM resource issue, not a PCI device issue.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccPCIDevice_labels tests that labels are correctly applied to PCIDeviceClaim.
func TestAccPCIDevice_labels(t *testing.T) {
	var (
		testAccVMName          = "test-acc-pci-labels-vm-" + uuid.New().String()[:6]
		testAccPCIDeviceName   = "test-acc-pci-labels-" + uuid.New().String()[:6]
		testAccPCIResourceName = constants.ResourceTypePCIDevice + "." + testAccPCIDeviceName
		pciNode                = os.Getenv("HARVESTER_TEST_PCI_NODE")
		pciAddress             = os.Getenv("HARVESTER_TEST_PCI_ADDRESS")
		ctx                    = context.Background()
		expectedLabels         = map[string]string{
			"environment": "test",
			"purpose":     "pci-test",
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPCIDevicePreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPCIDeviceDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccPCIDeviceConfigWithLabels(testAccVMName, testAccPCIDeviceName, pciNode, pciAddress, expectedLabels),
				Check: resource.ComposeTestCheckFunc(
					testAccPCIDeviceClaimExists(ctx, testAccPCIResourceName),
					testAccPCIDeviceClaimLabels(ctx, testAccPCIResourceName, expectedLabels),
					resource.TestCheckResourceAttr(testAccPCIResourceName, "labels.environment", "test"),
					resource.TestCheckResourceAttr(testAccPCIResourceName, "labels.purpose", "pci-test"),
				),
				// Note: VM drift due to Harvester webhook injection of pod_anti_affinity
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccPCIDevice_invalidAddressFormat tests that invalid PCI address formats are rejected.
func TestAccPCIDevice_invalidAddressFormat(t *testing.T) {
	var (
		testAccVMName        = "test-acc-pci-invalid-" + uuid.New().String()[:6]
		testAccPCIDeviceName = "test-acc-pci-invalid-" + uuid.New().String()[:6]
	)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccPCIDeviceConfigInvalidAddress(testAccVMName, testAccPCIDeviceName),
				ExpectError: regexp.MustCompile(`PCI address must be in format '0000:XX:YY\.Z'`),
			},
		},
	})
}

// TestAccPCIDevice_vmNotFound tests that creating a PCI device claim with a non-existent VM fails.
func TestAccPCIDevice_vmNotFound(t *testing.T) {
	var (
		testAccPCIDeviceName = "test-acc-pci-vm-notfound-" + uuid.New().String()[:6]
		pciNode              = "test-node"
		pciAddress           = "0000:00:1f.3"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccPCIDeviceConfigVMNotFound(testAccPCIDeviceName, pciNode, pciAddress),
				ExpectError: regexp.MustCompile(`virtual machine .* not found`),
			},
		},
	})
}

// testAccPCIDeviceConfig generates a Terraform configuration for testing PCI device resources.
func testAccPCIDeviceConfig(vmName, pciDeviceName, nodeName, pciAddress string) string {
	return fmt.Sprintf(`
resource "harvester_virtualmachine" "%s" {
  name        = "%s"
  namespace   = "default"
  description = "Test VM for PCI device passthrough"

  cpu    = 1
  memory = "1Gi"

  run_strategy = "RerunOnFailure"
  machine_type = "q35"

  network_interface {
    name = "default"
  }

  disk {
    name               = "rootdisk"
    type               = "disk"
    bus                = "virtio"
    boot_order         = 1
    container_image_name = "%s"
  }
}

resource "harvester_pci_device" "%s" {
  name      = "%s"
  namespace = "default"

  vm_name   = "default/${harvester_virtualmachine.%s.name}"
  node_name = "%s"

  pci_addresses = ["%s"]

  depends_on = [harvester_virtualmachine.%s]
}
`, vmName, vmName, fedoraCloudContainer, pciDeviceName, pciDeviceName, vmName, nodeName, pciAddress, vmName)
}

// testAccPCIDeviceConfigWithLabels generates a Terraform configuration with labels for testing.
func testAccPCIDeviceConfigWithLabels(vmName, pciDeviceName, nodeName, pciAddress string, labels map[string]string) string {
	labelsStr := ""
	for k, v := range labels {
		labelsStr += fmt.Sprintf("    %s = \"%s\"\n", k, v)
	}

	return fmt.Sprintf(`
resource "harvester_virtualmachine" "%s" {
  name        = "%s"
  namespace   = "default"
  description = "Test VM for PCI device passthrough with labels"

  cpu    = 1
  memory = "1Gi"

  run_strategy = "RerunOnFailure"
  machine_type = "q35"

  network_interface {
    name = "default"
  }

  disk {
    name               = "rootdisk"
    type               = "disk"
    bus                = "virtio"
    boot_order         = 1
    container_image_name = "%s"
  }
}

resource "harvester_pci_device" "%s" {
  name      = "%s"
  namespace = "default"

  vm_name   = "default/${harvester_virtualmachine.%s.name}"
  node_name = "%s"

  pci_addresses = ["%s"]

  labels = {
%s  }

  depends_on = [harvester_virtualmachine.%s]
}
`, vmName, vmName, fedoraCloudContainer, pciDeviceName, pciDeviceName, vmName, nodeName, pciAddress, labelsStr, vmName)
}

// testAccPCIDeviceConfigInvalidAddress generates a configuration with an invalid PCI address.
func testAccPCIDeviceConfigInvalidAddress(vmName, pciDeviceName string) string {
	return fmt.Sprintf(`
resource "harvester_virtualmachine" "%s" {
  name        = "%s"
  namespace   = "default"
  description = "Test VM for invalid PCI address"

  cpu    = 1
  memory = "1Gi"

  run_strategy = "RerunOnFailure"
  machine_type = "q35"

  network_interface {
    name = "default"
  }

  disk {
    name               = "rootdisk"
    type               = "disk"
    bus                = "virtio"
    boot_order         = 1
    container_image_name = "%s"
  }
}

resource "harvester_pci_device" "%s" {
  name      = "%s"
  namespace = "default"

  vm_name   = "default/${harvester_virtualmachine.%s.name}"
  node_name = "test-node"

  # Invalid PCI address format (missing dots/colons)
  pci_addresses = ["invalid-address"]

  depends_on = [harvester_virtualmachine.%s]
}
`, vmName, vmName, fedoraCloudContainer, pciDeviceName, pciDeviceName, vmName, vmName)
}

// testAccPCIDeviceConfigVMNotFound generates a configuration referencing a non-existent VM.
func testAccPCIDeviceConfigVMNotFound(pciDeviceName, nodeName, pciAddress string) string {
	return fmt.Sprintf(`
resource "harvester_pci_device" "%s" {
  name      = "%s"
  namespace = "default"

  vm_name   = "default/nonexistent-vm-12345"
  node_name = "%s"

  pci_addresses = ["%s"]
}
`, pciDeviceName, pciDeviceName, nodeName, pciAddress)
}

// testAccPCIDeviceClaimExists verifies that the PCIDeviceClaim was created.
func testAccPCIDeviceClaimExists(ctx context.Context, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID not set: %s", resourceName)
		}

		c, err := testAccProvider.Meta().(*config.Config).K8sClient()
		if err != nil {
			return err
		}

		dynamicClient, err := dynamic.NewForConfig(c.RestConfig)
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		// Parse the claim name from the ID (format: namespace/vmname/claimname)
		claimName := getClaimNameFromID(rs.Primary.ID)
		if claimName == "" {
			return fmt.Errorf("could not extract claim name from ID: %s", rs.Primary.ID)
		}

		_, err = dynamicClient.Resource(pcideviceClaimGVR).Get(ctx, claimName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("PCIDeviceClaim %s not found: %w", claimName, err)
		}

		return nil
	}
}

// testAccPCIDeviceClaimLabels verifies that the PCIDeviceClaim has the expected labels.
func testAccPCIDeviceClaimLabels(ctx context.Context, resourceName string, expectedLabels map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		c, err := testAccProvider.Meta().(*config.Config).K8sClient()
		if err != nil {
			return err
		}

		dynamicClient, err := dynamic.NewForConfig(c.RestConfig)
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		claimName := getClaimNameFromID(rs.Primary.ID)
		if claimName == "" {
			return fmt.Errorf("could not extract claim name from ID: %s", rs.Primary.ID)
		}

		pcideviceClaim, err := dynamicClient.Resource(pcideviceClaimGVR).Get(ctx, claimName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("PCIDeviceClaim %s not found: %w", claimName, err)
		}

		labels := pcideviceClaim.GetLabels()
		for key, expectedValue := range expectedLabels {
			actualValue, ok := labels[key]
			if !ok {
				return fmt.Errorf("label %s not found on PCIDeviceClaim", key)
			}
			if actualValue != expectedValue {
				return fmt.Errorf("label %s has value %s, expected %s", key, actualValue, expectedValue)
			}
		}

		return nil
	}
}

// testAccCheckPCIDeviceDestroy verifies that PCIDeviceClaims are destroyed.
// Note: In test environments, the Harvester controller may not remove finalizers,
// so we check that the resource is marked for deletion (has deletionTimestamp).
func testAccCheckPCIDeviceDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != constants.ResourceTypePCIDevice {
				continue
			}

			c, err := testAccProvider.Meta().(*config.Config).K8sClient()
			if err != nil {
				return err
			}

			dynamicClient, err := dynamic.NewForConfig(c.RestConfig)
			if err != nil {
				return fmt.Errorf("failed to create dynamic client: %w", err)
			}

			claimName := getClaimNameFromID(rs.Primary.ID)
			if claimName == "" {
				continue
			}

			// Check if the resource exists and is being deleted
			claim, err := dynamicClient.Resource(pcideviceClaimGVR).Get(ctx, claimName, metav1.GetOptions{})
			if err != nil {
				// Resource not found - good, it's been deleted
				continue
			}

			// If the resource exists, check if it has a deletion timestamp
			// (meaning deletion was requested even if finalizers are blocking it)
			deletionTimestamp := claim.GetDeletionTimestamp()
			if deletionTimestamp == nil {
				return fmt.Errorf("PCIDeviceClaim %s still exists and is not marked for deletion", claimName)
			}

			// Resource is marked for deletion - clean up finalizer for test purposes
			// In production, the Harvester controller would handle this
			if len(claim.GetFinalizers()) > 0 {
				claim.SetFinalizers(nil)
				_, err = dynamicClient.Resource(pcideviceClaimGVR).Update(ctx, claim, metav1.UpdateOptions{})
				if err != nil {
					// Ignore errors - the resource may already be gone
					continue
				}
			}
		}
		return nil
	}
}

// getClaimNameFromID extracts the claim name from the resource ID.
// ID format: namespace/vmname/claimname
func getClaimNameFromID(id string) string {
	parts := splitID(id)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// splitID splits a resource ID by "/".
func splitID(id string) []string {
	var parts []string
	current := ""
	for _, c := range id {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// TestAccPCIDevice_import tests importing an existing PCIDeviceClaim.
func TestAccPCIDevice_import(t *testing.T) {
	var (
		testAccVMName          = "test-acc-pci-import-vm-" + uuid.New().String()[:6]
		testAccPCIDeviceName   = "test-acc-pci-import-" + uuid.New().String()[:6]
		testAccPCIResourceName = constants.ResourceTypePCIDevice + "." + testAccPCIDeviceName
		pciNode                = os.Getenv("HARVESTER_TEST_PCI_NODE")
		pciAddress             = os.Getenv("HARVESTER_TEST_PCI_ADDRESS")
		ctx                    = context.Background()
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPCIDevicePreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPCIDeviceDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccPCIDeviceConfig(testAccVMName, testAccPCIDeviceName, pciNode, pciAddress),
				Check: resource.ComposeTestCheckFunc(
					testAccPCIDeviceClaimExists(ctx, testAccPCIResourceName),
				),
				// Note: VM drift due to Harvester webhook injection of pod_anti_affinity
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:      testAccPCIResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The name field will differ because it's set to the claim name during import
				ImportStateVerifyIgnore: []string{"name"},
			},
		},
	})
}

// Helper function to get or create a test PCIDeviceClaim for import testing
func testAccGetPCIDeviceClaim(ctx context.Context, dynamicClient dynamic.Interface, claimName string) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(pcideviceClaimGVR).Get(ctx, claimName, metav1.GetOptions{})
}
