package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// For this lab, use the Azure subscription assigned to your CloudLabs account.
var subscriptionID string = "16c1b6b4-d7ef-4a39-8085-d207592a56f4"

func TestAzureLinuxVMCreation(t *testing.T) {
	terraformOptions := &terraform.Options{
		// The Terraform files are one folder above this /test folder.
		TerraformDir: "../",

		// This provides a value for var.labelPrefix in variables.tf.
		Vars: map[string]interface{}{
			"labelPrefix": "zaza0018",
		},
	}

	// This runs at the end of the test and removes the Azure resources.
	defer terraform.Destroy(t, terraformOptions)

	// This runs terraform init and terraform apply.
	terraform.InitAndApply(t, terraformOptions)

	// These values come from outputs.tf.
	vmName := terraform.Output(t, terraformOptions, "vm_name")
	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
	nicName := terraform.Output(t, terraformOptions, "nic_name")

	// Test 1: Confirm the VM exists in Azure.
	assert.True(t, azure.VirtualMachineExists(t, vmName, resourceGroupName, subscriptionID))

	// Test 2: Confirm the NIC exists in Azure.
	assert.True(t, azure.NetworkInterfaceExists(t, nicName, resourceGroupName, subscriptionID))

	// Test 3: Confirm the NIC is connected to the VM.
	vmNics := azure.GetVirtualMachineNics(t, vmName, resourceGroupName, subscriptionID)
	assert.Contains(t, vmNics, nicName)

	// Test 4: Confirm the VM is using the expected Ubuntu 22.04 image.
	vmImage := azure.GetVirtualMachineImage(t, vmName, resourceGroupName, subscriptionID)

	assert.Equal(t, "Canonical", vmImage.Publisher)
	assert.Equal(t, "0001-com-ubuntu-server-jammy", vmImage.Offer)
	assert.Equal(t, "22_04-lts-gen2", vmImage.SKU)
}
