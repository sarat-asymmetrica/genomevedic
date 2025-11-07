# GenomeVedic Installation Guide

Complete installation instructions for all environments.

---

## Table of Contents

1. [Terra.bio Installation](#terralio-installation)
2. [Google Colab Installation](#google-colab-installation)
3. [Local Jupyter Installation](#local-jupyter-installation)
4. [PyPI Package Build](#pypi-package-build)
5. [Development Setup](#development-setup)
6. [Verification](#verification)
7. [Troubleshooting](#troubleshooting)

---

## Terra.bio Installation

### Method 1: Simple Install (Recommended)

In a Terra notebook cell:

```python
!pip install genomevedic[terra]
```

**Time:** < 2 minutes
**Size:** ~50 MB (with dependencies)

### Method 2: From Source

```python
# Clone repository
!git clone https://github.com/genomevedic/genomevedic-python
!cd genomevedic-python/integrations/terra && pip install -e ".[terra]"
```

### Verification

```python
import genomevedic as gv
print(f"Version: {gv.__version__}")

# Test GCS access
gv.check_gcs_access("gs://your-bucket/test.bam")
```

### Terra-Specific Notes

- Authentication is automatic via workspace credentials
- GCS access uses Application Default Credentials (ADC)
- No additional configuration needed

---

## Google Colab Installation

### Step 1: Install Package

```python
!pip install genomevedic[full]
```

### Step 2: Authenticate with GCS (if using GCS files)

```python
from google.colab import auth
auth.authenticate_user()
```

### Step 3: Verify

```python
import genomevedic as gv
gv.show(bam_file="gs://your-bucket/sample.bam")
```

---

## Local Jupyter Installation

### Prerequisites

- Python 3.8 or higher
- pip or conda
- Jupyter Notebook or JupyterLab

### Step 1: Install Package

```bash
# Core installation
pip install genomevedic

# Full installation with GCS support
pip install genomevedic[full]

# Development installation
pip install genomevedic[dev]
```

### Step 2: Enable Jupyter Extensions

```bash
# Enable ipywidgets
jupyter nbextension enable --py widgetsnbextension

# For JupyterLab
jupyter labextension install @jupyter-widgets/jupyterlab-manager
```

### Step 3: Configure GCS Authentication (if needed)

```bash
# Method 1: gcloud CLI
gcloud auth application-default login

# Method 2: Service account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
```

### Step 4: Start Jupyter

```bash
jupyter notebook
# or
jupyter lab
```

---

## PyPI Package Build

### For Package Maintainers

#### Build Distribution

```bash
cd integrations/terra

# Install build tools
pip install build twine

# Build package
python -m build

# Result: dist/genomevedic-1.0.0.tar.gz and .whl
```

#### Verify Package

```bash
# Check package
twine check dist/*

# Test installation locally
pip install dist/genomevedic-1.0.0-py3-none-any.whl
```

#### Upload to PyPI

```bash
# Test PyPI (recommended first)
twine upload --repository testpypi dist/*

# Production PyPI
twine upload dist/*
```

#### PyPI Credentials

Create `~/.pypirc`:

```ini
[distutils]
index-servers =
    pypi
    testpypi

[pypi]
username = __token__
password = pypi-your-token-here

[testpypi]
repository = https://test.pypi.org/legacy/
username = __token__
password = pypi-your-test-token-here
```

---

## Development Setup

### Clone and Install

```bash
# Clone repository
git clone https://github.com/genomevedic/genomevedic-python
cd genomevedic-python/integrations/terra

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install in development mode
pip install -e ".[dev]"
```

### Run Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=genomevedic_python --cov-report=html

# View coverage report
open htmlcov/index.html
```

### Code Quality

```bash
# Format code
black genomevedic_python/

# Check style
flake8 genomevedic_python/

# Type checking
mypy genomevedic_python/
```

### Build Documentation

```bash
cd docs
pip install sphinx sphinx-rtd-theme
make html
open _build/html/index.html
```

---

## Verification

### Quick Test

```python
import genomevedic as gv

# Check version
print(f"GenomeVedic v{gv.__version__}")

# Test API client
client = gv.GenomeVedicAPIClient()
health = client.health_check()
print(f"API Status: {health['status']}")

# Test GCS client
if gv.check_gcs_access("gs://test-bucket/test.bam"):
    print("✓ GCS access working")
```

### Full Test Suite

```python
# Test all components
import genomevedic as gv

# 1. API Client
client = gv.GenomeVedicAPIClient(base_url="http://localhost:8080")
result = client.query_natural_language("Find BRCA1 variants")
print(f"✓ API client: {result['success']}")

# 2. GCS Client
from genomevedic import GCSClient
gcs = GCSClient()
info = gcs.get_file_info("gs://bucket/sample.bam")
print(f"✓ GCS client: {info['size']} bytes")

# 3. Jupyter Widget
widget = gv.GenomeVedicWidget(bam_file="gs://bucket/sample.bam")
print("✓ Widget created")

# 4. Configuration
gv.set_api_url("http://localhost:8080")
config = gv.get_config()
print(f"✓ Configuration: {config['api_url']}")
```

---

## Troubleshooting

### Issue: Module not found

```bash
# Ensure package is installed
pip list | grep genomevedic

# Reinstall if needed
pip install --upgrade --force-reinstall genomevedic
```

### Issue: Import errors

```python
# Check dependencies
import sys
print(f"Python: {sys.version}")

import requests
print(f"requests: {requests.__version__}")

import ipywidgets
print(f"ipywidgets: {ipywidgets.__version__}")

import IPython
print(f"IPython: {IPython.__version__}")
```

### Issue: Widgets not displaying

```bash
# Jupyter Notebook
jupyter nbextension list
jupyter nbextension enable --py widgetsnbextension

# JupyterLab
jupyter labextension list
jupyter labextension install @jupyter-widgets/jupyterlab-manager
```

### Issue: GCS authentication

```python
# Test authentication
from google.cloud import storage

try:
    client = storage.Client()
    buckets = list(client.list_buckets())
    print(f"✓ Authenticated - {len(buckets)} buckets accessible")
except Exception as e:
    print(f"✗ Authentication failed: {e}")
```

### Issue: API connection

```python
import requests

# Test API connection
try:
    response = requests.get("http://localhost:8080/api/v1/health")
    print(f"✓ API connected: {response.json()}")
except Exception as e:
    print(f"✗ API connection failed: {e}")
```

---

## System Requirements

### Minimum Requirements

- **Python:** 3.8+
- **RAM:** 2 GB
- **Disk:** 500 MB
- **Network:** Broadband internet (for GCS access)

### Recommended Requirements

- **Python:** 3.10+
- **RAM:** 8 GB
- **Disk:** 2 GB
- **Network:** High-speed internet
- **Browser:** Chrome, Firefox, or Safari (latest)

### Operating Systems

- Linux (tested on Ubuntu 20.04+)
- macOS (tested on macOS 11+)
- Windows 10/11 (with WSL recommended)

---

## Version Compatibility

| Component | Minimum Version | Recommended |
|-----------|----------------|-------------|
| Python | 3.8 | 3.10+ |
| requests | 2.28.0 | 2.31.0+ |
| ipywidgets | 8.0.0 | 8.1.0+ |
| IPython | 7.0.0 | 8.0.0+ |
| google-cloud-storage | 2.10.0 | 2.14.0+ |
| pysam | 0.21.0 | 0.22.0+ |
| Jupyter | 6.0 | 7.0+ |

---

## Uninstallation

```bash
# Remove package
pip uninstall genomevedic

# Remove with dependencies
pip uninstall genomevedic google-cloud-storage pysam

# Clean pip cache
pip cache purge
```

---

## Support

**Issues:** https://github.com/genomevedic/genomevedic-python/issues
**Email:** support@genomevedic.io
**Docs:** https://genomevedic.readthedocs.io

---

## License

MIT License - See LICENSE file for details.
