"""
GenomeVedic Python Package Setup

PyPI-ready setup configuration for the GenomeVedic Python library.
"""

from setuptools import setup, find_packages
from pathlib import Path

# Read README for long description
this_directory = Path(__file__).parent
try:
    long_description = (this_directory / "README.md").read_text()
except FileNotFoundError:
    long_description = """
    GenomeVedic Python Library

    Interactive genome visualization for Jupyter notebooks and Terra.bio.
    """

# Read version from package
version = {}
with open("genomevedic_python/__init__.py") as f:
    for line in f:
        if line.startswith("__version__"):
            exec(line, version)
            break

setup(
    name="genomevedic",
    version=version.get("__version__", "1.0.0"),
    author="GenomeVedic Team",
    author_email="contact@genomevedic.io",
    description="Interactive genome visualization for Jupyter notebooks and Terra.bio",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/genomevedic/genomevedic-python",
    project_urls={
        "Bug Tracker": "https://github.com/genomevedic/genomevedic-python/issues",
        "Documentation": "https://genomevedic.readthedocs.io",
        "Source Code": "https://github.com/genomevedic/genomevedic-python",
    },
    packages=find_packages(),
    classifiers=[
        # Development status
        "Development Status :: 4 - Beta",

        # Audience
        "Intended Audience :: Science/Research",
        "Intended Audience :: Healthcare Industry",
        "Intended Audience :: Developers",

        # Topics
        "Topic :: Scientific/Engineering :: Bio-Informatics",
        "Topic :: Scientific/Engineering :: Visualization",

        # License
        "License :: OSI Approved :: MIT License",

        # Python versions
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",

        # Operating systems
        "Operating System :: OS Independent",

        # Framework
        "Framework :: Jupyter",
        "Framework :: IPython",
    ],
    keywords=[
        "genomics",
        "bioinformatics",
        "visualization",
        "jupyter",
        "terra",
        "bam",
        "vcf",
        "genome",
        "variant-calling",
        "ngs",
        "sequencing"
    ],
    python_requires=">=3.8",
    install_requires=[
        "requests>=2.28.0",
        "ipywidgets>=8.0.0",
        "IPython>=7.0.0",
    ],
    extras_require={
        # Full installation with all features
        "full": [
            "google-cloud-storage>=2.10.0",
            "pysam>=0.21.0",
        ],

        # Terra.bio specific dependencies
        "terra": [
            "google-cloud-storage>=2.10.0",
            "pysam>=0.21.0",
        ],

        # Development dependencies
        "dev": [
            "pytest>=7.0.0",
            "pytest-cov>=4.0.0",
            "black>=23.0.0",
            "flake8>=6.0.0",
            "mypy>=1.0.0",
            "sphinx>=5.0.0",
            "sphinx-rtd-theme>=1.2.0",
        ],

        # Testing dependencies
        "test": [
            "pytest>=7.0.0",
            "pytest-cov>=4.0.0",
            "pytest-mock>=3.10.0",
        ],
    },
    entry_points={
        "console_scripts": [
            "genomevedic=genomevedic_python.cli:main",
        ],
    },
    include_package_data=True,
    zip_safe=False,
    platforms="any",
)
