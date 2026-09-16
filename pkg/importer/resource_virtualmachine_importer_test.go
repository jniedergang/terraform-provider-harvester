package importer

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/harvester/harvester/pkg/builder"

	"github.com/harvester/terraform-provider-harvester/pkg/constants"
)

func TestNetworkInterface(t *testing.T) {
	type testcase struct {
		importer    *VMImporter
		expectation []map[string]interface{}
		expectError error
	}

	const (
		networkName0   = "net0"
		networkName1   = "net1"
		networkName2   = "net2"
		interfaceName0 = "eth0"
		linkLocalIPv60 = "fe80::21f:bcff:fe13:405/64"
		ipv4Address0   = "192.168.178.64/24"
		ipv4Address1   = "192.168.180.64/24"
	)

	properties := []string{
		constants.FieldNetworkInterfaceName,
		constants.FieldNetworkInterfaceType,
		constants.FieldNetworkInterfaceModel,
		constants.FieldNetworkInterfaceMACAddress,
		constants.FieldNetworkInterfaceNetworkName,
		constants.FieldNetworkInterfaceBootOrder,
		constants.FieldNetworkInterfaceIPAddress,
		constants.FieldNetworkInterfaceInterfaceName,
		constants.FieldNetworkInterfaceWaitForLease,
	}

	testcases := []testcase{
		{
			// a VM that doesn't have any network interface
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{},
							},
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{},
									},
								},
							},
						},
					},
				},
				VirtualMachineInstance: &kubevirtv1.VirtualMachineInstance{},
			},
			expectation: []map[string]interface{}{},
			expectError: nil,
		},
		{
			// a VM that has a single minimal bridge network interface, but no IP
			// address
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{},
							},
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{1}[0],
											},
										},
									},
								},
							},
						},
					},
				},
				VirtualMachineInstance: &kubevirtv1.VirtualMachineInstance{},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldNetworkInterfaceName:         "",
					constants.FieldNetworkInterfaceType:         builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:        "",
					constants.FieldNetworkInterfaceMACAddress:   "",
					constants.FieldNetworkInterfaceNetworkName:  "",
					constants.FieldNetworkInterfaceBootOrder:    &[]uint{1}[0],
					constants.FieldNetworkInterfaceWaitForLease: false,
				},
			},
			expectError: nil,
		},
		{
			// a VM that has a single minimal bridge network interface, and only
			// a link-local IP addresses
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{},
							},
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												Name: networkName0,
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{1}[0],
											},
										},
									},
								},
							},
						},
					},
				},
				VirtualMachineInstance: &kubevirtv1.VirtualMachineInstance{
					Status: kubevirtv1.VirtualMachineInstanceStatus{
						Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
							{
								Name:          networkName0,
								InterfaceName: interfaceName0,
								IPs:           []string{"169.254.10.140/24", linkLocalIPv60},
							},
						},
					},
				},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldNetworkInterfaceName:         networkName0,
					constants.FieldNetworkInterfaceType:         builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:        "",
					constants.FieldNetworkInterfaceMACAddress:   "",
					constants.FieldNetworkInterfaceNetworkName:  "",
					constants.FieldNetworkInterfaceBootOrder:    &[]uint{1}[0],
					constants.FieldNetworkInterfaceWaitForLease: false,
				},
			},
			expectError: nil,
		},
		{
			// a VM that has a single minimal bridge network interface with IP
			// addresses
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{},
							},
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												Name: networkName0,
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{1}[0],
											},
										},
									},
								},
							},
						},
					},
				},
				VirtualMachineInstance: &kubevirtv1.VirtualMachineInstance{
					Status: kubevirtv1.VirtualMachineInstanceStatus{
						Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
							{
								Name:          networkName0,
								InterfaceName: interfaceName0,
								IPs:           []string{ipv4Address0, linkLocalIPv60},
							},
						},
					},
				},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldNetworkInterfaceName:          networkName0,
					constants.FieldNetworkInterfaceType:          builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:         "",
					constants.FieldNetworkInterfaceMACAddress:    "",
					constants.FieldNetworkInterfaceNetworkName:   "",
					constants.FieldNetworkInterfaceBootOrder:     &[]uint{1}[0],
					constants.FieldNetworkInterfaceWaitForLease:  false,
					constants.FieldNetworkInterfaceIPAddress:     ipv4Address0,
					constants.FieldNetworkInterfaceInterfaceName: interfaceName0,
				},
			},
			expectError: nil,
		},
		{
			// a VM that has multiple minimal bridge network interfaces with several IP
			// addresses
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{},
							},
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												Name: networkName0,
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{1}[0],
											},
											{
												Name: networkName1,
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{2}[0],
											},
											{
												Name: networkName2,
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{3}[0],
											},
										},
									},
								},
							},
						},
					},
				},
				VirtualMachineInstance: &kubevirtv1.VirtualMachineInstance{
					Status: kubevirtv1.VirtualMachineInstanceStatus{
						Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
							{
								Name:          networkName0,
								InterfaceName: interfaceName0,
								IPs:           []string{ipv4Address0, linkLocalIPv60},
							},
							{
								Name:          networkName1,
								InterfaceName: "eth1",
								IPs:           []string{"fe80::21f:bcff:fe13:406/64"},
							},
							{
								Name:          networkName2,
								InterfaceName: "eth2",
								IPs:           []string{ipv4Address1, "169.254.180.64/24", "201.168.180.64/24"},
							},
						},
					},
				},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldNetworkInterfaceName:          networkName0,
					constants.FieldNetworkInterfaceType:          builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:         "",
					constants.FieldNetworkInterfaceMACAddress:    "",
					constants.FieldNetworkInterfaceNetworkName:   "",
					constants.FieldNetworkInterfaceBootOrder:     &[]uint{1}[0],
					constants.FieldNetworkInterfaceWaitForLease:  false,
					constants.FieldNetworkInterfaceIPAddress:     ipv4Address0,
					constants.FieldNetworkInterfaceInterfaceName: interfaceName0,
				},
				{
					constants.FieldNetworkInterfaceName:         networkName1,
					constants.FieldNetworkInterfaceType:         builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:        "",
					constants.FieldNetworkInterfaceMACAddress:   "",
					constants.FieldNetworkInterfaceNetworkName:  "",
					constants.FieldNetworkInterfaceBootOrder:    &[]uint{2}[0],
					constants.FieldNetworkInterfaceWaitForLease: false,
				},
				{
					constants.FieldNetworkInterfaceName:          networkName2,
					constants.FieldNetworkInterfaceType:          builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:         "",
					constants.FieldNetworkInterfaceMACAddress:    "",
					constants.FieldNetworkInterfaceNetworkName:   "",
					constants.FieldNetworkInterfaceBootOrder:     &[]uint{3}[0],
					constants.FieldNetworkInterfaceWaitForLease:  false,
					constants.FieldNetworkInterfaceIPAddress:     ipv4Address1,
					constants.FieldNetworkInterfaceInterfaceName: "eth2",
				},
			},
			expectError: nil,
		},
		{
			// a VM that has a minimal bridge network interface with multiple IP
			// addresses in different order
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{},
							},
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												Name: networkName0,
												InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
													Bridge: &kubevirtv1.InterfaceBridge{},
												},
												BootOrder: &[]uint{1}[0],
											},
										},
									},
								},
							},
						},
					},
				},
				VirtualMachineInstance: &kubevirtv1.VirtualMachineInstance{
					Status: kubevirtv1.VirtualMachineInstanceStatus{
						Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
							{
								Name:          networkName0,
								InterfaceName: interfaceName0,
								IPs:           []string{"201.168.180.64/24", "169.254.180.64/24", ipv4Address1},
							},
						},
					},
				},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldNetworkInterfaceName:          networkName0,
					constants.FieldNetworkInterfaceType:          builder.NetworkInterfaceTypeBridge,
					constants.FieldNetworkInterfaceModel:         "",
					constants.FieldNetworkInterfaceMACAddress:    "",
					constants.FieldNetworkInterfaceNetworkName:   "",
					constants.FieldNetworkInterfaceBootOrder:     &[]uint{1}[0],
					constants.FieldNetworkInterfaceWaitForLease:  false,
					constants.FieldNetworkInterfaceIPAddress:     ipv4Address1,
					constants.FieldNetworkInterfaceInterfaceName: interfaceName0,
				},
			},
			expectError: nil,
		},
	}

	for _, tc := range testcases {
		outcome, err := tc.importer.NetworkInterface()

		if err != nil && tc.expectError == nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if err == nil && tc.expectError != nil {
			t.Errorf("Expected error %v, got nil", tc.expectError)
		}

		if len(outcome) != len(tc.expectation) {
			t.Errorf("Unexpected outcome length: %v, expected %v", len(outcome), len(tc.expectation))
		}

		for idx, out := range outcome {
			expect := tc.expectation[idx]

			for _, property := range properties {
				switch expect[property].(type) {
				case *uint:
					o := (out[property].(*uint))
					e := (expect[property].(*uint))
					if *o != *e {
						t.Errorf("Failed Importing NetworkInterface. Value for %v is %v, expeceted %v",
							property,
							*o,
							*e)
					}
				default:
					if out[property] != expect[property] {
						t.Errorf("Failed Importing NetworkInterface. Value for %v is %v, expeceted %v",
							property,
							out[property],
							expect[property])
					}
				}
			}
		}
	}
}

