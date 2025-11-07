// Package integrations - Export GenomeVedic annotations back to Galaxy
package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ExportFormat represents supported export file formats
type ExportFormat string

const (
	FormatBED  ExportFormat = "bed"
	FormatGTF  ExportFormat = "gtf"
	FormatGFF3 ExportFormat = "gff3"
	FormatVCF  ExportFormat = "vcf"
)

// GalaxyExportRequest represents a request to export annotations to Galaxy
type GalaxyExportRequest struct {
	SessionID     string       `json:"session_id"`
	HistoryID     string       `json:"history_id"`
	DatasetName   string       `json:"dataset_name"`
	Format        ExportFormat `json:"format"`
	Annotations   []Annotation `json:"annotations"`
	IncludeHeader bool         `json:"include_header"`
	Metadata      ExportMeta   `json:"metadata"`
}

// ExportMeta contains metadata about the export
type ExportMeta struct {
	GenomeBuild    string    `json:"genome_build"`
	Source         string    `json:"source"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	Description    string    `json:"description"`
	TotalFeatures  int       `json:"total_features"`
	AnalysisMethod string    `json:"analysis_method"`
}

// Annotation represents a genomic annotation
type Annotation struct {
	Chromosome  string                 `json:"chromosome"`
	Start       int64                  `json:"start"`
	End         int64                  `json:"end"`
	Name        string                 `json:"name"`
	Score       float64                `json:"score"`
	Strand      string                 `json:"strand"` // +, -, or .
	Type        string                 `json:"type"`   // gene, exon, mutation, etc.
	Attributes  map[string]string      `json:"attributes"`
	Source      string                 `json:"source"`
	Phase       int                    `json:"phase"` // For GTF/GFF3
	ExtraFields []string               `json:"extra_fields"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GalaxyExportResponse represents the response after exporting to Galaxy
type GalaxyExportResponse struct {
	Success       bool   `json:"success"`
	DatasetID     string `json:"dataset_id"`
	HistoryID     string `json:"history_id"`
	DatasetName   string `json:"dataset_name"`
	Format        string `json:"format"`
	FeatureCount  int    `json:"feature_count"`
	FileSize      int64  `json:"file_size_bytes"`
	GalaxyURL     string `json:"galaxy_url"`
	DownloadURL   string `json:"download_url"`
	ProcessTimeMs int64  `json:"process_time_ms"`
	Error         string `json:"error,omitempty"`
}

// GalaxyExporter handles exporting annotations to Galaxy
type GalaxyExporter struct {
	oauthClient *GalaxyOAuthClient
}

// NewGalaxyExporter creates a new Galaxy exporter
func NewGalaxyExporter(oauthClient *GalaxyOAuthClient) *GalaxyExporter {
	return &GalaxyExporter{
		oauthClient: oauthClient,
	}
}

// ExportToGalaxy exports annotations to a Galaxy history
func (e *GalaxyExporter) ExportToGalaxy(req GalaxyExportRequest, apiKey string) (*GalaxyExportResponse, error) {
	startTime := time.Now()

	// Validate API key
	apiClient, err := e.oauthClient.GetGalaxyAPIClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	// Convert annotations to requested format
	var content string
	switch req.Format {
	case FormatBED:
		content = e.convertToBED(req.Annotations, req.IncludeHeader, req.Metadata)
	case FormatGTF:
		content = e.convertToGTF(req.Annotations, req.IncludeHeader, req.Metadata)
	case FormatGFF3:
		content = e.convertToGFF3(req.Annotations, req.IncludeHeader, req.Metadata)
	case FormatVCF:
		content = e.convertToVCF(req.Annotations, req.IncludeHeader, req.Metadata)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	// Upload to Galaxy
	datasetID, downloadURL, err := apiClient.UploadToHistory(req.HistoryID, req.DatasetName, content, string(req.Format))
	if err != nil {
		return nil, fmt.Errorf("failed to upload to Galaxy: %w", err)
	}

	processingTime := time.Since(startTime).Milliseconds()

	response := &GalaxyExportResponse{
		Success:       true,
		DatasetID:     datasetID,
		HistoryID:     req.HistoryID,
		DatasetName:   req.DatasetName,
		Format:        string(req.Format),
		FeatureCount:  len(req.Annotations),
		FileSize:      int64(len(content)),
		GalaxyURL:     apiClient.baseURL,
		DownloadURL:   downloadURL,
		ProcessTimeMs: processingTime,
	}

	return response, nil
}

// convertToBED converts annotations to BED format
func (e *GalaxyExporter) convertToBED(annotations []Annotation, includeHeader bool, meta ExportMeta) string {
	var buf bytes.Buffer

	if includeHeader {
		buf.WriteString(fmt.Sprintf("# BED format export from GenomeVedic\n"))
		buf.WriteString(fmt.Sprintf("# Created: %s\n", meta.CreatedAt.Format(time.RFC3339)))
		buf.WriteString(fmt.Sprintf("# Genome build: %s\n", meta.GenomeBuild))
		buf.WriteString(fmt.Sprintf("# Total features: %d\n", meta.TotalFeatures))
		buf.WriteString(fmt.Sprintf("# Source: %s\n", meta.Source))
		if meta.Description != "" {
			buf.WriteString(fmt.Sprintf("# Description: %s\n", meta.Description))
		}
		buf.WriteString("#\n")
	}

	// Sort annotations by chromosome and position
	sortedAnnotations := make([]Annotation, len(annotations))
	copy(sortedAnnotations, annotations)
	sort.Slice(sortedAnnotations, func(i, j int) bool {
		if sortedAnnotations[i].Chromosome != sortedAnnotations[j].Chromosome {
			return sortedAnnotations[i].Chromosome < sortedAnnotations[j].Chromosome
		}
		return sortedAnnotations[i].Start < sortedAnnotations[j].Start
	})

	// BED format: chrom chromStart chromEnd name score strand
	for _, ann := range sortedAnnotations {
		score := int(ann.Score * 1000) // BED scores are 0-1000
		if score > 1000 {
			score = 1000
		}

		buf.WriteString(fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s",
			ann.Chromosome,
			ann.Start,
			ann.End,
			ann.Name,
			score,
			ann.Strand))

		// Add extra fields if present (BED12 format)
		if len(ann.ExtraFields) > 0 {
			buf.WriteString("\t")
			buf.WriteString(strings.Join(ann.ExtraFields, "\t"))
		}

		buf.WriteString("\n")
	}

	return buf.String()
}

// convertToGTF converts annotations to GTF format
func (e *GalaxyExporter) convertToGTF(annotations []Annotation, includeHeader bool, meta ExportMeta) string {
	var buf bytes.Buffer

	if includeHeader {
		buf.WriteString(fmt.Sprintf("#gtf-version 2.2\n"))
		buf.WriteString(fmt.Sprintf("#genome-build %s\n", meta.GenomeBuild))
		buf.WriteString(fmt.Sprintf("#date %s\n", meta.CreatedAt.Format("2006-01-02")))
		if meta.Description != "" {
			buf.WriteString(fmt.Sprintf("#description: %s\n", meta.Description))
		}
	}

	// GTF format: seqname source feature start end score strand frame attributes
	for _, ann := range annotations {
		source := ann.Source
		if source == "" {
			source = "GenomeVedic"
		}

		featureType := ann.Type
		if featureType == "" {
			featureType = "feature"
		}

		// Build attributes string
		var attrs []string
		if ann.Name != "" {
			attrs = append(attrs, fmt.Sprintf(`gene_id "%s"`, ann.Name))
			attrs = append(attrs, fmt.Sprintf(`transcript_id "%s"`, ann.Name))
		}

		for key, value := range ann.Attributes {
			attrs = append(attrs, fmt.Sprintf(`%s "%s"`, key, value))
		}

		attributeStr := strings.Join(attrs, "; ")
		if attributeStr != "" {
			attributeStr += ";"
		}

		buf.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%.3f\t%s\t%d\t%s\n",
			ann.Chromosome,
			source,
			featureType,
			ann.Start+1, // GTF is 1-based
			ann.End,
			ann.Score,
			ann.Strand,
			ann.Phase,
			attributeStr))
	}

	return buf.String()
}

