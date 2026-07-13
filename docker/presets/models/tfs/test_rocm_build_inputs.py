#!/usr/bin/env python3
"""Static guardrails for the ROCm KAITO image build inputs."""

from pathlib import Path


ROOT = Path(__file__).parents[4]
ROCM_REQUIREMENTS = ROOT / "presets/workspace/dependencies/requirements-rocm.txt"
DOCKERFILE = ROOT / "docker/presets/models/tfs/Dockerfile"


def test_rocm_requirements_do_not_install_cuda_runtime_packages():
    lines = ROCM_REQUIREMENTS.read_text().splitlines()
    forbidden = ("torch", "vllm", "nvidia-ml-py", "bitsandbytes", "lmcache")
    packages = [line.split("#", 1)[0].strip().lower() for line in lines]
    assert not [line for line in packages if line and line.startswith(forbidden)]


def test_dockerfile_selects_rocm_dependency_file_explicitly():
    dockerfile = DOCKERFILE.read_text()
    assert "ARG DEPENDENCIES_FILE=" in dockerfile
    assert "COPY ${DEPENDENCIES_FILE} /workspace/requirements.txt" in dockerfile
    assert "COPY presets/workspace/dependencies/requirements-rocm.txt" in dockerfile
    amd_path = dockerfile.split('if [ "${GPU_PROVIDER}" = "amd" ]; then', 1)[1].split(
        "else", 1
    )[0]
    assert "requirements-rocm.txt" in amd_path
    assert "requirements.txt" not in amd_path
    assert "cu129" not in amd_path
    assert "download.pytorch.org" not in amd_path
