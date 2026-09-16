package constants

const (
	ResourceTypeVirtualMachine = "harvester_virtualmachine"

	FieldVirtualMachineMachineType           = "machine_type"
	FieldVirtualMachineHostname              = "hostname"
	FieldVirtualMachineReservedMemory        = "reserved_memory"
	FieldVirtualMachineRestartAfterUpdate    = "restart_after_update"
	FieldVirtualMachineStart                 = "start"
	FieldVirtualMachineRunStrategy           = "run_strategy"
	FieldVirtualMachineCPU                   = "cpu"
	FieldVirtualMachineCPUModel              = "cpu_model"
	FieldVirtualMachineMemory                = "memory"
	FieldVirtualMachineRequests              = "requests"
	FieldRequestsCPU                         = "cpu"
	FieldRequestsMemory                      = "memory"
	FieldVirtualMachineSSHKeys               = "ssh_keys"
	FieldVirtualMachineCloudInit             = "cloudinit"
	FieldVirtualMachineDisk                  = "disk"
	FieldVirtualMachineNetworkInterface      = "network_interface"
	FieldVirtualMachineInput                 = "input"
	FieldVirtualMachineTPM                   = "tpm"
	FieldVirtualMachineInstanceNodeName      = "node_name"
	FieldVirtualMachineEFI                   = "efi"
	FieldVirtualMachineSecureBoot            = "secure_boot"
	FieldVirtualMachineCPUPinning            = "cpu_pinning"
	FieldVirtualMachineIsolateEmulatorThread = "isolate_emulator_thread"
	FieldVirtualMachineNodeSelector          = "node_selector"
	FieldVirtualMachineCreateInitialSnapshot = "create_initial_snapshot"
	FieldVirtualMachineHostDevice            = "host_device"
	FieldVirtualMachineToleration            = "toleration"

	FieldTolerationKey               = "key"
	FieldTolerationOperator          = "operator"
	FieldTolerationValue             = "value"
	FieldTolerationEffect            = "effect"
	FieldTolerationTolerationSeconds = "toleration_seconds"

	// Node Affinity - Controls VM scheduling based on node labels
	// Reference: https://docs.harvesterhci.io/v1.7/vm/index/#node-scheduling
	FieldVirtualMachineNodeAffinity = "node_affinity"
	FieldNodeAffinityRequired       = "required"  // requiredDuringSchedulingIgnoredDuringExecution
	FieldNodeAffinityPreferred      = "preferred" // preferredDuringSchedulingIgnoredDuringExecution
	FieldNodeSelectorTerm           = "node_selector_term"
	FieldMatchExpressions           = "match_expressions" // Match by node labels
	FieldMatchFields                = "match_fields"      // Match by node fields
	FieldExpressionKey              = "key"
	FieldExpressionOperator         = "operator" // In, NotIn, Exists, DoesNotExist, Gt, Lt
	FieldExpressionValues           = "values"
	FieldPreferredWeight            = "weight"     // 1-100, higher means more preferred
	FieldPreferredPreference        = "preference" // Node selector term for preferred scheduling

	// Pod Affinity/Anti-Affinity - Controls VM co-location with other pods
	// Reference: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity
	FieldVirtualMachinePodAffinity     = "pod_affinity"      // Co-locate VMs with matching pods
	FieldVirtualMachinePodAntiAffinity = "pod_anti_affinity" // Separate VMs from matching pods
	FieldPodAffinityRequired           = "required"          // requiredDuringSchedulingIgnoredDuringExecution
	FieldPodAffinityPreferred          = "preferred"         // preferredDuringSchedulingIgnoredDuringExecution
	FieldLabelSelector                 = "label_selector"    // Select pods by labels
	FieldMatchLabels                   = "match_labels"      // Exact label matching
	FieldNamespaces                    = "namespaces"        // Limit to specific namespaces
	FieldNamespaceSelector             = "namespace_selector"
	FieldTopologyKey                   = "topology_key" // e.g., kubernetes.io/hostname
	FieldPodAffinityTerm               = "pod_affinity_term"

	FieldVirtualMachineCPUSockets = "cpu_sockets"
	FieldVirtualMachineCPUThreads = "cpu_threads"

	FieldVirtualMachineEvictionStrategy              = "eviction_strategy"
	FieldVirtualMachineTerminationGracePeriodSeconds = "termination_grace_period_seconds"
	FieldVirtualMachineOSType                        = "os_type"

	AnnotationOSType = "harvesterhci.io/os"

	DefaultEvictionStrategy              = "LiveMigrateIfPossible"
	DefaultTerminationGracePeriodSeconds = 30

	FieldVirtualMachineHugepages         = "hugepages"
	FieldVirtualMachineInstallGuestAgent = "install_guest_agent"
	FieldVirtualMachineAccessCredentials = "access_credentials"
	FieldVirtualMachineDNSPolicy         = "dns_policy"
	FieldVirtualMachineDNSConfig         = "dns_config"
	FieldVirtualMachineHyperv            = "hyperv"
	FieldVirtualMachineHypervPassthrough = "hyperv_passthrough" // #nosec G101
	FieldVirtualMachineClock             = "clock"

	StateVirtualMachineStarting = "Starting"
	StateVirtualMachineRunning  = "Running"
	StateVirtualMachineStopping = "Stopping"
	StateVirtualMachineStopped  = "Off"
)

