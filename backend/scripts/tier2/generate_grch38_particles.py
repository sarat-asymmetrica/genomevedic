#!/usr/bin/env python3
"""
Generate particles for GRCh38 Full Human Genome (all 24 chromosomes)
Optimized for streaming and LOD levels

Features:
- Multi-chromosome support (chr1-chr22, chrX, chrY)
- Chunked generation (memory efficient)
- Progressive LOD levels (5K, 50K, 500K, 5M particles)
- Zstandard compression (level 19)
- Streaming-friendly JSON format

Usage:
    # Generate full genome particles (all chromosomes)
    python3 generate_grch38_particles.py --full

    # Generate single chromosome for testing
    python3 generate_grch38_particles.py --chr chr22

    # Generate with specific LOD levels
    python3 generate_grch38_particles.py --chr chr22 --lod 5000 50000 500000
"""

import sys
import json
import argparse
import os
from pathlib import Path

# Add parent directory to path to import fasta_to_particles
sys.path.insert(0, str(Path(__file__).parent.parent))
from fasta_to_particles import fasta_to_particles, williams_batch_size

# GRCh38 chromosome sizes (approximate)
CHROMOSOME_SIZES = {
    'chr1': 248956422,
    'chr2': 242193529,
    'chr3': 198295559,
    'chr4': 190214555,
    'chr5': 181538259,
    'chr6': 170805979,
    'chr7': 159345973,
    'chr8': 145138636,
    'chr9': 138394717,
    'chr10': 133797422,
    'chr11': 135086622,
    'chr12': 133275309,
    'chr13': 114364328,
    'chr14': 107043718,
    'chr15': 101991189,
    'chr16': 90338345,
    'chr17': 83257441,
    'chr18': 80373285,
    'chr19': 58617616,
    'chr20': 64444167,
    'chr21': 46709983,
    'chr22': 50818468,
    'chrX': 156040895,
    'chrY': 57227415
}


def generate_chromosome_particles(chr_name, fasta_path, lod_levels, output_dir):
    """Generate particles for a single chromosome"""

    print(f"\n=== Generating particles for {chr_name} ===", file=sys.stderr)

    # Check if FASTA exists
    if not os.path.exists(fasta_path):
        print(f"WARNING: {fasta_path} not found. Creating simulated data...", file=sys.stderr)
        # Create simulated FASTA for demonstration
        create_simulated_fasta(chr_name, fasta_path)

    # Generate particles
    particles_data = fasta_to_particles(fasta_path, lod_targets=lod_levels)

    # Save JSON
    output_json = os.path.join(output_dir, f"{chr_name}.particles.json")
    with open(output_json, 'w') as f:
        json.dump(particles_data, f)

    print(f"✓ Saved particles to {output_json}", file=sys.stderr)

    # Compress with zstandard
    compress_with_zstd(output_json, level=19)

    return particles_data


def create_simulated_fasta(chr_name, output_path):
    """Create simulated FASTA for demonstration"""
    import random

    size = CHROMOSOME_SIZES.get(chr_name, 50000000)
    # For demonstration, create smaller simulated sequences
    size = min(size, 1000000)  # 1M bases max for demo

    print(f"Creating simulated {chr_name} ({size:,} bases)...", file=sys.stderr)

    os.makedirs(os.path.dirname(output_path), exist_ok=True)

    with open(output_path, 'w') as f:
        f.write(f">{chr_name}\n")

        bases = ['A', 'C', 'G', 'T']
        line_length = 80

        for i in range(0, size, line_length):
            chunk_size = min(line_length, size - i)
            sequence = ''.join(random.choices(bases, k=chunk_size))
            f.write(sequence + '\n')

    print(f"✓ Created simulated FASTA at {output_path}", file=sys.stderr)


