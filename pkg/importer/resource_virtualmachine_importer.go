package importer

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/harvester/harvester/pkg/builder"
	harvesterutil "github.com/harvester/harvester/pkg/util"

	"github.com/harvester/terraform-provider-harvester/pkg/constants"
	"github.com/harvester/terraform-provider-harvester/pkg/helper"
)

type VMImporter struct {
	VirtualMachine         *kubevirtv1.VirtualMachine
	VirtualMachineInstance *kubevirtv1.VirtualMachineInstance
}

func (v *VMImporter) Name() string {
	return v.VirtualMachine.Name
}

func (v *VMImporter) Namespace() string {
	return v.VirtualMachine.Namespace
}

func (v *VMImporter) MachineType() string {
	return v.VirtualMachine.Spec.Template.Spec.Domain.Machine.Type
}

func (v *VMImporter) HostName() string {
	return v.VirtualMachine.Spec.Template.Spec.Hostname
}

func (v *VMImporter) ReservedMemory() string {
	return v.VirtualMachine.Annotations[harvesterutil.AnnotationReservedMemory]
}

func (v *VMImporter) Description() string {
	return v.VirtualMachine.Annotations[builder.AnnotationKeyDescription]
}

func (v *VMImporter) Memory() string {
	return v.VirtualMachine.Spec.Template.Spec.Domain.Resources.Limits.Memory().String()
}

func (v *VMImporter) CPU() int {
	return int(v.VirtualMachine.Spec.Template.Spec.Domain.CPU.Cores)
}

func (v *VMImporter) CPUModel() string {
	return v.VirtualMachine.Spec.Template.Spec.Domain.CPU.Model
}

func (v *VMImporter) Requests() []map[string]interface{} {
	req := v.VirtualMachine.Spec.Template.Spec.Domain.Resources.Requests
	result := map[string]interface{}{
		constants.FieldRequestsCPU:    "",
		constants.FieldRequestsMemory: "",
	}
	if req != nil {
		if cpu, ok := req[corev1.ResourceCPU]; ok {
			result[constants.FieldRequestsCPU] = cpu.String()
		}
		if mem, ok := req[corev1.ResourceMemory]; ok {
			result[constants.FieldRequestsMemory] = mem.String()
		}
	}
	return []map[string]interface{}{result}
}

func (v *VMImporter) DedicatedCPUPlacement() bool {
	return bool(v.VirtualMachine.Spec.Template.Spec.Domain.CPU.DedicatedCPUPlacement)
}

func (v *VMImporter) IsolateEmulatorThread() bool {
	return bool(v.VirtualMachine.Spec.Template.Spec.Domain.CPU.IsolateEmulatorThread)
}

func (v *VMImporter) EFI() bool {
	firmware := v.VirtualMachine.Spec.Template.Spec.Domain.Firmware
	return firmware != nil && firmware.Bootloader != nil && firmware.Bootloader.EFI != nil
}

func (v *VMImporter) SecureBoot() bool {
	return v.EFI() && *v.VirtualMachine.Spec.Template.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot
}

func (v *VMImporter) HugepagesSize() string {
	memory := v.VirtualMachine.Spec.Template.Spec.Domain.Memory
	if memory == nil || memory.Hugepages == nil {
		return ""
	}
	return memory.Hugepages.PageSize
}

func (v *VMImporter) CPUSockets() int {
	s := int(v.VirtualMachine.Spec.Template.Spec.Domain.CPU.Sockets)
	if s == 0 {
		return 1
	}
	return s
}

func (v *VMImporter) CPUThreads() int {
	t := int(v.VirtualMachine.Spec.Template.Spec.Domain.CPU.Threads)
	if t == 0 {
		return 1
	}
	return t
}

func (v *VMImporter) EvictionStrategy() string {
	if v.VirtualMachine.Spec.Template.Spec.EvictionStrategy == nil {
		return constants.DefaultEvictionStrategy
	}
	return string(*v.VirtualMachine.Spec.Template.Spec.EvictionStrategy)
}

func (v *VMImporter) TerminationGracePeriodSeconds() int {
	if v.VirtualMachine.Spec.Template.Spec.TerminationGracePeriodSeconds == nil {
		return constants.DefaultTerminationGracePeriodSeconds
	}
	return int(*v.VirtualMachine.Spec.Template.Spec.TerminationGracePeriodSeconds)
}

func (v *VMImporter) OSType() string {
	return v.VirtualMachine.Annotations[constants.AnnotationOSType]
}

func (v *VMImporter) SSHKeys() ([]string, error) {
	var sshKeys []string
	sshNames := v.VirtualMachine.Spec.Template.ObjectMeta.Annotations[builder.AnnotationKeyVirtualMachineSSHNames]
	if err := json.Unmarshal([]byte(sshNames), &sshKeys); err != nil {
		return nil, err
	}
	for i, sshKey := range sshKeys {
		sshKeyNamespacedName, err := helper.RebuildNamespacedName(sshKey, v.Namespace())
		if err != nil {
			return nil, err
		}
		sshKeys[i] = sshKeyNamespacedName
	}
	return sshKeys, nil
}

func (v *VMImporter) Input() ([]map[string]interface{}, error) {
	inputs := v.VirtualMachine.Spec.Template.Spec.Domain.Devices.Inputs
	var inputStates = make([]map[string]interface{}, 0, len(inputs))
	for _, input := range inputs {
		inputState := map[string]interface{}{
			constants.FieldInputName: input.Name,
			constants.FieldInputType: input.Type,
			constants.FieldInputBus:  input.Bus,
		}
		inputStates = append(inputStates, inputState)
	}
	return inputStates, nil
}

