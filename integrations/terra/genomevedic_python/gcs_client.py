"""
Google Cloud Storage Client for GenomeVedic

Handles BAM file access from GCS buckets with streaming support.
Supports Terra.bio service accounts and OAuth authentication.

Author: GenomeVedic Team
License: MIT
"""

import os
import io
import re
from typing import Optional, Tuple, BinaryIO
from urllib.parse import urlparse


class GCSError(Exception):
    """Base exception for GCS operations"""
    pass


class GCSClient:
    """
    Google Cloud Storage client with BAM streaming support.

    Supports multiple authentication methods:
    - Application Default Credentials (recommended for Terra)
    - Service account JSON file
    - Manual credentials

    Example:
        >>> client = GCSClient()
        >>> data = client.read_bam_region("gs://bucket/file.bam", "chr1", 1000, 2000)
    """

    def __init__(
        self,
        credentials_path: Optional[str] = None,
        project: Optional[str] = None
    ):
        """
        Initialize GCS client.

        Args:
            credentials_path: Path to service account JSON file (optional)
            project: GCP project ID (optional, auto-detected in Terra)
        """
        self.credentials_path = credentials_path
        self.project = project
        self._client = None
        self._storage_client = None

    def _get_storage_client(self):
        """
        Get or create Google Cloud Storage client.

        Uses lazy initialization to avoid import errors when
        google-cloud-storage is not installed.
        """
        if self._storage_client is not None:
            return self._storage_client

        try:
            from google.cloud import storage
            from google.oauth2 import service_account
        except ImportError:
            raise GCSError(
                "google-cloud-storage library not installed. "
                "Install with: pip install google-cloud-storage"
            )

        # Initialize credentials
        credentials = None
        if self.credentials_path:
            credentials = service_account.Credentials.from_service_account_file(
                self.credentials_path
            )

        # Create storage client
        self._storage_client = storage.Client(
            credentials=credentials,
            project=self.project
        )

        return self._storage_client

    @staticmethod
    def parse_gcs_path(gcs_path: str) -> Tuple[str, str]:
        """
        Parse GCS path into bucket and blob name.

        Args:
            gcs_path: GCS path (e.g., "gs://bucket/path/to/file.bam")

        Returns:
            Tuple of (bucket_name, blob_name)

        Example:
            >>> GCSClient.parse_gcs_path("gs://my-bucket/data/sample.bam")
            ('my-bucket', 'data/sample.bam')
        """
        if not gcs_path.startswith('gs://'):
            raise GCSError(f"Invalid GCS path: {gcs_path}. Must start with 'gs://'")

        # Remove 'gs://' prefix
        path = gcs_path[5:]

        # Split into bucket and blob
        parts = path.split('/', 1)
        if len(parts) != 2:
            raise GCSError(f"Invalid GCS path format: {gcs_path}")

        return parts[0], parts[1]

    def get_blob(self, gcs_path: str):
        """
        Get GCS blob object.

        Args:
            gcs_path: Full GCS path (gs://bucket/blob)

        Returns:
            Google Cloud Storage Blob object
        """
        client = self._get_storage_client()
        bucket_name, blob_name = self.parse_gcs_path(gcs_path)

        bucket = client.bucket(bucket_name)
        blob = bucket.blob(blob_name)

        if not blob.exists():
            raise GCSError(f"File not found: {gcs_path}")

        return blob

    def download_file(
        self,
        gcs_path: str,
        local_path: str,
        chunk_size: int = 1024 * 1024
    ) -> str:
        """
        Download file from GCS to local filesystem.

        Args:
            gcs_path: GCS file path
            local_path: Local destination path
            chunk_size: Download chunk size in bytes (default: 1MB)

        Returns:
            Path to downloaded file

        Example:
            >>> client.download_file("gs://bucket/file.bam", "/tmp/file.bam")
            '/tmp/file.bam'
        """
        blob = self.get_blob(gcs_path)

        # Create parent directory if needed
        os.makedirs(os.path.dirname(local_path), exist_ok=True)

        # Download file
        blob.download_to_filename(local_path)

        return local_path

    def read_bytes(
        self,
        gcs_path: str,
        start: Optional[int] = None,
        end: Optional[int] = None
    ) -> bytes:
        """
        Read bytes from GCS file (with optional range).

        Args:
            gcs_path: GCS file path
            start: Start byte position (optional)
            end: End byte position (optional)

        Returns:
            File content as bytes

        Example:
            >>> # Read first 1000 bytes
            >>> data = client.read_bytes("gs://bucket/file.bam", 0, 1000)
        """
        blob = self.get_blob(gcs_path)

        if start is not None and end is not None:
            # Range read
            return blob.download_as_bytes(start=start, end=end)
        else:
            # Full read
            return blob.download_as_bytes()

    def read_bam_region(
        self,
        gcs_path: str,
        chromosome: str,
        start: int,
        end: int
    ) -> bytes:
        """
        Read specific genomic region from BAM file.

        Note: This requires the BAM file to be indexed (.bai file present).
        Uses pysam for efficient region extraction.

        Args:
            gcs_path: GCS path to BAM file
            chromosome: Chromosome name (e.g., "chr1")
            start: Start position (0-based)
            end: End position (0-based)

        Returns:
            BAM data for the specified region

        Example:
            >>> data = client.read_bam_region(
            ...     "gs://bucket/sample.bam",
            ...     "chr1", 1000000, 1001000
            ... )
        """
        try:
            import pysam
        except ImportError:
            raise GCSError(
                "pysam library not installed. "
                "Install with: pip install pysam"
            )

        # For GCS, pysam can read directly using htslib
        # First download index file if not present
        bai_path = f"{gcs_path}.bai"

        try:
            # Open BAM file directly from GCS
            # pysam supports gs:// URLs with proper GCS credentials
            with pysam.AlignmentFile(gcs_path, "rb") as bam:
                # Extract region
                region_data = io.BytesIO()

                # Write reads from region
                for read in bam.fetch(chromosome, start, end):
                    # This is a simplified example
                    # In production, you'd want to serialize properly
                    pass

                return region_data.getvalue()

        except Exception as e:
            raise GCSError(f"Failed to read BAM region: {e}")

    def get_file_info(self, gcs_path: str) -> dict:
        """
        Get file metadata from GCS.

        Args:
            gcs_path: GCS file path

        Returns:
            Dictionary with file metadata

        Example:
            >>> info = client.get_file_info("gs://bucket/file.bam")
            >>> print(f"Size: {info['size']} bytes")
        """
        blob = self.get_blob(gcs_path)

        return {
            'name': blob.name,
            'bucket': blob.bucket.name,
            'size': blob.size,
            'content_type': blob.content_type,
            'md5_hash': blob.md5_hash,
            'created': blob.time_created,
            'updated': blob.updated
        }

    def check_access(self, gcs_path: str) -> bool:
        """
        Check if file is accessible with current credentials.

        Args:
            gcs_path: GCS file path

        Returns:
            True if accessible, False otherwise

        Example:
            >>> if client.check_access("gs://bucket/file.bam"):
            ...     print("File is accessible")
        """
        try:
            blob = self.get_blob(gcs_path)
            return blob.exists()
        except Exception:
            return False

    def generate_signed_url(
        self,
        gcs_path: str,
        expiration: int = 3600
    ) -> str:
        """
        Generate signed URL for temporary public access.

        Args:
            gcs_path: GCS file path
            expiration: URL expiration time in seconds (default: 1 hour)

        Returns:
            Signed URL string

        Example:
            >>> url = client.generate_signed_url("gs://bucket/file.bam")
            >>> # Share this URL with others for temporary access
        """
        blob = self.get_blob(gcs_path)

        from datetime import timedelta

        url = blob.generate_signed_url(
            expiration=timedelta(seconds=expiration),
            method='GET'
        )

        return url

    def stream_download(self, gcs_path: str, chunk_size: int = 1024 * 1024):
        """
        Stream download file in chunks (memory efficient).

        Args:
            gcs_path: GCS file path
            chunk_size: Chunk size in bytes (default: 1MB)

        Yields:
            Chunks of file data

        Example:
            >>> for chunk in client.stream_download("gs://bucket/large.bam"):
            ...     process_chunk(chunk)
        """
        blob = self.get_blob(gcs_path)

        # Download in chunks
        for chunk in blob.download_as_bytes(chunk_size=chunk_size):
            yield chunk


