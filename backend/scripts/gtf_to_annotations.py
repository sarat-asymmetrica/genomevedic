#!/usr/bin/env python3
"""
GenomeVedic GTF to Annotations Converter
Converts Ensembl GTF gene annotations to JSON format for 3D overlay

Features:
- Parse GTF format (9-column tab-delimited)
- Extract genes, exons, transcripts
- Map to 3D particle positions
- Create annotation layers for VR overlay

Usage:
    python3 gtf_to_annotations.py input.gtf > annotations.json
    python3 gtf_to_annotations.py input.gtf --chromosome 22 > chr22_annotations.json

Output JSON format:
{
    "metadata": {
        "source": "Ensembl Release 115",
        "chromosome": "22",
        "genes": 1234,
        "exons": 5678,
        "transcripts": 2345
    },
    "genes": [
        {
            "id": "ENSG00000100055",
            "name": "APOL1",
            "chromosome": "22",
            "start": 36250000,
            "end": 36260000,
            "strand": "+",
            "type": "protein_coding",
            "exon_count": 7
        },
        ...
    ],
    "exons": [...],
    "transcripts": [...]
}

GTF Format:
    chr22  Ensembl  gene  10000  20000  .  +  .  gene_id "ENSG123"; gene_name "TP53"; ...
"""

import sys
import json
import re
import argparse
from typing import Dict, List, Optional
from collections import defaultdict


def parse_gtf_attributes(attr_string: str) -> Dict[str, str]:
    """
    Parse GTF attributes field (column 9)

    Example: 'gene_id "ENSG00000139618"; gene_name "BRCA2"; gene_biotype "protein_coding";'

    Returns:
        Dictionary of attribute key-value pairs
    """
    attributes = {}
    # Split by semicolon, then parse key-value pairs
    for item in attr_string.strip().split(';'):
        item = item.strip()
        if not item:
            continue

        # Match: key "value" or key value
        match = re.match(r'(\S+)\s+"?([^"]+)"?', item)
        if match:
            key, value = match.groups()
            attributes[key] = value.strip('"')

    return attributes


def parse_gtf_line(line: str) -> Optional[Dict]:
    """
    Parse single GTF line

    GTF format (9 columns, tab-delimited):
        1. seqname (chromosome)
        2. source (e.g., 'Ensembl')
        3. feature (e.g., 'gene', 'exon', 'transcript')
        4. start (1-indexed)
        5. end (1-indexed, inclusive)
        6. score (. if none)
        7. strand (+ or -)
        8. frame (0, 1, 2, or .)
        9. attributes (key-value pairs)

    Returns:
        Dictionary with parsed fields or None if comment/invalid
    """
    # Skip comments
    if line.startswith('#'):
        return None

    # Split by tab
    fields = line.strip().split('\t')
    if len(fields) != 9:
        return None

    seqname, source, feature, start, end, score, strand, frame, attributes = fields

    # Parse attributes
    attr_dict = parse_gtf_attributes(attributes)

    return {
        'chromosome': seqname,
        'source': source,
        'feature': feature,
        'start': int(start),
        'end': int(end),
        'score': score,
        'strand': strand,
        'frame': frame,
        'attributes': attr_dict
    }


