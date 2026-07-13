// Copyright (c) KAITO authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sku

import (
	"fmt"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kaito-project/kaito/pkg/apis"
	"github.com/kaito-project/kaito/pkg/utils/consts"
)

// DefaultSKUHandler is the package-level CloudSKUHandler used by callers that
// do not want to resolve a handler from the environment on every call. It is
// expected to be initialized once at process startup via GetSKUHandler.
var DefaultSKUHandler CloudSKUHandler = nil

// GetSKUHandler returns the CloudSKUHandler for the current cloud provider
// as configured via the CLOUD_PROVIDER environment variable.
func GetSKUHandler() (CloudSKUHandler, error) {
	provider := os.Getenv("CLOUD_PROVIDER")

	if provider == "" {
		return nil, apis.ErrMissingField("CLOUD_PROVIDER environment variable must be set")
	}
	skuHandler := GetCloudSKUHandler(provider)
	if skuHandler == nil {
		return nil, apis.ErrInvalidValue(fmt.Sprintf("Unsupported cloud provider %s", provider), "CLOUD_PROVIDER")
	}

	return skuHandler, nil
}

// IsAzureCloudProvider reports whether the current cloud provider is Azure.
func IsAzureCloudProvider() bool {
	return os.Getenv("CLOUD_PROVIDER") == consts.AzureCloudName
}

// GetGPUConfigBySKU returns the GPUConfig for the given instance type using
// the cloud provider configured via the CLOUD_PROVIDER environment variable.
func GetGPUConfigBySKU(instanceType string) (*GPUConfig, error) {
	handler := DefaultSKUHandler
	if handler == nil {
		h, err := GetSKUHandler()
		if err != nil {
			return nil, apis.ErrInvalidValue(fmt.Sprintf("Failed to get SKU handler: %v", err), "sku")
		}
		handler = h
	}

	config := handler.GetGPUConfigBySKU(instanceType)
	if config == nil {
		return nil, apis.ErrInvalidValue(fmt.Sprintf("Unsupported SKU '%s' for cloud provider", instanceType), "sku")
	}

	return config, nil
}

// GetGPUConfigFromNodeLabelsForProvider extracts GPU configuration using the
// provider-specific label contract. AMD labels are supplied by an AMD node
// labeller or cluster bootstrap configuration.
func GetGPUConfigFromNodeLabelsForProvider(node *corev1.Node, provider GPUProvider) (*GPUConfig, error) {
	var productKey, countKey, memoryKey, architectureKey string
	var vendor, resourceName string
	switch provider {
	case GPUProviderAMD:
		productKey = "amd.com/gpu.product"
		countKey = "amd.com/gpu.count"
		memoryKey = "amd.com/gpu.memory"
		architectureKey = "amd.com/gpu.arch"
		vendor = string(GPUProviderAMD)
		resourceName = "amd.com/gpu"
	case GPUProviderNvidia:
		productKey = consts.NvidiaGPUProduct
		countKey = consts.NvidiaGPUCount
		memoryKey = consts.NvidiaGPUMemory
		architectureKey = ""
		vendor = string(GPUProviderNvidia)
		resourceName = consts.NvidiaGPU
	default:
		return nil, fmt.Errorf("unsupported GPU provider %q", provider)
	}

	product, hasProduct := node.Labels[productKey]
	countValue, hasCount := node.Labels[countKey]
	memoryValue, hasMemory := node.Labels[memoryKey]
	if !hasProduct || !hasCount || !hasMemory {
		return nil, fmt.Errorf("missing required %s GPU labels on node %s", vendor, node.Name)
	}
	count, err := strconv.Atoi(countValue)
	if err != nil || count < 1 {
		return nil, fmt.Errorf("invalid %s GPU count on node %s: %s", vendor, node.Name, countValue)
	}
	memoryMiB, err := strconv.Atoi(memoryValue)
	if err != nil || memoryMiB < 1 {
		return nil, fmt.Errorf("invalid %s GPU memory on node %s: %s", vendor, node.Name, memoryValue)
	}
	config := &GPUConfig{
		SKU:             "unknown",
		GPUCount:        count,
		GPUModel:        product,
		GPUVendor:       vendor,
		GPUResourceName: resourceName,
		GPUMem:          *resource.NewQuantity(int64((float64(memoryMiB)/1024)+0.5)*int64(count)*consts.GiBToBytes, resource.BinarySI),
	}
	if provider == GPUProviderNvidia {
		var capability float64
		if major, err := strconv.Atoi(node.Labels[consts.NvidiaCUDAComputeCapMajor]); err == nil {
			capability = float64(major)
			if minor, err := strconv.Atoi(node.Labels[consts.NvidiaCUDAComputeCapMinor]); err == nil {
				capability += float64(minor) / 10
			}
		}
		config.CUDAComputeCapability = capability
	}
	if architectureKey != "" {
		config.GPUArchitecture = node.Labels[architectureKey]
	}
	return config, nil
}

// GetGPUConfigFromNodeLabels retains the NVIDIA-compatible API for callers
// that have not yet selected a provider explicitly.
func GetGPUConfigFromNodeLabels(node *corev1.Node) (*GPUConfig, error) {
	return GetGPUConfigFromNodeLabelsForProvider(node, GPUProviderNvidia)
}
