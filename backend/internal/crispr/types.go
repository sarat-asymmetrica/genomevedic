package crispr

import "time"

// CasEnzyme represents different CRISPR-Cas enzymes
type CasEnzyme string

const (
	Cas9      CasEnzyme = "SpCas9"      // Streptococcus pyogenes Cas9
	Cas9HF1   CasEnzyme = "SpCas9-HF1"  // High-fidelity Cas9
	xCas9     CasEnzyme = "xCas9"       // Expanded PAM Cas9
	Cas12a    CasEnzyme = "Cas12a"      // Cpf1
	Cas13     CasEnzyme = "Cas13"       // RNA targeting
	SaCas9    CasEnzyme = "SaCas9"      // Staphylococcus aureus Cas9
	NmeCas9   CasEnzyme = "NmeCas9"     // Neisseria meningitidis Cas9
)

// PAMSequence represents PAM site requirements for each Cas enzyme
type PAMSequence struct {
	Enzyme       CasEnzyme
	Pattern      string   // Regex pattern for PAM
	Offset       int      // Offset from guide sequence (-3 for NGG)
	GuideLength  int      // Length of guide RNA (20 for Cas9, 23 for Cas12a)
	Orientation  string   // "3prime" or "5prime"
}

// GuideRNA represents a single CRISPR guide RNA
type GuideRNA struct {
	ID              string    `json:"id"`
	Sequence        string    `json:"sequence"`        // 20bp guide sequence
	Chromosome      string    `json:"chromosome"`
	Position        int       `json:"position"`        // Genomic position
	Strand          string    `json:"strand"`          // "+" or "-"
	PAMSequence     string    `json:"pam_sequence"`    // PAM site (e.g., "NGG")
	Enzyme          CasEnzyme `json:"enzyme"`

	// Scoring metrics
	DoenchScore     float64   `json:"doench_score"`    // On-target efficiency (0-1)
	GCContent       float64   `json:"gc_content"`      // GC percentage (0-100)
	SelfCompScore   float64   `json:"self_comp_score"` // Self-complementarity
	OffTargetCount  int       `json:"off_target_count"`// Number of off-targets
	OffTargetScore  float64   `json:"off_target_score"`// Off-target specificity (0-100)

	// Composite score
	RankScore       float64   `json:"rank_score"`      // Final ranking score

	// Metadata
	GeneName        string    `json:"gene_name,omitempty"`
	Exon            int       `json:"exon,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// DesignRequest represents a CRISPR design request
type DesignRequest struct {
	// Target specification
	GeneName    string    `json:"gene_name,omitempty"`
	Chromosome  string    `json:"chromosome,omitempty"`
	Start       int       `json:"start,omitempty"`
	End         int       `json:"end,omitempty"`
	Sequence    string    `json:"sequence,omitempty"` // Direct sequence input

	// Design parameters
	Enzyme      CasEnzyme `json:"enzyme"`
	MaxGuides   int       `json:"max_guides"`   // Number of guides to return (default: 10)
	MinDoench   float64   `json:"min_doench"`   // Minimum Doench score (default: 0.2)
	MaxOffTarget int      `json:"max_off_target"` // Max allowed off-targets (default: 5)

	// Optional filters
	GCMin       float64   `json:"gc_min,omitempty"` // Min GC% (default: 40)
	GCMax       float64   `json:"gc_max,omitempty"` // Max GC% (default: 60)
	ExcludePolyT bool     `json:"exclude_poly_t"`   // Exclude TTTT runs
}

// DesignResponse represents the CRISPR design output
type DesignResponse struct {
	Guides        []GuideRNA `json:"guides"`
	TotalFound    int        `json:"total_found"`
	Region        string     `json:"region"`
	ProcessingTime float64   `json:"processing_time_ms"`
	Warnings      []string   `json:"warnings,omitempty"`
}

// OffTargetSite represents a potential off-target binding site
type OffTargetSite struct {
	Chromosome    string  `json:"chromosome"`
	Position      int     `json:"position"`
	Sequence      string  `json:"sequence"`
	Mismatches    int     `json:"mismatches"`     // Number of mismatches
	MismatchPos   []int   `json:"mismatch_pos"`   // Positions of mismatches
	InGene        bool    `json:"in_gene"`
	GeneName      string  `json:"gene_name,omitempty"`
	Score         float64 `json:"score"`          // CFD score or similar
}

// ExportFormat represents export file formats
type ExportFormat string

const (
	ExportCSV     ExportFormat = "csv"
	ExportGenBank ExportFormat = "genbank"
	ExportPDF     ExportFormat = "pdf"
	ExportJSON    ExportFormat = "json"
)

// ExportRequest represents an export request
type ExportRequest struct {
	Guides  []GuideRNA   `json:"guides"`
	Format  ExportFormat `json:"format"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// GetPAMSequence returns PAM configuration for a given enzyme
func GetPAMSequence(enzyme CasEnzyme) PAMSequence {
	switch enzyme {
	case Cas9, Cas9HF1:
		return PAMSequence{
			Enzyme:      enzyme,
			Pattern:     "[ACGT]GG",  // NGG
			Offset:      -3,
			GuideLength: 20,
			Orientation: "3prime",
		}
	case xCas9:
		return PAMSequence{
			Enzyme:      enzyme,
			Pattern:     "[ACGT]G[ACGT]",  // NGA, NGC, NGT
			Offset:      -3,
			GuideLength: 20,
			Orientation: "3prime",
		}
	case Cas12a:
		return PAMSequence{
			Enzyme:      enzyme,
			Pattern:     "TTT[ACGT]",  // TTTV
			Offset:      4,
			GuideLength: 23,
			Orientation: "5prime",
		}
	case Cas13:
		return PAMSequence{
			Enzyme:      enzyme,
			Pattern:     "",  // No strict PAM for Cas13
			Offset:      0,
			GuideLength: 28,
			Orientation: "5prime",
		}
	case SaCas9:
		return PAMSequence{
			Enzyme:      enzyme,
			Pattern:     "[ACGT][ACGT]GRRT",  // NNGRRT
			Offset:      -6,
			GuideLength: 21,
			Orientation: "3prime",
		}
	case NmeCas9:
		return PAMSequence{
			Enzyme:      enzyme,
			Pattern:     "[ACGT]{8}G[ACGT]TT",  // NNNNGATT
			Offset:      -8,
			GuideLength: 24,
			Orientation: "3prime",
		}
	default:
		// Default to SpCas9
		return PAMSequence{
			Enzyme:      Cas9,
			Pattern:     "[ACGT]GG",
			Offset:      -3,
			GuideLength: 20,
			Orientation: "3prime",
		}
	}
}