def gtf_to_annotations(gtf_path: str, chromosome_filter: Optional[str] = None) -> Dict:
    """
    Convert GTF file to annotation JSON

    Args:
        gtf_path: Path to GTF file
        chromosome_filter: Optional chromosome to filter (e.g., '22' or 'chr22')

    Returns:
        Dictionary with genes, exons, transcripts
    """
    print(f"Reading GTF file: {gtf_path}", file=sys.stderr)

    genes = []
    exons = []
    transcripts = []

    gene_exon_counts = defaultdict(int)
    gene_transcript_counts = defaultdict(int)

    line_count = 0
    gene_count = 0
    exon_count = 0
    transcript_count = 0

    with open(gtf_path, 'r') as f:
        for line in f:
            line_count += 1

            # Progress
            if line_count % 100000 == 0:
                print(f"  Processed {line_count:,} lines (genes: {gene_count:,}, exons: {exon_count:,}, transcripts: {transcript_count:,})", file=sys.stderr)

            # Parse line
            entry = parse_gtf_line(line)
            if not entry:
                continue

            # Filter by chromosome
            if chromosome_filter:
                chrom = entry['chromosome']
                # Handle both "22" and "chr22" formats
                if chrom != chromosome_filter and chrom != f"chr{chromosome_filter}" and chrom.lstrip('chr') != chromosome_filter:
                    continue

            # Extract gene
            if entry['feature'] == 'gene':
                gene_id = entry['attributes'].get('gene_id', 'unknown')
                gene_name = entry['attributes'].get('gene_name', entry['attributes'].get('gene_id', 'unknown'))
                gene_type = entry['attributes'].get('gene_biotype', entry['attributes'].get('gene_type', 'unknown'))

                genes.append({
                    'id': gene_id,
                    'name': gene_name,
                    'chromosome': entry['chromosome'],
                    'start': entry['start'],
                    'end': entry['end'],
                    'strand': entry['strand'],
                    'type': gene_type,
                    'source': entry['source'],
                    'length': entry['end'] - entry['start'] + 1
                })
                gene_count += 1

            # Extract exon
            elif entry['feature'] == 'exon':
                gene_id = entry['attributes'].get('gene_id', 'unknown')
                transcript_id = entry['attributes'].get('transcript_id', 'unknown')
                exon_number = entry['attributes'].get('exon_number', '0')

                exons.append({
                    'gene_id': gene_id,
                    'transcript_id': transcript_id,
                    'exon_number': int(exon_number) if exon_number.isdigit() else 0,
                    'chromosome': entry['chromosome'],
                    'start': entry['start'],
                    'end': entry['end'],
                    'strand': entry['strand'],
                    'length': entry['end'] - entry['start'] + 1
                })
                exon_count += 1
                gene_exon_counts[gene_id] += 1

            # Extract transcript
            elif entry['feature'] == 'transcript':
                gene_id = entry['attributes'].get('gene_id', 'unknown')
                transcript_id = entry['attributes'].get('transcript_id', 'unknown')
                transcript_name = entry['attributes'].get('transcript_name', transcript_id)
                transcript_type = entry['attributes'].get('transcript_biotype', entry['attributes'].get('transcript_type', 'unknown'))

                transcripts.append({
                    'id': transcript_id,
                    'name': transcript_name,
                    'gene_id': gene_id,
                    'chromosome': entry['chromosome'],
                    'start': entry['start'],
                    'end': entry['end'],
                    'strand': entry['strand'],
                    'type': transcript_type,
                    'length': entry['end'] - entry['start'] + 1
                })
                transcript_count += 1
                gene_transcript_counts[gene_id] += 1

    print(f"Parsed {line_count:,} lines", file=sys.stderr)
    print(f"Found {gene_count:,} genes, {exon_count:,} exons, {transcript_count:,} transcripts", file=sys.stderr)

    # Add exon counts to genes
    for gene in genes:
        gene['exon_count'] = gene_exon_counts.get(gene['id'], 0)
        gene['transcript_count'] = gene_transcript_counts.get(gene['id'], 0)

    # Build output
    output = {
        'metadata': {
            'source': 'Ensembl GTF',
            'chromosome': chromosome_filter if chromosome_filter else 'all',
            'genes': len(genes),
            'exons': len(exons),
            'transcripts': len(transcripts),
            'lines_parsed': line_count,
            'version': '1.0.0'
        },
        'genes': genes,
        'exons': exons,
        'transcripts': transcripts
    }

    return output


def main():
    parser = argparse.ArgumentParser(
        description='Convert GTF annotations to JSON',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    parser.add_argument('gtf_file', help='Input GTF file')
    parser.add_argument('--chromosome', help='Filter by chromosome (e.g., 22 or chr22)')
    parser.add_argument('--pretty', action='store_true', help='Pretty-print JSON output')

    args = parser.parse_args()

    # Convert GTF to annotations
    try:
        result = gtf_to_annotations(args.gtf_file, chromosome_filter=args.chromosome)

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