func (v *VMImporter) TPM() []map[string]interface{} {
	tpm := v.VirtualMachine.Spec.Template.Spec.Domain.Devices.TPM
	tpmStates := make([]map[string]interface{}, 0, 1)
	if tpm != nil {
		tpmState := map[string]interface{}{}
		tpmStates = append(tpmStates, tpmState)
	}
	return tpmStates
}

func (v *VMImporter) NetworkInterface() ([]map[string]interface{}, error) {
	var (
		waitForLeaseInterfaces   []string
		waitForLeaseInterfaceMap = map[string]struct{}{}
	)

	waitForLeaseInterfaceNames := v.VirtualMachine.Spec.Template.ObjectMeta.Annotations[builder.AnnotationKeyVirtualMachineWaitForLeaseInterfaceNames]
	if waitForLeaseInterfaceNames != "" {
		if err := json.Unmarshal([]byte(waitForLeaseInterfaceNames), &waitForLeaseInterfaces); err != nil {
			return nil, err
		}
		for _, waitForLeaseInterface := range waitForLeaseInterfaces {
			waitForLeaseInterfaceMap[waitForLeaseInterface] = struct{}{}
		}
	}

	interfaceStatusMap := map[string]kubevirtv1.VirtualMachineInstanceNetworkInterface{}
	if v.VirtualMachineInstance != nil {
		interfaceStatuses := v.VirtualMachineInstance.Status.Interfaces
		for _, interfaceStatus := range interfaceStatuses {
			interfaceStatusMap[interfaceStatus.Name] = interfaceStatus
		}
	}

	interfaces := v.VirtualMachine.Spec.Template.Spec.Domain.Devices.Interfaces
	var networkInterfaceStates = make([]map[string]interface{}, 0, len(interfaces))
	for _, networkInterface := range interfaces {
		var interfaceType string
		if networkInterface.Bridge != nil {
			interfaceType = builder.NetworkInterfaceTypeBridge
		} else if networkInterface.Masquerade != nil {
			interfaceType = builder.NetworkInterfaceTypeMasquerade
		} else {
			return nil, fmt.Errorf("unsupported type found on network %s. ", networkInterface.Name)
		}
		var networkName string
		for _, network := range v.VirtualMachine.Spec.Template.Spec.Networks {
			if network.Name == networkInterface.Name {
				if network.Multus != nil {
					networkName = network.Multus.NetworkName
				}
				break
			}
		}

		networkInterfaceState := map[string]interface{}{
			constants.FieldNetworkInterfaceName:        networkInterface.Name,
			constants.FieldNetworkInterfaceType:        interfaceType,
			constants.FieldNetworkInterfaceModel:       networkInterface.Model,
			constants.FieldNetworkInterfaceMACAddress:  networkInterface.MacAddress,
			constants.FieldNetworkInterfaceNetworkName: networkName,
			constants.FieldNetworkInterfaceBootOrder:   networkInterface.BootOrder,
		}
		if interfaceStatus, ok := interfaceStatusMap[networkInterface.Name]; ok {
			// disregard any link-local addresses
			ips := slices.DeleteFunc(
				slices.DeleteFunc(interfaceStatus.IPs, helper.IsIPv6LinkLocal),
				helper.IsIPv4LinkLocal)
			slices.Sort(ips)
			if len(ips) > 0 {
				networkInterfaceState[constants.FieldNetworkInterfaceIPAddress] = slices.Min(ips)
				networkInterfaceState[constants.FieldNetworkInterfaceInterfaceName] = interfaceStatus.InterfaceName
			}
		}
		_, ok := waitForLeaseInterfaceMap[networkInterface.Name]
		networkInterfaceState[constants.FieldNetworkInterfaceWaitForLease] = ok
		networkInterfaceStates = append(networkInterfaceStates, networkInterfaceState)
	}
	return networkInterfaceStates, nil
}

func (v *VMImporter) pvcVolume(volume kubevirtv1.Volume, state map[string]interface{}) error {
	pvc := volume.PersistentVolumeClaim
	pvcName := pvc.ClaimName
	state[constants.FieldDiskVolumeName] = pvcName
	state[constants.FieldDiskHotPlug] = pvc.Hotpluggable
	var (
		isInPVCTemplates bool
		pvcTemplates     []*corev1.PersistentVolumeClaim
	)
	volumeClaimTemplates := v.VirtualMachine.Annotations[harvesterutil.AnnotationVolumeClaimTemplates]
	if volumeClaimTemplates != "" {
		if err := json.Unmarshal([]byte(volumeClaimTemplates), &pvcTemplates); err != nil {
			return err
		}
		for _, pvcTemplate := range pvcTemplates {
			if pvcTemplate.Name == pvcName {
				state[constants.FieldDiskSize] = pvcTemplate.Spec.Resources.Requests.Storage().String()
				if imageID := pvcTemplate.Annotations[builder.AnnotationKeyImageID]; imageID != "" {
					imageNamespacedName, err := helper.RebuildNamespacedName(imageID, v.Namespace())
					if err != nil {
						return err
					}
					state[constants.FieldVolumeImage] = imageNamespacedName
				}
				if pvcTemplate.Spec.VolumeMode != nil {
					state[constants.FieldVolumeMode] = string(*pvcTemplate.Spec.VolumeMode)
				}
				if accessModes := pvcTemplate.Spec.AccessModes; len(accessModes) > 0 {
					state[constants.FieldVolumeAccessMode] = string(pvcTemplate.Spec.AccessModes[0])
				}
				if pvcTemplate.Spec.StorageClassName != nil {
					state[constants.FieldVolumeStorageClassName] = *pvcTemplate.Spec.StorageClassName
				}
				state[constants.FieldDiskAutoDelete] = pvcTemplate.Annotations[constants.AnnotationDiskAutoDelete] == "true"
				isInPVCTemplates = true
				break
			}
		}
	}
	if !isInPVCTemplates {
		state[constants.FieldDiskExistingVolumeName] = pvcName
	}
	return nil
}