func TestTolerations(t *testing.T) {
	type testcase struct {
		name        string
		importer    *VMImporter
		expectation []map[string]interface{}
	}

	int64Ptr := func(v int64) *int64 { return &v }

	testcases := []testcase{
		{
			name: "nil tolerations returns empty slice",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{},
						},
					},
				},
			},
			expectation: []map[string]interface{}{},
		},
		{
			name: "standard toleration",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Tolerations: []corev1.Toleration{
									{
										Key:      "key1",
										Operator: corev1.TolerationOpEqual,
										Value:    "value1",
										Effect:   corev1.TaintEffectNoSchedule,
									},
								},
							},
						},
					},
				},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldTolerationKey:               "key1",
					constants.FieldTolerationOperator:          "Equal",
					constants.FieldTolerationValue:             "value1",
					constants.FieldTolerationEffect:            "NoSchedule",
					constants.FieldTolerationTolerationSeconds: 0,
				},
			},
		},
		{
			name: "Exists operator with TolerationSeconds",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Tolerations: []corev1.Toleration{
									{
										Key:               "node.kubernetes.io/not-ready",
										Operator:          corev1.TolerationOpExists,
										Effect:            corev1.TaintEffectNoExecute,
										TolerationSeconds: int64Ptr(300),
									},
								},
							},
						},
					},
				},
			},
			expectation: []map[string]interface{}{
				{
					constants.FieldTolerationKey:               "node.kubernetes.io/not-ready",
					constants.FieldTolerationOperator:          "Exists",
					constants.FieldTolerationValue:             "",
					constants.FieldTolerationEffect:            "NoExecute",
					constants.FieldTolerationTolerationSeconds: 300,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.importer.Tolerations()
			if len(result) != len(tc.expectation) {
				t.Fatalf("expected %d tolerations, got %d", len(tc.expectation), len(result))
			}
			for idx, r := range result {
				e := tc.expectation[idx]
				for k, ev := range e {
					if r[k] != ev {
						t.Errorf("toleration[%d][%s] = %v, expected %v", idx, k, r[k], ev)
					}
				}
			}
		})
	}
}

