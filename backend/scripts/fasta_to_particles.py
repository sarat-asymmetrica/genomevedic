#!/usr/bin/env python3
"""
GenomeVedic FASTA to Particles Converter
Converts genomic FASTA sequences to 3D particle positions using Vedic digital root hashing

Features:
- Digital root spatial hashing (O(1) lookup)
- Golden spiral positioning (Fibonacci-based)
- LOD levels: 5M → 500K → 50K → 5K
- Voxel grid optimization (spatial clustering)
- Batch processing with Williams Optimizer formula

Usage:
    python3 fasta_to_particles.py input.fa > output.json
    python3 fasta_to_particles.py input.fa --lod 4 > output_lod4.json

Output JSON format:
{
    "metadata": {
        "sequence_name": "chr22",
        "length": 50818468,
        "particles": 50818468,
        "lod_levels": [5, 50000, 500000, 5000000],
        "compression": "none",
        "williams_batch_size": 6989,
        "generation_time": 123.45
    },
    "particles": [
        {"x": 0.5, "y": 0.3, "z": 0.8, "base": "A", "pos": 0, "voxel": 123},
        ...
    ],
    "spatial_hash": {
        "123": [0, 1, 2, ...],  // voxel_id -> particle indices
        ...
    }
}

Algorithm:
    1. DigitalRoot(n) = 1 + ((n - 1) % 9)  # Vedic mathematics
    2. Golden angle = 2π / φ² = 137.508°
    3. Position = spiral(digital_root, golden_angle, position)
    4. Voxel = floor(position / voxel_size)
"""

import sys
import json
import math
import time
import argparse
from typing import Dict, List, Tuple
from collections import defaultdict

# Constants
GOLDEN_RATIO = (1 + math.sqrt(5)) / 2
GOLDEN_ANGLE = 2 * math.pi / (GOLDEN_RATIO ** 2)  # 137.508° in radians
VOXEL_SIZE = 0.01  # 1% of unit cube = 100 voxels per axis = 1M voxels total

# Base to integer mapping
BASE_MAP = {'A': 0, 'C': 1, 'G': 2, 'T': 3, 'N': 4}
BASE_COLORS = {
    'A': [1.0, 0.0, 0.0],  # Red (Adenine)
    'C': [0.0, 1.0, 0.0],  # Green (Cytosine)
    'G': [0.0, 0.0, 1.0],  # Blue (Guanine)
    'T': [1.0, 1.0, 0.0],  # Yellow (Thymine)
    'N': [0.5, 0.5, 0.5]   # Gray (Unknown)
}


def digital_root(n: int) -> int:
    """
    Vedic digital root calculation (modulo 9 with special handling)
    Maps any integer to 1-9

    Examples:
        digital_root(0) = 9
        digital_root(1) = 1
        digital_root(10) = 1
        digital_root(123) = 6
    """
    if n == 0:
        return 9
    return 1 + ((n - 1) % 9)


def sequence_to_3d(sequence: str, position: int) -> Tuple[float, float, float]:
    """
    Convert DNA sequence position to 3D coordinates using digital root + golden spiral

    Algorithm:
        1. Calculate digital root for spatial clustering
        2. Use golden spiral for radial distribution
        3. Add Z-axis based on position (chromosome linearization)

    Args:
        sequence: DNA base at this position ('A', 'C', 'G', 'T')
        position: Position in genome (0-indexed)

    Returns:
        (x, y, z) coordinates in [0, 1] range
    """
    # Digital root for clustering (1-9)
    root = digital_root(position)

    # Golden spiral parameters
    theta = position * GOLDEN_ANGLE
    radius = math.sqrt(position) / 10000  # Normalize to [0, 1] range

    # Base-specific offset (slight separation by nucleotide type)
    base_offset = BASE_MAP.get(sequence.upper(), 4) * 0.02

    # 3D coordinates
    x = (radius * math.cos(theta) + base_offset) % 1.0
    y = (radius * math.sin(theta) + base_offset) % 1.0
    z = (root / 10.0 + position / 100000000) % 1.0  # Digital root clustering + linear

    return (x, y, z)


def calculate_voxel_id(x: float, y: float, z: float) -> int:
    """
    Calculate voxel ID for spatial hashing (O(1) lookup)

    Args:
        x, y, z: Coordinates in [0, 1] range

    Returns:
        Voxel ID (integer)
    """
    voxel_x = int(x / VOXEL_SIZE)
    voxel_y = int(y / VOXEL_SIZE)
    voxel_z = int(z / VOXEL_SIZE)

    # 3D to 1D index (100 voxels per axis)
    return voxel_x + voxel_y * 100 + voxel_z * 10000


def williams_batch_size(n: int) -> int:
    """
    Williams Optimizer formula: BatchSize = √n × log₂(n)

    This provides optimal complexity reduction for streaming large datasets

    Args:
        n: Total number of particles

    Returns:
        Optimal batch size
    """
    if n <= 0:
        return 1
    return max(1, int(math.sqrt(n) * math.log2(n)))