// stripInjectedSSHUser removes the "user: <name>" line the provider appends to
// cloudinit.user_data from the VM's `ssh-user` tag on create, so reading the
// VM back does not report a permanent diff.
func (v *VMImporter) stripInjectedSSHUser(userData string) string {
	sshUser := v.VirtualMachine.Labels[builder.LabelPrefixHarvesterTag+constants.LabelSSHUsername]
	if sshUser == "" {
		return userData
	}
	if userData == fmt.Sprintf("#cloud-config\nuser: %s\n", sshUser) {
		return ""
	}
	if suffix := fmt.Sprintf("\nuser: %s\n", sshUser); strings.HasSuffix(userData, suffix) {
		return strings.TrimSuffix(userData, suffix)
	}
	return userData
}

// stripGuestAgentSnippet reverses the install_guest_agent injection so the
// user_data field remains idempotent across reads. The snippet is either set as
// the whole user_data (when the configured user_data was empty) or appended with
// a leading newline to the configured user_data.
func stripGuestAgentSnippet(userData string) string {
	if userData == "#cloud-config\n"+constants.GuestAgentCloudInitSnippet {
		return ""
	}
	if suffix := "\n" + constants.GuestAgentCloudInitSnippet; strings.HasSuffix(userData, suffix) {
		return strings.TrimSuffix(userData, suffix)
	}
	return userData
}

func (v *VMImporter) cloudInit(volume kubevirtv1.Volume) []map[string]interface{} {
	var cloudInitState = make([]map[string]interface{}, 0, 1)
	if volume.CloudInitNoCloud != nil {
		cloudInitState = append(cloudInitState, map[string]interface{}{
			constants.FieldCloudInitType:              builder.CloudInitTypeNoCloud,
			constants.FieldCloudInitUserData:          v.stripInjectedSSHUser(stripGuestAgentSnippet(volume.CloudInitNoCloud.UserData)),
			constants.FieldCloudInitUserDataBase64:    volume.CloudInitNoCloud.UserDataBase64,
			constants.FieldCloudInitNetworkData:       volume.CloudInitNoCloud.NetworkData,
			constants.FieldCloudInitNetworkDataBase64: volume.CloudInitNoCloud.NetworkDataBase64,
		})
		if volume.CloudInitNoCloud.UserDataSecretRef != nil {
			cloudInitState[0][constants.FieldCloudInitUserDataSecretName] = volume.CloudInitNoCloud.UserDataSecretRef.Name
		}
		if volume.CloudInitNoCloud.NetworkDataSecretRef != nil {
			cloudInitState[0][constants.FieldCloudInitNetworkDataSecretName] = volume.CloudInitNoCloud.NetworkDataSecretRef.Name
		}
	} else if volume.CloudInitConfigDrive != nil {
		cloudInitState = append(cloudInitState, map[string]interface{}{
			constants.FieldCloudInitType:              builder.CloudInitTypeConfigDrive,
			constants.FieldCloudInitUserData:          v.stripInjectedSSHUser(stripGuestAgentSnippet(volume.CloudInitConfigDrive.UserData)),
			constants.FieldCloudInitUserDataBase64:    volume.CloudInitConfigDrive.UserDataBase64,
			constants.FieldCloudInitNetworkData:       volume.CloudInitConfigDrive.NetworkData,
			constants.FieldCloudInitNetworkDataBase64: volume.CloudInitConfigDrive.NetworkDataBase64,
		})
		if volume.CloudInitConfigDrive.UserDataSecretRef != nil {
			cloudInitState[0][constants.FieldCloudInitUserDataSecretName] = volume.CloudInitConfigDrive.UserDataSecretRef.Name
		}
		if volume.CloudInitConfigDrive.NetworkDataSecretRef != nil {
			cloudInitState[0][constants.FieldCloudInitNetworkDataSecretName] = volume.CloudInitConfigDrive.NetworkDataSecretRef.Name
		}
	}
	return cloudInitState
}

