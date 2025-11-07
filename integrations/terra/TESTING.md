# GenomeVedic Testing Guide

Comprehensive testing instructions for Terra.bio integration.

---

## Quick Test (5 minutes)

### Terra.bio Environment

```python
# 1. Install
!pip install genomevedic[terra] -q

# 2. Import
import genomevedic as gv

# 3. Test basic functionality
gv.show(bam_file="gs://your-test-bucket/sample.bam")
```

**Expected Result:** Widget displays with visualization controls

---

## Complete Test Suite

### Test 1: Installation

```python
import sys
import genomevedic as gv

print("=== Installation Test ===")
print(f"Python version: {sys.version}")
print(f"GenomeVedic version: {gv.__version__}")
print(f"Import successful: ✓")
```

**Expected Output:**
```
=== Installation Test ===
Python version: 3.10.x
GenomeVedic version: 1.0.0
Import successful: ✓
```

### Test 2: API Client

```python
from genomevedic import GenomeVedicAPIClient

print("\n=== API Client Test ===")

# Initialize client
client = GenomeVedicAPIClient(base_url="http://localhost:8080")

# Health check
health = client.health_check()
print(f"API health: {health['status']}")

# Natural language query
result = client.query_natural_language("Find variants in BRCA1")
print(f"Query success: {result['success']}")
print(f"Generated SQL: {result['generated_sql'][:50]}...")

# Variant explanation
explanation = client.explain_variant(
    gene="BRCA1",
    variant="c.68_69delAG"
)
print(f"Explanation received: ✓")
```

**Expected Output:**
```
=== API Client Test ===
API health: healthy
Query success: True
Generated SQL: SELECT * FROM variants WHERE gene = 'BRCA1'...
Explanation received: ✓
```

### Test 3: GCS Client

```python
from genomevedic import GCSClient, TerraGCSClient
import os

print("\n=== GCS Client Test ===")

# Test file path
TEST_BAM = "gs://your-test-bucket/sample.bam"

# Check Terra environment
in_terra = os.getenv('WORKSPACE_NAMESPACE') is not None
print(f"Terra environment: {in_terra}")

# Initialize client
if in_terra:
    client = TerraGCSClient()
    print(f"Workspace bucket: {client.get_workspace_bucket()}")
else:
    client = GCSClient()

# Check access
accessible = client.check_access(TEST_BAM)
print(f"File accessible: {accessible}")

# Get file info
if accessible:
    info = client.get_file_info(TEST_BAM)
    print(f"File size: {info['size']:,} bytes")
    print(f"Bucket: {info['bucket']}")
```

**Expected Output:**
```
=== GCS Client Test ===
Terra environment: True
Workspace bucket: fc-secure-xxxxx
File accessible: True
File size: 123,456,789 bytes
Bucket: your-test-bucket
```

### Test 4: Jupyter Widget

```python
import genomevedic as gv

print("\n=== Widget Test ===")

# Create widget
widget = gv.GenomeVedicWidget(
    bam_file="gs://test-bucket/sample.bam",
    reference="hg38",
    initial_region="chr1:1000000-1010000"
)

print("Widget created: ✓")

# Test methods
url = widget.get_session_url()
print(f"Session URL generated: ✓")

# Show widget
widget.show()
print("Widget displayed: ✓")
```

**Expected Output:**
```
=== Widget Test ===
Widget created: ✓
Session URL generated: ✓
Widget displayed: ✓
```
*Plus interactive widget in notebook output*

### Test 5: Natural Language Queries

```python
import genomevedic as gv

print("\n=== Natural Language Query Test ===")

widget = gv.load_bam("gs://test-bucket/sample.bam")

test_queries = [
    "Find all variants in BRCA1",
    "Show SNPs on chromosome 1",
    "List pathogenic mutations"
]

for query in test_queries:
    result = widget.query(query)
    print(f"✓ '{query}'")
    print(f"  SQL: {result['generated_sql'][:60]}...")
```

**Expected Output:**
```
=== Natural Language Query Test ===
✓ 'Find all variants in BRCA1'
  SQL: SELECT * FROM variants WHERE gene = 'BRCA1'...
✓ 'Show SNPs on chromosome 1'
  SQL: SELECT * FROM variants WHERE chr = '1' AND type = 'SNP'...
✓ 'List pathogenic mutations'
  SQL: SELECT * FROM variants WHERE clinical_significance = 'path...
```

### Test 6: Performance

```python
import time
import genomevedic as gv

print("\n=== Performance Test ===")

# Test 1: Import speed
start = time.time()
import genomevedic as gv
import_time = time.time() - start
print(f"Import time: {import_time:.3f}s")

# Test 2: Widget creation
start = time.time()
widget = gv.GenomeVedicWidget(bam_file="gs://bucket/sample.bam")
widget_time = time.time() - start
print(f"Widget creation: {widget_time:.3f}s")

# Test 3: API query
start = time.time()
result = widget.query("Find BRCA1 variants")
query_time = time.time() - start
print(f"Query time: {query_time:.3f}s")

# Benchmarks
print("\nBenchmarks:")
print(f"  Import: {'PASS' if import_time < 5 else 'SLOW'} (< 5s)")
print(f"  Widget: {'PASS' if widget_time < 3 else 'SLOW'} (< 3s)")
print(f"  Query: {'PASS' if query_time < 2 else 'SLOW'} (< 2s)")
```

**Expected Output:**
```
=== Performance Test ===
Import time: 0.234s
Widget creation: 0.456s
Query time: 1.123s

Benchmarks:
  Import: PASS (< 5s)
  Widget: PASS (< 3s)
  Query: PASS (< 2s)
```