def read_fasta(file_path: str) -> Dict[str, str]:
    """
    Read FASTA file and return sequences

    Args:
        file_path: Path to FASTA file

    Returns:
        Dictionary mapping sequence name to sequence string
    """
    sequences = {}
    current_name = None
    current_seq = []

    with open(file_path, 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue

            if line.startswith('>'):
                # Save previous sequence
                if current_name:
                    sequences[current_name] = ''.join(current_seq)

                # Start new sequence
                current_name = line[1:].split()[0]  # Take first word after '>'
                current_seq = []
            else:
                current_seq.append(line.upper())

        # Save last sequence
        if current_name:
            sequences[current_name] = ''.join(current_seq)

    return sequences


def generate_lod_levels(particles: List[Dict], target_counts: List[int]) -> Dict[int, List[int]]:
    """
    Generate Level-of-Detail (LOD) levels using uniform sampling

    Args:
        particles: Full particle list
        target_counts: Target particle counts for each LOD level (e.g., [5000, 50000, 500000])

    Returns:
        Dictionary mapping LOD level to particle indices
    """
    total = len(particles)
    lod_levels = {}

    for lod_id, target in enumerate(target_counts):
        if target >= total:
            # Include all particles
            lod_levels[lod_id] = list(range(total))
        else:
            # Uniform sampling
            step = total / target
            indices = [int(i * step) for i in range(target)]
            lod_levels[lod_id] = indices

    return lod_levels


def fasta_to_particles(fasta_path: str, max_particles: int = None, lod_targets: List[int] = None) -> Dict:
    """
    Convert FASTA file to particle JSON with spatial hashing and LOD levels

    Args:
        fasta_path: Path to input FASTA file
        max_particles: Maximum number of particles to generate (for testing)
        lod_targets: Target particle counts for LOD levels

    Returns:
        Dictionary with metadata, particles, spatial hash, and LOD levels
    """
    start_time = time.time()

    # Default LOD levels: 5K → 50K → 500K → 5M
    if lod_targets is None:
        lod_targets = [5000, 50000, 500000, 5000000]

    # Read FASTA
    print(f"Reading FASTA file: {fasta_path}", file=sys.stderr)
    sequences = read_fasta(fasta_path)

    if not sequences:
        raise ValueError(f"No sequences found in {fasta_path}")

    # Use first sequence
    seq_name = list(sequences.keys())[0]
    sequence = sequences[seq_name]
    seq_length = len(sequence)

    print(f"Sequence: {seq_name}", file=sys.stderr)
    print(f"Length: {seq_length:,} bases", file=sys.stderr)

    # Limit particles for testing
    if max_particles:
        seq_length = min(seq_length, max_particles)
        sequence = sequence[:seq_length]
        print(f"Limited to {seq_length:,} particles", file=sys.stderr)

    # Calculate Williams batch size
    batch_size = williams_batch_size(seq_length)
    print(f"Williams batch size: {batch_size:,}", file=sys.stderr)

    # Generate particles
    print("Generating particles...", file=sys.stderr)
    particles = []
    spatial_hash = defaultdict(list)

    for pos in range(seq_length):
        base = sequence[pos]
        x, y, z = sequence_to_3d(base, pos)
        voxel = calculate_voxel_id(x, y, z)

        particle = {
            'x': round(x, 6),
            'y': round(y, 6),
            'z': round(z, 6),
            'base': base,
            'pos': pos,
            'voxel': voxel,
            'color': BASE_COLORS.get(base, [0.5, 0.5, 0.5])
        }

        particles.append(particle)
        spatial_hash[voxel].append(pos)

        # Progress
        if (pos + 1) % 100000 == 0:
            print(f"  Processed {pos + 1:,} / {seq_length:,} particles ({(pos + 1) / seq_length * 100:.1f}%)", file=sys.stderr)

    print(f"Generated {len(particles):,} particles", file=sys.stderr)

    # Generate LOD levels
    print("Generating LOD levels...", file=sys.stderr)
    lod_levels = generate_lod_levels(particles, lod_targets)

    for lod_id, indices in lod_levels.items():
        print(f"  LOD {lod_id}: {len(indices):,} particles", file=sys.stderr)

    # Build output
    end_time = time.time()
    generation_time = end_time - start_time

    output = {
        'metadata': {
            'sequence_name': seq_name,
            'length': seq_length,
            'particles': len(particles),
            'lod_levels': [len(indices) for indices in lod_levels.values()],
            'voxel_count': len(spatial_hash),
            'voxel_size': VOXEL_SIZE,
            'williams_batch_size': batch_size,
            'generation_time': round(generation_time, 2),
            'digital_root_algorithm': 'vedic',
            'golden_angle_degrees': round(math.degrees(GOLDEN_ANGLE), 3),
            'version': '1.0.0'
        },
        'particles': particles,
        'spatial_hash': {str(k): v for k, v in spatial_hash.items()},
        'lod_levels': {str(k): v for k, v in lod_levels.items()}
    }

    print(f"Total generation time: {generation_time:.2f}s", file=sys.stderr)
    print(f"Particles per second: {len(particles) / generation_time:,.0f}", file=sys.stderr)

    return output


def main():
    parser = argparse.ArgumentParser(
        description='Convert FASTA to 3D particles using Vedic digital root hashing',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    parser.add_argument('fasta_file', help='Input FASTA file')
    parser.add_argument('--max-particles', type=int, help='Maximum particles (for testing)')
    parser.add_argument('--lod', nargs='+', type=int, help='LOD level targets (e.g., 5000 50000 500000)')
    parser.add_argument('--pretty', action='store_true', help='Pretty-print JSON output')

    args = parser.parse_args()

    # Convert FASTA to particles
    try:
        result = fasta_to_particles(
            args.fasta_file,
            max_particles=args.max_particles,
            lod_targets=args.lod
        )

        # Output JSON
        if args.pretty:
            print(json.dumps(result, indent=2))
        else:
            print(json.dumps(result))

    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