func (v *VMImporter) Volume() ([]map[string]interface{}, []map[string]interface{}, error) {
	var (
		disks          = v.VirtualMachine.Spec.Template.Spec.Domain.Devices.Disks
		volumes        = v.VirtualMachine.Spec.Template.Spec.Volumes
		volumesMap     = make(map[string]kubevirtv1.Volume, len(volumes))
		cloudInitState = make([]map[string]interface{}, 0, 1)
		diskStates     = make([]map[string]interface{}, 0, len(disks))
	)

	for _, volume := range volumes {
		volumesMap[volume.Name] = volume
	}

	for _, disk := range disks {
		volume, hasVolume := volumesMap[disk.Name]

		// The cloud-init disk is represented by the cloudinit block, not by an
		// explicit disk block, so handle it separately and do not report it as a
		// disk (otherwise it would drift on every plan).
		if hasVolume && (volume.CloudInitNoCloud != nil || volume.CloudInitConfigDrive != nil) {
			cloudInitState = v.cloudInit(volume)
			continue
		}

		diskState := make(map[string]interface{})
		switch {
		case disk.CDRom != nil:
			diskState[constants.FieldDiskType] = builder.DiskTypeCDRom
			diskState[constants.FieldDiskBus] = string(disk.CDRom.Bus)
			diskState[constants.FieldDiskEject] = disk.CDRom.Tray == kubevirtv1.TrayStateOpen
		case disk.Disk != nil:
			diskState[constants.FieldDiskType] = builder.DiskTypeDisk
			diskState[constants.FieldDiskBus] = string(disk.Disk.Bus)
			diskState[constants.FieldDiskEject] = false
		default:
			return nil, nil, fmt.Errorf("unsupported disk type found on disk %s. ", disk.Name)
		}
		diskState[constants.FieldDiskName] = disk.Name
		diskState[constants.FieldDiskBootOrder] = disk.BootOrder
		diskState[constants.FieldDiskCacheMode] = disk.Cache

		if hasVolume {
			switch {
			case volume.PersistentVolumeClaim != nil:
				if err := v.pvcVolume(volume, diskState); err != nil {
					return nil, nil, err
				}
			case volume.ContainerDisk != nil:
				diskState[constants.FieldDiskContainerImageName] = volume.ContainerDisk.Image
			case volume.Sysprep != nil:
				if volume.Sysprep.Secret != nil {
					diskState[constants.FieldDiskSysprepSecretName] = volume.Sysprep.Secret.Name
				}
				if volume.Sysprep.ConfigMap != nil {
					diskState[constants.FieldDiskSysprepConfigMapName] = volume.Sysprep.ConfigMap.Name
				}
			case volume.ConfigMap != nil:
				diskState[constants.FieldDiskConfigMapName] = volume.ConfigMap.Name
			case volume.Secret != nil:
				diskState[constants.FieldDiskSecretName] = volume.Secret.SecretName
			default:
				return nil, nil, fmt.Errorf("unsupported volume type found on volume %s. ", volume.Name)
			}
		}
		diskStates = append(diskStates, diskState)
	}
	return diskStates, cloudInitState, nil
}

func (v *VMImporter) Tolerations() []map[string]interface{} {
	tolerations := v.VirtualMachine.Spec.Template.Spec.Tolerations
	result := make([]map[string]interface{}, 0, len(tolerations))
	for _, t := range tolerations {
		tolMap := map[string]interface{}{
			constants.FieldTolerationKey:               t.Key,
			constants.FieldTolerationOperator:          string(t.Operator),
			constants.FieldTolerationValue:             t.Value,
			constants.FieldTolerationEffect:            string(t.Effect),
			constants.FieldTolerationTolerationSeconds: 0,
		}
		if t.TolerationSeconds != nil {
			tolMap[constants.FieldTolerationTolerationSeconds] = int(*t.TolerationSeconds)
		}
		result = append(result, tolMap)
	}
	return result
}

func (v *VMImporter) InstallGuestAgent() bool {
	for _, volume := range v.VirtualMachine.Spec.Template.Spec.Volumes {
		var userData string
		if volume.CloudInitNoCloud != nil {
			userData = volume.CloudInitNoCloud.UserData
		} else if volume.CloudInitConfigDrive != nil {
			userData = volume.CloudInitConfigDrive.UserData
		}
		if userData != "" && strings.Contains(userData, "qemu-guest-agent") {
			return true
		}
	}
	return false
}

func (v *VMImporter) AccessCredentials() []map[string]interface{} {
	acs := v.VirtualMachine.Spec.Template.Spec.AccessCredentials
	result := make([]map[string]interface{}, 0, len(acs))
	for _, ac := range acs {
		entry := map[string]interface{}{}
		if ac.SSHPublicKey != nil {
			ssh := map[string]interface{}{}
			if ac.SSHPublicKey.Source.Secret != nil {
				ssh[constants.FieldAccessCredentialSecretName] = ac.SSHPublicKey.Source.Secret.SecretName
			}
			pm := ac.SSHPublicKey.PropagationMethod
			switch {
			case pm.ConfigDrive != nil:
				ssh[constants.FieldAccessCredentialPropagationMethod] = "configDrive"
			case pm.NoCloud != nil:
				ssh[constants.FieldAccessCredentialPropagationMethod] = "noCloud"
			case pm.QemuGuestAgent != nil:
				ssh[constants.FieldAccessCredentialPropagationMethod] = "qemuGuestAgent"
				ssh[constants.FieldAccessCredentialUsers] = pm.QemuGuestAgent.Users
			}
			entry[constants.FieldAccessCredentialSSHPublicKey] = []interface{}{ssh}
			entry[constants.FieldAccessCredentialUserPassword] = []interface{}{}
		} else if ac.UserPassword != nil {
			pw := map[string]interface{}{}
			if ac.UserPassword.Source.Secret != nil {
				pw[constants.FieldAccessCredentialSecretName] = ac.UserPassword.Source.Secret.SecretName
			}
			entry[constants.FieldAccessCredentialUserPassword] = []interface{}{pw}
			entry[constants.FieldAccessCredentialSSHPublicKey] = []interface{}{}
		}
		result = append(result, entry)
	}
	return result
}

func (v *VMImporter) DNSPolicy() string {
	return string(v.VirtualMachine.Spec.Template.Spec.DNSPolicy)
}

