/**
 * GenomeVedic Database Schema Documentation
 *
 * This file contains the database schema documentation used to teach
 * GPT-4 how to convert natural language queries to SQL.
 *
 * Security: This schema is used for query generation but actual database
 * access is restricted through validation and parameterized queries.
 */

package ai

// SchemaDocumentation contains the complete database schema for GPT-4
const SchemaDocumentation = `
# GenomeVedic Database Schema

## Table: variants

The main table containing genomic variant information.

### Columns:
- id (TEXT): Unique identifier for the variant (UUID format)
- gene (TEXT): Gene symbol (e.g., TP53, BRCA1, KRAS, EGFR)
- chromosome (TEXT): Chromosome identifier (1-22, X, Y, MT)
- position (INTEGER): Genomic position in base pairs (bp)
- ref_allele (TEXT): Reference allele from reference genome
- alt_allele (TEXT): Alternate (variant) allele
- hgvs (TEXT): HGVS notation (e.g., c.524G>A, p.Arg175His)
- af (REAL): Allele frequency (0.0-1.0), also known as MAF
- pathogenicity (TEXT): ClinVar classification (Pathogenic, Likely Pathogenic, Benign, Likely Benign, Uncertain)
- mutation_type (TEXT): Type of mutation (Missense, Nonsense, Frameshift, Splice, Inframe, Synonymous)
- sample_id (TEXT): Associated sample identifier
- sample_count (INTEGER): Number of samples with this variant
- cosmic_id (TEXT): COSMIC database identifier (if applicable)
- rsid (TEXT): dbSNP reference SNP identifier (if applicable)

### Example Queries:

1. Find all TP53 mutations:
   SELECT * FROM variants WHERE gene = 'TP53'

2. Find variants with high allele frequency:
   SELECT * FROM variants WHERE af > 0.01

3. Find pathogenic variants in BRCA1:
   SELECT * FROM variants WHERE gene = 'BRCA1' AND pathogenicity = 'Pathogenic'

4. Find all variants on chromosome 17:
   SELECT * FROM variants WHERE chromosome = '17'

5. Find missense mutations in KRAS:
   SELECT * FROM variants WHERE gene = 'KRAS' AND mutation_type = 'Missense'

6. Find hotspot mutations (high sample count):
   SELECT * FROM variants WHERE sample_count > 100 ORDER BY sample_count DESC

7. Find rare variants:
   SELECT * FROM variants WHERE af < 0.001

8. Find variants in a specific region:
   SELECT * FROM variants WHERE chromosome = '17' AND position BETWEEN 7571720 AND 7590868

## Important Notes:

- Gene names are uppercase (TP53, not tp53)
- Chromosome values are strings: '1'-'22', 'X', 'Y', 'MT'
- Allele frequency (af) is a decimal between 0.0 and 1.0
- Pathogenicity values: 'Pathogenic', 'Likely Pathogenic', 'Benign', 'Likely Benign', 'Uncertain'
- Mutation types: 'Missense', 'Nonsense', 'Frameshift', 'Splice', 'Inframe', 'Synonymous'
- Position is 1-indexed (first base pair is position 1)

## Common Gene Symbols:
- TP53: Tumor protein p53 (chromosome 17)
- BRCA1: Breast cancer 1 (chromosome 17)
- BRCA2: Breast cancer 2 (chromosome 13)
- KRAS: Kirsten rat sarcoma viral oncogene (chromosome 12)
- EGFR: Epidermal growth factor receptor (chromosome 7)
- PTEN: Phosphatase and tensin homolog (chromosome 10)
- PIK3CA: Phosphatidylinositol-4,5-bisphosphate 3-kinase (chromosome 3)
- APC: Adenomatous polyposis coli (chromosome 5)
- BRAF: B-Raf proto-oncogene (chromosome 7)
`

