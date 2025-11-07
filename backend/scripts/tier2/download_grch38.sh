#!/bin/bash
# Download GRCh38 Full Human Genome
# All 24 chromosomes (chr1-chr22, chrX, chrY)
# Total size: ~3 GB FASTA → 1 GB compressed particles

set -e

DOWNLOAD_DIR="/home/user/genomevedic/data/tier2/grch38"
UCSC_BASE="https://hgdownload.soe.ucsc.edu/goldenPath/hg38/chromosomes"

echo "=== GRCh38 Full Genome Download ==="
echo "Target directory: $DOWNLOAD_DIR"
echo ""

# Create download directory
mkdir -p "$DOWNLOAD_DIR/raw"

# Download all 24 chromosomes
CHROMOSOMES=(chr{1..22} chrX chrY)

for chr in "${CHROMOSOMES[@]}"; do
    echo "Downloading $chr..."

    # Download gzipped FASTA
    if [ ! -f "$DOWNLOAD_DIR/raw/${chr}.fa.gz" ]; then
        wget -q --show-progress \
            -O "$DOWNLOAD_DIR/raw/${chr}.fa.gz" \
            "${UCSC_BASE}/${chr}.fa.gz"
        echo "  ✓ Downloaded ${chr}.fa.gz"
    else
        echo "  ✓ Already downloaded ${chr}.fa.gz"
    fi

    # Decompress
    if [ ! -f "$DOWNLOAD_DIR/raw/${chr}.fa" ]; then
        gunzip -k "$DOWNLOAD_DIR/raw/${chr}.fa.gz"
        echo "  ✓ Decompressed to ${chr}.fa"
    else
        echo "  ✓ Already decompressed ${chr}.fa"
    fi
done

echo ""
echo "=== Download Summary ==="
echo "Chromosomes downloaded: 24"
echo "Raw FASTA files:"
du -sh "$DOWNLOAD_DIR/raw"
echo ""
echo "Next steps:"
echo "  1. Run generate_grch38_particles.py to convert to particles"
echo "  2. Compress with zstandard (level 19)"
echo "  3. Generate LOD levels (5K, 50K, 500K, 5M)"
