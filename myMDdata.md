# LAB6 Terratest Notes – Terraform + Go + Azure

## 1. What this lab was about

This lab added **Terratest integration tests** to a Terraform Azure web server deployment.

Terratest is a Go testing library. It lets us test real infrastructure by running Terraform from Go code.

The full test flow was:

```text
Go test starts
  -> Terratest runs terraform init
  -> Terratest runs terraform apply
  -> Azure resources are created
  -> Go/Terratest checks VM, NIC, NIC attachment, and Ubuntu image
  -> Terratest runs terraform destroy
Go test ends with PASS
```

Final result:

```text
Destroy complete! Resources: 8 destroyed.
--- PASS: TestAzureLinuxVMCreation
PASS
ok      command-line-arguments
```

Important: Terratest destroyed the **Azure resources**, not local project files.

---

## 2. Branch used

Work was done on:

```bash
test-vm-creation
```

Check branch:

```bash
git branch --show-current
```

Switch branch:

```bash
git switch test-vm-creation
```

Push branch:

```bash
git push origin test-vm-creation
```

---

## 3. Main project files

| File | Purpose |
|---|---|
| `providers.tf` | Defines Terraform providers such as AzureRM and cloud-init. |
| `variables.tf` | Defines variables such as `labelPrefix`, `region`, and `admin_username`. |
| `main.tf` | Creates Azure resource group, VNet, subnet, NSG, public IP, NIC, and Linux VM. |
| `outputs.tf` | Prints VM name, NIC name, public IP, and resource group name. |
| `init.sh` | Startup script used by cloud-init to install Apache on the VM. |
| `test/azure_webserver_test.go` | Go Terratest file. |
| `test/go.mod` | Go module/dependency file, similar to Maven `pom.xml`. |
| `test/go.sum` | Dependency checksum file. |

---

## 4. Manual Terraform test

Before Terratest, the infrastructure was tested manually.

Login:

```bash
az login
az account show -o table
```

Run Terraform:

```bash
terraform init
terraform validate
terraform apply -var="labelPrefix=zaza0018"
```

Save outputs:

```bash
RG=$(terraform output -raw resource_group_name)
VM=$(terraform output -raw vm_name)
NIC=$(terraform output -raw nic_name)
IP=$(terraform output -raw public_ip)
```

Check resources:

```bash
az resource list   --resource-group $RG   --query "[].{name:name,type:type,location:location}"   -o table
```

Check VM:

```bash
az vm show   --resource-group $RG   --name $VM   --show-details   --query "{name:name,powerState:powerState,publicIps:publicIps,size:hardwareProfile.vmSize}"   -o table
```

Check web server:

```bash
curl -I http://$IP
```

SSH into VM:

```bash
ssh -i ~/.ssh/id_rsa azureadmin@$IP
```

Inside VM:

```bash
cat /etc/os-release
systemctl status apache2 --no-pager
exit
```

Destroy manual deployment:

```bash
terraform destroy -var="labelPrefix=zaza0018"
```

---

## 5. SSH key issue

Terraform failed first because this file did not exist:

```bash
~/.ssh/id_rsa.pub
```

Fix:

```bash
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
```

Check:

```bash
ls -la ~/.ssh
```

Expected:

```text
id_rsa
id_rsa.pub
```

Meaning:

| File | Meaning |
|---|---|
| `id_rsa` | Private key. Never share or commit. |
| `id_rsa.pub` | Public key. Safe to use in Terraform. |

In `main.tf`, the public key line was changed to:

```hcl
public_key = file(pathexpand("~/.ssh/id_rsa.pub"))
```

This helps Terraform correctly expand the `~` home directory path.

---

## 6. VM size policy issue

Azure policy blocked:

```hcl
size = "Standard_B1s"
```

The fix was:

```hcl
size = "Standard_B2s"
```

Command used:

```bash
sed -i 's/Standard_B1s/Standard_B2s/g' main.tf
terraform fmt
terraform validate
```

---

## 7. Go installation in WSL Ubuntu

Go was not installed:

```text
Command 'go' not found
```

Installed Go using the official tarball method.

```bash
sudo apt update
sudo apt install -y curl tar
cd /tmp
GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | head -n 1)
echo $GO_VERSION
curl -LO https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${GO_VERSION}.linux-amd64.tar.gz
```

Add Go to PATH:

