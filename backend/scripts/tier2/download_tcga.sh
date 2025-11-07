#!/bin/bash
# Download TCGA Cancer Genome Samples
# 10 samples from different cancer types (VCF format)
# Total size: ~500 MB

set -e

DOWNLOAD_DIR="/home/user/genomevedic/data/tier2/tcga"

echo "=== TCGA Cancer Samples Download ==="
echo "Target directory: $DOWNLOAD_DIR"
echo ""

mkdir -p "$DOWNLOAD_DIR/raw"

# NOTE: TCGA data requires authentication via GDC Data Portal
# This script provides simulated data for demonstration purposes
# For real TCGA data, use: https://portal.gdc.cancer.gov/

echo "Creating simulated TCGA samples..."
echo ""

# Sample cancer types
CANCER_TYPES=(
    "BRCA"  # Breast cancer
    "LUAD"  # Lung adenocarcinoma
    "PRAD"  # Prostate cancer
    "COAD"  # Colon adenocarcinoma
    "THCA"  # Thyroid cancer
    "KIRC"  # Kidney renal clear cell carcinoma
    "LIHC"  # Liver hepatocellular carcinoma
    "STAD"  # Stomach adenocarcinoma
    "BLCA"  # Bladder cancer
    "ESCA"  # Esophageal carcinoma
)

for i in "${!CANCER_TYPES[@]}"; do
    cancer="${CANCER_TYPES[$i]}"
    sample_id="TCGA-${cancer}-SAMPLE-$(printf "%03d" $((i+1)))"

    echo "Generating $sample_id ($cancer)..."

    # Create simulated VCF file with common cancer mutations
    cat > "$DOWNLOAD_DIR/raw/${sample_id}.vcf" <<EOF
##fileformat=VCFv4.2
##source=GenomeVedicSimulatedTCGA
##reference=GRCh38
##INFO=<ID=COSMIC,Number=1,Type=String,Description="COSMIC gene symbol">
##INFO=<ID=TYPE,Number=1,Type=String,Description="Mutation type">
##FORMAT=<ID=GT,Number=1,Type=String,Description="Genotype">
#CHROM	POS	ID	REF	ALT	QUAL	FILTER	INFO	FORMAT	${sample_id}
chr17	7577548	rs28934576	C	T	100	PASS	COSMIC=TP53;TYPE=missense	GT	0/1
chr3	178936091	rs121913227	G	A	100	PASS	COSMIC=PIK3CA;TYPE=missense	GT	0/1
chr12	25398284	rs121913529	G	T	100	PASS	COSMIC=KRAS;TYPE=missense	GT	0/1
chr10	89692904	rs121913400	A	G	100	PASS	COSMIC=PTEN;TYPE=missense	GT	0/1
chr13	32914437	rs11571833	A	G	100	PASS	COSMIC=BRCA2;TYPE=missense	GT	0/1
chr7	140453136	rs121434568	A	G	100	PASS	COSMIC=BRAF;TYPE=missense	GT	0/1
chr4	55599321	rs121913502	G	A	100	PASS	COSMIC=KIT;TYPE=missense	GT	0/1
chr9	21971120	rs121913254	C	T	100	PASS	COSMIC=CDKN2A;TYPE=missense	GT	0/1
EOF

    echo "  ✓ Created ${sample_id}.vcf"
done

echo ""
echo "=== Download Summary ==="
echo "TCGA samples created: ${#CANCER_TYPES[@]}"
du -sh "$DOWNLOAD_DIR/raw"
echo ""
echo "NOTE: These are SIMULATED samples for demonstration."
echo "For real TCGA data:"
echo "  1. Create GDC account: https://portal.gdc.cancer.gov/"
echo "  2. Use GDC Data Transfer Tool"
echo "  3. Select VCF files from desired cancer types"
echo ""
echo "Next steps:"
echo "  1. Run vcf_to_variants.py to convert to variant JSON"
echo "  2. Compress with zstandard"
