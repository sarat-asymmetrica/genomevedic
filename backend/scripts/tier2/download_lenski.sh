#!/bin/bash
# Download Lenski E. coli Long-Term Evolution Experiment Data
# 50,000 generations of E. coli evolution
# Total size: ~100 MB

set -e

DOWNLOAD_DIR="/home/user/genomevedic/data/tier2/lenski"
BARRICK_LAB_BASE="https://raw.githubusercontent.com/barricklab/LTEE-Ecoli/master"

echo "=== Lenski LTEE Data Download ==="
echo "Target directory: $DOWNLOAD_DIR"
echo ""

mkdir -p "$DOWNLOAD_DIR/raw"

echo "Downloading Lenski LTEE genome data..."
echo ""

# Download reference genomes for different timepoints
# NOTE: This is a simulated dataset based on the Lenski LTEE structure
# Real data available at: https://barricklab.org/twiki/bin/view/Lab/EvolvingE.coliStrains

# Create simulated evolution timeline
GENERATIONS=(0 2000 5000 10000 15000 20000 30000 40000 50000)

echo "Creating simulated evolution data for ${#GENERATIONS[@]} timepoints..."

for gen in "${GENERATIONS[@]}"; do
    sample_id="REL606_gen${gen}"

    echo "Generating $sample_id..."

    # Create simulated VCF with increasing mutations over time
    # Real LTEE shows ~2 mutations per 1000 generations
    num_mutations=$((gen / 500))

    cat > "$DOWNLOAD_DIR/raw/${sample_id}.vcf" <<EOF
##fileformat=VCFv4.2
##source=GenomeVedicSimulatedLTEE
##reference=REL606
##INFO=<ID=GEN,Number=1,Type=Integer,Description="Generation number">
##INFO=<ID=TYPE,Number=1,Type=String,Description="Mutation type">
##FORMAT=<ID=GT,Number=1,Type=String,Description="Genotype">
#CHROM	POS	ID	REF	ALT	QUAL	FILTER	INFO	FORMAT	${sample_id}
EOF

    # Add mutations (scale with generations)
    if [ "$num_mutations" -gt 0 ]; then
        for ((i=1; i<=num_mutations; i++)); do
            pos=$((1000000 + i * 10000))
            echo -e "NC_012967.1\t${pos}\t.\tA\tT\t100\tPASS\tGEN=${gen};TYPE=SNP\tGT\t1/1" >> "$DOWNLOAD_DIR/raw/${sample_id}.vcf"
        done
    fi

    echo "  ✓ Created ${sample_id}.vcf ($num_mutations mutations)"
done

# Create metadata file
cat > "$DOWNLOAD_DIR/metadata.json" <<EOF
{
  "experiment": "Long-Term Evolution Experiment (LTEE)",
  "organism": "Escherichia coli B",
  "strain": "REL606",
  "principal_investigator": "Richard Lenski",
  "start_date": "1988-02-15",
  "generations_total": 50000,
  "timepoints": $(printf '%s\n' "${GENERATIONS[@]}" | jq -R . | jq -s .),
  "population_count": 12,
  "description": "Longest-running evolution experiment. E. coli populations grown for 50,000+ generations, showing adaptation and evolutionary dynamics.",
  "key_findings": [
    "Fitness improvement over time",
    "Citrate+ mutation (Cit+) at ~31,500 generations",
    "Parallel evolution across populations",
    "Diminishing returns epistasis"
  ],
  "references": [
    "Lenski RE et al. (1991) Long-term experimental evolution in Escherichia coli.",
    "Blount ZD et al. (2012) Historical contingency and the evolution of a key innovation."
  ],
  "source": "https://barricklab.org/twiki/bin/view/Lab/EvolvingE.coliStrains",
  "data_type": "simulated_for_demonstration"
}
EOF

echo ""
echo "=== Download Summary ==="
echo "Evolution timepoints: ${#GENERATIONS[@]}"
du -sh "$DOWNLOAD_DIR/raw"
cat "$DOWNLOAD_DIR/metadata.json" | jq .
echo ""
echo "NOTE: This is SIMULATED data for demonstration."
echo "For real Lenski LTEE data:"
echo "  1. Visit: https://barricklab.org/twiki/bin/view/Lab/EvolvingE.coliStrains"
echo "  2. Download genome sequences from NCBI SRA"
echo "  3. Run variant calling pipeline"
echo ""
echo "Next steps:"
echo "  1. Run vcf_to_variants.py to convert to variant JSON"
echo "  2. Generate evolution animation trails"
echo "  3. Compress with zstandard"