func (v *VMImporter) DNSConfig() []map[string]interface{} {
	dc := v.VirtualMachine.Spec.Template.Spec.DNSConfig
	if dc == nil {
		return nil
	}
	result := map[string]interface{}{
		constants.FieldDNSConfigNameservers: dc.Nameservers,
		constants.FieldDNSConfigSearches:    dc.Searches,
	}
	opts := make([]map[string]interface{}, 0, len(dc.Options))
	for _, o := range dc.Options {
		opt := map[string]interface{}{
			constants.FieldDNSOptionName: o.Name,
		}
		if o.Value != nil {
			opt[constants.FieldDNSOptionValue] = *o.Value
		} else {
			opt[constants.FieldDNSOptionValue] = ""
		}
		opts = append(opts, opt)
	}
	result[constants.FieldDNSConfigOptions] = opts
	return []map[string]interface{}{result}
}

func (v *VMImporter) Hyperv() []map[string]interface{} {
	features := v.VirtualMachine.Spec.Template.Spec.Domain.Features
	if features == nil || features.Hyperv == nil {
		return nil
	}
	hv := features.Hyperv
	getBool := func(fs *kubevirtv1.FeatureState) bool {
		return fs != nil && fs.Enabled != nil && *fs.Enabled
	}
	result := map[string]interface{}{
		constants.FieldHypervRelaxed:         getBool(hv.Relaxed),
		constants.FieldHypervVAPIC:           getBool(hv.VAPIC),
		constants.FieldHypervVPIndex:         getBool(hv.VPIndex),
		constants.FieldHypervRuntime:         getBool(hv.Runtime),
		constants.FieldHypervSyNIC:           getBool(hv.SyNIC),
		constants.FieldHypervReset:           getBool(hv.Reset),
		constants.FieldHypervFrequencies:     getBool(hv.Frequencies),
		constants.FieldHypervReenlightenment: getBool(hv.Reenlightenment),
		constants.FieldHypervTLBFlush:        getBool(hv.TLBFlush),
		constants.FieldHypervIPI:             getBool(hv.IPI),
		constants.FieldHypervEVMCS:           getBool(hv.EVMCS),
	}

	if hv.Spinlocks != nil && hv.Spinlocks.Enabled != nil && *hv.Spinlocks.Enabled {
		result[constants.FieldHypervSpinlocks] = true
		retries := 4096
		if hv.Spinlocks.Retries != nil {
			retries = int(*hv.Spinlocks.Retries)
		}
		result[constants.FieldHypervSpinlocksRetries] = retries
	} else {
		result[constants.FieldHypervSpinlocks] = false
		result[constants.FieldHypervSpinlocksRetries] = 4096
	}

	if hv.SyNICTimer != nil && hv.SyNICTimer.Enabled != nil && *hv.SyNICTimer.Enabled {
		result[constants.FieldHypervSyNICTimer] = true
		result[constants.FieldHypervSyNICTimerDirect] = getBool(hv.SyNICTimer.Direct)
	} else {
		result[constants.FieldHypervSyNICTimer] = false
		result[constants.FieldHypervSyNICTimerDirect] = false
	}

	if hv.VendorID != nil && hv.VendorID.Enabled != nil && *hv.VendorID.Enabled {
		result[constants.FieldHypervVendorID] = true
		result[constants.FieldHypervVendorIDValue] = hv.VendorID.VendorID
	} else {
		result[constants.FieldHypervVendorID] = false
		result[constants.FieldHypervVendorIDValue] = ""
	}

	return []map[string]interface{}{result}
}

func (v *VMImporter) HypervPassthrough() bool {
	features := v.VirtualMachine.Spec.Template.Spec.Domain.Features
	return features != nil && features.HypervPassthrough != nil &&
		features.HypervPassthrough.Enabled != nil && *features.HypervPassthrough.Enabled
}

func (v *VMImporter) Clock() []map[string]interface{} {
	clock := v.VirtualMachine.Spec.Template.Spec.Domain.Clock
	if clock == nil {
		return nil
	}
	result := map[string]interface{}{
		constants.FieldClockTimezone:         "",
		constants.FieldClockUTCOffsetSeconds: 0,
	}
	if clock.Timezone != nil {
		result[constants.FieldClockTimezone] = string(*clock.Timezone)
	}
	if clock.UTC != nil && clock.UTC.OffsetSeconds != nil {
		result[constants.FieldClockUTCOffsetSeconds] = *clock.UTC.OffsetSeconds
	}

	if clock.Timer != nil {
		result[constants.FieldClockTimer] = []interface{}{importClockTimers(clock.Timer)}
	} else {
		result[constants.FieldClockTimer] = []interface{}{}
	}

	return []map[string]interface{}{result}
}

