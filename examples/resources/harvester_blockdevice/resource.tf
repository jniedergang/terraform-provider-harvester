# Adopt a new disk discovered by the node disk manager and add it to Longhorn
resource "harvester_blockdevice" "nvme_data" {
  name      = "blockdevice-pci-0000-04-00-0-abcdef123456"
  namespace = "longhorn-system"

  description = "NVMe data drive on node-01"

  tags = {
    role = "data"
  }

  labels = {
    tier = "fast"
  }

  # Provisioned = device is formatted and used by Longhorn
  provision = true

  # A new disk has no filesystem yet and must be formatted before it can be
  # provisioned. Also required to reuse a disk that has a filesystem: its data
  # is erased.
  force_formatted = true

  # Optional: Longhorn with the V1 engine is used when the block is omitted,
  # like in the Harvester UI.
  disk_provisioner {
    longhorn {
      engine_version = "LonghornV1"
    }
  }
}

# Adopt a device without provisioning (monitoring only)
resource "harvester_blockdevice" "spare_disk" {
  name      = "blockdevice-pci-0000-05-00-0-fedcba654321"
  namespace = "longhorn-system"

  description = "Spare disk - not yet provisioned"

  provision = false
}
