<script>
/**
 * CRISPR Guide RNA Designer Component
 *
 * Features:
 * - Target region selection (gene name or coordinates)
 * - Enzyme selection (Cas9, Cas12a, etc.)
 * - Guide design with Doench scoring
 * - Off-target prediction
 * - Export to CSV, GenBank, PDF
 * - Visual guide ranking display
 */

import { onMount } from 'svelte';

// Props
export let apiEndpoint = 'http://localhost:8080/api/v1';
export let onGuidesDesigned = null; // Callback when guides are designed

// Design form state
let targetType = 'gene'; // 'gene', 'coordinates', 'sequence'
let geneName = '';
let chromosome = '';
let startPos = '';
let endPos = '';
let targetSequence = '';
let selectedEnzyme = 'SpCas9';
let maxGuides = 10;
let minDoench = 0.2;
let maxOffTargets = 5;
let gcMin = 40;
let gcMax = 60;
let excludePolyT = true;

// UI state
let isLoading = false;
let showResults = false;
let guides = [];
let designResponse = null;
let errorMessage = '';
let enzymes = [];
let selectedGuide = null;
let exportFormat = 'csv';
let showExportModal = false;

// Load available enzymes on mount
onMount(async () => {
    await loadEnzymes();
});

// Load available Cas enzymes
async function loadEnzymes() {
    try {
        const response = await fetch(`${apiEndpoint}/crispr/enzymes`);
        const data = await response.json();

        if (data.success) {
            enzymes = data.enzymes || [];
            if (enzymes.length > 0) {
                selectedEnzyme = enzymes[0].name;
            }
        }
    } catch (error) {
        console.error('Failed to load enzymes:', error);
    }
}

// Design guides
async function designGuides() {
    errorMessage = '';
    isLoading = true;

    try {
        // Build request
        const request = {
            enzyme: selectedEnzyme,
            max_guides: parseInt(maxGuides),
            min_doench: parseFloat(minDoench),
            max_off_target: parseInt(maxOffTargets),
            gc_min: parseFloat(gcMin),
            gc_max: parseFloat(gcMax),
            exclude_poly_t: excludePolyT
        };

        // Add target specification
        if (targetType === 'gene') {
            request.gene_name = geneName;
        } else if (targetType === 'coordinates') {
            request.chromosome = chromosome;
            request.start = parseInt(startPos);
            request.end = parseInt(endPos);
        } else if (targetType === 'sequence') {
            request.sequence = targetSequence.toUpperCase();
        }

        // Call API
        const response = await fetch(`${apiEndpoint}/crispr/design`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(request)
        });

        const data = await response.json();

        if (response.ok && data.guides) {
            guides = data.guides;
            designResponse = data;
            showResults = true;

            // Callback
            if (onGuidesDesigned) {
                onGuidesDesigned(guides);
            }
        } else {
            errorMessage = data.error || 'Failed to design guides';
        }
    } catch (error) {
        errorMessage = `Error: ${error.message}`;
        console.error('Design failed:', error);
    } finally {
        isLoading = false;
    }
}

// Export guides
async function exportGuides() {
    try {
        const request = {
            guides: guides,
            format: exportFormat
        };

        const response = await fetch(`${apiEndpoint}/crispr/export`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(request)
        });

        if (response.ok) {
            // Download file
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;

            const extension = exportFormat === 'genbank' ? 'gb' : exportFormat;
            a.download = `crispr_guides.${extension}`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            window.URL.revokeObjectURL(url);

            showExportModal = false;
        } else {
            alert('Export failed');
        }
    } catch (error) {
        console.error('Export failed:', error);
        alert('Export failed: ' + error.message);
    }
}

// Get efficiency badge color
function getEfficiencyColor(score) {
    if (score >= 0.7) return '#10b981'; // green
    if (score >= 0.4) return '#f59e0b'; // orange
    if (score >= 0.2) return '#ef4444'; // red
    return '#6b7280'; // gray
}

// Get efficiency label
function getEfficiencyLabel(score) {
    if (score >= 0.7) return 'High';
    if (score >= 0.4) return 'Medium';
    if (score >= 0.2) return 'Low';
    return 'Very Low';
}

