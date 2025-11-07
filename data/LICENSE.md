# GenomeVedic Dataset Licenses and Attributions

This document provides complete attribution and licensing information for all genomic datasets used in GenomeVedic. All datasets are publicly available and used in accordance with their respective licenses.

---

## 1. Human Chromosome 22 (GRCh38)

**Source:** UCSC Genome Browser
**URL:** https://hgdownload.soe.ucsc.edu/goldenPath/hg38/chromosomes/
**File:** chr22.fa.gz
**Assembly:** GRCh38 (hg38)
**Size:** ~50 MB (uncompressed)

**License:** Public Domain
**Citation:**
> Kent WJ, Sugnet CW, Furey TS, Roskin KM, Pringle TH, Zahler AM, Haussler D.
> The human genome browser at UCSC. Genome Res. 2002 Jun;12(6):996-1006.
> doi: 10.1101/gr.229102

**Terms of Use:**
The UCSC Genome Browser data are freely available for academic, nonprofit, and personal use. Commercial use requires a license. GenomeVedic is an open-source academic/research tool.

**Attribution:**
```
Human Chromosome 22 sequence from UCSC Genome Browser (GRCh38/hg38)
Genome Reference Consortium Human Build 38 (GRCh38)
https://genome.ucsc.edu/
```

---

## 2. E. coli K-12 Genome

**Source:** NCBI RefSeq
**URL:** https://ftp.ncbi.nlm.nih.gov/genomes/all/GCF/000/005/845/GCF_000005845.2_ASM584v2/
**File:** GCF_000005845.2_ASM584v2_genomic.fna.gz
**Accession:** GCF_000005845.2
**Assembly:** ASM584v2
**Size:** ~4.6 MB (uncompressed)

**License:** Public Domain (U.S. Government Work)
**Citation:**
> Blattner FR, Plunkett G 3rd, Bloch CA, Perna NT, Burland V, Riley M, et al.
> The complete genome sequence of Escherichia coli K-12. Science. 1997 Sep 5;277(5331):1453-62.
> doi: 10.1126/science.277.5331.1453

**Terms of Use:**
NCBI data are produced by the U.S. Government and are in the public domain. No restrictions on use or distribution.

**Attribution:**
```
E. coli K-12 MG1655 genome from NCBI RefSeq
Accession: GCF_000005845.2 (ASM584v2)
https://www.ncbi.nlm.nih.gov/genome/?term=escherichia+coli+K12
```

---

## 3. COSMIC Cancer Gene Census

**Source:** Sanger Institute COSMIC Database
**URL:** https://cancer.sanger.ac.uk/cosmic
**File:** cosmic_top100_simulated.tsv (SIMULATED DATA)

**IMPORTANT NOTE:**
Real COSMIC data requires registration and acceptance of terms. For development and demonstration purposes, GenomeVedic includes a **simulated dataset** with known cancer genes from public sources (NCBI Gene, PubMed).

**Simulated Data Sources:**
- Gene names: HUGO Gene Nomenclature Committee (HGNC)
- Chromosomal positions: NCBI Gene database (public domain)
- Cancer gene classifications: Published literature (PubMed)

**License (Real COSMIC Data):** Academic License Required
**License (Simulated Data):** Public Domain Compilation

**For Production Use:**
To use real COSMIC data:
1. Register at: https://cancer.sanger.ac.uk/cosmic/download
2. Accept Terms and Conditions (free for academic use)
3. Download authenticated data
4. Replace simulated data with real COSMIC data

**Citation (Real COSMIC):**
> Tate JG, Bamford S, Jubb HC, Sondka Z, Beare DM, Bindal N, et al.
> COSMIC: the Catalogue Of Somatic Mutations In Cancer. Nucleic Acids Res. 2019 Jan 8;47(D1):D941-D947.
> doi: 10.1093/nar/gky1015

**Attribution:**
```
Simulated cancer gene dataset compiled from public sources
Real COSMIC data available at https://cancer.sanger.ac.uk/cosmic
(Requires registration for production use)
```

---

## 4. Ensembl GTF Annotations (Release 115)

**Source:** Ensembl
**URL:** https://ftp.ensembl.org/pub/release-115/gtf/homo_sapiens/
**File:** Homo_sapiens.GRCh38.115.gtf.gz
**Assembly:** GRCh38
**Release:** 115 (2024)
**Size:** ~50 MB (compressed)