func TestInstallGuestAgent(t *testing.T) {
	type testcase struct {
		name     string
		importer *VMImporter
		expected bool
	}

	testcases := []testcase{
		{
			name: "no cloud-init volumes returns false",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "cloud-init with guest agent snippet returns true",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Volumes: []kubevirtv1.Volume{
									{
										Name: "cloudinitdisk",
										VolumeSource: kubevirtv1.VolumeSource{
											CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
												UserData: "#cloud-config\npackage_update: true\npackages:\n  - qemu-guest-agent\nruncmd:\n  - - systemctl\n    - enable\n    - '--now'\n    - qemu-ga",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "cloud-init without guest agent returns false",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Volumes: []kubevirtv1.Volume{
									{
										Name: "cloudinitdisk",
										VolumeSource: kubevirtv1.VolumeSource{
											CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
												UserData: "#cloud-config\nuser: sles\n",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "base64 user data returns false",
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Volumes: []kubevirtv1.Volume{
									{
										Name: "cloudinitdisk",
										VolumeSource: kubevirtv1.VolumeSource{
											CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
												UserDataBase64: "I2Nsb3VkLWNvbmZpZwpwYWNrYWdlczoKICAtIHFlbXUtZ3Vlc3QtYWdlbnQ=",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.importer.InstallGuestAgent()
			if result != tc.expected {
				t.Errorf("InstallGuestAgent() = %v, expected %v", result, tc.expected)
			}
		})
	}
}

func TestCPU(t *testing.T) {
	type testcase struct {
		importer      *VMImporter
		expectedCores int
		expectedModel string
	}

	testcases := []testcase{
		{
			// VM with basic CPU configuration (no model specified)
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									CPU: &kubevirtv1.CPU{
										Cores: 2,
									},
								},
							},
						},
					},
				},
			},
			expectedCores: 2,
			expectedModel: "",
		},
		{
			// VM with CPU model set to specific Intel model
			importer: &VMImporter{
				VirtualMachine: &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									CPU: &kubevirtv1.CPU{
										Cores: 8,
										Model: "Skylake-Client-IBRS",
									},
								},
							},
						},
					},
				},
			},
			expectedCores: 8,
			expectedModel: "Skylake-Client-IBRS",
		},
	}

	for idx, tc := range testcases {
		cores := tc.importer.CPU()
		if cores != tc.expectedCores {
			t.Errorf("Test case %d: CPU() returned %d, expected %d", idx, cores, tc.expectedCores)
		}

		model := tc.importer.CPUModel()
		if model != tc.expectedModel {
			t.Errorf("Test case %d: CPUModel() returned %q, expected %q", idx, model, tc.expectedModel)
		}
	}
}