// ExampleMappings provides example natural language to SQL mappings
var ExampleMappings = []QueryExample{
	{
		NaturalLanguage: "Show me all TP53 mutations",
		SQL:             "SELECT * FROM variants WHERE gene = 'TP53'",
		Description:     "Find all mutations in the TP53 gene",
	},
	{
		NaturalLanguage: "What are variants with MAF > 0.01?",
		SQL:             "SELECT * FROM variants WHERE af > 0.01",
		Description:     "Find variants with minor allele frequency greater than 1%",
	},
	{
		NaturalLanguage: "Find pathogenic variants in BRCA1",
		SQL:             "SELECT * FROM variants WHERE gene = 'BRCA1' AND pathogenicity = 'Pathogenic'",
		Description:     "Find pathogenic mutations in BRCA1 gene",
	},
	{
		NaturalLanguage: "List all variants on chromosome 17",
		SQL:             "SELECT * FROM variants WHERE chromosome = '17'",
		Description:     "Find all variants on chromosome 17",
	},
	{
		NaturalLanguage: "Show missense mutations in KRAS",
		SQL:             "SELECT * FROM variants WHERE gene = 'KRAS' AND mutation_type = 'Missense'",
		Description:     "Find missense mutations in KRAS gene",
	},
	{
		NaturalLanguage: "Find hotspot mutations",
		SQL:             "SELECT * FROM variants WHERE sample_count > 100 ORDER BY sample_count DESC",
		Description:     "Find mutations with high sample counts (hotspots)",
	},
	{
		NaturalLanguage: "Show me rare variants",
		SQL:             "SELECT * FROM variants WHERE af < 0.001",
		Description:     "Find rare variants with frequency < 0.1%",
	},
	{
		NaturalLanguage: "Find variants in TP53 region on chromosome 17",
		SQL:             "SELECT * FROM variants WHERE chromosome = '17' AND position BETWEEN 7571720 AND 7590868",
		Description:     "Find variants in TP53 genomic region",
	},
	{
		NaturalLanguage: "What are the most common mutations?",
		SQL:             "SELECT gene, COUNT(*) as count FROM variants GROUP BY gene ORDER BY count DESC LIMIT 10",
		Description:     "Count mutations by gene and show top 10",
	},
	{
		NaturalLanguage: "Show pathogenic mutations ordered by frequency",
		SQL:             "SELECT * FROM variants WHERE pathogenicity = 'Pathogenic' ORDER BY af DESC",
		Description:     "Find pathogenic mutations sorted by allele frequency",
	},
	{
		NaturalLanguage: "Find nonsense mutations",
		SQL:             "SELECT * FROM variants WHERE mutation_type = 'Nonsense'",
		Description:     "Find all nonsense (stop-gain) mutations",
	},
	{
		NaturalLanguage: "Show variants with COSMIC ID",
		SQL:             "SELECT * FROM variants WHERE cosmic_id IS NOT NULL AND cosmic_id != ''",
		Description:     "Find variants present in COSMIC database",
	},
	{
		NaturalLanguage: "Find frameshift mutations in tumor suppressor genes",
		SQL:             "SELECT * FROM variants WHERE mutation_type = 'Frameshift' AND gene IN ('TP53', 'BRCA1', 'BRCA2', 'PTEN', 'APC')",
		Description:     "Find frameshift mutations in known tumor suppressors",
	},
	{
		NaturalLanguage: "What variants are in EGFR?",
		SQL:             "SELECT * FROM variants WHERE gene = 'EGFR'",
		Description:     "Find all EGFR mutations",
	},
	{
		NaturalLanguage: "Show splice site mutations",
		SQL:             "SELECT * FROM variants WHERE mutation_type = 'Splice'",
		Description:     "Find splice site mutations",
	},
	{
		NaturalLanguage: "Find mutations on sex chromosomes",
		SQL:             "SELECT * FROM variants WHERE chromosome IN ('X', 'Y')",
		Description:     "Find variants on X and Y chromosomes",
	},
	{
		NaturalLanguage: "Show common variants in BRCA2",
		SQL:             "SELECT * FROM variants WHERE gene = 'BRCA2' AND af > 0.01",
		Description:     "Find common variants (>1% frequency) in BRCA2",
	},
	{
		NaturalLanguage: "Find mutations in DNA repair genes",
		SQL:             "SELECT * FROM variants WHERE gene IN ('BRCA1', 'BRCA2', 'MLH1', 'MSH2', 'PTEN')",
		Description:     "Find mutations in key DNA repair pathway genes",
	},
	{
		NaturalLanguage: "What are high frequency pathogenic variants?",
		SQL:             "SELECT * FROM variants WHERE pathogenicity = 'Pathogenic' AND af > 0.005",
		Description:     "Find pathogenic variants with frequency > 0.5%",
	},
	{
		NaturalLanguage: "Show me mitochondrial mutations",
		SQL:             "SELECT * FROM variants WHERE chromosome = 'MT'",
		Description:     "Find mitochondrial DNA variants",
	},
}

// QueryExample represents an example query mapping
type QueryExample struct {
	NaturalLanguage string
	SQL             string
	Description     string
}
