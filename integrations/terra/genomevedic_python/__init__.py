"""
GenomeVedic Python Library

A Python client for GenomeVedic genome visualization platform.
Designed for Jupyter notebooks, Terra.bio, and Google Colab.

Quick Start:
    >>> import genomevedic as gv
    >>> gv.show(bam_file="gs://my-bucket/sample.bam")

Advanced Usage:
    >>> widget = gv.GenomeVedicWidget(
    ...     bam_file="gs://bucket/tumor.bam",
    ...     reference="hg38",
    ...     initial_region="chr17:41196311-41277500"  # BRCA1
    ... )
    >>> widget.show()
    >>> results = widget.query("Find pathogenic variants")

Author: GenomeVedic Team
License: MIT
Version: 1.0.0
"""

__version__ = "1.0.0"
__author__ = "GenomeVedic Team"
__license__ = "MIT"

# Public API
from .api_client import GenomeVedicAPIClient, GenomeVedicAPIError
from .gcs_client import GCSClient, TerraGCSClient, GCSError
from .jupyter_widget import (
    GenomeVedicWidget,
    show,
    create_comparison_view
)

# Configuration management
_global_config = {
    'api_url': 'http://localhost:8080',
    'api_key': None,
    'default_reference': 'hg38'
}


def set_api_url(url: str):
    """
    Set default GenomeVedic API URL.

    Args:
        url: API server URL

    Example:
        >>> import genomevedic as gv
        >>> gv.set_api_url("https://genomevedic.example.com")
    """
    global _global_config
    _global_config['api_url'] = url.rstrip('/')


def set_api_key(api_key: str):
    """
    Set API key for authentication.

    Args:
        api_key: Your GenomeVedic API key

    Example:
        >>> import genomevedic as gv
        >>> gv.set_api_key("your-api-key-here")
    """
    global _global_config
    _global_config['api_key'] = api_key


def set_default_reference(reference: str):
    """
    Set default reference genome.

    Args:
        reference: Reference genome (e.g., 'hg38', 'hg19', 'mm10')

    Example:
        >>> import genomevedic as gv
        >>> gv.set_default_reference("hg19")
    """
    global _global_config
    _global_config['default_reference'] = reference


def get_config() -> dict:
    """
    Get current configuration.

    Returns:
        Configuration dictionary

    Example:
        >>> import genomevedic as gv
        >>> config = gv.get_config()
        >>> print(config['api_url'])
    """
    return _global_config.copy()


def load_bam(
    bam_file: str,
    reference: str = None,
    **kwargs
) -> GenomeVedicWidget:
    """
    Load and visualize BAM file (alternative to show()).

    Args:
        bam_file: Path to BAM file
        reference: Reference genome (uses default if not specified)
        **kwargs: Additional arguments for GenomeVedicWidget

    Returns:
        GenomeVedicWidget instance

    Example:
        >>> import genomevedic as gv
        >>> widget = gv.load_bam("gs://bucket/sample.bam")
        >>> widget.show()
    """
    if reference is None:
        reference = _global_config['default_reference']

    widget = GenomeVedicWidget(
        bam_file=bam_file,
        reference=reference,
        api_url=_global_config['api_url'],
        **kwargs
    )

    widget.show()
    return widget


def query(natural_language: str) -> dict:
    """
    Execute natural language query without visualization.

    Args:
        natural_language: Query in plain English

    Returns:
        Query results including generated SQL

    Example:
        >>> import genomevedic as gv
        >>> results = gv.query("Find variants in BRCA1 gene")
        >>> print(results['generated_sql'])
    """
    client = GenomeVedicAPIClient(
        base_url=_global_config['api_url'],
        api_key=_global_config['api_key']
    )
    return client.query_natural_language(natural_language)


