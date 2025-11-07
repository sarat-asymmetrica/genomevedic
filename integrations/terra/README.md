# GenomeVedic Python Library

[![PyPI version](https://badge.fury.io/py/genomevedic.svg)](https://badge.fury.io/py/genomevedic)
[![Python Versions](https://img.shields.io/pypi/pyversions/genomevedic.svg)](https://pypi.org/project/genomevedic/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Interactive genome visualization for Jupyter notebooks, Terra.bio, and Google Colab.

## Features

- **One-line visualization**: `gv.show(bam_file="gs://bucket/sample.bam")`
- **Terra.bio native**: Works seamlessly in Terra notebooks
- **GCS integration**: Direct bucket access, no downloads needed
- **AI-powered**: Natural language queries and variant explanations
- **Interactive widgets**: Zoom, pan, filter directly in notebooks

## Installation

### Basic Installation
```bash
pip install genomevedic
```

### Full Installation (with GCS support)
```bash
pip install genomevedic[full]
```

### Terra.bio Installation
```bash
# In Terra notebook
pip install genomevedic[terra]
```

## Quick Start

### Basic Usage
```python
import genomevedic as gv

# Visualize BAM file
gv.show(bam_file="gs://my-bucket/sample.bam")
```

### Advanced Usage
```python
import genomevedic as gv

# Create widget with options
widget = gv.GenomeVedicWidget(
    bam_file="gs://bucket/tumor.bam",
    reference="hg38",
    initial_region="chr17:41196311-41277500"  # BRCA1
)
widget.show()

# Query with natural language
results = widget.query("Find pathogenic variants in BRCA1")
print(results['generated_sql'])

# Get AI explanation
explanation = widget.explain_variant(
    gene="BRCA1",
    variant="c.68_69delAG",
    cancer_type="breast cancer"
)
print(explanation['summary'])
```

### Terra.bio Example
```python
import genomevedic as gv

# Auto-detects Terra environment
gv.terra_show("gs://fc-secure-bucket/sample.bam")
```

### Compare Multiple Samples
```python
import genomevedic as gv

gv.create_comparison_view(
    bam_files=[
        "gs://bucket/tumor.bam",
        "gs://bucket/normal.bam"
    ],
    labels=["Tumor", "Normal"]
)
```

## Configuration

```python
import genomevedic as gv

# Set API URL
gv.set_api_url("https://genomevedic.example.com")

# Set API key (if required)
gv.set_api_key("your-api-key")

# Set default reference
gv.set_default_reference("hg19")
```

## GCS Authentication

### Terra.bio
In Terra, authentication is automatic via workspace credentials.

### Local Development
```bash
# Authenticate with gcloud
gcloud auth application-default login

# Or set credentials file
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
```

### Service Account
```python
from genomevedic import GCSClient

client = GCSClient(credentials_path="/path/to/service-account.json")
```

## API Reference

### Main Functions

- `show(bam_file, ...)` - Quick display function
- `load_bam(bam_file, ...)` - Load BAM and return widget
- `query(natural_language)` - Execute natural language query
- `explain_variant(gene, variant, ...)` - Get variant explanation

### Configuration

- `set_api_url(url)` - Set GenomeVedic API URL
- `set_api_key(key)` - Set authentication key
- `set_default_reference(ref)` - Set default genome

### Classes

- `GenomeVedicWidget` - Interactive visualization widget
- `GenomeVedicAPIClient` - REST API client
- `GCSClient` - Google Cloud Storage client
- `TerraGCSClient` - Terra-optimized GCS client

## Examples

See the `examples/` directory for complete Jupyter notebooks:

- `basic_usage.ipynb` - Getting started
- `terra_demo.ipynb` - Terra.bio integration
- `advanced_features.ipynb` - Natural language queries and AI
- `tcga_analysis.ipynb` - TCGA dataset analysis

## Requirements

- Python 3.8+
- Jupyter or JupyterLab
- (Optional) Google Cloud SDK for GCS access

## Development

```bash
# Clone repository
git clone https://github.com/genomevedic/genomevedic-python
cd genomevedic-python

# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black genomevedic_python/
```

## Documentation

Full documentation available at: https://genomevedic.readthedocs.io

## Support

- Issues: https://github.com/genomevedic/genomevedic-python/issues
- Email: support@genomevedic.io
- Slack: https://genomevedic.slack.com

## License

MIT License - see LICENSE file for details.

## Citation

If you use GenomeVedic in your research, please cite:

```bibtex
@software{genomevedic2025,
  title = {GenomeVedic: Interactive Genome Visualization},
  author = {GenomeVedic Team},
  year = {2025},
  url = {https://github.com/genomevedic/genomevedic-python}
}
```

## Acknowledgments

- Built for the Broad Institute Terra.bio platform
- Supports TCGA and other genomics datasets
- Powered by AI for natural language queries
