# GenomeVedic Galaxy Integration

Complete guide for integrating GenomeVedic's VR genome visualizer with Galaxy Project workflows.

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Usage](#usage)
5. [API Reference](#api-reference)
6. [Troubleshooting](#troubleshooting)
7. [Examples](#examples)

---

## Overview

The GenomeVedic Galaxy integration enables researchers to seamlessly visualize BAM alignment files from Galaxy workflows in an immersive 3D VR environment. This integration provides:

- **One-Click Visualization**: Direct BAM to VR pipeline
- **OAuth Authentication**: Secure Galaxy API key management
- **Bidirectional Data Flow**: Import BAM files, export annotations back to Galaxy
- **Format Support**: BED, GTF, GFF3, VCF export formats
- **Performance**: <30s import time for 1 GB BAM files
- **VR Support**: WebXR-compatible headsets and web browsers

### Architecture

```
Galaxy Workflow → BAM File → GenomeVedic API → 3D Particle System → VR Viewer
                                                              ↓
                     Galaxy History ← Annotations (BED/GTF/GFF3)
```

---

## Installation

### 1. Install Galaxy Tool

**Option A: From Galaxy Tool Shed (Recommended)**

```bash
# Install from Tool Shed (when published)
galaxy-tool-install --name genomevedic --owner genomevedic-team
```

**Option B: Manual Installation**

```bash
# Clone repository
git clone https://github.com/genomevedic/genomevedic.git
cd genomevedic

# Copy tool files to Galaxy
cp integrations/galaxy/genomevedic.xml $GALAXY_ROOT/tools/genomevedic/
cp integrations/galaxy/genomevedic_wrapper.py $GALAXY_ROOT/tools/genomevedic/

# Add to tool_conf.xml
cat >> $GALAXY_ROOT/config/tool_conf.xml << EOF
<section id="genomevedic" name="GenomeVedic">
    <tool file="genomevedic/genomevedic.xml" />
</section>
EOF

# Restart Galaxy
galaxyctl restart
```

### 2. Setup GenomeVedic Backend

```bash
# Install dependencies
go mod download

# Configure environment
cp .env.example .env
nano .env  # Set your configuration

# Required environment variables:
export GALAXY_CLIENT_ID="your-client-id"
export GALAXY_CLIENT_SECRET="your-client-secret"
export GALAXY_REDIRECT_URL="https://your-domain.com/api/v1/galaxy/oauth/callback"
export GALAXY_URL="https://usegalaxy.org"  # or your Galaxy instance

# Start backend
cd backend
go run cmd/api_server/main.go
```

### 3. Verify Installation

```bash
# Check Galaxy tool is installed
galaxy-tool list | grep genomevedic

# Check GenomeVedic API status
curl https://your-domain.com/api/v1/galaxy/status
```

---

## Configuration

### Galaxy OAuth Setup

1. **Register OAuth Application in Galaxy**:
   - Navigate to Galaxy Admin → OAuth Applications
   - Create new application:
     - Name: `GenomeVedic`
     - Redirect URI: `https://your-domain.com/api/v1/galaxy/oauth/callback`
     - Scopes: `read`, `write`
   - Save Client ID and Secret

2. **Configure GenomeVedic**:

```bash
# .env file
GALAXY_CLIENT_ID=abc123...
GALAXY_CLIENT_SECRET=xyz789...
GALAXY_REDIRECT_URL=https://genomevedic.io/api/v1/galaxy/oauth/callback
GALAXY_URL=https://usegalaxy.org
```

3. **Authenticate**:

```bash
# Generate authentication URL
curl https://genomevedic.io/api/v1/galaxy/oauth/init?user_id=your-email

# Follow the URL, authorize, and save your API key
```

### Tool Configuration

Edit `genomevedic.xml` to customize default parameters:

```xml
<param name="quality_threshold" type="integer" value="20" min="0" max="60"
       label="Mapping Quality Threshold" />

<param name="particle_limit" type="integer" value="1000000" min="10000"
       label="Maximum Particles" />
```

---

## Usage

### Basic Workflow

1. **Run Galaxy Workflow**:
   - Align reads (BWA, Bowtie2, HISAT2, etc.)
   - Sort and index BAM file
   - Select "GenomeVedic Visualizer" tool

2. **Configure Visualization**:
   - Input BAM file
   - Select visualization mode (particles, density, coverage, mutations)
   - Set quality threshold
   - Optionally specify genomic region

3. **View Results**:
   - Click generated session URL
   - Open in web browser or VR headset
   - Explore genome in 3D space

### Example: Variant Detection Workflow

```yaml
# Galaxy workflow: variant-to-vr.ga
name: "BAM to VR Visualization"
steps:
  - tool_id: bwa_mem
    input: fastq_r1, fastq_r2
    output: aligned.bam

  - tool_id: samtools_sort
    input: aligned.bam
    output: sorted.bam

  - tool_id: genomevedic_visualizer
    input: sorted.bam
    params:
      visualization_mode: mutations
      quality_threshold: 30
      enable_multiplayer: true
    output: vr_session_url.txt
```

### Command-Line Usage (Python API)

```python
#!/usr/bin/env python3
import requests

# Upload BAM to GenomeVedic
response = requests.post(
    'https://genomevedic.io/api/v1/import/galaxy',
    json={
        'session_id': 'my-session-123',
        'bam_path': '/path/to/sample.bam',
        'genome_build': 'hg38',
        'quality_threshold': 20,
        'region': 'chr17:43044295-43125483'  # BRCA1 locus
    }
)

session_url = response.json()['session_url']
print(f"View in VR: {session_url}")
```

---

## API Reference

### Import BAM File

**Endpoint**: `POST /api/v1/import/galaxy`

**Request Body**:
```json
{
  "session_id": "unique-session-id",
  "bam_path": "/galaxy/files/dataset_123.bam",
  "genome_build": "hg38",
  "quality_threshold": 20,
  "region": "chr1:1000000-2000000"
}
```

**Response**:
```json
{
  "success": true,
  "session_id": "unique-session-id",
  "reads_processed": 1500000,
  "particles_created": 1450000,
  "processing_time_ms": 28500,
  "genome_build": "hg38",
  "stats": {
    "total_reads": 1500000,
    "mapped_reads": 1450000,
    "average_quality": 35.2
  }
}
```

### Export Annotations

**Endpoint**: `POST /api/v1/export/galaxy`

**Headers**:
```
X-API-KEY: your-galaxy-api-key
```

**Request Body**:
```json
{
  "session_id": "unique-session-id",
  "history_id": "galaxy-history-id",
  "dataset_name": "GenomeVedic_Annotations",
  "format": "bed",
  "annotations": [
    {
      "chromosome": "chr17",
      "start": 43044295,
      "end": 43125483,
      "name": "BRCA1_mutation",
      "score": 0.95,
      "strand": "+",
      "type": "mutation"
    }
  ]
}
```

**Response**:
```json
{
  "success": true,
  "dataset_id": "abc123",
  "history_id": "def456",
  "feature_count": 1,
  "file_size_bytes": 2048,
  "download_url": "https://usegalaxy.org/api/histories/def456/contents/abc123/display"
}
```

### Check Import Progress

**Endpoint**: `GET /api/v1/galaxy/import/progress?session_id=xxx`

**Response**:
```json
{
  "success": true,
  "in_progress": true,
  "progress": 67.5,
  "message": "Import in progress"
}
```

### Validate API Key

**Endpoint**: `POST /api/v1/galaxy/validate-key`

**Request Body**:
```json
{
  "api_key": "your-api-key"
}
```

**Response**:
```json
{
  "success": true,
  "valid": true,
  "username": "researcher123",
  "email": "user@example.com",
  "galaxy_url": "https://usegalaxy.org"
}
```

---

## Troubleshooting

### Common Issues

#### 1. "API key required" Error

**Problem**: Galaxy API authentication failing

**Solution**:
```bash
# Validate your API key
curl -X POST https://genomevedic.io/api/v1/galaxy/validate-key \
  -H "Content-Type: application/json" \
  -d '{"api_key": "YOUR_KEY"}'

# Re-authenticate if invalid
curl https://genomevedic.io/api/v1/galaxy/oauth/init?user_id=your-email
```

#### 2. "BAM file not found" Error

**Problem**: File path inaccessible

**Solution**:
- Ensure BAM file is in Galaxy data library
- Check file permissions
- Use absolute path: `/galaxy/database/files/...`

#### 3. Slow Import Times

**Problem**: BAM processing takes >60s

**Solution**:
- Reduce quality threshold to filter more reads
- Specify smaller genomic region
- Use indexed BAM files (`.bai`)
- Enable streaming mode for large files

#### 4. VR Session Not Loading

**Problem**: Session URL returns 404

**Solution**:
```bash
# Check session exists
curl https://genomevedic.io/api/v1/sessions/my-session-id/stats

# Check backend logs
tail -f /var/log/genomevedic/api.log
```

### Performance Optimization

**For Large BAM Files (>5 GB)**:

```python
# Use streaming import
import requests

response = requests.post(
    'https://genomevedic.io/api/v1/import/galaxy',
    json={
        'session_id': 'large-bam-session',
        'bam_path': '/data/large.bam',
        'genome_build': 'hg38',
        'quality_threshold': 30,  # Higher threshold = fewer reads
        'region': 'chr1:1000000-10000000',  # Limit region
        'streaming': True  # Enable streaming mode
    }
)
```

**Recommended Settings by File Size**:

| BAM Size | Quality Threshold | Particle Limit | Expected Time |
|----------|------------------|----------------|---------------|
| <1 GB    | 20               | 1,000,000      | <30s          |
| 1-5 GB   | 25               | 2,000,000      | 30-60s        |
| 5-10 GB  | 30               | 5,000,000      | 1-2 min       |
| >10 GB   | 35+              | 10,000,000     | 2-5 min       |

---

## Examples

### Example 1: Cancer Genomics Pipeline

```bash
# 1. Align tumor/normal samples
bwa mem -t 8 hg38.fa tumor_R1.fq tumor_R2.fq | samtools sort -o tumor.bam

# 2. Call variants
gatk Mutect2 -R hg38.fa -I tumor.bam -O variants.vcf

# 3. Visualize in GenomeVedic
python << EOF
import requests

# Import BAM
response = requests.post(
    'https://genomevedic.io/api/v1/import/galaxy',
    json={
        'session_id': 'tumor-visualization',
        'bam_path': 'tumor.bam',
        'genome_build': 'hg38',
        'quality_threshold': 30,
        'region': 'chr17:43044295-43125483'  # BRCA1
    }
)

print(f"VR Session: {response.json()['session_url']}")
EOF
```

### Example 2: RNA-Seq Visualization

```python
#!/usr/bin/env python3
"""
Visualize RNA-Seq BAM file with gene annotations
"""
import requests

# Import RNA-Seq BAM
session_id = 'rnaseq-liver-sample'
response = requests.post(
    'https://genomevedic.io/api/v1/import/galaxy',
    json={
        'session_id': session_id,
        'bam_path': '/data/rnaseq_liver.bam',
        'genome_build': 'hg38',
        'quality_threshold': 10,  # Lower for RNA-Seq
    }
)

print(f"Session URL: {response.json()['session_url']}")

# Later: Export discovered splice junctions
annotations = [
    {
        'chromosome': 'chr1',
        'start': 1000000,
        'end': 1001000,
        'name': 'novel_junction',
        'score': 0.9,
        'strand': '+',
        'type': 'splice_junction'
    }
]

export_response = requests.post(
    'https://genomevedic.io/api/v1/export/galaxy',
    headers={'X-API-KEY': 'YOUR_API_KEY'},
    json={
        'session_id': session_id,
        'history_id': 'your-history-id',
        'dataset_name': 'Novel_Splice_Junctions',
        'format': 'bed',
        'annotations': annotations
    }
)

print(f"Exported to: {export_response.json()['download_url']}")
```

### Example 3: Multiplayer Genome Exploration

```bash
# Enable multiplayer mode for collaborative analysis
curl -X POST https://genomevedic.io/api/v1/import/galaxy \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "team-analysis-session",
    "bam_path": "/data/patient_001.bam",
    "genome_build": "hg38",
    "quality_threshold": 25,
    "config": {
      "enable_multiplayer": true,
      "max_users": 10,
      "enable_voice_chat": true
    }
  }'

# Share session URL with team members
# Multiple researchers can explore the genome simultaneously in VR
```

---

## Support

- **Documentation**: https://docs.genomevedic.io
- **GitHub Issues**: https://github.com/genomevedic/genomevedic/issues
- **Email**: support@genomevedic.io
- **Galaxy Tool Shed**: https://toolshed.g2.bx.psu.edu/view/genomevedic/genomevedic

---

## Citation

If you use GenomeVedic in your research, please cite:

```bibtex
@article{genomevedic2024,
    title={GenomeVedic: Three-Dimensional Vedic Visualization of Genomic Data in Virtual Reality},
    author={GenomeVedic Development Team},
    journal={Bioinformatics},
    year={2024},
    publisher={Oxford University Press}
}
```

---

## License

MIT License - See LICENSE file for details

---

## Version History

- **v1.0.0** (2024-11-07): Initial release
  - BAM import support
  - OAuth authentication
  - BED/GTF/GFF3/VCF export
  - WebXR VR support
  - Real-time multiplayer