def explain_variant(
    gene: str,
    variant: str,
    cancer_type: str = None
) -> dict:
    """
    Get AI-powered explanation of genomic variant.

    Args:
        gene: Gene name (e.g., "BRCA1")
        variant: Variant notation (e.g., "c.68_69delAG")
        cancer_type: Optional cancer type for context

    Returns:
        Variant explanation with clinical significance

    Example:
        >>> import genomevedic as gv
        >>> result = gv.explain_variant("BRCA1", "c.68_69delAG", "breast cancer")
        >>> print(result['explanation'])
    """
    client = GenomeVedicAPIClient(
        base_url=_global_config['api_url'],
        api_key=_global_config['api_key']
    )
    return client.explain_variant(gene, variant, cancer_type)


def check_gcs_access(gcs_path: str) -> bool:
    """
    Check if GCS file is accessible with current credentials.

    Args:
        gcs_path: GCS path (e.g., "gs://bucket/file.bam")

    Returns:
        True if accessible, False otherwise

    Example:
        >>> import genomevedic as gv
        >>> if gv.check_gcs_access("gs://my-bucket/sample.bam"):
        ...     print("File is accessible!")
    """
    client = GCSClient()
    return client.check_access(gcs_path)


def download_from_gcs(
    gcs_path: str,
    local_path: str
) -> str:
    """
    Download file from Google Cloud Storage.

    Args:
        gcs_path: GCS path (e.g., "gs://bucket/file.bam")
        local_path: Local destination path

    Returns:
        Path to downloaded file

    Example:
        >>> import genomevedic as gv
        >>> path = gv.download_from_gcs(
        ...     "gs://bucket/sample.bam",
        ...     "/tmp/sample.bam"
        ... )
    """
    client = GCSClient()
    return client.download_file(gcs_path, local_path)


# Convenience functions for Terra.bio
def terra_show(
    bam_file: str,
    **kwargs
):
    """
    Show BAM file in Terra.bio notebook (auto-detects Terra environment).

    Args:
        bam_file: BAM file path (can be workspace path or GCS)
        **kwargs: Additional arguments for GenomeVedicWidget

    Example:
        >>> import genomevedic as gv
        >>> # In Terra notebook:
        >>> gv.terra_show("gs://fc-secure-bucket/sample.bam")
    """
    # Auto-detect if in Terra
    import os
    if os.getenv('WORKSPACE_NAMESPACE'):
        print(f"Terra workspace detected: {os.getenv('WORKSPACE_NAMESPACE')}")

    return show(bam_file=bam_file, **kwargs)


# Package metadata
__all__ = [
    # Main classes
    'GenomeVedicWidget',
    'GenomeVedicAPIClient',
    'GCSClient',
    'TerraGCSClient',

    # Quick functions
    'show',
    'load_bam',
    'query',
    'explain_variant',

    # Configuration
    'set_api_url',
    'set_api_key',
    'set_default_reference',
    'get_config',

    # GCS utilities
    'check_gcs_access',
    'download_from_gcs',

    # Terra utilities
    'terra_show',

    # Advanced
    'create_comparison_view',

    # Exceptions
    'GenomeVedicAPIError',
    'GCSError',

    # Metadata
    '__version__',
]


# Module-level docstring for help()
def help():
    """
    Display GenomeVedic help information.

    Example:
        >>> import genomevedic as gv
        >>> gv.help()
    """
    help_text = """
    GenomeVedic Python Library
    ==========================

    Quick Start:
        import genomevedic as gv
        gv.show(bam_file="gs://my-bucket/sample.bam")

    Main Functions:
        show()              - Display BAM file in widget
        load_bam()          - Load BAM and return widget instance
        query()             - Natural language query
        explain_variant()   - AI variant explanation

    Configuration:
        set_api_url()       - Set API server URL
        set_api_key()       - Set authentication key
        set_default_reference() - Set default genome

    Terra.bio:
        terra_show()        - Auto-configured for Terra

    For full documentation:
        https://genomevedic.readthedocs.io
    """
    print(help_text)
