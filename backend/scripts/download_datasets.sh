#!/bin/bash
################################################################################
# GenomeVedic Real Dataset Downloader
# Downloads and validates Tier 1 genomic datasets for production use
#
# Datasets:
# 1. Human Chromosome 22 (UCSC GRCh38) - 50 MB FASTA
# 2. E. coli K-12 (NCBI RefSeq) - 4.6 MB FASTA
# 3. COSMIC Top 100 Cancer Genes (Sanger) - 10 MB TSV
# 4. Ensembl GTF Annotations (Release 115) - 50 MB compressed
# 5. 1000 Genomes chr22 VCF sample - 100 MB VCF
#
# Usage: ./download_datasets.sh
# Output: data/raw/
################################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RAW_DATA_DIR="$PROJECT_ROOT/data/raw"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      GenomeVedic Real Dataset Downloader v1.0             ║${NC}"
echo -e "${BLUE}║      Tier 1 Starter Pack (500 MB → 150 MB compressed)     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo

# Create directories
mkdir -p "$RAW_DATA_DIR"
cd "$RAW_DATA_DIR"

# Function to download with progress and checksum validation
download_and_validate() {
    local url="$1"
    local output_file="$2"
    local expected_md5="$3"
    local description="$4"

    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}📦 Downloading: ${description}${NC}"
    echo -e "${BLUE}   URL: ${url}${NC}"
    echo -e "${BLUE}   Output: ${output_file}${NC}"
    echo

    # Skip if already downloaded and validated
    if [ -f "$output_file" ]; then
        echo -e "${YELLOW}File exists. Validating checksum...${NC}"
        if [ "$expected_md5" != "SKIP" ]; then
            local actual_md5=$(md5sum "$output_file" | awk '{print $1}')
            if [ "$actual_md5" = "$expected_md5" ]; then
                echo -e "${GREEN}✓ Checksum validated. Skipping download.${NC}"
                echo
                return 0
            else
                echo -e "${RED}✗ Checksum mismatch. Re-downloading...${NC}"
                rm -f "$output_file"
            fi
        else
            echo -e "${YELLOW}⚠ Checksum validation skipped (SKIP flag)${NC}"
            echo
            return 0
        fi
    fi

    # Download with resume capability
    wget -c --progress=bar:force:noscroll -O "$output_file" "$url" || {
        echo -e "${RED}✗ Download failed: $url${NC}"
        exit 1
    }

    # Validate checksum
    if [ "$expected_md5" != "SKIP" ]; then
        echo -e "${YELLOW}Validating checksum...${NC}"
        local actual_md5=$(md5sum "$output_file" | awk '{print $1}')
        if [ "$actual_md5" = "$expected_md5" ]; then
            echo -e "${GREEN}✓ Checksum validated: $actual_md5${NC}"
        else
            echo -e "${RED}✗ Checksum mismatch!${NC}"
            echo -e "${RED}   Expected: $expected_md5${NC}"
            echo -e "${RED}   Got:      $actual_md5${NC}"
            exit 1
        fi
    fi

    echo -e "${GREEN}✓ Download complete${NC}"
    echo
}

################################################################################
# Dataset 1: Human Chromosome 22 (UCSC GRCh38)
################################################################################
download_and_validate \
    "https://hgdownload.soe.ucsc.edu/goldenPath/hg38/chromosomes/chr22.fa.gz" \
    "chr22.fa.gz" \
    "SKIP" \
    "Human Chromosome 22 (GRCh38) - 50 MB FASTA"

# Decompress
if [ ! -f "chr22.fa" ]; then
    echo -e "${YELLOW}Decompressing chr22.fa.gz...${NC}"
    gunzip -k chr22.fa.gz
    echo -e "${GREEN}✓ Decompressed${NC}"
    echo
fi

################################################################################
# Dataset 2: E. coli K-12 (NCBI RefSeq)
################################################################################
download_and_validate \
    "https://ftp.ncbi.nlm.nih.gov/genomes/all/GCF/000/005/845/GCF_000005845.2_ASM584v2/GCF_000005845.2_ASM584v2_genomic.fna.gz" \
    "ecoli_k12.fna.gz" \
    "SKIP" \
    "E. coli K-12 (NCBI RefSeq) - 4.6 MB FASTA"

# Decompress
if [ ! -f "ecoli_k12.fna" ]; then
    echo -e "${YELLOW}Decompressing ecoli_k12.fna.gz...${NC}"
    gunzip -k ecoli_k12.fna.gz
    echo -e "${GREEN}✓ Decompressed${NC}"
    echo
fi

################################################################################
# Dataset 3: COSMIC Top 100 Cancer Genes (Simulated - COSMIC requires license)
################################################################################
# NOTE: Real COSMIC data requires registration at https://cancer.sanger.ac.uk/cosmic
# For demo purposes, we'll create a simulated dataset with known cancer genes
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📦 Creating: COSMIC Top 100 Cancer Genes (Simulated)${NC}"
echo -e "${YELLOW}⚠ Note: Real COSMIC data requires license registration${NC}"
echo -e "${YELLOW}   This is a simulated dataset for development/testing${NC}"
echo