const (
	ResourceVirtualMachine = "virtualmachines"
	SubresourceRestart     = "restart"
)

// GuestAgentCloudInitSnippet is appended to cloudinit.user_data when
// install_guest_agent is enabled. It is also used to strip the injection back
// out on read so the user_data field stays idempotent.
const GuestAgentCloudInitSnippet = "package_update: true\npackages:\n  - qemu-guest-agent\nruncmd:\n  - - systemctl\n    - enable\n    - '--now'\n    - qemu-ga"

const (
	FieldCloudInitType                  = "type"
	FieldCloudInitNetworkData           = "network_data"
	FieldCloudInitNetworkDataBase64     = "network_data_base64"
	FieldCloudInitNetworkDataSecretName = "network_data_secret_name"
	FieldCloudInitUserData              = "user_data"
	FieldCloudInitUserDataBase64        = "user_data_base64"
	FieldCloudInitUserDataSecretName    = "user_data_secret_name"
)

const (
	FieldNetworkInterfaceName          = "name"
	FieldNetworkInterfaceType          = "type"
	FieldNetworkInterfaceModel         = "model"
	FieldNetworkInterfaceMACAddress    = "mac_address"
	FieldNetworkInterfaceIPAddress     = "ip_address"
	FieldNetworkInterfaceInterfaceName = "interface_name"
	FieldNetworkInterfaceWaitForLease  = "wait_for_lease"
	FieldNetworkInterfaceNetworkName   = "network_name"
	FieldNetworkInterfaceBootOrder     = "boot_order"
)

const (
	FieldDiskName                 = "name"
	FieldDiskType                 = "type"
	FieldDiskSize                 = "size"
	FieldDiskBus                  = "bus"
	FieldDiskCacheMode            = "cache_mode"
	FieldDiskBootOrder            = "boot_order"
	FieldDiskExistingVolumeName   = "existing_volume_name"
	FieldDiskContainerImageName   = "container_image_name"
	FieldDiskHotPlug              = "hot_plug"
	FieldDiskAutoDelete           = "auto_delete"
	FieldDiskVolumeName           = "volume_name"
	FieldDiskEject                = "eject"
	FieldDiskConfigMapName        = "configmap_name"
	FieldDiskSecretName           = "secret_name"
	FieldDiskSysprepSecretName    = "sysprep_secret_name" // #nosec G101
	FieldDiskSysprepConfigMapName = "sysprep_configmap_name"

	AnnotationDiskAutoDelete = "terraform-provider-harvester-auto-delete"
)

const (
	FieldInputName = "name"
	FieldInputType = "type"
	FieldInputBus  = "bus"
)

const (
	FieldTPMName = "name"
)

const (
	LabelSSHUsername = "ssh-user"
)

const (
	DiskCacheModeNone         = "none"
	DiskCacheModeWriteBack    = "writeback"
	DiskCacheModeWriteThrough = "writethrough"
)

const (
	FieldHostDeviceName       = "name"
	FieldHostDeviceDeviceName = "device_name"
)

const (
	FieldAccessCredentialSSHPublicKey      = "ssh_public_key"     // #nosec G101
	FieldAccessCredentialUserPassword      = "user_password"      // #nosec G101
	FieldAccessCredentialSecretName        = "secret_name"        // #nosec G101
	FieldAccessCredentialPropagationMethod = "propagation_method" // #nosec G101
	FieldAccessCredentialUsers             = "users"
)

const (
	FieldDNSConfigNameservers = "nameservers"
	FieldDNSConfigSearches    = "searches"
	FieldDNSConfigOptions     = "options"
	FieldDNSOptionName        = "name"
	FieldDNSOptionValue       = "value"
)

const (
	FieldHypervRelaxed          = "relaxed"
	FieldHypervVAPIC            = "vapic"
	FieldHypervVPIndex          = "vpindex"
	FieldHypervRuntime          = "runtime"
	FieldHypervSyNIC            = "synic"
	FieldHypervReset            = "reset"
	FieldHypervFrequencies      = "frequencies"
	FieldHypervReenlightenment  = "reenlightenment"
	FieldHypervTLBFlush         = "tlbflush"
	FieldHypervIPI              = "ipi"
	FieldHypervEVMCS            = "evmcs"
	FieldHypervSpinlocks        = "spinlocks"
	FieldHypervSpinlocksRetries = "spinlocks_retries"
	FieldHypervSyNICTimer       = "synictimer"
	FieldHypervSyNICTimerDirect = "synictimer_direct"
	FieldHypervVendorID         = "vendorid"
	FieldHypervVendorIDValue    = "vendorid_value"
)

const (
	FieldClockTimezone         = "timezone"
	FieldClockUTCOffsetSeconds = "utc_offset_seconds"
	FieldClockTimer            = "timer"
	FieldTimerHPET             = "hpet"
	FieldTimerKVM              = "kvm"
	FieldTimerPIT              = "pit"
	FieldTimerRTC              = "rtc"
	FieldTimerHyperv           = "hyperv"
	FieldTimerEnabled          = "enabled"
	FieldTimerTickPolicy       = "tick_policy"
	FieldTimerTrack            = "track"
)