func TestResourceRequestsImport(t *testing.T) {
	// Test with explicit requests
	vm := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						Resources: kubevirtv1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
					},
				},
			},
		},
	}
	importer := &VMImporter{VirtualMachine: vm}

	reqs := importer.Requests()
	if len(reqs) != 1 {
		t.Fatalf("Requests() returned %d entries, want 1", len(reqs))
	}
	if got := reqs[0][constants.FieldRequestsCPU]; got != "500m" {
		t.Errorf("Requests() cpu = %q, want %q", got, "500m")
	}
	if got := reqs[0][constants.FieldRequestsMemory]; got != "512Mi" {
		t.Errorf("Requests() memory = %q, want %q", got, "512Mi")
	}

	// Test without requests (empty)
	vmNoReq := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						Resources: kubevirtv1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
					},
				},
			},
		},
	}
	importerNoReq := &VMImporter{VirtualMachine: vmNoReq}

	reqsNoReq := importerNoReq.Requests()
	if len(reqsNoReq) != 1 {
		t.Fatalf("Requests() no requests returned %d entries, want 1", len(reqsNoReq))
	}
	if got := reqsNoReq[0][constants.FieldRequestsCPU]; got != "" {
		t.Errorf("Requests() no requests cpu = %q, want empty", got)
	}
	if got := reqsNoReq[0][constants.FieldRequestsMemory]; got != "" {
		t.Errorf("Requests() no requests memory = %q, want empty", got)
	}

	// Test with nil Requests map
	vmNilReq := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						Resources: kubevirtv1.ResourceRequirements{
							Requests: nil,
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
					},
				},
			},
		},
	}
	importerNilReq := &VMImporter{VirtualMachine: vmNilReq}

	reqsNil := importerNilReq.Requests()
	if len(reqsNil) != 1 {
		t.Fatalf("Requests() nil returned %d entries, want 1", len(reqsNil))
	}
	if got := reqsNil[0][constants.FieldRequestsCPU]; got != "" {
		t.Errorf("Requests() nil cpu = %q, want empty", got)
	}
	if got := reqsNil[0][constants.FieldRequestsMemory]; got != "" {
		t.Errorf("Requests() nil memory = %q, want empty", got)
	}
}