cat > cosmic_top100_simulated.tsv << 'EOF'
Gene	Chromosome	Start	End	Mutations	Tier	Type
TP53	17	7661779	7687550	10000	1	Tumor Suppressor
KRAS	12	25205246	25250929	8500	1	Oncogene
PIK3CA	3	179148114	179240093	7200	1	Oncogene
BRAF	7	140719327	140924928	5800	1	Oncogene
EGFR	7	55019017	55211628	4900	1	Oncogene
PTEN	10	87863113	87971930	4500	1	Tumor Suppressor
BRCA1	17	43044295	43170245	4200	1	Tumor Suppressor
BRCA2	13	32315086	32400266	3800	1	Tumor Suppressor
APC	5	112707498	112846239	3500	1	Tumor Suppressor
RB1	13	48303751	48481890	3200	1	Tumor Suppressor
MYC	8	127735434	127742951	2900	1	Oncogene
ERBB2	17	39687914	39730426	2700	1	Oncogene
CDKN2A	9	21967751	21995300	2500	1	Tumor Suppressor
VHL	3	10141618	10154220	2300	1	Tumor Suppressor
NF1	17	31094927	31377677	2100	1	Tumor Suppressor
ATM	11	108222484	108369102	2000	1	Tumor Suppressor
ALK	2	29192774	29921586	1900	1	Oncogene
RET	10	43572517	43625799	1800	1	Oncogene
IDH1	2	208236227	208255328	1700	2	Metabolic
IDH2	15	90088606	90101916	1600	2	Metabolic
NRAS	1	114704469	114716894	1500	2	Oncogene
HRAS	11	534242	535550	1400	2	Oncogene
STK11	19	1189289	1228434	1300	2	Tumor Suppressor
SMAD4	18	51028394	51085045	1250	2	Tumor Suppressor
FBXW7	4	152322178	152427717	1200	2	Tumor Suppressor
EOF

echo -e "${GREEN}✓ COSMIC simulated data created (25 genes)${NC}"
echo

################################################################################
# Dataset 4: Ensembl GTF Annotations (Release 115 - Human chr22)
################################################################################
download_and_validate \
    "https://ftp.ensembl.org/pub/release-115/gtf/homo_sapiens/Homo_sapiens.GRCh38.115.chr.gtf.gz" \
    "Homo_sapiens.GRCh38.115.gtf.gz" \
    "SKIP" \
    "Ensembl GTF Annotations (Release 115) - 50 MB compressed"

# Decompress and extract chr22 only (to reduce size)
if [ ! -f "Homo_sapiens.GRCh38.115.chr22.gtf" ]; then
    echo -e "${YELLOW}Extracting chr22 annotations...${NC}"
    gunzip -c Homo_sapiens.GRCh38.115.gtf.gz | grep "^22\s" > Homo_sapiens.GRCh38.115.chr22.gtf || {
        # Some GTF files use "chr22" instead of "22"
        gunzip -c Homo_sapiens.GRCh38.115.gtf.gz | grep "^chr22\s" > Homo_sapiens.GRCh38.115.chr22.gtf || {
            echo -e "${RED}✗ Failed to extract chr22 annotations${NC}"
            echo -e "${YELLOW}⚠ Will use full GTF file${NC}"
            gunzip -k Homo_sapiens.GRCh38.115.gtf.gz
        }
    }
    if [ -f "Homo_sapiens.GRCh38.115.chr22.gtf" ] && [ -s "Homo_sapiens.GRCh38.115.chr22.gtf" ]; then
        echo -e "${GREEN}✓ chr22 annotations extracted${NC}"
    else
        echo -e "${YELLOW}⚠ Using full GTF file${NC}"
        gunzip -k Homo_sapiens.GRCh38.115.gtf.gz 2>/dev/null || true
    fi
    echo
fi

################################################################################
# Dataset 5: 1000 Genomes chr22 VCF (Phase 3)
################################################################################
download_and_validate \
    "https://ftp.1000genomes.ebi.ac.uk/vol1/ftp/release/20130502/ALL.chr22.phase3_shapeit2_mvncall_integrated_v5b.20130502.genotypes.vcf.gz" \
    "1000genomes_chr22.vcf.gz" \
    "SKIP" \
    "1000 Genomes chr22 VCF (Phase 3) - 100 MB VCF"

# Decompress (keep compressed for now, parsers will handle it)
echo -e "${YELLOW}Note: VCF file kept compressed. Parser will handle decompression.${NC}"
echo

################################################################################
# Summary
################################################################################
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                 DOWNLOAD COMPLETE                          ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo
echo -e "${BLUE}Downloaded datasets:${NC}"
ls -lh "$RAW_DATA_DIR" | grep -E '\.(fa|fna|tsv|gtf|vcf)' | awk '{printf "  %-40s %8s\n", $9, $5}'
echo
echo -e "${BLUE}Total size:${NC}"
du -sh "$RAW_DATA_DIR"
echo
echo -e "${GREEN}Next steps:${NC}"
echo -e "  1. Parse FASTA files:   python3 backend/scripts/fasta_to_particles.py"
echo -e "  2. Parse GTF files:     python3 backend/scripts/gtf_to_annotations.py"
echo -e "  3. Parse VCF files:     python3 backend/scripts/vcf_to_variants.py"
echo -e "  4. Compress with zstd:  zstd -19 data/tier1/*.json"
echo
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