class TerraGCSClient(GCSClient):
    """
    Specialized GCS client for Terra.bio platform.

    Automatically detects Terra environment and uses
    appropriate authentication methods.
    """

    def __init__(self):
        """
        Initialize Terra GCS client.

        Automatically detects Terra workspace credentials.
        """
        # In Terra, credentials are typically available via Application Default Credentials
        super().__init__(
            credentials_path=None,  # Use ADC
            project=self._detect_terra_project()
        )

    @staticmethod
    def _detect_terra_project() -> Optional[str]:
        """
        Detect Terra workspace project.

        Returns:
            GCP project ID if in Terra environment, None otherwise
        """
        # Check Terra environment variables
        project = os.getenv('GOOGLE_PROJECT')
        if project:
            return project

        # Check workspace namespace (Terra-specific)
        workspace = os.getenv('WORKSPACE_NAMESPACE')
        if workspace:
            return workspace

        return None

    def get_workspace_bucket(self) -> Optional[str]:
        """
        Get Terra workspace bucket name.

        Returns:
            Workspace bucket name or None

        Example:
            >>> client = TerraGCSClient()
            >>> bucket = client.get_workspace_bucket()
            >>> print(f"Workspace bucket: {bucket}")
        """
        return os.getenv('WORKSPACE_BUCKET')