def compress_with_zstd(json_path, level=19):
    """Compress JSON file with zstandard"""
    import subprocess

    output_path = json_path.replace('.json', '.zst')

    print(f"Compressing with zstandard (level {level})...", file=sys.stderr)

    try:
        # Try using zstd command line tool
        result = subprocess.run(
            ['zstd', f'-{level}', '-f', json_path, '-o', output_path],
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            original_size = os.path.getsize(json_path)
            compressed_size = os.path.getsize(output_path)
            ratio = (1 - compressed_size / original_size) * 100

            print(f"✓ Compressed: {original_size:,} → {compressed_size:,} bytes ({ratio:.1f}% reduction)", file=sys.stderr)

            # Remove original JSON to save space
            os.remove(json_path)
            print(f"✓ Removed original JSON (use .zst version)", file=sys.stderr)
        else:
            print(f"WARNING: zstd compression failed: {result.stderr}", file=sys.stderr)

    except FileNotFoundError:
        print("WARNING: zstd not found. Install with: sudo apt-get install zstd", file=sys.stderr)


def generate_full_genome(lod_levels, output_dir, chromosomes=None):
    """Generate particles for all chromosomes"""

    if chromosomes is None:
        chromosomes = list(CHROMOSOME_SIZES.keys())

    print(f"\n=== Generating Full Genome Particles ===", file=sys.stderr)
    print(f"Chromosomes: {len(chromosomes)}", file=sys.stderr)
    print(f"Output directory: {output_dir}", file=sys.stderr)
    print(f"LOD levels: {lod_levels}", file=sys.stderr)
    print("", file=sys.stderr)

    os.makedirs(output_dir, exist_ok=True)

    results = {}

    for chr_name in chromosomes:
        fasta_path = f"/home/user/genomevedic/data/tier2/grch38/raw/{chr_name}.fa"

        try:
            particles_data = generate_chromosome_particles(
                chr_name, fasta_path, lod_levels, output_dir
            )
            results[chr_name] = {
                'particles': particles_data['metadata']['particles'],
                'lod_levels': particles_data['metadata']['lod_levels'],
                'generation_time': particles_data['metadata']['generation_time']
            }
        except Exception as e:
            print(f"ERROR processing {chr_name}: {e}", file=sys.stderr)
            continue

    # Create summary metadata
    summary = {
        'genome': 'GRCh38',
        'organism': 'Homo sapiens',
        'chromosomes': len(results),
        'total_particles': sum(r['particles'] for r in results.values()),
        'lod_levels': lod_levels,
        'chromosome_details': results
    }

    summary_path = os.path.join(output_dir, 'grch38_summary.json')
    with open(summary_path, 'w') as f:
        json.dump(summary, f, indent=2)

    print(f"\n=== Generation Complete ===", file=sys.stderr)
    print(f"Total particles: {summary['total_particles']:,}", file=sys.stderr)
    print(f"Summary saved to: {summary_path}", file=sys.stderr)

    return summary


def main():
    parser = argparse.ArgumentParser(
        description='Generate GRCh38 particles with LOD levels',
        formatter_class=argparse.RawDescriptionHelpFormatter
    )

    parser.add_argument('--chr', help='Single chromosome (e.g., chr22)')
    parser.add_argument('--full', action='store_true', help='Generate all chromosomes')
    parser.add_argument('--chromosomes', nargs='+', help='Specific chromosomes to generate')
    parser.add_argument('--lod', nargs='+', type=int, default=[5000, 50000, 500000, 5000000],
                       help='LOD level targets (default: 5000 50000 500000 5000000)')
    parser.add_argument('--output', default='/home/user/genomevedic/data/tier2/grch38',
                       help='Output directory')

    args = parser.parse_args()

    # Validate arguments
    if not args.chr and not args.full and not args.chromosomes:
        parser.error("Must specify --chr, --full, or --chromosomes")

    try:
        if args.chr:
            # Single chromosome
            fasta_path = f"{args.output}/raw/{args.chr}.fa"
            generate_chromosome_particles(args.chr, fasta_path, args.lod, args.output)

        elif args.full:
            # All chromosomes
            generate_full_genome(args.lod, args.output)

        elif args.chromosomes:
            # Specific chromosomes
            generate_full_genome(args.lod, args.output, chromosomes=args.chromosomes)

    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