// Format position
function formatPosition(guide) {
    return `${guide.chromosome}:${guide.position.toLocaleString()}`;
}

// Copy sequence to clipboard
function copySequence(sequence) {
    navigator.clipboard.writeText(sequence);
    alert('Sequence copied to clipboard!');
}
</script>

<style>
.crispr-designer {
    background: white;
    border-radius: 12px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    padding: 24px;
    max-width: 1400px;
    margin: 20px auto;
}

.header {
    margin-bottom: 24px;
}

.header h2 {
    margin: 0 0 8px 0;
    color: #1f2937;
    font-size: 24px;
}

.header p {
    margin: 0;
    color: #6b7280;
    font-size: 14px;
}

.design-form {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    margin-bottom: 24px;
}

.form-group {
    display: flex;
    flex-direction: column;
}

.form-group label {
    font-weight: 600;
    color: #374151;
    margin-bottom: 6px;
    font-size: 14px;
}

.form-group input,
.form-group select,
.form-group textarea {
    padding: 10px;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 14px;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.target-type-tabs {
    display: flex;
    gap: 10px;
    margin-bottom: 16px;
}

.tab {
    padding: 8px 16px;
    border: 1px solid #d1d5db;
    background: white;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    transition: all 0.2s;
}

.tab:hover {
    background: #f3f4f6;
}

.tab.active {
    background: #3b82f6;
    color: white;
    border-color: #3b82f6;
}

.design-button {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 12px 32px;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    font-size: 16px;
    cursor: pointer;
    transition: transform 0.2s;
    width: fit-content;
}

.design-button:hover {
    transform: translateY(-2px);
}

.design-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
}

.guides-list {
    margin-top: 24px;
}

.guide-card {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 12px;
    transition: all 0.2s;
}

.guide-card:hover {
    border-color: #3b82f6;
    box-shadow: 0 2px 8px rgba(59, 130, 246, 0.1);
}

.guide-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}

.guide-rank {
    background: #3b82f6;
    color: white;
    padding: 4px 12px;
    border-radius: 16px;
    font-weight: 600;
    font-size: 14px;
}

.guide-score {
    font-size: 18px;
    font-weight: 700;
    color: #1f2937;
}

.sequence-display {
    font-family: 'Courier New', monospace;
    font-size: 16px;
    background: white;
    padding: 12px;
    border-radius: 6px;
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.sequence-text {
    font-weight: 600;
}

.copy-btn {
    background: #3b82f6;
    color: white;
    border: none;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
}

.copy-btn:hover {
    background: #2563eb;
}

.metrics-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
}

.metric {
    display: flex;
    flex-direction: column;
}

.metric-label {
    font-size: 12px;
    color: #6b7280;
    margin-bottom: 4px;
}

.metric-value {
    font-size: 16px;
    font-weight: 600;
    color: #1f2937;
}

.efficiency-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 12px;
    color: white;
    font-size: 12px;
    font-weight: 600;
}

.export-section {
    margin-top: 24px;
    padding-top: 24px;
    border-top: 1px solid #e5e7eb;
}

.export-buttons {
    display: flex;
    gap: 12px;
}

.export-btn {
    padding: 10px 20px;
    border: 1px solid #d1d5db;
    background: white;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.2s;
}

.export-btn:hover {
    background: #f3f4f6;
    border-color: #3b82f6;
}

.error-message {
    background: #fee2e2;
    border: 1px solid #fecaca;
    color: #991b1b;
    padding: 12px;
    border-radius: 6px;
    margin-bottom: 16px;
}

.loading {
    text-align: center;
    padding: 40px;
    color: #6b7280;
}

.spinner {
    border: 3px solid #f3f4f6;
    border-top: 3px solid #3b82f6;
    border-radius: 50%;
    width: 40px;
    height: 40px;
    animation: spin 1s linear infinite;
    margin: 0 auto 16px auto;
}

@keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
}

.warnings {
    background: #fef3c7;
    border: 1px solid #fde68a;
    color: #92400e;
    padding: 12px;
    border-radius: 6px;
    margin-bottom: 16px;
    font-size: 14px;
}

