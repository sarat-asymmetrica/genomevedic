#!/usr/bin/env python3
"""
GenomeVedic Galaxy Tool Wrapper

This script acts as a bridge between Galaxy and GenomeVedic's backend API,
converting BAM files into 3D VR visualizations.

Author: GenomeVedic Development Team
License: MIT
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path
from typing import Dict, Any, Optional
import urllib.request
import urllib.error
import urllib.parse


class GenomeVedicAPIClient:
    """Client for GenomeVedic API interactions"""

    def __init__(self, api_endpoint: str, api_key: Optional[str] = None):
        self.api_endpoint = api_endpoint.rstrip('/')
        self.api_key = api_key
        self.session_id = None

    def _make_request(self, endpoint: str, method: str = 'POST',
                      data: Optional[Dict] = None, files: Optional[Dict] = None) -> Dict:
        """Make HTTP request to GenomeVedic API"""
        url = f"{self.api_endpoint}/{endpoint.lstrip('/')}"

        headers = {
            'Content-Type': 'application/json',
            'User-Agent': 'GenomeVedic-Galaxy/1.0'
        }

        if self.api_key:
            headers['Authorization'] = f'Bearer {self.api_key}'

        request_data = None
        if data:
            request_data = json.dumps(data).encode('utf-8')

        req = urllib.request.Request(url, data=request_data, headers=headers, method=method)

        try:
            with urllib.request.urlopen(req, timeout=300) as response:
                response_data = response.read().decode('utf-8')
                return json.loads(response_data)
        except urllib.error.HTTPError as e:
            error_msg = e.read().decode('utf-8') if e.fp else str(e)
            raise RuntimeError(f"API request failed: {e.code} {e.reason}\n{error_msg}")
        except urllib.error.URLError as e:
            raise RuntimeError(f"Network error: {e.reason}")

    def create_session(self, session_name: str, config: Dict[str, Any]) -> str:
        """Create a new visualization session"""
        data = {
            'session_name': session_name,
            'source': 'galaxy',
            'config': config
        }

        response = self._make_request('/api/v1/sessions/create', data=data)

        if not response.get('success'):
            raise RuntimeError(f"Failed to create session: {response.get('error', 'Unknown error')}")

        self.session_id = response['session_id']
        return self.session_id

    def upload_bam(self, bam_path: str, genome_build: str,
                   quality_threshold: int, region: Optional[str] = None) -> Dict:
        """Upload and process BAM file"""

        if not self.session_id:
            raise RuntimeError("No active session. Call create_session first.")

        # For Galaxy integration, we send BAM metadata and let the backend fetch it
        # In production, this would use Galaxy's data library API
        data = {
            'session_id': self.session_id,
            'bam_path': bam_path,
            'genome_build': genome_build,
            'quality_threshold': quality_threshold,
            'region': region
        }

        response = self._make_request('/api/v1/import/galaxy', data=data)

        if not response.get('success'):
            raise RuntimeError(f"BAM upload failed: {response.get('error', 'Unknown error')}")

        return response

    def get_session_url(self) -> str:
        """Get the visualization URL for the session"""
        if not self.session_id:
            raise RuntimeError("No active session")

        # Return direct link to visualization
        return f"{self.api_endpoint.replace('/api/v1', '')}/view/{self.session_id}"

    def get_session_stats(self) -> Dict:
        """Get processing statistics for the session"""
        if not self.session_id:
            raise RuntimeError("No active session")

        response = self._make_request(f'/api/v1/sessions/{self.session_id}/stats', method='GET')
        return response


def parse_arguments():
    """Parse command-line arguments"""
    parser = argparse.ArgumentParser(description='GenomeVedic Galaxy Tool Wrapper')

    # Required arguments
    parser.add_argument('--bam-input', required=True, help='Input BAM file path')
    parser.add_argument('--genome-build', required=True, help='Genome build (e.g., hg38)')
    parser.add_argument('--visualization-mode', required=True,
                        choices=['particles', 'density', 'coverage', 'mutations'])
    parser.add_argument('--quality-threshold', type=int, default=20)
    parser.add_argument('--session-name', required=True, help='Name for visualization session')
    parser.add_argument('--api-endpoint', required=True, help='GenomeVedic API endpoint URL')

    # Optional arguments
    parser.add_argument('--api-key', help='API key for authentication')
    parser.add_argument('--region', help='Genomic region (chr:start-end)')
    parser.add_argument('--enable-lod', action='store_true', help='Enable LOD rendering')
    parser.add_argument('--lod-levels', type=int, default=5)
    parser.add_argument('--enable-multiplayer', action='store_true')
    parser.add_argument('--particle-limit', type=int, default=1000000)
    parser.add_argument('--color-scheme', default='vedic',
                        choices=['vedic', 'quality', 'gc_content', 'mutation_freq'])

    # Output files
    parser.add_argument('--output-url', required=True, help='Output file for session URL')
    parser.add_argument('--output-stats', required=True, help='Output file for statistics JSON')
    parser.add_argument('--output-log', required=True, help='Output file for processing log')

    return parser.parse_args()


def setup_logging(log_file: str):
    """Setup logging to file"""
    class Logger:
        def __init__(self, filename):
            self.terminal = sys.stdout
            self.log = open(filename, 'w')

        def write(self, message):
            self.terminal.write(message)
            self.log.write(message)
            self.log.flush()

        def flush(self):
            self.terminal.flush()
            self.log.flush()

    sys.stdout = Logger(log_file)
    sys.stderr = sys.stdout


def validate_bam_file(bam_path: str) -> bool:
    """Validate BAM file exists and is readable"""
    if not os.path.exists(bam_path):
        raise FileNotFoundError(f"BAM file not found: {bam_path}")

    if not os.path.isfile(bam_path):
        raise ValueError(f"BAM path is not a file: {bam_path}")

    file_size = os.path.getsize(bam_path)
    if file_size == 0:
        raise ValueError("BAM file is empty")

    print(f"✓ BAM file validated: {bam_path} ({file_size:,} bytes)")
    return True


def main():
    """Main execution function"""
    args = parse_arguments()

    # Setup logging
    setup_logging(args.output_log)

    print("=" * 80)
    print("GenomeVedic Galaxy Integration - BAM to VR Visualization")
    print("=" * 80)
    print()

    start_time = time.time()

    try:
        # Step 1: Validate input
        print("[1/5] Validating BAM file...")
        validate_bam_file(args.bam_input)
        print()

        # Step 2: Initialize API client
        print("[2/5] Connecting to GenomeVedic API...")
        print(f"  Endpoint: {args.api_endpoint}")
        client = GenomeVedicAPIClient(args.api_endpoint, args.api_key)
        print("✓ API client initialized")
        print()

        # Step 3: Create session
        print("[3/5] Creating visualization session...")
        session_config = {
            'visualization_mode': args.visualization_mode,
            'enable_lod': args.enable_lod,
            'lod_levels': args.lod_levels,
            'enable_multiplayer': args.enable_multiplayer,
            'particle_limit': args.particle_limit,
            'color_scheme': args.color_scheme
        }

        session_id = client.create_session(args.session_name, session_config)
        print(f"✓ Session created: {session_id}")
        print()

        # Step 4: Upload and process BAM
        print("[4/5] Uploading and processing BAM file...")
        print(f"  Quality threshold: {args.quality_threshold}")
        if args.region:
            print(f"  Region: {args.region}")

        upload_result = client.upload_bam(
            args.bam_input,
            args.genome_build,
            args.quality_threshold,
            args.region
        )

        print(f"✓ BAM processed successfully")
        print(f"  Reads processed: {upload_result.get('reads_processed', 'N/A'):,}")
        print(f"  Particles created: {upload_result.get('particles_created', 'N/A'):,}")
        print(f"  Processing time: {upload_result.get('processing_time_ms', 0)/1000:.2f}s")
        print()

        # Step 5: Get session URL and stats
        print("[5/5] Generating outputs...")
        session_url = client.get_session_url()
        session_stats = client.get_session_stats()

        # Write output files
        with open(args.output_url, 'w') as f:
            f.write(f"{session_url}\n")
            f.write(f"\nSession ID: {session_id}\n")
            f.write(f"Visualization Mode: {args.visualization_mode}\n")
            f.write(f"\nOpen this URL in a web browser or VR headset to view your genome!\n")

        with open(args.output_stats, 'w') as f:
            stats = {
                'session_id': session_id,
                'session_url': session_url,
                'session_name': args.session_name,
                'bam_file': os.path.basename(args.bam_input),
                'genome_build': args.genome_build,
                'visualization_mode': args.visualization_mode,
                'region': args.region,
                'quality_threshold': args.quality_threshold,
                'processing_stats': upload_result,
                'session_stats': session_stats,
                'total_time_seconds': time.time() - start_time,
                'timestamp': time.strftime('%Y-%m-%d %H:%M:%S')
            }
            json.dump(stats, f, indent=2)

        print(f"✓ Session URL written to: {args.output_url}")
        print(f"✓ Statistics written to: {args.output_stats}")
        print()

        # Summary
        total_time = time.time() - start_time
        print("=" * 80)
        print("SUCCESS! Your genome is ready to explore in VR!")
        print("=" * 80)
        print(f"Session URL: {session_url}")
        print(f"Total processing time: {total_time:.2f} seconds")
        print()
        print("Next steps:")
        print("  1. Open the session URL in a web browser")
        print("  2. Click 'Enter VR' if you have a VR headset")
        print("  3. Use WASD/arrows to navigate, mouse to look around")
        print("  4. Click on particles to see read details")
        print()

        return 0

    except Exception as e:
        print()
        print("=" * 80)
        print("ERROR: Visualization failed")
        print("=" * 80)
        print(f"Error: {str(e)}")
        print()
        print("Troubleshooting:")
        print("  1. Check that GenomeVedic API endpoint is accessible")
        print("  2. Verify BAM file is properly formatted and indexed")
        print("  3. Ensure API key is valid (if using authentication)")
        print("  4. Check network connectivity")
        print()

        # Write error outputs
        with open(args.output_url, 'w') as f:
            f.write(f"ERROR: {str(e)}\n")

        with open(args.output_stats, 'w') as f:
            json.dump({'success': False, 'error': str(e)}, f, indent=2)

        return 1


if __name__ == '__main__':
    sys.exit(main())