func importClockTimers(t *kubevirtv1.Timer) map[string]interface{} {
	timer := map[string]interface{}{}

	if t.HPET != nil {
		h := map[string]interface{}{
			constants.FieldTimerEnabled:    t.HPET.Enabled == nil || *t.HPET.Enabled,
			constants.FieldTimerTickPolicy: string(t.HPET.TickPolicy),
		}
		timer[constants.FieldTimerHPET] = []interface{}{h}
	} else {
		timer[constants.FieldTimerHPET] = []interface{}{}
	}

	if t.KVM != nil {
		timer[constants.FieldTimerKVM] = []interface{}{map[string]interface{}{
			constants.FieldTimerEnabled: t.KVM.Enabled == nil || *t.KVM.Enabled,
		}}
	} else {
		timer[constants.FieldTimerKVM] = []interface{}{}
	}

	if t.PIT != nil {
		p := map[string]interface{}{
			constants.FieldTimerEnabled:    t.PIT.Enabled == nil || *t.PIT.Enabled,
			constants.FieldTimerTickPolicy: string(t.PIT.TickPolicy),
		}
		timer[constants.FieldTimerPIT] = []interface{}{p}
	} else {
		timer[constants.FieldTimerPIT] = []interface{}{}
	}

	if t.RTC != nil {
		r := map[string]interface{}{
			constants.FieldTimerEnabled:    t.RTC.Enabled == nil || *t.RTC.Enabled,
			constants.FieldTimerTickPolicy: string(t.RTC.TickPolicy),
			constants.FieldTimerTrack:      string(t.RTC.Track),
		}
		timer[constants.FieldTimerRTC] = []interface{}{r}
	} else {
		timer[constants.FieldTimerRTC] = []interface{}{}
	}

	if t.Hyperv != nil {
		timer[constants.FieldTimerHyperv] = []interface{}{map[string]interface{}{
			constants.FieldTimerEnabled: t.Hyperv.Enabled == nil || *t.Hyperv.Enabled,
		}}
	} else {
		timer[constants.FieldTimerHyperv] = []interface{}{}
	}

	return timer
}

func (v *VMImporter) NodeName() string {
	if v.VirtualMachineInstance == nil {
		return ""
	}
	return v.VirtualMachineInstance.Status.NodeName
}

// exportLabelSelectorRequirements exports label selector requirements to Terraform state format
func exportLabelSelectorRequirements(requirements []metav1.LabelSelectorRequirement) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(requirements))
	for _, req := range requirements {
		values := make([]interface{}, 0, len(req.Values))
		for _, v := range req.Values {
			values = append(values, v)
		}
		result = append(result, map[string]interface{}{
			constants.FieldExpressionKey:      req.Key,
			constants.FieldExpressionOperator: string(req.Operator),
			constants.FieldExpressionValues:   values,
		})
	}
	return result
}

// exportLabelSelector exports a label selector to Terraform state format
func exportLabelSelector(selector *metav1.LabelSelector) []map[string]interface{} {
	if selector == nil {
		return nil
	}
	result := map[string]interface{}{}

	if len(selector.MatchLabels) > 0 {
		matchLabels := make(map[string]interface{})
		for k, v := range selector.MatchLabels {
			matchLabels[k] = v
		}
		result[constants.FieldMatchLabels] = matchLabels
	}

	if len(selector.MatchExpressions) > 0 {
		result[constants.FieldMatchExpressions] = exportLabelSelectorRequirements(selector.MatchExpressions)
	}

	if len(result) == 0 {
		return nil
	}
	return []map[string]interface{}{result}
}

// isHarvesterInjectedNodeSelector returns true if the requirement was injected
// by the Harvester webhook. The whole `network.harvesterhci.io/` prefix is
// matched because Harvester injects one readiness expression per cluster
// network the VM is attached to (e.g. network.harvesterhci.io/mgmt), keyed by
// the cluster network name.
func isHarvesterInjectedNodeSelector(req corev1.NodeSelectorRequirement) bool {
	return strings.HasPrefix(req.Key, "network.harvesterhci.io/")
}

// exportNodeSelectorRequirements exports node selector requirements to Terraform state format,
// filtering out Harvester-injected expressions (e.g. network.harvesterhci.io/mgmt)
func exportNodeSelectorRequirements(requirements []corev1.NodeSelectorRequirement) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(requirements))
	for _, req := range requirements {
		if isHarvesterInjectedNodeSelector(req) {
			continue
		}
		values := make([]interface{}, 0, len(req.Values))
		for _, v := range req.Values {
			values = append(values, v)
		}
		result = append(result, map[string]interface{}{
			constants.FieldExpressionKey:      req.Key,
			constants.FieldExpressionOperator: string(req.Operator),
			constants.FieldExpressionValues:   values,
		})
	}
	return result
}

// exportNodeSelectorTerms exports node selector terms to Terraform state format.
// Terms that end up empty once the Harvester-injected expressions are filtered
// out (e.g. the term added for VMs on the mgmt cluster network) are dropped
// entirely, otherwise a VM without user affinity would report a phantom empty
// term and drift on every plan.
func exportNodeSelectorTerms(terms []corev1.NodeSelectorTerm) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(terms))
	for _, term := range terms {
		termMap := map[string]interface{}{}
		if expressions := exportNodeSelectorRequirements(term.MatchExpressions); len(expressions) > 0 {
			termMap[constants.FieldMatchExpressions] = expressions
		}
		if fields := exportNodeSelectorRequirements(term.MatchFields); len(fields) > 0 {
			termMap[constants.FieldMatchFields] = fields
		}
		if len(termMap) == 0 {
			continue
		}
		result = append(result, termMap)
	}
	return result
}