// TestVolume verifies that Volume() reports explicit disks (PVC, container,
// cd-rom) while representing the auto-created cloud-init disk via the cloudinit
// block only. Reporting the cloud-init disk as a disk block would otherwise
// cause a perpetual diff on every plan.
func TestVolume(t *testing.T) {
	bootOrder := func(n uint) *uint { return &n }
	disk := func(name, diskType, bus string, order uint, cache string) kubevirtv1.Disk {
		d := kubevirtv1.Disk{Name: name, BootOrder: bootOrder(order), Cache: kubevirtv1.DriverCache(cache)}
		if diskType == builder.DiskTypeCDRom {
			d.DiskDevice = kubevirtv1.DiskDevice{CDRom: &kubevirtv1.CDRomTarget{Bus: kubevirtv1.DiskBus(bus)}}
		} else {
			d.DiskDevice = kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: kubevirtv1.DiskBus(bus)}}
		}
		return d
	}
	pvcVol := func(name string) kubevirtv1.Volume {
		return kubevirtv1.Volume{Name: name, VolumeSource: kubevirtv1.VolumeSource{PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{ClaimName: name}}}}
	}
	containerVol := func(name, image string) kubevirtv1.Volume {
		return kubevirtv1.Volume{Name: name, VolumeSource: kubevirtv1.VolumeSource{ContainerDisk: &kubevirtv1.ContainerDiskSource{Image: image}}}
	}
	noCloudVol := func(name string) kubevirtv1.Volume {
		return kubevirtv1.Volume{Name: name, VolumeSource: kubevirtv1.VolumeSource{CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{UserData: "#cloud-config\n"}}}
	}
	configDriveVol := func(name string) kubevirtv1.Volume {
		return kubevirtv1.Volume{Name: name, VolumeSource: kubevirtv1.VolumeSource{CloudInitConfigDrive: &kubevirtv1.CloudInitConfigDriveSource{UserData: "#cloud-config\n"}}}
	}
	build := func(disks []kubevirtv1.Disk, volumes []kubevirtv1.Volume) *VMImporter {
		return &VMImporter{VirtualMachine: &kubevirtv1.VirtualMachine{Spec: kubevirtv1.VirtualMachineSpec{Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{Spec: kubevirtv1.VirtualMachineInstanceSpec{Domain: kubevirtv1.DomainSpec{Devices: kubevirtv1.Devices{Disks: disks}}, Volumes: volumes}}}}}
	}
	// fields holds the per-disk values the importer derives directly (the surface
	// the refactor moves around); volume-specific fields like volume_name are set
	// by pvcVolume and asserted via extra.
	type wantDisk struct {
		name      string
		diskType  string
		bus       string
		bootOrder uint
		cache     string
		extra     map[string]interface{}
	}

	testcases := []struct {
		name          string
		importer      *VMImporter
		wantDisks     []wantDisk
		wantCloudInit bool
		wantErr       bool
	}{
		{
			name:          "pvc root disk + noCloud cloud-init is skipped",
			importer:      build([]kubevirtv1.Disk{disk("rootdisk", builder.DiskTypeDisk, "virtio", 1, "writeback"), disk(builder.CloudInitDiskName, builder.DiskTypeDisk, "virtio", 0, "")}, []kubevirtv1.Volume{pvcVol("rootdisk"), noCloudVol(builder.CloudInitDiskName)}),
			wantDisks:     []wantDisk{{name: "rootdisk", diskType: builder.DiskTypeDisk, bus: "virtio", bootOrder: 1, cache: "writeback", extra: map[string]interface{}{constants.FieldDiskVolumeName: "rootdisk"}}},
			wantCloudInit: true,
		},
		{
			name:          "configDrive cloud-init is also skipped",
			importer:      build([]kubevirtv1.Disk{disk("rootdisk", builder.DiskTypeDisk, "virtio", 1, ""), disk(builder.CloudInitDiskName, builder.DiskTypeDisk, "sata", 0, "")}, []kubevirtv1.Volume{pvcVol("rootdisk"), configDriveVol(builder.CloudInitDiskName)}),
			wantDisks:     []wantDisk{{name: "rootdisk", diskType: builder.DiskTypeDisk, bus: "virtio", bootOrder: 1}},
			wantCloudInit: true,
		},
		{
			name:          "container disk reported, no cloud-init",
			importer:      build([]kubevirtv1.Disk{disk("rootdisk", builder.DiskTypeDisk, "virtio", 1, "")}, []kubevirtv1.Volume{containerVol("rootdisk", "example/image:latest")}),
			wantDisks:     []wantDisk{{name: "rootdisk", diskType: builder.DiskTypeDisk, bus: "virtio", bootOrder: 1, extra: map[string]interface{}{constants.FieldDiskContainerImageName: "example/image:latest"}}},
			wantCloudInit: false,
		},
		{
			// cloud-init in the middle of the disk list must still be skipped and
			// must not shift the surrounding disks.
			name:     "cd-rom and a cloud-init disk in the middle",
			importer: build([]kubevirtv1.Disk{disk("rootdisk", builder.DiskTypeDisk, "scsi", 1, ""), disk(builder.CloudInitDiskName, builder.DiskTypeDisk, "virtio", 0, ""), disk("iso", builder.DiskTypeCDRom, "sata", 2, "")}, []kubevirtv1.Volume{pvcVol("rootdisk"), noCloudVol(builder.CloudInitDiskName)}),
			wantDisks: []wantDisk{
				{name: "rootdisk", diskType: builder.DiskTypeDisk, bus: "scsi", bootOrder: 1, extra: map[string]interface{}{constants.FieldDiskVolumeName: "rootdisk"}},
				{name: "iso", diskType: builder.DiskTypeCDRom, bus: "sata", bootOrder: 2},
			},
			wantCloudInit: true,
		},
		{
			name:          "no cloud-init at all",
			importer:      build([]kubevirtv1.Disk{disk("rootdisk", builder.DiskTypeDisk, "virtio", 1, "")}, []kubevirtv1.Volume{pvcVol("rootdisk")}),
			wantDisks:     []wantDisk{{name: "rootdisk", diskType: builder.DiskTypeDisk, bus: "virtio", bootOrder: 1}},
			wantCloudInit: false,
		},
		{
			// not reachable via this provider (a VM always has a real disk), but the
			// importer must not panic and yields an empty disk list.
			name:          "cloud-init only yields an empty disk list",
			importer:      build([]kubevirtv1.Disk{disk(builder.CloudInitDiskName, builder.DiskTypeDisk, "virtio", 0, "")}, []kubevirtv1.Volume{noCloudVol(builder.CloudInitDiskName)}),
			wantDisks:     []wantDisk{},
			wantCloudInit: true,
		},
		{
			name:     "disk with neither cdrom nor disk errors",
			importer: build([]kubevirtv1.Disk{{Name: "broken"}}, nil),
			wantErr:  true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			diskStates, cloudInitState, err := tc.importer.Volume()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Volume() error: %v", err)
			}
			if len(diskStates) != len(tc.wantDisks) {
				t.Fatalf("got %d disks, want %d: %v", len(diskStates), len(tc.wantDisks), diskStates)
			}
			for i, want := range tc.wantDisks {
				got := diskStates[i]
				if got[constants.FieldDiskName] != want.name {
					t.Errorf("disk[%d] name = %v, want %v", i, got[constants.FieldDiskName], want.name)
				}
				if got[constants.FieldDiskType] != want.diskType {
					t.Errorf("disk[%d] type = %v, want %v", i, got[constants.FieldDiskType], want.diskType)
				}
				if got[constants.FieldDiskBus] != want.bus {
					t.Errorf("disk[%d] bus = %v, want %v", i, got[constants.FieldDiskBus], want.bus)
				}
				if bo, ok := got[constants.FieldDiskBootOrder].(*uint); !ok || bo == nil || *bo != want.bootOrder {
					t.Errorf("disk[%d] boot_order = %v, want %v", i, got[constants.FieldDiskBootOrder], want.bootOrder)
				}
				if string(got[constants.FieldDiskCacheMode].(kubevirtv1.DriverCache)) != want.cache {
					t.Errorf("disk[%d] cache_mode = %v, want %q", i, got[constants.FieldDiskCacheMode], want.cache)
				}
				for k, v := range want.extra {
					if got[k] != v {
						t.Errorf("disk[%d] %s = %v, want %v", i, k, got[k], v)
					}
				}
			}
			if hasCloudInit := len(cloudInitState) > 0; hasCloudInit != tc.wantCloudInit {
				t.Errorf("cloud-init present = %v, want %v", hasCloudInit, tc.wantCloudInit)
			}
		})
	}
}