// convertToGFF3 converts annotations to GFF3 format
func (e *GalaxyExporter) convertToGFF3(annotations []Annotation, includeHeader bool, meta ExportMeta) string {
	var buf bytes.Buffer

	if includeHeader {
		buf.WriteString("##gff-version 3\n")
		buf.WriteString(fmt.Sprintf("##genome-build %s\n", meta.GenomeBuild))
		buf.WriteString(fmt.Sprintf("##date %s\n", meta.CreatedAt.Format("2006-01-02")))
	}

	// GFF3 format: seqid source type start end score strand phase attributes
	for i, ann := range annotations {
		source := ann.Source
		if source == "" {
			source = "GenomeVedic"
		}

		featureType := ann.Type
		if featureType == "" {
			featureType = "region"
		}

		// Build attributes string (key=value format)
		var attrs []string
		if ann.Name != "" {
			attrs = append(attrs, fmt.Sprintf("ID=%s_%d", ann.Name, i))
			attrs = append(attrs, fmt.Sprintf("Name=%s", ann.Name))
		} else {
			attrs = append(attrs, fmt.Sprintf("ID=feature_%d", i))
		}

		for key, value := range ann.Attributes {
			attrs = append(attrs, fmt.Sprintf("%s=%s", key, value))
		}

		attributeStr := strings.Join(attrs, ";")

		buf.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%.3f\t%s\t%d\t%s\n",
			ann.Chromosome,
			source,
			featureType,
			ann.Start+1, // GFF3 is 1-based
			ann.End,
			ann.Score,
			ann.Strand,
			ann.Phase,
			attributeStr))
	}

	return buf.String()
}