// exportPodAffinityTerms exports pod affinity terms to Terraform state format,
// filtering out Harvester-injected terms
func exportPodAffinityTerms(terms []corev1.PodAffinityTerm) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(terms))
	for _, term := range terms {
		if isHarvesterInjectedPodAffinityTerm(term) {
			continue
		}
		termMap := map[string]interface{}{
			constants.FieldTopologyKey: term.TopologyKey,
		}
		if term.LabelSelector != nil {
			termMap[constants.FieldLabelSelector] = exportLabelSelector(term.LabelSelector)
		}
		if len(term.Namespaces) > 0 {
			namespaces := make([]interface{}, 0, len(term.Namespaces))
			for _, ns := range term.Namespaces {
				namespaces = append(namespaces, ns)
			}
			termMap[constants.FieldNamespaces] = namespaces
		}
		if term.NamespaceSelector != nil {
			termMap[constants.FieldNamespaceSelector] = exportLabelSelector(term.NamespaceSelector)
		}
		result = append(result, termMap)
	}
	return result
}

// isHarvesterInjectedPodAffinityTerm returns true if the term was injected by
// the Harvester webhook. Only the exact `harvesterhci.io/creator` key is
// matched: users may legitimately select pods by other harvesterhci.io labels
// (e.g. harvesterhci.io/vmName) and those terms must be kept.
func isHarvesterInjectedPodAffinityTerm(term corev1.PodAffinityTerm) bool {
	if term.LabelSelector == nil {
		return false
	}
	for _, expr := range term.LabelSelector.MatchExpressions {
		if expr.Key == "harvesterhci.io/creator" {
			return true
		}
	}
	return false
}

// exportWeightedPodAffinityTerms exports weighted pod affinity terms to Terraform state format,
// filtering out Harvester-injected terms (e.g. harvesterhci.io/creator)
func exportWeightedPodAffinityTerms(terms []corev1.WeightedPodAffinityTerm) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(terms))
	for _, term := range terms {
		if isHarvesterInjectedPodAffinityTerm(term.PodAffinityTerm) {
			continue
		}
		termMap := map[string]interface{}{
			constants.FieldPreferredWeight: int(term.Weight),
			constants.FieldPodAffinityTerm: exportPodAffinityTerms([]corev1.PodAffinityTerm{term.PodAffinityTerm}),
		}
		result = append(result, termMap)
	}
	return result
}

// NodeAffinity exports node affinity to Terraform state format
func (v *VMImporter) NodeAffinity() []map[string]interface{} {
	affinity := v.VirtualMachine.Spec.Template.Spec.Affinity
	if affinity == nil || affinity.NodeAffinity == nil {
		return nil
	}
	nodeAffinity := affinity.NodeAffinity
	result := map[string]interface{}{}

	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		terms := exportNodeSelectorTerms(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		if len(terms) > 0 {
			result[constants.FieldNodeAffinityRequired] = []map[string]interface{}{
				{constants.FieldNodeSelectorTerm: terms},
			}
		}
	}

	if len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		preferred := make([]map[string]interface{}, 0, len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution))
		for _, pref := range nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			prefMap := map[string]interface{}{
				constants.FieldPreferredWeight:     int(pref.Weight),
				constants.FieldPreferredPreference: exportNodeSelectorTerms([]corev1.NodeSelectorTerm{pref.Preference}),
			}
			preferred = append(preferred, prefMap)
		}
		result[constants.FieldNodeAffinityPreferred] = preferred
	}

	if len(result) == 0 {
		return nil
	}
	return []map[string]interface{}{result}
}

// PodAffinity exports pod affinity to Terraform state format
func (v *VMImporter) PodAffinity() []map[string]interface{} {
	affinity := v.VirtualMachine.Spec.Template.Spec.Affinity
	if affinity == nil || affinity.PodAffinity == nil {
		return nil
	}
	podAffinity := affinity.PodAffinity
	result := map[string]interface{}{}

	if len(podAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
		if terms := exportPodAffinityTerms(podAffinity.RequiredDuringSchedulingIgnoredDuringExecution); len(terms) > 0 {
			result[constants.FieldPodAffinityRequired] = terms
		}
	}

	if len(podAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		if terms := exportWeightedPodAffinityTerms(podAffinity.PreferredDuringSchedulingIgnoredDuringExecution); len(terms) > 0 {
			result[constants.FieldPodAffinityPreferred] = terms
		}
	}

	if len(result) == 0 {
		return nil
	}
	return []map[string]interface{}{result}
}

// PodAntiAffinity exports pod anti-affinity to Terraform state format
func (v *VMImporter) PodAntiAffinity() []map[string]interface{} {
	affinity := v.VirtualMachine.Spec.Template.Spec.Affinity
	if affinity == nil || affinity.PodAntiAffinity == nil {
		return nil
	}
	podAntiAffinity := affinity.PodAntiAffinity
	result := map[string]interface{}{}

	if len(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
		if terms := exportPodAffinityTerms(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution); len(terms) > 0 {
			result[constants.FieldPodAffinityRequired] = terms
		}
	}

	if len(podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		if terms := exportWeightedPodAffinityTerms(podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution); len(terms) > 0 {
			result[constants.FieldPodAffinityPreferred] = terms
		}
	}

	if len(result) == 0 {
		return nil
	}
	return []map[string]interface{}{result}
}

func (v *VMImporter) State(networkInterfaces []map[string]interface{}, oldInstanceUID string) string {
	if v.VirtualMachineInstance == nil {
		return constants.StateVirtualMachineStopped
	}
	switch v.VirtualMachineInstance.Status.Phase {
	case "Pending", "Scheduling", "Scheduled":
		return constants.StateVirtualMachineStarting
	case "Running":
		if string(v.VirtualMachineInstance.UID) == oldInstanceUID {
			return constants.StateVirtualMachineRunning
		}

		for _, networkInterface := range networkInterfaces {
			wait_for_lease := networkInterface[constants.FieldNetworkInterfaceWaitForLease].(bool)
			ip_string, has_ip_string := networkInterface[constants.FieldNetworkInterfaceIPAddress].(string)
			if wait_for_lease && (!has_ip_string || ip_string == "") {
				return constants.StateVirtualMachineRunning
			}
		}
		return constants.StateCommonReady
	case "Succeeded":
		return constants.StateVirtualMachineStopping
	case "Failed":
		return constants.StateCommonFailed
	default:
		return constants.StateCommonUnknown
	}
}