func TestStripInjectedSSHUser(t *testing.T) {
	newImporter := func(sshUser string) *VMImporter {
		labels := map[string]string{}
		if sshUser != "" {
			labels[builder.LabelPrefixHarvesterTag+constants.LabelSSHUsername] = sshUser
		}
		return &VMImporter{
			VirtualMachine: &kubevirtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
			},
		}
	}
	testcases := []struct {
		name     string
		sshUser  string
		input    string
		expected string
	}{
		{
			name:     "no ssh-user tag leaves user_data unchanged",
			sshUser:  "",
			input:    "#cloud-config\nuser: sles\n",
			expected: "#cloud-config\nuser: sles\n",
		},
		{
			name:     "fully injected user_data reads back as empty",
			sshUser:  "sles",
			input:    "#cloud-config\nuser: sles\n",
			expected: "",
		},
		{
			name:     "injected trailing user line is stripped",
			sshUser:  "sles",
			input:    "#cloud-config\npackages:\n  - qemu-guest-agent\n\nuser: sles\n",
			expected: "#cloud-config\npackages:\n  - qemu-guest-agent\n",
		},
		{
			name:     "user line in the middle is kept",
			sshUser:  "sles",
			input:    "#cloud-config\nuser: sles\npackages:\n  - vim\n",
			expected: "#cloud-config\nuser: sles\npackages:\n  - vim\n",
		},
		{
			name:     "trailing user line for another user is kept",
			sshUser:  "sles",
			input:    "#cloud-config\n\nuser: opensuse\n",
			expected: "#cloud-config\n\nuser: opensuse\n",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := newImporter(tc.sshUser).stripInjectedSSHUser(tc.input); got != tc.expected {
				t.Errorf("stripInjectedSSHUser() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func affinityImporter(affinity *corev1.Affinity) *VMImporter {
	return &VMImporter{
		VirtualMachine: &kubevirtv1.VirtualMachine{
			Spec: kubevirtv1.VirtualMachineSpec{
				Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
					Spec: kubevirtv1.VirtualMachineInstanceSpec{
						Affinity: affinity,
					},
				},
			},
		},
	}
}

// TestNodeAffinityImport verifies that Harvester-injected node selector
// expressions (network.harvesterhci.io/*) are filtered out and that terms left
// empty by the filter are not exported as phantom empty terms.
func TestNodeAffinityImport(t *testing.T) {
	injected := corev1.NodeSelectorRequirement{
		Key:      "network.harvesterhci.io/mgmt",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"true"},
	}
	user := corev1.NodeSelectorRequirement{
		Key:      "kubernetes.io/hostname",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"node1"},
	}

	testcases := []struct {
		name     string
		affinity *corev1.Affinity
		expected int // number of exported node_affinity blocks
	}{
		{
			name:     "nil affinity exports nothing",
			affinity: nil,
			expected: 0,
		},
		{
			name: "injected-only term exports nothing (no phantom empty term)",
			affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchExpressions: []corev1.NodeSelectorRequirement{injected}},
						},
					},
				},
			},
			expected: 0,
		},
		{
			name: "user expression is kept when mixed with injected one",
			affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchExpressions: []corev1.NodeSelectorRequirement{user, injected}},
						},
					},
				},
			},
			expected: 1,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := affinityImporter(tc.affinity).NodeAffinity()
			if len(got) != tc.expected {
				t.Fatalf("NodeAffinity() exported %d blocks, want %d (%v)", len(got), tc.expected, got)
			}
			if tc.expected == 1 {
				required := got[0][constants.FieldNodeAffinityRequired].([]map[string]interface{})
				terms := required[0][constants.FieldNodeSelectorTerm].([]map[string]interface{})
				expressions := terms[0][constants.FieldMatchExpressions].([]map[string]interface{})
				if len(expressions) != 1 || expressions[0][constants.FieldExpressionKey] != "kubernetes.io/hostname" {
					t.Errorf("expected only the user expression, got %v", expressions)
				}
			}
		})
	}
}

// TestPodAntiAffinityImport verifies that the Harvester-injected creator term
// is filtered while user terms selecting other harvesterhci.io labels are kept.
func TestPodAntiAffinityImport(t *testing.T) {
	creatorTerm := corev1.WeightedPodAffinityTerm{
		Weight: 1,
		PodAffinityTerm: corev1.PodAffinityTerm{
			TopologyKey: "kubernetes.io/hostname",
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "harvesterhci.io/creator", Operator: metav1.LabelSelectorOpExists},
				},
			},
		},
	}
	userTerm := corev1.WeightedPodAffinityTerm{
		Weight: 100,
		PodAffinityTerm: corev1.PodAffinityTerm{
			TopologyKey: "kubernetes.io/hostname",
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "harvesterhci.io/vmName", Operator: metav1.LabelSelectorOpIn, Values: []string{"other-vm"}},
				},
			},
		},
	}

	affinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{creatorTerm, userTerm},
		},
	}
	got := affinityImporter(affinity).PodAntiAffinity()
	if len(got) != 1 {
		t.Fatalf("PodAntiAffinity() exported %d blocks, want 1", len(got))
	}
	preferred := got[0][constants.FieldPodAffinityPreferred].([]map[string]interface{})
	if len(preferred) != 1 {
		t.Fatalf("expected exactly the user term after filtering, got %d terms", len(preferred))
	}
	if preferred[0][constants.FieldPreferredWeight] != 100 {
		t.Errorf("expected user term weight 100, got %v", preferred[0][constants.FieldPreferredWeight])
	}

	// creator-only anti-affinity (plain VM) must export nothing at all
	affinityInjectedOnly := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{creatorTerm},
		},
	}
	if got := affinityImporter(affinityInjectedOnly).PodAntiAffinity(); len(got) != 0 {
		t.Errorf("creator-only anti-affinity should export nothing, got %v", got)
	}
}

