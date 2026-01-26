# CLAUDE.md - Notes de Developpement

## Feature: create_initial_snapshot pour harvester_virtualmachine

### Description
Ajout d'un parametre `create_initial_snapshot` dans la ressource `harvester_virtualmachine` pour creer automatiquement un snapshot apres la creation de la VM.

### Fichiers modifies

1. **pkg/constants/constants_virtualmachine.go**
   - Ajout de la constante `FieldVirtualMachineCreateInitialSnapshot = "create_initial_snapshot"`

2. **internal/provider/virtualmachine/schema_virtualmachine.go**
   - Ajout du champ schema TypeBool, Optional, Default false

3. **internal/provider/virtualmachine/resource_virtualmachine.go**
   - Ajout des imports: `fmt`, `unstructured`, `k8sschema`, `dynamic`, `client`
   - Ajout de la variable `vmBackupGVR` pour l'API Harvester VirtualMachineBackup
   - Modification de `resourceVirtualMachineCreate` pour appeler `createInitialSnapshot`
   - Ajout de la fonction `createInitialSnapshot` qui cree un `VirtualMachineBackup` de type `snapshot`

### API utilisee

**IMPORTANT:** Harvester n'utilise PAS l'API KubeVirt `virtualmachinesnapshots` (snapshot.kubevirt.io/v1beta1).
Harvester utilise sa propre API `VirtualMachineBackup` (harvesterhci.io/v1beta1) avec deux types:
- `type: snapshot` - Snapshot local (rapide, stocke sur le meme storage)
- `type: backup` - Backup vers une cible externe (S3, NFS, etc.)

### Exemple d'utilisation

```hcl
resource "harvester_virtualmachine" "example" {
  name      = "my-vm"
  namespace = "default"

  cpu    = 2
  memory = "4Gi"

  create_initial_snapshot = true

  disk {
    name  = "root"
    size  = "20Gi"
    image = "default/ubuntu-22.04"
  }

  network_interface {
    name = "nic-1"
  }
}
```

### Comportement

- **`create_initial_snapshot = false` (defaut):** Aucun changement
- **`create_initial_snapshot = true`:**
  - La VM est creee normalement
  - On attend que la VM soit prete (Ready ou Off selon run_strategy)
  - Un snapshot nomme `{vm-name}-initial` est cree via l'API Harvester
  - Si le snapshot echoue, un warning est retourne (la VM reste valide)

### Test effectue le 2026-01-26

**Configuration de test:** `/root/workspace/TERRAFORM/TESTBACKUP/virtualmachine.tf` - VM `test-vm2`

**Resultat: SUCCES**
```
harvester_virtualmachine.test-vm2: Creation complete after 19s [id=default/test-vm2]
harvester_schedule_backup.vm2_backup: Creation complete after 0s

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.
```

**Verification du snapshot:**
```bash
$ kubectl get virtualmachinebackups.harvesterhci.io -n default
NAME                TYPE       READY   VM
test-vm2-initial    snapshot   true    test-vm2
```

**Verification de la VM:**
```bash
$ kubectl get vm -n default test-vm2 -o wide
NAME       AGE   STATUS    READY
test-vm2   30s   Running   True
```

### Notes techniques

- Le snapshot est cree via l'API Harvester `harvesterhci.io/v1beta1` (VirtualMachineBackup)
- Le type est `snapshot` (pas `backup` qui necessite une cible externe)
- Le nom du snapshot est derive du nom de la VM (`{vm-name}-initial`)
- Le snapshot n'est pas gere par Terraform (pas de delete automatique)
- En cas d'echec du snapshot, un warning est retourne mais la VM reste valide

### Provider location

Le provider de developpement est installe dans:
- `/root/terraform-provider-harvester/` (override dans ~/.terraformrc)
- Binary compile depuis: `/root/projects/terraform-provider-harvester/`

### Commandes utiles

```bash
# Compiler le provider
cd /root/projects/terraform-provider-harvester
go build -o terraform-provider-harvester

# Copier vers l'emplacement de l'override
cp terraform-provider-harvester /root/terraform-provider-harvester/

# Verifier les snapshots Harvester
KUBECONFIG=/root/workspace/TERRAFORM/TESTBACKUP/rke2.yaml kubectl get virtualmachinebackups.harvesterhci.io -n default

# Verifier les VMs
KUBECONFIG=/root/workspace/TERRAFORM/TESTBACKUP/rke2.yaml kubectl get vm -n default
```

### Historique des modifications

1. **Premiere tentative** - Utilisation de l'API KubeVirt `virtualmachinesnapshots` (snapshot.kubevirt.io/v1beta1)
   - Resultat: Echec - "snapshot feature gate not enabled"
   - Cause: Harvester n'utilise pas cette API

2. **Correction** - Utilisation de l'API Harvester `virtualmachinebackups` (harvesterhci.io/v1beta1) avec `type: snapshot`
   - Resultat: Succes - Snapshot cree correctement