// convertToVCF converts annotations to VCF format (for variants)
func (e *GalaxyExporter) convertToVCF(annotations []Annotation, includeHeader bool, meta ExportMeta) string {
	var buf bytes.Buffer

	if includeHeader {
		buf.WriteString("##fileformat=VCFv4.3\n")
		buf.WriteString(fmt.Sprintf("##fileDate=%s\n", meta.CreatedAt.Format("20060102")))
		buf.WriteString(fmt.Sprintf("##source=%s\n", meta.Source))
		buf.WriteString(fmt.Sprintf("##reference=%s\n", meta.GenomeBuild))
		buf.WriteString("##INFO=<ID=TYPE,Number=1,Type=String,Description=\"Type of variant\">\n")
		buf.WriteString("##INFO=<ID=SOURCE,Number=1,Type=String,Description=\"Source of annotation\">\n")
		buf.WriteString("#CHROM\tPOS\tID\tREF\tALT\tQUAL\tFILTER\tINFO\n")
	}

	// VCF format for variants
	for _, ann := range annotations {
		id := ann.Name
		if id == "" {
			id = "."
		}

		// Get REF and ALT from attributes
		ref := ann.Attributes["ref"]
		if ref == "" {
			ref = "N"
		}

		alt := ann.Attributes["alt"]
		if alt == "" {
			alt = "."
		}

		qual := int(ann.Score)
		if qual == 0 {
			qual = 30 // Default quality
		}

		filter := "PASS"

		// Build INFO field
		var infoFields []string
		if ann.Type != "" {
			infoFields = append(infoFields, fmt.Sprintf("TYPE=%s", ann.Type))
		}
		if ann.Source != "" {
			infoFields = append(infoFields, fmt.Sprintf("SOURCE=%s", ann.Source))
		}

		for key, value := range ann.Attributes {
			if key != "ref" && key != "alt" {
				infoFields = append(infoFields, fmt.Sprintf("%s=%s", strings.ToUpper(key), value))
			}
		}

		info := "."
		if len(infoFields) > 0 {
			info = strings.Join(infoFields, ";")
		}

		buf.WriteString(fmt.Sprintf("%s\t%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
			ann.Chromosome,
			ann.Start+1, // VCF is 1-based
			id,
			ref,
			alt,
			qual,
			filter,
			info))
	}

	return buf.String()
}

// UploadToHistory uploads content to a Galaxy history
func (g *GalaxyAPIClient) UploadToHistory(historyID, name, content, format string) (string, string, error) {
	uploadURL := fmt.Sprintf("%s/api/tools", g.baseURL)

	// Prepare upload payload
	payload := map[string]interface{}{
		"tool_id": "upload1",
		"history_id": historyID,
		"inputs": map[string]interface{}{
			"files_0|type":     "upload_dataset",
			"files_0|NAME":     name,
			"files_0|file_data": content,
			"dbkey":            "?",
			"file_type":        format,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Outputs []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"outputs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	if len(result.Outputs) == 0 {
		return "", "", fmt.Errorf("no outputs returned from upload")
	}

	datasetID := result.Outputs[0].ID
	downloadURL := fmt.Sprintf("%s/api/histories/%s/contents/%s/display", g.baseURL, historyID, datasetID)

	return datasetID, downloadURL, nil
}

// CreateExampleAnnotations creates example annotations for testing
func CreateExampleAnnotations() []Annotation {
	return []Annotation{
		{
			Chromosome: "chr1",
			Start:      1000000,
			End:        1001000,
			Name:       "BRCA1_mutation",
			Score:      0.95,
			Strand:     "+",
			Type:       "mutation",
			Source:     "GenomeVedic",
			Attributes: map[string]string{
				"gene":       "BRCA1",
				"mutation":   "missense",
				"impact":     "HIGH",
				"ref":        "A",
				"alt":        "G",
				"frequency":  "0.05",
			},
		},
		{
			Chromosome: "chr17",
			Start:      43044295,
			End:        43125483,
			Name:       "BRCA1",
			Score:      1.0,
			Strand:     "-",
			Type:       "gene",
			Source:     "GenomeVedic",
			Attributes: map[string]string{
				"gene_name":    "BRCA1",
				"gene_type":    "protein_coding",
				"description":  "Breast cancer type 1 susceptibility protein",
			},
		},
		{
			Chromosome: "chr13",
			Start:      32315086,
			End:        32400266,
			Name:       "BRCA2",
			Score:      1.0,
			Strand:     "+",
			Type:       "gene",
			Source:     "GenomeVedic",
			Attributes: map[string]string{
				"gene_name":    "BRCA2",
				"gene_type":    "protein_coding",
				"description":  "Breast cancer type 2 susceptibility protein",
			},
		},
	}
}
