"""
GenomeVedic Jupyter Widget

Interactive visualization widget for Jupyter notebooks.
Embeds GenomeVedic viewer directly in notebook cells.

Author: GenomeVedic Team
License: MIT
"""

from typing import Optional, Dict, Any, List
import json

try:
    from IPython.display import IFrame, HTML, display
    import ipywidgets as widgets
    from ipywidgets import Layout
    JUPYTER_AVAILABLE = True
except ImportError:
    JUPYTER_AVAILABLE = False
    # Fallback classes for non-Jupyter environments
    class Layout:
        pass

from .api_client import GenomeVedicAPIClient
from .gcs_client import GCSClient, TerraGCSClient


class GenomeVedicWidget:
    """
    Interactive GenomeVedic viewer for Jupyter notebooks.

    Supports:
    - GCS bucket paths (gs://bucket/file.bam)
    - Local file paths
    - Direct URLs
    - Interactive controls (zoom, pan, filter)

    Example:
        >>> import genomevedic as gv
        >>> widget = gv.GenomeVedicWidget(
        ...     bam_file="gs://my-bucket/sample.bam",
        ...     reference="hg38"
        ... )
        >>> widget.show()
    """

    def __init__(
        self,
        bam_file: Optional[str] = None,
        reference: str = "hg38",
        api_url: str = "http://localhost:8080",
        width: str = "100%",
        height: str = "800px",
        initial_region: Optional[str] = None,
        auto_load: bool = False
    ):
        """
        Initialize GenomeVedic widget.

        Args:
            bam_file: Path to BAM file (local, GCS, or URL)
            reference: Reference genome (default: hg38)
            api_url: GenomeVedic API URL
            width: Widget width (CSS format)
            height: Widget height (CSS format)
            initial_region: Initial genomic region (e.g., "chr1:1000-2000")
            auto_load: Automatically load and display widget
        """
        if not JUPYTER_AVAILABLE:
            raise ImportError(
                "Jupyter dependencies not available. "
                "Install with: pip install ipywidgets IPython"
            )

        self.bam_file = bam_file
        self.reference = reference
        self.api_url = api_url.rstrip('/')
        self.width = width
        self.height = height
        self.initial_region = initial_region

        # Initialize clients
        self.api_client = GenomeVedicAPIClient(base_url=api_url)

        # Detect if running in Terra
        self.in_terra = self._detect_terra_environment()

        if self.in_terra:
            self.gcs_client = TerraGCSClient()
        else:
            self.gcs_client = GCSClient()

        # Widget state
        self._iframe = None
        self._controls = None
        self._output = widgets.Output()

        if auto_load and bam_file:
            self.load(bam_file)

    @staticmethod
    def _detect_terra_environment() -> bool:
        """Check if running in Terra.bio environment."""
        import os
        return (
            os.getenv('WORKSPACE_NAMESPACE') is not None or
            os.getenv('GOOGLE_PROJECT') is not None
        )

    def load(self, bam_file: str, reference: Optional[str] = None):
        """
        Load BAM file for visualization.

        Args:
            bam_file: Path to BAM file
            reference: Reference genome (optional)

        Example:
            >>> widget.load("gs://my-bucket/sample.bam", reference="hg38")
        """
        self.bam_file = bam_file
        if reference:
            self.reference = reference

        # Check GCS access if needed
        if bam_file.startswith('gs://'):
            if not self.gcs_client.check_access(bam_file):
                print(f"Warning: Cannot access {bam_file}")
                print("Check your GCS credentials and bucket permissions")

        # Update iframe if already created
        if self._iframe is not None:
            self._update_iframe()

    def _build_viewer_url(self) -> str:
        """
        Build GenomeVedic viewer URL with parameters.

        Returns:
            Complete viewer URL
        """
        params = []

        if self.bam_file:
            # Handle GCS URLs
            if self.bam_file.startswith('gs://'):
                # Generate signed URL for direct access
                try:
                    signed_url = self.gcs_client.generate_signed_url(self.bam_file)
                    params.append(f"bam={signed_url}")
                except Exception as e:
                    print(f"Warning: Could not generate signed URL: {e}")
                    params.append(f"bam={self.bam_file}")
            else:
                params.append(f"bam={self.bam_file}")

        if self.reference:
            params.append(f"ref={self.reference}")

        if self.initial_region:
            params.append(f"region={self.initial_region}")

        # Add embed mode
        params.append("embed=true")

        param_string = "&".join(params)
        return f"{self.api_url}?{param_string}"

    def _create_iframe(self):
        """Create IFrame widget for viewer."""
        url = self._build_viewer_url()

        self._iframe = IFrame(
            src=url,
            width=self.width,
            height=self.height
        )

    def _update_iframe(self):
        """Update existing IFrame with new URL."""
        if self._iframe is not None:
            url = self._build_viewer_url()
            self._iframe.src = url

    def _create_controls(self):
        """Create interactive control panel."""
        # Region input
        region_input = widgets.Text(
            value=self.initial_region or "",
            placeholder='chr1:1000-2000',
            description='Region:',
            layout=Layout(width='300px')
        )

        # Go button
        go_button = widgets.Button(
            description='Go',
            button_style='primary',
            layout=Layout(width='80px')
        )

        # Reference selector
        reference_select = widgets.Dropdown(
            options=['hg19', 'hg38', 'mm10', 'mm39'],
            value=self.reference,
            description='Reference:',
            layout=Layout(width='200px')
        )

        # Zoom controls
        zoom_in = widgets.Button(
            description='Zoom In',
            button_style='info',
            layout=Layout(width='100px')
        )

        zoom_out = widgets.Button(
            description='Zoom Out',
            button_style='info',
            layout=Layout(width='100px')
        )

        # File info button
        info_button = widgets.Button(
            description='File Info',
            button_style='',
            layout=Layout(width='100px')
        )

        # Event handlers
        def on_go_clicked(b):
            self.initial_region = region_input.value
            self._update_iframe()

        def on_reference_change(change):
            self.reference = change['new']
            self._update_iframe()

        def on_zoom_in(b):
            # This would require JavaScript communication
            # For now, just update the region
            pass

        def on_zoom_out(b):
            # This would require JavaScript communication
            pass

        def on_info_clicked(b):
            with self._output:
                self._output.clear_output()
                if self.bam_file and self.bam_file.startswith('gs://'):
                    try:
                        info = self.gcs_client.get_file_info(self.bam_file)
                        print("File Information:")
                        print(f"  Name: {info['name']}")
                        print(f"  Bucket: {info['bucket']}")
                        print(f"  Size: {info['size']:,} bytes ({info['size']/1e9:.2f} GB)")
                        print(f"  Created: {info['created']}")
                        print(f"  Updated: {info['updated']}")
                    except Exception as e:
                        print(f"Error getting file info: {e}")
                else:
                    print("File info only available for GCS files")

        # Attach handlers
        go_button.on_click(on_go_clicked)
        reference_select.observe(on_reference_change, names='value')
        zoom_in.on_click(on_zoom_in)
        zoom_out.on_click(on_zoom_out)
        info_button.on_click(on_info_clicked)

        # Layout controls
        controls_box = widgets.HBox([
            reference_select,
            region_input,
            go_button,
            zoom_in,
            zoom_out,
            info_button
        ])

        self._controls = widgets.VBox([controls_box, self._output])

    def show(self):
        """
        Display the GenomeVedic widget in notebook.

        Example:
            >>> widget.show()
        """
        if self._iframe is None:
            self._create_iframe()

        if self._controls is None:
            self._create_controls()

        # Display controls and viewer
        display(self._controls)
        display(self._iframe)

    def get_session_url(self) -> str:
        """
        Get shareable session URL.

        Returns:
            URL that can be shared with others

        Example:
            >>> url = widget.get_session_url()
            >>> print(f"Share this URL: {url}")
        """
        return self._build_viewer_url()

    def query(self, natural_language: str) -> Dict[str, Any]:
        """
        Execute natural language query on loaded data.

        Args:
            natural_language: Query in plain English

        Returns:
            Query results

        Example:
            >>> results = widget.query("Find variants in BRCA1")
            >>> print(results['generated_sql'])
        """
        return self.api_client.query_natural_language(natural_language)

    def explain_variant(
        self,
        gene: str,
        variant: str,
        cancer_type: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Get AI explanation for a variant.

        Args:
            gene: Gene name
            variant: Variant notation
            cancer_type: Optional cancer type context

        Returns:
            Variant explanation

        Example:
            >>> explanation = widget.explain_variant("BRCA1", "c.68_69delAG")
            >>> print(explanation['summary'])
        """
        return self.api_client.explain_variant(
            gene=gene,
            variant=variant,
            cancer_type=cancer_type
        )


def show(
    bam_file: str,
    reference: str = "hg38",
    api_url: str = "http://localhost:8080",
    width: str = "100%",
    height: str = "800px",
    initial_region: Optional[str] = None
):
    """
    Quick display function for GenomeVedic viewer.

    This is the primary function users will call for
    simple use cases.

    Args:
        bam_file: Path to BAM file (local, GCS, or URL)
        reference: Reference genome (default: hg38)
        api_url: GenomeVedic API URL
        width: Widget width
        height: Widget height
        initial_region: Initial genomic region to display

    Example:
        >>> import genomevedic as gv
        >>> gv.show(bam_file="gs://my-bucket/sample.bam")
    """
    widget = GenomeVedicWidget(
        bam_file=bam_file,
        reference=reference,
        api_url=api_url,
        width=width,
        height=height,
        initial_region=initial_region,
        auto_load=False
    )

    widget.show()
    return widget


def create_comparison_view(
    bam_files: List[str],
    labels: Optional[List[str]] = None,
    reference: str = "hg38",
    **kwargs
) -> widgets.Widget:
    """
    Create side-by-side comparison of multiple BAM files.

    Args:
        bam_files: List of BAM file paths
        labels: Optional labels for each file
        reference: Reference genome
        **kwargs: Additional arguments passed to GenomeVedicWidget

    Returns:
        Widget with comparison view

    Example:
        >>> gv.create_comparison_view([
        ...     "gs://bucket/tumor.bam",
        ...     "gs://bucket/normal.bam"
        ... ], labels=["Tumor", "Normal"])
    """
    if not JUPYTER_AVAILABLE:
        raise ImportError("Jupyter widgets not available")

    if labels is None:
        labels = [f"Sample {i+1}" for i in range(len(bam_files))]

    # Create widgets for each file
    tabs = []
    for bam_file, label in zip(bam_files, labels):
        widget = GenomeVedicWidget(
            bam_file=bam_file,
            reference=reference,
            **kwargs
        )
        widget.show()
        tabs.append(widget._iframe)

    # Create tabbed interface
    tab_widget = widgets.Tab(children=tabs)
    for i, label in enumerate(labels):
        tab_widget.set_title(i, label)

    display(tab_widget)
    return tab_widget