.summary {
    background: #eff6ff;
    border: 1px solid #dbeafe;
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 20px;
}

.summary h3 {
    margin: 0 0 12px 0;
    color: #1e40af;
}

.summary-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
}
</style>

<div class="crispr-designer">
    <div class="header">
        <h2>🧬 CRISPR Guide RNA Designer</h2>
        <p>Design optimized guide RNAs with Doench 2016 efficiency scoring and off-target prediction</p>
    </div>

    <!-- Design Form -->
    <div class="design-form">
        <!-- Target Type Selection -->
        <div class="form-group" style="grid-column: 1 / -1;">
            <label>Target Specification</label>
            <div class="target-type-tabs">
                <button
                    class="tab"
                    class:active={targetType === 'gene'}
                    on:click={() => targetType = 'gene'}>
                    Gene Name
                </button>
                <button
                    class="tab"
                    class:active={targetType === 'coordinates'}
                    on:click={() => targetType = 'coordinates'}>
                    Coordinates
                </button>
                <button
                    class="tab"
                    class:active={targetType === 'sequence'}
                    on:click={() => targetType = 'sequence'}>
                    Direct Sequence
                </button>
            </div>
        </div>

        <!-- Gene Name Input -->
        {#if targetType === 'gene'}
            <div class="form-group">
                <label>Gene Name</label>
                <input
                    type="text"
                    bind:value={geneName}
                    placeholder="e.g., TP53, BRCA1, KRAS">
            </div>
        {/if}

        <!-- Coordinates Input -->
        {#if targetType === 'coordinates'}
            <div class="form-group">
                <label>Chromosome</label>
                <input
                    type="text"
                    bind:value={chromosome}
                    placeholder="e.g., chr17">
            </div>
            <div class="form-group">
                <label>Start Position</label>
                <input
                    type="number"
                    bind:value={startPos}
                    placeholder="e.g., 7676154">
            </div>
            <div class="form-group">
                <label>End Position</label>
                <input
                    type="number"
                    bind:value={endPos}
                    placeholder="e.g., 7676300">
            </div>
        {/if}

        <!-- Sequence Input -->
        {#if targetType === 'sequence'}
            <div class="form-group" style="grid-column: 1 / -1;">
                <label>Target Sequence (DNA)</label>
                <textarea
                    bind:value={targetSequence}
                    placeholder="Enter target DNA sequence (ATGC)..."
                    rows="4"></textarea>
            </div>
        {/if}

        <!-- Enzyme Selection -->
        <div class="form-group">
            <label>Cas Enzyme</label>
            <select bind:value={selectedEnzyme}>
                {#each enzymes as enzyme}
                    <option value={enzyme.name}>
                        {enzyme.name} - {enzyme.pam} ({enzyme.description})
                    </option>
                {/each}
            </select>
        </div>

        <!-- Max Guides -->
        <div class="form-group">
            <label>Number of Guides</label>
            <input
                type="number"
                bind:value={maxGuides}
                min="1"
                max="50">
        </div>

        <!-- Min Doench Score -->
        <div class="form-group">
            <label>Min Efficiency Score</label>
            <input
                type="number"
                bind:value={minDoench}
                min="0"
                max="1"
                step="0.1">
        </div>

        <!-- Max Off-Targets -->
        <div class="form-group">
            <label>Max Off-Targets</label>
            <input
                type="number"
                bind:value={maxOffTargets}
                min="0"
                max="20">
        </div>

        <!-- GC Range -->
        <div class="form-group">
            <label>GC Content Range (%)</label>
            <div style="display: flex; gap: 8px; align-items: center;">
                <input
                    type="number"
                    bind:value={gcMin}
                    min="0"
                    max="100"
                    style="width: 80px;">
                <span>to</span>
                <input
                    type="number"
                    bind:value={gcMax}
                    min="0"
                    max="100"
                    style="width: 80px;">
            </div>
        </div>

        <!-- Exclude Poly-T -->
        <div class="form-group">
            <label>
                <input type="checkbox" bind:checked={excludePolyT}>
                Exclude guides with TTTT (poly-T runs)
            </label>
        </div>
    </div>

    <!-- Design Button -->
    <button
        class="design-button"
        on:click={designGuides}
        disabled={isLoading}>
        {isLoading ? 'Designing...' : '🔬 Design Guides'}
    </button>

    <!-- Error Message -->
    {#if errorMessage}
        <div class="error-message">
            ⚠️ {errorMessage}
        </div>
    {/if}

    <!-- Loading -->
    {#if isLoading}
        <div class="loading">
            <div class="spinner"></div>
            <p>Designing CRISPR guides...</p>
        </div>
    {/if}

    <!-- Results -->
    {#if showResults && designResponse}
        <!-- Summary -->
        <div class="summary">
            <h3>Design Summary</h3>
            <div class="summary-stats">
                <div>
                    <div class="metric-label">Guides Found</div>
                    <div class="metric-value">{designResponse.total_found}</div>
                </div>
                <div>
                    <div class="metric-label">Target Region</div>
                    <div class="metric-value" style="font-size: 14px;">{designResponse.region}</div>
                </div>
                <div>
                    <div class="metric-label">Processing Time</div>
                    <div class="metric-value">{designResponse.processing_time_ms.toFixed(0)} ms</div>
                </div>
            </div>
        </div>

        <!-- Warnings -->
        {#if designResponse.warnings && designResponse.warnings.length > 0}
            <div class="warnings">
                <strong>⚠️ Warnings:</strong>
                <ul style="margin: 8px 0 0 20px; padding: 0;">
                    {#each designResponse.warnings as warning}
                        <li>{warning}</li>
                    {/each}
                </ul>
            </div>
        {/if}

        <!-- Guides List -->
        <div class="guides-list">
            <h3>Top Guide RNAs</h3>
            {#each guides as guide, index}
                <div class="guide-card">
                    <div class="guide-header">
                        <span class="guide-rank">#{index + 1}</span>
                        <span class="guide-score">Score: {guide.rank_score.toFixed(3)}</span>
                    </div>

                    <div class="sequence-display">
                        <div class="sequence-text">
                            5'-{guide.sequence}-{guide.pam_sequence}-3'
                        </div>
                        <button class="copy-btn" on:click={() => copySequence(guide.sequence)}>
                            📋 Copy
                        </button>
                    </div>

                    <div class="metrics-grid">
                        <div class="metric">
                            <div class="metric-label">Efficiency</div>
                            <div class="metric-value">
                                <span
                                    class="efficiency-badge"
                                    style="background: {getEfficiencyColor(guide.doench_score)}">
                                    {getEfficiencyLabel(guide.doench_score)}
                                </span>
                                {guide.doench_score.toFixed(3)}
                            </div>
                        </div>

                        <div class="metric">
                            <div class="metric-label">Off-Targets</div>
                            <div class="metric-value">{guide.off_target_count}</div>
                        </div>

                        <div class="metric">
                            <div class="metric-label">Specificity</div>
                            <div class="metric-value">{guide.off_target_score.toFixed(1)}</div>
                        </div>

                        <div class="metric">
                            <div class="metric-label">GC Content</div>
                            <div class="metric-value">{guide.gc_content.toFixed(1)}%</div>
                        </div>

                        <div class="metric">
                            <div class="metric-label">Position</div>
                            <div class="metric-value" style="font-size: 12px;">{formatPosition(guide)}</div>
                        </div>

                        <div class="metric">
                            <div class="metric-label">Strand</div>
                            <div class="metric-value">{guide.strand}</div>
                        </div>
                    </div>
                </div>
            {/each}
        </div>

        <!-- Export Section -->
        <div class="export-section">
            <h3>Export Guides</h3>
            <div style="display: flex; gap: 16px; align-items: center;">
                <select bind:value={exportFormat} style="padding: 10px; border-radius: 6px;">
                    <option value="csv">CSV</option>
                    <option value="genbank">GenBank</option>
                    <option value="pdf">PDF Report</option>
                    <option value="json">JSON</option>
                </select>
                <button class="export-btn" on:click={exportGuides}>
                    📥 Download
                </button>
            </div>
        </div>
    {/if}
</div>
