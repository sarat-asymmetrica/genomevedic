# Terra.bio Integration Guide

Complete guide for using GenomeVedic in Terra.bio notebooks.

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Quick Start](#quick-start)
4. [Authentication](#authentication)
5. [Working with GCS](#working-with-gcs)
6. [Advanced Features](#advanced-features)
7. [Troubleshooting](#troubleshooting)
8. [Examples](#examples)

---

## Overview

GenomeVedic integrates seamlessly with Terra.bio, allowing researchers to visualize genomic data directly in notebook environments without downloading files.

### Key Features

- **Direct GCS Access**: Stream BAM files from Google Cloud Storage buckets
- **No Downloads Required**: Visualize multi-GB files instantly
- **Terra Native**: Auto-detects Terra workspace credentials
- **AI-Powered**: Natural language queries and variant explanations
- **Interactive**: Zoom, pan, filter genomic regions in real-time

### Architecture

```
Terra Notebook (Jupyter)
    ↓
GenomeVedic Python Library
    ↓
┌──────────────┬──────────────────┐
│ GCS Client   │ API Client       │
│ (BAM files)  │ (Queries/AI)     │
└──────────────┴──────────────────┘
    ↓                 ↓
Google Cloud      GenomeVedic
Storage           Backend
```

---

## Installation

### In Terra Notebook

```python
# Basic installation
!pip install genomevedic

# Full installation with all features
!pip install genomevedic[terra]

# Verify installation
import genomevedic as gv
print(f"GenomeVedic version: {gv.__version__}")
```

### Installation Time
- Expected: < 2 minutes
- Package size: ~500KB (core) / ~50MB (with dependencies)

### Requirements
- Python 3.8+
- Jupyter/JupyterLab (included in Terra)
- Terra workspace with GCS access

---

## Quick Start

### Minimal Example (1 line)

```python
import genomevedic as gv
gv.show(bam_file="gs://your-workspace-bucket/sample.bam")
```

### Complete Example (5 minutes)

```python
import genomevedic as gv

# 1. Configure (optional - auto-detects Terra)
gv.set_api_url("http://localhost:8080")  # Local GenomeVedic instance
gv.set_default_reference("hg38")

# 2. Load BAM file from workspace bucket
widget = gv.load_bam(
    bam_file="gs://fc-secure-bucket/TCGA/BRCA/tumor.bam",
    reference="hg38",
    initial_region="chr17:41196311-41277500"  # BRCA1 gene
)

# 3. Query with natural language
results = widget.query("Find all pathogenic variants in BRCA1")
print(f"Generated SQL: {results['generated_sql']}")

# 4. Get AI explanations
explanation = widget.explain_variant(
    gene="BRCA1",
    variant="c.68_69delAG",
    cancer_type="breast cancer"
)
print(explanation['summary'])
```

---

## Authentication

### Automatic (Recommended)

In Terra, authentication happens automatically via workspace credentials:

```python
import genomevedic as gv

# No configuration needed - just use it!
gv.show(bam_file="gs://fc-secure-bucket/sample.bam")
```

### Manual Configuration

For custom service accounts or local testing:

```python
from genomevedic import GCSClient

# Option 1: Service account JSON
client = GCSClient(credentials_path="/path/to/service-account.json")

# Option 2: Environment variable
import os
os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = "/path/to/credentials.json"
client = GCSClient()

# Option 3: gcloud auth (local development)
# Run in terminal: gcloud auth application-default login
client = GCSClient()
```

### Verify Access

```python
import genomevedic as gv

# Check if file is accessible
bam_file = "gs://your-bucket/sample.bam"
if gv.check_gcs_access(bam_file):
    print("✓ File is accessible")
    gv.show(bam_file=bam_file)
else:
    print("✗ Cannot access file - check credentials")
```

---

## Working with GCS

### List Workspace Files

```python
from google.cloud import storage

# Get Terra workspace bucket
import os
bucket_name = os.getenv('WORKSPACE_BUCKET')
print(f"Workspace bucket: {bucket_name}")

# List BAM files
client = storage.Client()
bucket = client.bucket(bucket_name)
blobs = bucket.list_blobs(prefix="bam/")

for blob in blobs:
    if blob.name.endswith('.bam'):
        print(f"gs://{bucket_name}/{blob.name}")
```

### Stream Large Files

```python
import genomevedic as gv

# GenomeVedic streams data - no need to download!
# This works even for 100+ GB BAM files
gv.show(bam_file="gs://bucket/huge-file-200GB.bam")
```

### Download Specific Regions

```python
from genomevedic import GCSClient

client = GCSClient()

# Extract specific genomic region
data = client.read_bam_region(
    gcs_path="gs://bucket/sample.bam",
    chromosome="chr17",
    start=41196311,
    end=41277500  # BRCA1 region
)

# Process region data
# (data is in BAM format)
```

### Generate Shareable URLs

```python
from genomevedic import GCSClient

client = GCSClient()

# Create temporary signed URL (1 hour expiration)
url = client.generate_signed_url(
    gcs_path="gs://bucket/sample.bam",
    expiration=3600
)

print(f"Share this URL: {url}")
```

---

## Advanced Features

### Natural Language Queries

```python
import genomevedic as gv

# Create widget
widget = gv.load_bam("gs://bucket/sample.bam")

# Query examples
queries = [
    "Find all variants in BRCA1 gene",
    "Show mutations with MAF > 0.01",
    "List pathogenic variants in cancer genes",
    "Find SNPs on chromosome 17",
    "Show variants in exon regions"
]

for query in queries:
    result = widget.query(query)
    print(f"Query: {query}")
    print(f"SQL: {result['generated_sql']}")
    print(f"Explanation: {result['explanation']}\n")
```

### Variant Explanations

```python
import genomevedic as gv

# Single variant
result = gv.explain_variant(
    gene="BRCA1",
    variant="c.68_69delAG",
    cancer_type="breast cancer"
)

print(f"Clinical Significance: {result['clinical_significance']}")
print(f"Summary: {result['summary']}")
print(f"Evidence: {result['evidence']}")

# Batch explanations
variants = [
    {'gene': 'BRCA1', 'variant': 'c.68_69delAG'},
    {'gene': 'TP53', 'variant': 'c.215C>G'},
    {'gene': 'EGFR', 'variant': 'p.L858R'}
]

from genomevedic import GenomeVedicAPIClient
client = GenomeVedicAPIClient()
results = client.batch_explain_variants(variants)

for r in results:
    print(f"{r['gene']} {r['variant']}: {r['summary']}")
```

### Compare Multiple Samples

```python
import genomevedic as gv

# Side-by-side comparison
gv.create_comparison_view(
    bam_files=[
        "gs://bucket/patient1-tumor.bam",
        "gs://bucket/patient1-normal.bam"
    ],
    labels=["Tumor", "Normal"],
    reference="hg38",
    initial_region="chr17:7571719-7590868"  # TP53
)
```

### Custom Widget Configuration

```python
import genomevedic as gv

widget = gv.GenomeVedicWidget(
    bam_file="gs://bucket/sample.bam",
    reference="hg38",
    api_url="http://localhost:8080",
    width="100%",
    height="600px",
    initial_region="chr1:1000000-2000000"
)

# Show widget
widget.show()

# Get session URL for sharing
url = widget.get_session_url()
print(f"Share: {url}")
```

---

## Troubleshooting

### Issue: "Cannot access GCS file"

**Solution:**
```python
# Check workspace permissions
import os
print(f"Workspace: {os.getenv('WORKSPACE_NAMESPACE')}")
print(f"Project: {os.getenv('GOOGLE_PROJECT')}")

# Verify bucket access
import genomevedic as gv
gv.check_gcs_access("gs://your-bucket/file.bam")

# Check IAM permissions in Terra workspace settings
```

### Issue: "ipywidgets not displaying"

**Solution:**
```python
# Enable widgets
!jupyter nbextension enable --py widgetsnbextension

# Verify installation
import ipywidgets
print(f"ipywidgets version: {ipywidgets.__version__}")

# Restart kernel and try again
```

### Issue: "API connection failed"

**Solution:**
```python
import genomevedic as gv

# Check API health
client = gv.GenomeVedicAPIClient(base_url="http://localhost:8080")
health = client.health_check()
print(health)

# Update API URL if needed
gv.set_api_url("https://your-genomevedic-instance.com")
```

### Issue: "BAM index not found"

**Solution:**
```bash
# Ensure .bai index file exists
# For gs://bucket/sample.bam, need gs://bucket/sample.bam.bai

# Create index if missing:
samtools index sample.bam
gsutil cp sample.bam.bai gs://bucket/
```

### Issue: "Slow loading"

**Optimization:**
```python
# Use indexed regions for faster access
widget = gv.load_bam(
    bam_file="gs://bucket/large.bam",
    initial_region="chr1:1000000-1010000"  # Start with small region
)

# Or download file locally for repeated access
local_path = gv.download_from_gcs(
    "gs://bucket/sample.bam",
    "/tmp/sample.bam"
)
gv.show(bam_file=local_path)
```

---

## Examples

### Example 1: TCGA Breast Cancer Analysis

```python
import genomevedic as gv

# Load TCGA tumor sample
widget = gv.load_bam(
    bam_file="gs://tcga-data/BRCA/TCGA-A1-A0SB-tumor.bam",
    reference="hg38",
    initial_region="chr17:41196311-41277500"  # BRCA1
)

# Find pathogenic variants
results = widget.query("Find pathogenic BRCA1 mutations")

# Explain findings
for variant in results.get('variants', []):
    explanation = widget.explain_variant(
        gene=variant['gene'],
        variant=variant['notation'],
        cancer_type="breast cancer"
    )
    print(f"{variant['gene']}: {explanation['summary']}")
```

### Example 2: Cohort Analysis

```python
import genomevedic as gv
import pandas as pd

# List of samples
samples = pd.DataFrame({
    'sample_id': ['S1', 'S2', 'S3'],
    'bam_path': [
        'gs://bucket/sample1.bam',
        'gs://bucket/sample2.bam',
        'gs://bucket/sample3.bam'
    ]
})

# Analyze each sample
for _, row in samples.iterrows():
    print(f"\nAnalyzing {row['sample_id']}...")

    widget = gv.load_bam(
        bam_file=row['bam_path'],
        initial_region="chr17:7571719-7590868"  # TP53
    )

    # Query for TP53 mutations
    results = widget.query("Find TP53 mutations")
    print(f"Found {results['result_count']} variants")
```

### Example 3: Reference Comparison

```python
import genomevedic as gv

# Compare same sample against different references
bam_file = "gs://bucket/sample.bam"

print("hg19 view:")
gv.show(bam_file=bam_file, reference="hg19", height="400px")

print("\nhg38 view:")
gv.show(bam_file=bam_file, reference="hg38", height="400px")
```

### Example 4: Export Results

```python
import genomevedic as gv
import pandas as pd

widget = gv.load_bam("gs://bucket/sample.bam")

# Query for variants
result = widget.query("Find all variants with MAF > 0.05")

# Convert to DataFrame
df = pd.DataFrame(result.get('results', []))
print(df.head())

# Export to CSV
df.to_csv('variants.csv', index=False)
print(f"Exported {len(df)} variants to variants.csv")
```

---

## Performance Benchmarks

### Setup Time
- Package install: < 2 minutes
- First import: < 5 seconds
- Widget load: < 3 seconds

### Data Access
- GCS streaming (100GB BAM): < 1 second (initial load)
- Region fetch (1MB): < 200ms
- Natural language query: < 2 seconds

### Resource Usage
- Memory: ~100MB (base) + ~10MB per open widget
- CPU: Minimal (streaming architecture)
- Network: ~1MB/s for active visualization

---

## Best Practices

1. **Always use GCS paths in Terra**
   ```python
   # Good
   gv.show("gs://bucket/file.bam")

   # Avoid (unnecessary download)
   !gsutil cp gs://bucket/file.bam .
   gv.show("file.bam")
   ```

2. **Start with specific regions for large files**
   ```python
   # Better performance
   gv.show(
       bam_file="gs://bucket/huge.bam",
       initial_region="chr1:1000000-1100000"
   )
   ```

3. **Reuse widget instances**
   ```python
   # Create once
   widget = gv.load_bam("gs://bucket/file.bam")

   # Reuse for multiple queries
   widget.query("Find BRCA1 variants")
   widget.query("Find TP53 variants")
   ```

4. **Check access before heavy operations**
   ```python
   if gv.check_gcs_access(bam_file):
       gv.show(bam_file=bam_file)
   ```

---

## Support

- **Documentation**: https://genomevedic.readthedocs.io
- **Issues**: https://github.com/genomevedic/genomevedic-python/issues
- **Terra Forum**: https://support.terra.bio
- **Email**: support@genomevedic.io

---

## Changelog

### Version 1.0.0 (2025-01-07)
- Initial release
- Terra.bio integration
- GCS streaming support
- Natural language queries
- AI variant explanations
- Jupyter widget implementation