func TestCPUTopologyImport(t *testing.T) {
	// Test with explicit values
	vm := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{
							Sockets: 2,
							Threads: 4,
						},
					},
				},
			},
		},
	}
	importer := &VMImporter{VirtualMachine: vm}

	if got := importer.CPUSockets(); got != 2 {
		t.Errorf("CPUSockets() = %d, want 2", got)
	}
	if got := importer.CPUThreads(); got != 4 {
		t.Errorf("CPUThreads() = %d, want 4", got)
	}

	// Test with zero values (defaults)
	vmZero := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{
							Sockets: 0,
							Threads: 0,
						},
					},
				},
			},
		},
	}
	importerZero := &VMImporter{VirtualMachine: vmZero}

	if got := importerZero.CPUSockets(); got != 1 {
		t.Errorf("CPUSockets() zero value = %d, want 1", got)
	}
	if got := importerZero.CPUThreads(); got != 1 {
		t.Errorf("CPUThreads() zero value = %d, want 1", got)
	}
}

func TestVMRuntimeImport(t *testing.T) {
	// Test with explicit values
	strategy := kubevirtv1.EvictionStrategy("LiveMigrate")
	grace := int64(60)
	vm := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				constants.AnnotationOSType: "linux",
			},
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					EvictionStrategy:              &strategy,
					TerminationGracePeriodSeconds: &grace,
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{},
					},
				},
			},
		},
	}
	importer := &VMImporter{VirtualMachine: vm}

	if got := importer.EvictionStrategy(); got != "LiveMigrate" {
		t.Errorf("EvictionStrategy() = %q, want %q", got, "LiveMigrate")
	}
	if got := importer.TerminationGracePeriodSeconds(); got != 60 {
		t.Errorf("TerminationGracePeriodSeconds() = %d, want 60", got)
	}
	if got := importer.OSType(); got != "linux" {
		t.Errorf("OSType() = %q, want %q", got, "linux")
	}

	// Test with nil values (defaults)
	vmNil := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{},
					},
				},
			},
		},
	}
	importerNil := &VMImporter{VirtualMachine: vmNil}

	if got := importerNil.EvictionStrategy(); got != constants.DefaultEvictionStrategy {
		t.Errorf("EvictionStrategy() nil = %q, want %q", got, constants.DefaultEvictionStrategy)
	}
	if got := importerNil.TerminationGracePeriodSeconds(); got != constants.DefaultTerminationGracePeriodSeconds {
		t.Errorf("TerminationGracePeriodSeconds() nil = %d, want %d", got, constants.DefaultTerminationGracePeriodSeconds)
	}
	if got := importerNil.OSType(); got != "" {
		t.Errorf("OSType() nil = %q, want %q", got, "")
	}
}

func TestHugepagesImport(t *testing.T) {
	// Test with hugepages set
	vm := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						Memory: &kubevirtv1.Memory{
							Hugepages: &kubevirtv1.Hugepages{PageSize: "2Mi"},
						},
					},
				},
			},
		},
	}
	importer := &VMImporter{VirtualMachine: vm}
	if got := importer.HugepagesSize(); got != "2Mi" {
		t.Errorf("HugepagesSize() = %q, want %q", got, "2Mi")
	}

	// Test with nil memory
	vmNilMem := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{},
				},
			},
		},
	}
	importerNilMem := &VMImporter{VirtualMachine: vmNilMem}
	if got := importerNilMem.HugepagesSize(); got != "" {
		t.Errorf("HugepagesSize() nil memory = %q, want %q", got, "")
	}

	// Test with nil hugepages
	vmNilHuge := &kubevirtv1.VirtualMachine{
		Spec: kubevirtv1.VirtualMachineSpec{
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						Memory: &kubevirtv1.Memory{},
					},
				},
			},
		},
	}
	importerNilHuge := &VMImporter{VirtualMachine: vmNilHuge}
	if got := importerNilHuge.HugepagesSize(); got != "" {
		t.Errorf("HugepagesSize() nil hugepages = %q, want %q", got, "")
	}
}