---

## Manual Testing Checklist

### Pre-requisites

- [ ] Terra workspace created
- [ ] Test BAM file uploaded to GCS
- [ ] BAM index (.bai) file present
- [ ] GenomeVedic API running
- [ ] Jupyter notebook ready

### Installation Tests

- [ ] `pip install genomevedic` completes without errors
- [ ] `import genomevedic` works
- [ ] Version number displays correctly
- [ ] All dependencies installed

### API Client Tests

- [ ] Health check returns status
- [ ] Natural language query generates SQL
- [ ] Variant explanation returns result
- [ ] Batch variant explanation works
- [ ] Error handling for invalid inputs

### GCS Client Tests

- [ ] Parse GCS paths correctly
- [ ] Check file access (existing file)
- [ ] Check file access (non-existing file)
- [ ] Get file metadata
- [ ] Generate signed URLs
- [ ] Terra environment auto-detection

### Widget Tests

- [ ] Widget displays in notebook
- [ ] Controls are interactive
- [ ] Region selector works
- [ ] Reference selector works
- [ ] File info button shows data
- [ ] Session URL generated

### Integration Tests

- [ ] Load BAM from GCS
- [ ] Execute natural language query
- [ ] Get variant explanation
- [ ] Compare multiple samples
- [ ] Share session URL
- [ ] Export results

### Performance Tests

- [ ] Import < 5 seconds
- [ ] Widget creation < 3 seconds
- [ ] API query < 2 seconds
- [ ] Large file (>100GB) loads
- [ ] Memory usage acceptable

---

## Automated Testing

### Unit Tests

```bash
# Run unit tests
cd integrations/terra
pytest tests/

# Run specific test
pytest tests/test_api_client.py

# Run with coverage
pytest --cov=genomevedic_python --cov-report=html
```

### Integration Tests

```python
# tests/test_integration.py
import pytest
import genomevedic as gv

def test_full_workflow():
    """Test complete GenomeVedic workflow"""

    # Setup
    bam_file = "gs://test-bucket/sample.bam"

    # Load BAM
    widget = gv.load_bam(bam_file)
    assert widget is not None

    # Query
    result = widget.query("Find BRCA1 variants")
    assert result['success'] == True

    # Explain
    explanation = widget.explain_variant("BRCA1", "c.68_69delAG")
    assert 'summary' in explanation

def test_gcs_access():
    """Test GCS file access"""
    from genomevedic import GCSClient

    client = GCSClient()
    info = client.get_file_info("gs://test-bucket/sample.bam")

    assert info['size'] > 0
    assert info['bucket'] == 'test-bucket'
```

### Performance Tests

```python
# tests/test_performance.py
import pytest
import time
import genomevedic as gv

def test_import_performance():
    """Test import time"""
    start = time.time()
    import genomevedic
    duration = time.time() - start

    assert duration < 5.0, f"Import too slow: {duration}s"

def test_query_performance():
    """Test query performance"""
    widget = gv.load_bam("gs://test-bucket/sample.bam")

    start = time.time()
    result = widget.query("Find variants")
    duration = time.time() - start

    assert duration < 2.0, f"Query too slow: {duration}s"
```

---

## Test Data

### Sample Files

Use these public datasets for testing:

```python
# 1000 Genomes BAM (small)
TEST_BAM_SMALL = "gs://genomics-public-data/1000-genomes/bam/NA12878.chrom20.ILLUMINA.bwa.CEU.low_coverage.20121211.bam"

# TCGA dataset (medium)
TEST_BAM_MEDIUM = "gs://genomics-public-data/tcga/BRCA/DNA-Seq/*.bam"

# Create test GCS bucket (your own)
TEST_BAM_CUSTOM = "gs://your-test-bucket/sample.bam"
```

### Test Regions

```python
# Small region (fast)
REGION_SMALL = "chr1:1000000-1010000"  # 10kb

# Gene region (BRCA1)
REGION_BRCA1 = "chr17:41196311-41277500"  # 81kb

# Larger region
REGION_LARGE = "chr1:1000000-2000000"  # 1Mb
```

---

## Known Issues

### Issue 1: Widget Not Displaying

**Symptom:** Widget code runs but nothing appears
**Solution:**
```bash
jupyter nbextension enable --py widgetsnbextension
# Restart kernel
```

### Issue 2: GCS Permission Denied

**Symptom:** `GCSError: Permission denied`
**Solution:**
```bash
# Terra: Check workspace permissions
# Local: Authenticate
gcloud auth application-default login
```

### Issue 3: API Connection Failed

**Symptom:** `GenomeVedicAPIError: Connection refused`
**Solution:**
```python
# Check API is running
import requests
requests.get("http://localhost:8080/api/v1/health")

# Update API URL
gv.set_api_url("http://your-api-server:8080")
```

---

## Test Results Template

```markdown
## Test Results - [Date]

### Environment
- Python: 3.10.x
- GenomeVedic: 1.0.0
- Platform: Terra.bio / Colab / Local

### Test Results

| Test | Status | Time | Notes |
|------|--------|------|-------|
| Installation | ✓ PASS | 1.2s | |
| API Client | ✓ PASS | 0.5s | |
| GCS Client | ✓ PASS | 0.8s | |
| Widget | ✓ PASS | 1.1s | |
| Queries | ✓ PASS | 1.5s | |
| Performance | ✓ PASS | - | All < targets |

### Issues Found
- None

### Overall: PASS ✓
```

---

## Support

**Report Issues:** https://github.com/genomevedic/genomevedic-python/issues
**Testing Help:** support@genomevedic.io
