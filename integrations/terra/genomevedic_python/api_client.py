"""
GenomeVedic API Client

Provides a Python interface to the GenomeVedic REST API for
natural language queries and variant explanations.

Author: GenomeVedic Team
License: MIT
"""

import requests
import json
import time
from typing import Dict, List, Optional, Any
from urllib.parse import urljoin


class GenomeVedicAPIError(Exception):
    """Base exception for GenomeVedic API errors"""
    pass


class GenomeVedicAPIClient:
    """
    Client for interacting with GenomeVedic API.

    This client provides methods for:
    - Natural language queries
    - Variant explanations
    - Session management
    - Health checks

    Example:
        >>> client = GenomeVedicAPIClient(base_url="http://localhost:8080")
        >>> result = client.query_natural_language("Find variants in BRCA1")
        >>> print(result['generated_sql'])
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        timeout: int = 30,
        max_retries: int = 3
    ):
        """
        Initialize GenomeVedic API client.

        Args:
            base_url: Base URL of GenomeVedic API server
            api_key: Optional API key for authentication (future use)
            timeout: Request timeout in seconds
            max_retries: Maximum number of retry attempts for failed requests
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = timeout
        self.max_retries = max_retries
        self.session = requests.Session()

        # Set default headers
        self.session.headers.update({
            'Content-Type': 'application/json',
            'User-Agent': 'GenomeVedic-Python-Client/1.0'
        })

        if api_key:
            self.session.headers['Authorization'] = f'Bearer {api_key}'

    def _make_request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict] = None,
        params: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """
        Make HTTP request with retry logic.

        Args:
            method: HTTP method (GET, POST, etc.)
            endpoint: API endpoint path
            data: Request body data
            params: URL query parameters

        Returns:
            Response JSON data

        Raises:
            GenomeVedicAPIError: If request fails after retries
        """
        url = urljoin(self.base_url, endpoint)

        for attempt in range(self.max_retries):
            try:
                response = self.session.request(
                    method=method,
                    url=url,
                    json=data,
                    params=params,
                    timeout=self.timeout
                )

                # Raise for HTTP errors
                response.raise_for_status()

                return response.json()

            except requests.exceptions.RequestException as e:
                if attempt == self.max_retries - 1:
                    raise GenomeVedicAPIError(f"Request failed after {self.max_retries} attempts: {e}")

                # Exponential backoff
                wait_time = 2 ** attempt
                time.sleep(wait_time)

        raise GenomeVedicAPIError("Unexpected error in request retry logic")

    def health_check(self) -> Dict[str, Any]:
        """
        Check API server health.

        Returns:
            Health status information

        Example:
            >>> client.health_check()
            {'success': True, 'status': 'healthy', 'time': 1699564800}
        """
        return self._make_request('GET', '/api/v1/health')

    def query_natural_language(
        self,
        query: str,
        user_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Convert natural language query to SQL.

        Args:
            query: Natural language query (e.g., "Find variants in BRCA1")
            user_id: Optional user identifier for rate limiting

        Returns:
            Query result containing generated SQL and explanation

        Example:
            >>> result = client.query_natural_language("Find variants in BRCA1")
            >>> print(result['generated_sql'])
            SELECT * FROM variants WHERE gene = 'BRCA1'
        """
        data = {
            'query': query
        }

        if user_id:
            data['user_id'] = user_id

        return self._make_request('POST', '/api/v1/query/natural-language', data=data)

    def get_query_examples(self) -> List[str]:
        """
        Get example queries supported by the system.

        Returns:
            List of example natural language queries

        Example:
            >>> examples = client.get_query_examples()
            >>> print(examples[0])
            "Find all mutations in the BRCA1 gene"
        """
        response = self._make_request('GET', '/api/v1/query/examples')
        return response.get('examples', [])

    def explain_variant(
        self,
        gene: str,
        variant: str,
        cancer_type: Optional[str] = None,
        clinical_context: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Get AI-powered explanation of a genomic variant.

        Args:
            gene: Gene name (e.g., "BRCA1")
            variant: Variant notation (e.g., "c.68_69delAG")
            cancer_type: Optional cancer type for context
            clinical_context: Optional clinical context

        Returns:
            Variant explanation including clinical significance

        Example:
            >>> result = client.explain_variant(
            ...     gene="BRCA1",
            ...     variant="c.68_69delAG",
            ...     cancer_type="breast cancer"
            ... )
            >>> print(result['explanation'])
        """
        data = {
            'gene': gene,
            'variant': variant
        }

        if cancer_type:
            data['cancer_type'] = cancer_type

        if clinical_context:
            data['clinical_context'] = clinical_context

        return self._make_request('POST', '/api/v1/variants/explain', data=data)

    def batch_explain_variants(
        self,
        variants: List[Dict[str, str]]
    ) -> List[Dict[str, Any]]:
        """
        Get explanations for multiple variants in batch.

        Args:
            variants: List of variant dictionaries with 'gene' and 'variant' keys

        Returns:
            List of variant explanations

        Example:
            >>> variants = [
            ...     {'gene': 'BRCA1', 'variant': 'c.68_69delAG'},
            ...     {'gene': 'TP53', 'variant': 'c.215C>G'}
            ... ]
            >>> results = client.batch_explain_variants(variants)
        """
        response = self._make_request('POST', '/api/v1/variants/batch-explain', data=variants)
        return response.get('responses', [])

    def get_cache_stats(self) -> Dict[str, Any]:
        """
        Get AI cache statistics (for performance monitoring).

        Returns:
            Cache statistics including hit rate and size

        Example:
            >>> stats = client.get_cache_stats()
            >>> print(f"Cache hit rate: {stats['hit_rate']:.2%}")
        """
        return self._make_request('GET', '/api/v1/cache/stats')

    def create_session(
        self,
        bam_file: str,
        reference: str = "hg38",
        metadata: Optional[Dict] = None
    ) -> str:
        """
        Create a GenomeVedic visualization session.

        Args:
            bam_file: Path or URL to BAM file
            reference: Reference genome (default: hg38)
            metadata: Optional session metadata

        Returns:
            Session URL for embedding or sharing

        Note:
            This endpoint may need to be implemented on the backend
        """
        data = {
            'bam_file': bam_file,
            'reference': reference,
            'metadata': metadata or {}
        }

        # This is a placeholder - actual endpoint needs backend implementation
        try:
            response = self._make_request('POST', '/api/v1/sessions/create', data=data)
            return response.get('session_url', '')
        except GenomeVedicAPIError:
            # Fallback to direct URL construction
            return f"{self.base_url}?bam={bam_file}&ref={reference}"

    def close(self):
        """Close the HTTP session."""
        self.session.close()

    def __enter__(self):
        """Context manager support."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager cleanup."""
        self.close()