func TestDiskEjectImport(t *testing.T) {
	makeVM := func(disks []kubevirtv1.Disk, volumes []kubevirtv1.Volume) *VMImporter {
		return &VMImporter{
			VirtualMachine: &kubevirtv1.VirtualMachine{
				Spec: kubevirtv1.VirtualMachineSpec{
					Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
						Spec: kubevirtv1.VirtualMachineInstanceSpec{
							Domain: kubevirtv1.DomainSpec{
								Devices: kubevirtv1.Devices{
									Disks: disks,
								},
							},
							Volumes: volumes,
						},
					},
				},
			},
		}
	}

	// CD-ROM with tray open (ejected)
	imp := makeVM(
		[]kubevirtv1.Disk{{
			Name: "cdrom-disk",
			DiskDevice: kubevirtv1.DiskDevice{
				CDRom: &kubevirtv1.CDRomTarget{
					Bus:  kubevirtv1.DiskBusSATA,
					Tray: kubevirtv1.TrayStateOpen,
				},
			},
		}},
		[]kubevirtv1.Volume{{
			Name: "cdrom-disk",
			VolumeSource: kubevirtv1.VolumeSource{
				ContainerDisk: &kubevirtv1.ContainerDiskSource{
					Image: "test-image",
				},
			},
		}},
	)
	diskStates, _, err := imp.Volume()
	if err != nil {
		t.Fatalf("Volume() error: %v", err)
	}
	if len(diskStates) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(diskStates))
	}
	if eject, ok := diskStates[0][constants.FieldDiskEject].(bool); !ok || !eject {
		t.Errorf("CD-ROM with TrayStateOpen: eject = %v, want true", diskStates[0][constants.FieldDiskEject])
	}

	// CD-ROM with tray closed (not ejected)
	imp2 := makeVM(
		[]kubevirtv1.Disk{{
			Name: "cdrom-disk",
			DiskDevice: kubevirtv1.DiskDevice{
				CDRom: &kubevirtv1.CDRomTarget{
					Bus:  kubevirtv1.DiskBusSATA,
					Tray: kubevirtv1.TrayStateClosed,
				},
			},
		}},
		[]kubevirtv1.Volume{{
			Name: "cdrom-disk",
			VolumeSource: kubevirtv1.VolumeSource{
				ContainerDisk: &kubevirtv1.ContainerDiskSource{
					Image: "test-image",
				},
			},
		}},
	)
	diskStates2, _, err := imp2.Volume()
	if err != nil {
		t.Fatalf("Volume() error: %v", err)
	}
	if eject, ok := diskStates2[0][constants.FieldDiskEject].(bool); !ok || eject {
		t.Errorf("CD-ROM with TrayStateClosed: eject = %v, want false", diskStates2[0][constants.FieldDiskEject])
	}

	// Regular disk (not CD-ROM) should have eject=false
	imp3 := makeVM(
		[]kubevirtv1.Disk{{
			Name: "rootdisk",
			DiskDevice: kubevirtv1.DiskDevice{
				Disk: &kubevirtv1.DiskTarget{
					Bus: kubevirtv1.DiskBusVirtio,
				},
			},
		}},
		[]kubevirtv1.Volume{{
			Name: "rootdisk",
			VolumeSource: kubevirtv1.VolumeSource{
				ContainerDisk: &kubevirtv1.ContainerDiskSource{
					Image: "test-image",
				},
			},
		}},
	)
	diskStates3, _, err := imp3.Volume()
	if err != nil {
		t.Fatalf("Volume() error: %v", err)
	}
	if eject, ok := diskStates3[0][constants.FieldDiskEject].(bool); !ok || eject {
		t.Errorf("Regular disk: eject = %v, want false", diskStates3[0][constants.FieldDiskEject])
	}

	// CD-ROM with no Tray field set should default to eject=false
	imp4 := makeVM(
		[]kubevirtv1.Disk{{
			Name: "cdrom-no-tray",
			DiskDevice: kubevirtv1.DiskDevice{
				CDRom: &kubevirtv1.CDRomTarget{
					Bus: kubevirtv1.DiskBusSATA,
				},
			},
		}},
		[]kubevirtv1.Volume{{
			Name: "cdrom-no-tray",
			VolumeSource: kubevirtv1.VolumeSource{
				ContainerDisk: &kubevirtv1.ContainerDiskSource{
					Image: "test-image",
				},
			},
		}},
	)
	diskStates4, _, err := imp4.Volume()
	if err != nil {
		t.Fatalf("Volume() error: %v", err)
	}
	if eject, ok := diskStates4[0][constants.FieldDiskEject].(bool); !ok || eject {
		t.Errorf("CD-ROM with no Tray set: eject = %v, want false", diskStates4[0][constants.FieldDiskEject])
	}
}

func TestStripGuestAgentSnippet(t *testing.T) {
	snippet := constants.GuestAgentCloudInitSnippet
	testcases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no injection is left untouched",
			input:    "#cloud-config\npackages:\n  - vim\n",
			expected: "#cloud-config\npackages:\n  - vim\n",
		},
		{
			name:     "injection appended to existing user_data is removed",
			input:    "#cloud-config\npackages:\n  - vim\n\n" + snippet,
			expected: "#cloud-config\npackages:\n  - vim\n",
		},
		{
			name:     "injection into empty user_data returns empty",
			input:    "#cloud-config\n" + snippet,
			expected: "",
		},
		{
			name:     "empty user_data stays empty",
			input:    "",
			expected: "",
		},
	}
	for _, tc := range testcases {
		if got := stripGuestAgentSnippet(tc.input); got != tc.expected {
			t.Errorf("%s: stripGuestAgentSnippet(%q) = %q, want %q", tc.name, tc.input, got, tc.expected)
		}
	}
}
