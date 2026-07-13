# KAITO AMD fork incremental rollout status

Date: 2026-07-13
Fork: `/home/anirudh/Projects/kaito-amd`
Branch: `feat/amd-rocm-poc`
Base: upstream `a62ac20`

## Completed increments

### AMD provider plumbing

- Provider-aware GPU config with NVIDIA/AMD providers.
- AMD node-label parser using `amd.com/gpu.product`, `amd.com/gpu.count`, `amd.com/gpu.memory`, and `amd.com/gpu.arch`.
- `amd.com/gpu` resource-name propagation into generated inference resources.
- AMD GPU and `gpu=amd:NoSchedule` tolerations.
- AMD provider selection in controller via `--gpu-provider=nvidia|amd`.
- AMD-aware BYO admission and node estimation.

### Runtime image selection

- Chart values now expose `gpuProvider` and `runtimeImage`.
- Controller deployment passes `--gpu-provider` and `KAITO_RUNTIME_IMAGE`.
- Empty `runtimeImage` preserves the upstream embedded preset image.
- Non-empty `runtimeImage` overrides the generated main inference container image.

### ROCm wrapper safety

- `pynvml` is optional at import time.
- AMD runtime bypasses NVML and uses `KAITO_GPU_MEMORY_UTILIZATION` with a conservative default of `0.84`.
- NVIDIA retains dynamic NVML sizing.
- Runtime image Dockerfile has explicit provider/base-image build arguments and does not silently pretend the CUDA install path is a ROCm build.

## Validation completed

Focused Go tests:

```text
ok github.com/kaito-project/kaito/pkg/sku
ok github.com/kaito-project/kaito/api/v1beta1
ok github.com/kaito-project/kaito/pkg/workspace/inference
ok github.com/kaito-project/kaito/pkg/workspace/estimator/nodesestimator
```

Python wrapper test:

```text
28 passed
```

Build/lint:

```text
make build-workspace: passed
helm lint charts/kaito/workspace: passed
```

The fork's working tree is clean. Commits are signed and DCO-signed.

## Current blockers before a real AMD canary

1. **AMD node metadata**: live production currently exposes `amd.com/gpu: 1` and generic AMD feature labels, but not the fork's product/count/memory metadata contract. Add node labels through the authoritative Ansible/node-labeller path, or change the fork to use an explicit provider profile. Do not hand-label production as an undocumented hotfix.
2. **ROCm image build**: `docker/presets/models/tfs/Dockerfile` now has provider/base-image hooks, but the AMD install path must be built from a real ROCm/vLLM base and validated. The current production `rocm/vllm-dev:nightly` image is a useful runtime reference but is not yet a pinned KAITO wrapper image.
3. **Image publishing**: publish a pinned digest to an approved registry before any cluster deployment.
4. **Generated-workload verification**: instantiate an AMD Workspace with a fake/fixture GPU config and verify the complete StatefulSet, including image, resource requests, tolerations, commands, probes, device injection, and model-weight flow.
5. **Real GPU validation**: dev has no GPU worker. A controlled production canary on x1-pro or a separate AMD test worker is required for model load and API verification.
6. **Scope warning**: RAG, tuning, model streaming, multi-node, and automatic provisioning are not covered by this first POC.

## Next safe implementation increment

Build a dedicated ROCm KAITO image, using the exact vLLM/ROCm lineage already working on x1-pro, with:

- Python 3.12.
- ROCm 7.1-compatible PyTorch/vLLM.
- `gfx1150` support.
- KAITO `/workspace/vllm` scripts.
- Optional NVML wrapper.
- `GPU_PROVIDER=amd` and an explicit `KAITO_GPU_MEMORY_UTILIZATION`.
- Pinned image digest.

Then add a separate forked-controller HelmRelease in dev or a dedicated test namespace. Do not replace stock production KAITO or existing vLLM deployments until the image starts a real model.

## Production gate

The fork is not production-ready until a real AMD Workspace reaches:

```text
ResourceReady=True
InferenceReady=True
WorkspaceSucceeded=True
GET /v1/models succeeds
POST /v1/chat/completions succeeds
```

For a model requiring tool calling or multimodal input, those must be separately tested. The final canary must remain outside LiteLLM routing until direct service tests and memory/context checks pass.