```bash
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

Verify:

```bash
go version
which go
go env GOPATH
go env GOROOT
```

Example result:

```text
go version go1.26.5 linux/amd64
/usr/local/go/bin/go
/home/ilyas_zazai/go
/usr/local/go
```

Meaning:

| Item | Meaning |
|---|---|
| `go version` | Confirms Go is installed. |
| `which go` | Shows the Go binary path. |
| `GOPATH` | Where Go stores packages/tools. |
| `GOROOT` | Where Go itself is installed. |

---

## 8. Go module setup

Go to the test folder:

```bash
cd test
```

Initialize module:

```bash
go mod init github.com/Ilyzazai/cst8918-w24-a06-terratest
```

There was an ambiguous import issue when Go selected newer Terratest packages automatically.

Fix: pin Terratest version.

```bash
go get github.com/gruntwork-io/terratest@v0.48.2
go get github.com/stretchr/testify@v1.11.1
go mod tidy
```

Expected files:

```text
azure_webserver_test.go
go.mod
go.sum
```

`go.mod` is like `pom.xml` in Java Maven. It stores the module name, Go version, and dependencies.

`go.sum` stores checksums to verify downloaded dependencies.

---

## 9. Go/Terratest file explanation

File:

```bash
test/azure_webserver_test.go
```

Main imports:

```go
import (
    "testing"

    "github.com/gruntwork-io/terratest/modules/azure"
    "github.com/gruntwork-io/terratest/modules/terraform"
    "github.com/stretchr/testify/assert"
)
```

| Go code | Meaning |
|---|---|
| `package test` | This file belongs to the `test` package. |
| `_test.go` | Go recognizes this as a test file. |
| `func TestAzureLinuxVMCreation(t *testing.T)` | Main test function. Go test functions start with `Test`. |
| `terraform.Options` | Terratest settings for Terraform. |
| `TerraformDir: "../"` | Terraform files are one folder above `test`. |
| `Vars` | Passes Terraform variables from Go. |
| `terraform.InitAndApply` | Runs Terraform init and apply. |
| `terraform.Output` | Reads values from `outputs.tf`. |
| `defer terraform.Destroy` | Destroys Azure resources after the test. |
| `assert.True` | Confirms something is true. |
| `assert.Equal` | Confirms actual value equals expected value. |

---

## 10. Tests added

### Test 1: VM exists

```go
assert.True(t, azure.VirtualMachineExists(t, vmName, resourceGroupName, subscriptionID))
```

### Test 2: NIC exists

```go
assert.True(t, azure.NetworkInterfaceExists(t, nicName, resourceGroupName, subscriptionID))
```

### Test 3: NIC is connected to VM

```go
vmNics := azure.GetVirtualMachineNics(t, vmName, resourceGroupName, subscriptionID)
assert.Contains(t, vmNics, nicName)
```

### Test 4: Correct Ubuntu image

```go
vmImage := azure.GetVirtualMachineImage(t, vmName, resourceGroupName, subscriptionID)

assert.Equal(t, "Canonical", vmImage.Publisher)
assert.Equal(t, "0001-com-ubuntu-server-jammy", vmImage.Offer)
assert.Equal(t, "22_04-lts-gen2", vmImage.SKU)
```

---

## 11. Compile-only test

This checks the Go test file without creating Azure resources:

```bash
go test -c -o /tmp/azure_webserver_test.test
ls -lh /tmp/azure_webserver_test.test
rm /tmp/azure_webserver_test.test
```

If a binary is created, Go compiled successfully.

---

## 12. Run real Terratest

Run from the `test` folder:

```bash
go test -v -timeout 30m azure_webserver_test.go
```

What happened:

1. Go compiled the test.
2. Terratest ran `terraform init`.
3. Terratest ran `terraform apply`.
4. Terraform created 8 Azure resources.
5. Go/Terratest checked VM, NIC, NIC connection, and Ubuntu image.
6. Terratest ran `terraform destroy`.
7. Test ended with `PASS`.

Final result:

```text
Destroy complete! Resources: 8 destroyed.
--- PASS: TestAzureLinuxVMCreation
PASS
ok      command-line-arguments
```

---

## 13. Why Azure resources disappeared

This line caused cleanup:

```go
defer terraform.Destroy(t, terraformOptions)
```

Meaning:

```text
After the test finishes, delete the Azure infrastructure created by this test.
```

This is correct. Terratest should clean up cloud resources to avoid cost and conflicts.

---

## 14. `.gitignore`

Recommended `.gitignore`:

```gitignore
.terraform/
*.tfstate
*.tfstate.*
crash.log
crash.*.log
*.tfvars
*.tfplan
```

Do not commit:

| Item | Reason |
|---|---|
| `.terraform/` | Local provider cache. |
| `terraform.tfstate` | Can expose infrastructure details. |
| `*.tfvars` | May contain local values or secrets. |
| `*.tfplan` | Local generated plan file. |

---

## 15. Push commands

From repo root:

```bash
git branch --show-current
git status
```

Add files:

```bash
git add .gitignore main.tf .terraform.lock.hcl test/azure_webserver_test.go test/go.mod test/go.sum LAB6_TERRATEST_NOTES.md
```

Commit:

```bash
git commit -m "add Terratest validation and lab notes"
```

Push:

```bash
git push origin test-vm-creation
```

Submit branch link:

```text
https://github.com/Ilyzazai/cst8918-w24-a06-terratest/tree/test-vm-creation
```

---

## 16. Quick command summary

```bash
# Terraform check
terraform fmt
terraform validate

# Go setup
cd test
go mod init github.com/Ilyzazai/cst8918-w24-a06-terratest
go get github.com/gruntwork-io/terratest@v0.48.2
go get github.com/stretchr/testify@v1.11.1
go mod tidy

# Compile-only check
go test -c -o /tmp/azure_webserver_test.test
rm /tmp/azure_webserver_test.test

# Real Terratest
go test -v -timeout 30m azure_webserver_test.go

# Push
cd ..
git add .gitignore main.tf .terraform.lock.hcl test/azure_webserver_test.go test/go.mod test/go.sum LAB6_TERRATEST_NOTES.md
git commit -m "add Terratest validation and lab notes"
git push origin test-vm-creation
```

---

## 17. Main learning summary

| Topic | What I learned |
|---|---|
| Terraform | Creates Azure infrastructure from code. |
| Terratest | Tests real infrastructure using Go. |
| Go module | `go.mod` manages dependencies. |
| Go test | `_test.go` files are run by `go test`. |
| Azure policy | CloudLabs may block some VM sizes. |
| SSH key | Linux VM needs a public SSH key. |
| `defer terraform.Destroy` | Cleans up cloud resources after the test. |
| Integration testing | Tests real deployed cloud resources, not only Terraform syntax. |
