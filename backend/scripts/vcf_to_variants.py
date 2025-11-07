#!/usr/bin/env python3
"""
GenomeVedic VCF to Variants Converter
Converts VCF (Variant Call Format) files to JSON for 3D visualization

Features:
- Parse VCF format (1000 Genomes, COSMIC, etc.)
- Extract variants (SNPs, INDELs)
- Calculate allele frequencies
- Annotate variant impact (if available)
- Support compressed VCF (.vcf.gz)

Usage:
    python3 vcf_to_variants.py input.vcf > variants.json
    python3 vcf_to_variants.py input.vcf.gz --max-variants 1000 > sample.json

Output JSON format:
{
    "metadata": {
        "source": "1000 Genomes Phase 3",
        "chromosome": "22",
        "variants": 123456,
        "samples": 2504
    },
    "variants": [
        {
            "id": "rs123456",
            "chromosome": "22",
            "position": 16050000,
            "ref": "A",
            "alt": ["G"],
            "quality": 100,
            "filter": "PASS",
            "info": {
                "AF": 0.25,
                "AC": 1252,
                "AN": 5008
            },
            "type": "SNP"
        },
        ...
    ]
}

VCF Format (columns):
    1. CHROM - Chromosome
    2. POS - Position (1-indexed)
    3. ID - Variant ID (e.g., rs123456)
    4. REF - Reference allele
    5. ALT - Alternate allele(s)
    6. QUAL - Quality score
    7. FILTER - Filter status (PASS, etc.)
    8. INFO - Additional information (key=value pairs)
    9+ Sample genotypes (optional)
"""

import sys
import json
import gzip
import argparse
from typing import Dict, List, Optional


def parse_vcf_info(info_string: str) -> Dict:
    """
    Parse VCF INFO field (key=value pairs)

    Example: "AF=0.25;AC=1252;AN=5008;DP=12345"

    Returns:
        Dictionary of info fields
    """
    info = {}
    for item in info_string.split(';'):
        if '=' in item:
            key, value = item.split('=', 1)
            # Try to convert to appropriate type
            try:
                if ',' in value:
                    # Multiple values
                    info[key] = [float(v) if '.' in v else int(v) for v in value.split(',')]
                elif '.' in value:
                    info[key] = float(value)
                else:
                    info[key] = int(value)
            except ValueError:
                info[key] = value
        else:
            # Flag (no value)
            info[item] = True

    return info


def parse_vcf_line(line: str, sample_count: int = 0) -> Optional[Dict]:
    """
    Parse single VCF variant line

    Args:
        line: VCF line
        sample_count: Number of samples in VCF (for parsing genotypes)

    Returns:
        Dictionary with variant fields or None if header/comment
    """
    # Skip headers and comments
    if line.startswith('#'):
        return None

    # Split by tab
    fields = line.strip().split('\t')
    if len(fields) < 8:
        return None

    chrom, pos, vid, ref, alt, qual, filt, info = fields[:8]

    # Parse INFO field
    info_dict = parse_vcf_info(info)

    # Determine variant type
    alt_alleles = alt.split(',')
    variant_type = 'SNP' if all(len(ref) == 1 and len(a) == 1 for a in alt_alleles) else 'INDEL'

    # Extract key info fields
    allele_freq = info_dict.get('AF', None)
    allele_count = info_dict.get('AC', None)
    allele_number = info_dict.get('AN', None)

    variant = {
        'id': vid if vid != '.' else None,
        'chromosome': chrom,
        'position': int(pos),
        'ref': ref,
        'alt': alt_alleles,
        'quality': float(qual) if qual != '.' else None,
        'filter': filt,
        'info': info_dict,
        'type': variant_type
    }

    # Add simplified fields for easy access
    if allele_freq is not None:
        variant['allele_frequency'] = allele_freq if isinstance(allele_freq, float) else allele_freq[0]
    if allele_count is not None:
        variant['allele_count'] = allele_count if isinstance(allele_count, int) else allele_count[0]
    if allele_number is not None:
        variant['allele_number'] = allele_number

    return variant


def open_vcf(vcf_path: str):
    """
    Open VCF file (supports .vcf and .vcf.gz)

    Args:
        vcf_path: Path to VCF file

    Returns:
        File handle
    """
    if vcf_path.endswith('.gz'):
        return gzip.open(vcf_path, 'rt')
    else:
        return open(vcf_path, 'r')


def vcf_to_variants(vcf_path: str, max_variants: Optional[int] = None, chromosome_filter: Optional[str] = None) -> Dict:
    """
    Convert VCF file to variant JSON

    Args:
        vcf_path: Path to VCF file
        max_variants: Maximum variants to parse (for testing)
        chromosome_filter: Optional chromosome to filter (e.g., '22' or 'chr22')

    Returns:
        Dictionary with metadata and variants
    """
    print(f"Reading VCF file: {vcf_path}", file=sys.stderr)

    variants = []
    metadata = {
        'source': vcf_path,
        'version': None,
        'samples': 0,
        'contigs': []
    }

    line_count = 0
    variant_count = 0
    sample_count = 0

    with open_vcf(vcf_path) as f:
        for line in f:
            line_count += 1

            # Parse headers
            if line.startswith('##'):
                # Extract metadata
                if line.startswith('##fileformat='):
                    metadata['version'] = line.split('=', 1)[1].strip()
                elif line.startswith('##contig='):
                    # Extract contig info (optional)
                    pass
                continue

            # Parse column header
            if line.startswith('#CHROM'):
                columns = line.strip().split('\t')
                if len(columns) > 9:
                    sample_count = len(columns) - 9
                    metadata['samples'] = sample_count
                    print(f"Found {sample_count} samples", file=sys.stderr)
                continue

            # Progress
            if variant_count > 0 and variant_count % 10000 == 0:
                print(f"  Processed {variant_count:,} variants", file=sys.stderr)

            # Parse variant
            variant = parse_vcf_line(line, sample_count)
            if not variant:
                continue

            # Filter by chromosome
            if chromosome_filter:
                chrom = variant['chromosome']
                if chrom != chromosome_filter and chrom != f"chr{chromosome_filter}" and chrom.lstrip('chr') != chromosome_filter:
                    continue

            variants.append(variant)
            variant_count += 1

            # Max variants limit
            if max_variants and variant_count >= max_variants:
                print(f"Reached max variants limit: {max_variants:,}", file=sys.stderr)
                break

    print(f"Parsed {line_count:,} lines", file=sys.stderr)
    print(f"Found {variant_count:,} variants", file=sys.stderr)

    # Update metadata
    metadata['variants'] = len(variants)
    if variants:
        metadata['chromosome'] = variants[0]['chromosome']

    # Build output
    output = {
        'metadata': metadata,
        'variants': variants
    }

    return output


def main():
    parser = argparse.ArgumentParser(
        description='Convert VCF to JSON',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    parser.add_argument('vcf_file', help='Input VCF file (.vcf or .vcf.gz)')
    parser.add_argument('--max-variants', type=int, help='Maximum variants to parse (for testing)')
    parser.add_argument('--chromosome', help='Filter by chromosome (e.g., 22 or chr22)')
    parser.add_argument('--pretty', action='store_true', help='Pretty-print JSON output')

    args = parser.parse_args()

    # Convert VCF to variants
    try:
        result = vcf_to_variants(
            args.vcf_file,
            max_variants=args.max_variants,
            chromosome_filter=args.chromosome
        )

        # Output JSON
        if args.pretty:
            print(json.dumps(result, indent=2))
        else:
            print(json.dumps(result))

    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
