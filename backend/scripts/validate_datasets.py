#!/usr/bin/env python3
"""
GenomeVedic Dataset Validator
Validates processed datasets for correctness and performance

Usage:
    python3 validate_datasets.py data/tier1/ecoli_k12.particles.json
"""

import sys
import json
import time
from pathlib import Path


def validate_particle_data(file_path: str):
    """Validate particle JSON structure and data integrity"""
    print(f"Validating: {file_path}")
    print("=" * 60)

    # Load JSON
    start_time = time.time()
    with open(file_path, 'r') as f:
        data = json.load(f)
    load_time = time.time() - start_time

    # Check metadata
    metadata = data.get('metadata', {})
    print(f"✓ Metadata found")
    print(f"  - Sequence: {metadata.get('sequence_name')}")
    print(f"  - Length: {metadata.get('length'):,}")
    print(f"  - Particles: {metadata.get('particles'):,}")
    print(f"  - Voxels: {metadata.get('voxel_count'):,}")
    print(f"  - Williams batch size: {metadata.get('williams_batch_size'):,}")
    print(f"  - Generation time: {metadata.get('generation_time')}s")
    print(f"  - Load time: {load_time:.2f}s")
    print()

    # Check particles
    particles = data.get('particles', [])
    print(f"✓ Particles: {len(particles):,}")

    if len(particles) > 0:
        sample = particles[0]
        print(f"  Sample particle: {sample}")

        # Validate structure
        required_fields = ['x', 'y', 'z', 'base', 'pos', 'voxel', 'color']
        for field in required_fields:
            if field not in sample:
                print(f"✗ Missing field: {field}")
                return False
        print(f"  ✓ All required fields present")
    print()

    # Check spatial hash
    spatial_hash = data.get('spatial_hash', {})
    print(f"✓ Spatial hash: {len(spatial_hash):,} voxels")

    # Check LOD levels
    lod_levels = data.get('lod_levels', {})
    print(f"✓ LOD levels: {len(lod_levels)}")
    for lod_id, indices in lod_levels.items():
        print(f"  - LOD {lod_id}: {len(indices):,} particles")
    print()

    # Validate particle positions
    print("Validating particle positions...")
    for i, p in enumerate(particles[:100]):  # Check first 100
        if not (0 <= p['x'] <= 1 and 0 <= p['y'] <= 1 and 0 <= p['z'] <= 1):
            print(f"✗ Invalid position at particle {i}: ({p['x']}, {p['y']}, {p['z']})")
            return False
    print("  ✓ Particle positions valid (sampled 100)")
    print()

    # File size
    file_size = Path(file_path).stat().st_size
    print(f"File size: {file_size / 1024 / 1024:.2f} MB")
    print(f"Bytes per particle: {file_size / len(particles):.2f}")
    print()

    return True


def validate_annotation_data(file_path: str):
    """Validate annotation JSON structure"""
    print(f"Validating: {file_path}")
    print("=" * 60)

    # Load JSON
    start_time = time.time()
    with open(file_path, 'r') as f:
        data = json.load(f)
    load_time = time.time() - start_time

    # Check metadata
    metadata = data.get('metadata', {})
    print(f"✓ Metadata found")
    print(f"  - Source: {metadata.get('source')}")
    print(f"  - Chromosome: {metadata.get('chromosome')}")
    print(f"  - Genes: {metadata.get('genes'):,}")
    print(f"  - Exons: {metadata.get('exons'):,}")
    print(f"  - Transcripts: {metadata.get('transcripts'):,}")
    print(f"  - Load time: {load_time:.2f}s")
    print()

    # Check genes
    genes = data.get('genes', [])
    print(f"✓ Genes: {len(genes):,}")
    if len(genes) > 0:
        sample = genes[0]
        print(f"  Sample gene: {sample.get('name')} ({sample.get('id')})")
        print(f"    Position: chr{sample.get('chromosome')}:{sample.get('start')}-{sample.get('end')}")
        print(f"    Type: {sample.get('type')}")
        print(f"    Exons: {sample.get('exon_count')}")
    print()

    # File size
    file_size = Path(file_path).stat().st_size
    print(f"File size: {file_size / 1024 / 1024:.2f} MB")
    print()

    return True


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 validate_datasets.py <file.json>")
        sys.exit(1)

    file_path = sys.argv[1]

    if not Path(file_path).exists():
        print(f"ERROR: File not found: {file_path}")
        sys.exit(1)

    # Determine file type
    if 'particles' in file_path:
        valid = validate_particle_data(file_path)
    elif 'annotations' in file_path:
        valid = validate_annotation_data(file_path)
    else:
        print("ERROR: Unknown file type")
        sys.exit(1)

    if valid:
        print("✓ VALIDATION PASSED")
    else:
        print("✗ VALIDATION FAILED")
        sys.exit(1)


if __name__ == '__main__':
    main()