**License:** Apache 2.0 License
**Citation:**
> Cunningham F, Allen JE, Allen J, Alvarez-Jarreta J, Amode MR, Armean IM, et al.
> Ensembl 2022. Nucleic Acids Res. 2022 Jan 7;50(D1):D988-D995.
> doi: 10.1093/nar/gkab1049

**Terms of Use:**
Ensembl data are freely available under the Apache 2.0 license. No restrictions on use for academic or commercial purposes with proper attribution.

**Attribution:**
```
Gene annotations from Ensembl Release 115 (GRCh38)
European Bioinformatics Institute (EMBL-EBI)
https://www.ensembl.org/
```

---

## 5. 1000 Genomes Project (Phase 3)

**Source:** 1000 Genomes Project
**URL:** https://ftp.1000genomes.ebi.ac.uk/vol1/ftp/release/20130502/
**File:** ALL.chr22.phase3_shapeit2_mvncall_integrated_v5b.20130502.genotypes.vcf.gz
**Chromosome:** 22
**Phase:** 3 (Final Release)
**Size:** ~100 MB (compressed)
**Samples:** 2,504 individuals from 26 populations

**License:** Public Domain (Fort Lauderdale Agreement)
**Citation:**
> 1000 Genomes Project Consortium, Auton A, Brooks LD, Durbin RM, Garrison EP, Kang HM, et al.
> A global reference for human genetic variation. Nature. 2015 Oct 1;526(7571):68-74.
> doi: 10.1038/nature15393

**Terms of Use:**
The 1000 Genomes Project data are freely available in the public domain. Users may freely download, analyze, and publish results based on these data. Proper citation is requested.

**Fort Lauderdale Principles:**
Data producers reserve the right to publish initial analyses. Subsequent uses should cite both the data resource and primary publications.

**Attribution:**
```
Variant data from 1000 Genomes Project Phase 3
International Genome Sample Resource (IGSR)
https://www.internationalgenome.org/
```

---

## Summary of Licenses

| Dataset | License | Commercial Use | Attribution Required |
|---------|---------|----------------|---------------------|
| UCSC chr22 | Public Domain | ✓ (with license) | ✓ |
| NCBI E. coli | Public Domain | ✓ | ✓ |
| COSMIC (simulated) | Public Domain | ✓ | ✓ |
| Ensembl GTF | Apache 2.0 | ✓ | ✓ |
| 1000 Genomes | Public Domain | ✓ | ✓ |

---

## GenomeVedic License

GenomeVedic is an open-source project. The software itself is licensed under [LICENSE TBD].

**Processed Data:**
The processed particle data, spatial hashes, and derived JSON files are considered derivative works and inherit the licenses of their source datasets. They are freely available for academic and research use.

**Commercial Use:**
If you intend to use GenomeVedic or its datasets for commercial purposes, please review each dataset's license terms carefully. Some datasets (e.g., UCSC) may require additional licensing for commercial applications.

---

## Data Citation Guidelines

When publishing results using GenomeVedic, please cite:

1. **GenomeVedic Software:**
   ```
   GenomeVedic: Real-Time 3D Visualization of Genomic Data
   [GitHub Repository URL]
   Version 1.0.0 (2025)
   ```

2. **Relevant Datasets:**
   - Include citations for all datasets used in your analysis
   - Follow each dataset's recommended citation format (listed above)

3. **Methods:**
   - Describe the Vedic digital root hashing algorithm
   - Cite the Williams Optimizer formula
   - Reference spatial clustering methods

---

## Contact and Questions

For questions about dataset licenses:
- **UCSC:** genome@soe.ucsc.edu
- **NCBI:** info@ncbi.nlm.nih.gov
- **COSMIC:** cosmic@sanger.ac.uk
- **Ensembl:** helpdesk@ensembl.org
- **1000 Genomes:** info@internationalgenome.org

For questions about GenomeVedic:
- **GitHub Issues:** [Repository URL]/issues
- **Email:** [Contact Email]

---

## Disclaimer

GenomeVedic provides these datasets for research and educational purposes. The data are provided "as-is" without warranty of any kind. Users are responsible for verifying the accuracy and suitability of data for their specific applications.

For clinical or diagnostic use, always consult primary data sources and follow appropriate regulatory guidelines.

---

**Last Updated:** 2025-11-07
**Version:** 1.0.0
**Maintainer:** GenomeVedic Team
