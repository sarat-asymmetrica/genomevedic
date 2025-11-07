package crispr

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler handles HTTP requests for CRISPR design
type Handler struct {
	designers map[CasEnzyme]*Designer
	exporter  *Exporter
}

// NewHandler creates a new CRISPR handler
func NewHandler() *Handler {
	// Pre-create designers for common enzymes
	designers := make(map[CasEnzyme]*Designer)
	enzymes := []CasEnzyme{Cas9, Cas9HF1, xCas9, Cas12a, Cas13, SaCas9}

	for _, enzyme := range enzymes {
		designers[enzyme] = NewDesigner(enzyme)
	}

	return &Handler{
		designers: designers,
		exporter:  NewExporter(),
	}
}

// HandleDesign handles POST /api/v1/crispr/design
func (h *Handler) HandleDesign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req DesignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate enzyme
	if req.Enzyme == "" {
		req.Enzyme = Cas9 // Default to SpCas9
	}

	// Get or create designer for enzyme
	designer, exists := h.designers[req.Enzyme]
	if !exists {
		designer = NewDesigner(req.Enzyme)
		h.designers[req.Enzyme] = designer
	}

	// Design guides
	response, err := designer.Design(req)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, response)
}

// HandleBatchDesign handles POST /api/v1/crispr/design/batch
func (h *Handler) HandleBatchDesign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var requests []DesignRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(requests) == 0 {
		h.sendError(w, http.StatusBadRequest, "at least one design request required")
		return
	}

	// Process each request
	responses := make([]*DesignResponse, 0, len(requests))
	for _, req := range requests {
		if req.Enzyme == "" {
			req.Enzyme = Cas9
		}

		designer, exists := h.designers[req.Enzyme]
		if !exists {
			designer = NewDesigner(req.Enzyme)
			h.designers[req.Enzyme] = designer
		}

		response, err := designer.Design(req)
		if err != nil {
			response = &DesignResponse{
				Warnings: []string{fmt.Sprintf("Design failed: %v", err)},
			}
		}
		responses = append(responses, response)
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"count":     len(responses),
		"responses": responses,
	})
}

// HandleExport handles POST /api/v1/crispr/export
func (h *Handler) HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Guides) == 0 {
		h.sendError(w, http.StatusBadRequest, "no guides to export")
		return
	}

	if req.Format == "" {
		req.Format = ExportCSV // Default to CSV
	}

	// Export
	data, err := h.exporter.Export(req)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Set appropriate content type
	contentType := "text/csv"
	filename := "guides.csv"
	switch req.Format {
	case ExportCSV:
		contentType = "text/csv"
		filename = "guides.csv"
	case ExportGenBank:
		contentType = "text/plain"
		filename = "guides.gb"
	case ExportPDF:
		contentType = "application/pdf"
		filename = "guides.pdf"
	case ExportJSON:
		contentType = "application/json"
		filename = "guides.json"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// HandleGetEnzymes handles GET /api/v1/crispr/enzymes
func (h *Handler) HandleGetEnzymes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	enzymes := []map[string]interface{}{
		{
			"name":        string(Cas9),
			"pam":         "NGG",
			"guide_length": 20,
			"description": "Streptococcus pyogenes Cas9 (most common)",
		},
		{
			"name":        string(Cas9HF1),
			"pam":         "NGG",
			"guide_length": 20,
			"description": "High-fidelity Cas9 variant",
		},
		{
			"name":        string(xCas9),
			"pam":         "NG",
			"guide_length": 20,
			"description": "Expanded PAM Cas9 (NG, NGA, NGC, NGT)",
		},
		{
			"name":        string(Cas12a),
			"pam":         "TTTV",
			"guide_length": 23,
			"description": "Cas12a/Cpf1 (TTTV PAM, 5' PAM)",
		},
		{
			"name":        string(Cas13),
			"pam":         "none",
			"guide_length": 28,
			"description": "RNA-targeting Cas13",
		},
		{
			"name":        string(SaCas9),
			"pam":         "NNGRRT",
			"guide_length": 21,
			"description": "Staphylococcus aureus Cas9 (smaller)",
		},
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"enzymes": enzymes,
		"count":   len(enzymes),
	})
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, statusCode int, message string) {
	h.sendJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// RegisterRoutes registers CRISPR routes with a mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/v1/crispr/design", corsMiddleware(h.HandleDesign))
	mux.HandleFunc("/api/v1/crispr/design/batch", corsMiddleware(h.HandleBatchDesign))
	mux.HandleFunc("/api/v1/crispr/export", corsMiddleware(h.HandleExport))
	mux.HandleFunc("/api/v1/crispr/enzymes", corsMiddleware(h.HandleGetEnzymes))
}