func NewVMImporter(vm *kubevirtv1.VirtualMachine, vmi *kubevirtv1.VirtualMachineInstance) *VMImporter {
	return &VMImporter{
		VirtualMachine:         vm,
		VirtualMachineInstance: vmi,
	}
}

func ResourceVirtualMachineStateGetter(vm *kubevirtv1.VirtualMachine, vmi *kubevirtv1.VirtualMachineInstance, oldInstanceUID string) (*StateGetter, error) {
	vmImporter := NewVMImporter(vm, vmi)
	networkInterface, err := vmImporter.NetworkInterface()
	if err != nil {
		return nil, err
	}
	disk, cloudInit, err := vmImporter.Volume()
	if err != nil {
		return nil, err
	}
	input, err := vmImporter.Input()
	if err != nil {
		return nil, err
	}
	sshKeys, err := vmImporter.SSHKeys()
	if err != nil {
		return nil, err
	}
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return nil, err
	}
	return &StateGetter{
		ID:           helper.BuildID(vm.Namespace, vm.Name),
		Name:         vm.Name,
		ResourceType: constants.ResourceTypeVirtualMachine,
		States: map[string]interface{}{
			constants.FieldCommonNamespace:                     vm.Namespace,
			constants.FieldCommonName:                          vm.Name,
			constants.FieldCommonDescription:                   GetDescriptions(vm.Annotations),
			constants.FieldCommonTags:                          GetTags(vm.Labels),
			constants.FieldCommonLabels:                        GetLabels(vm.Labels),
			constants.FieldCommonState:                         vmImporter.State(networkInterface, oldInstanceUID),
			constants.FieldVirtualMachineCPU:                   vmImporter.CPU(),
			constants.FieldVirtualMachineCPUModel:              vmImporter.CPUModel(),
			constants.FieldVirtualMachineMemory:                vmImporter.Memory(),
			constants.FieldVirtualMachineRequests:              vmImporter.Requests(),
			constants.FieldVirtualMachineHostname:              vmImporter.HostName(),
			constants.FieldVirtualMachineReservedMemory:        vmImporter.ReservedMemory(),
			constants.FieldVirtualMachineMachineType:           vmImporter.MachineType(),
			constants.FieldVirtualMachineRunStrategy:           string(runStrategy),
			constants.FieldVirtualMachineNetworkInterface:      networkInterface,
			constants.FieldVirtualMachineDisk:                  disk,
			constants.FieldVirtualMachineInput:                 input,
			constants.FieldVirtualMachineTPM:                   vmImporter.TPM(),
			constants.FieldVirtualMachineCloudInit:             cloudInit,
			constants.FieldVirtualMachineSSHKeys:               sshKeys,
			constants.FieldVirtualMachineInstanceNodeName:      vmImporter.NodeName(),
			constants.FieldVirtualMachineEFI:                   vmImporter.EFI(),
			constants.FieldVirtualMachineSecureBoot:            vmImporter.SecureBoot(),
			constants.FieldVirtualMachineCPUPinning:            vmImporter.DedicatedCPUPlacement(),
			constants.FieldVirtualMachineIsolateEmulatorThread: vmImporter.IsolateEmulatorThread(),
			constants.FieldVirtualMachineNodeSelector:          vm.Spec.Template.Spec.NodeSelector,
			constants.FieldVirtualMachineNodeAffinity:          vmImporter.NodeAffinity(),
			constants.FieldVirtualMachinePodAffinity:           vmImporter.PodAffinity(),
			constants.FieldVirtualMachinePodAntiAffinity:       vmImporter.PodAntiAffinity(),
			constants.FieldVirtualMachineCPUSockets:            vmImporter.CPUSockets(),
			constants.FieldVirtualMachineCPUThreads:            vmImporter.CPUThreads(),

			constants.FieldVirtualMachineEvictionStrategy:              vmImporter.EvictionStrategy(),
			constants.FieldVirtualMachineTerminationGracePeriodSeconds: vmImporter.TerminationGracePeriodSeconds(),
			constants.FieldVirtualMachineOSType:                        vmImporter.OSType(),
			constants.FieldVirtualMachineHugepages:                     vmImporter.HugepagesSize(),
			constants.FieldVirtualMachineToleration:                    vmImporter.Tolerations(),
			constants.FieldVirtualMachineInstallGuestAgent:             vmImporter.InstallGuestAgent(),
			constants.FieldVirtualMachineAccessCredentials:             vmImporter.AccessCredentials(),
			constants.FieldVirtualMachineDNSPolicy:                     vmImporter.DNSPolicy(),
			constants.FieldVirtualMachineDNSConfig:                     vmImporter.DNSConfig(),
			constants.FieldVirtualMachineHyperv:                        vmImporter.Hyperv(),
			constants.FieldVirtualMachineHypervPassthrough:             vmImporter.HypervPassthrough(),
			constants.FieldVirtualMachineClock:                         vmImporter.Clock(),
		},
	}, nil
}
