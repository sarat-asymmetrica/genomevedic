<script>
/**
 * Natural Language Query Bar Component
 *
 * Features:
 * - Search input with autocomplete
 * - Example queries on empty state
 * - Query history (recent searches)
 * - Result summary and SQL display
 * - Error handling and validation
 */

import { onMount } from 'svelte';

// Props
export let apiEndpoint = 'http://localhost:8080/api/v1';
export let onResultsUpdate = null; // Callback when results are available

// State
let query = '';
let isLoading = false;
let showExamples = true;
let showHistory = false;
let showResults = false;
let examples = [];
let history = [];
let currentResult = null;
let errorMessage = '';
let showAutocomplete = false;
let filteredExamples = [];

// Result stats
let resultSummary = '';

// Load examples on mount
onMount(async () => {
    await loadExamples();
    loadHistory();
});

// Load example queries from API
async function loadExamples() {
    try {
        const response = await fetch(`${apiEndpoint}/query/examples`);
        const data = await response.json();

        if (data.success) {
            examples = data.examples || [];
        }
    } catch (error) {
        console.error('Failed to load examples:', error);
    }
}

// Load query history from localStorage
function loadHistory() {
    const stored = localStorage.getItem('genomevedic_query_history');
    if (stored) {
        try {
            history = JSON.parse(stored);
        } catch (e) {
            history = [];
        }
    }
}

// Save query to history
function saveToHistory(query, result) {
    const entry = {
        query,
        sql: result.generated_sql,
        timestamp: new Date().toISOString(),
        isValid: result.is_valid,
    };

    // Add to beginning of history, limit to 20 entries
    history = [entry, ...history.filter(h => h.query !== query)].slice(0, 20);
    localStorage.setItem('genomevedic_query_history', JSON.stringify(history));
}

// Handle input change for autocomplete
function handleInput(event) {
    query = event.target.value;

    if (query.length >= 3) {
        // Filter examples based on input
        const lowerQuery = query.toLowerCase();
        filteredExamples = examples.filter(ex =>
            ex.natural_language.toLowerCase().includes(lowerQuery) ||
            ex.description.toLowerCase().includes(lowerQuery)
        ).slice(0, 5);

        showAutocomplete = filteredExamples.length > 0;
    } else {
        showAutocomplete = false;
        filteredExamples = [];
    }

    showExamples = query.length === 0;
    showResults = false;
}

// Execute query
async function executeQuery() {
    if (!query.trim()) {
        return;
    }

    isLoading = true;
    errorMessage = '';
    showResults = false;
    showAutocomplete = false;

    try {
        const response = await fetch(`${apiEndpoint}/query/natural-language`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                query: query.trim(),
                user_id: getUserId(),
            }),
        });

        const data = await response.json();

        if (data.success && data.is_valid) {
            currentResult = data;
            showResults = true;
            showExamples = false;

            // Save to history
            saveToHistory(query, data);

            // Generate result summary
            generateResultSummary(data);

            // Callback for parent component
            if (onResultsUpdate) {
                onResultsUpdate(data);
            }
        } else {
            errorMessage = data.validation_error || data.error || 'Query validation failed';
            currentResult = data;
            showResults = true;
        }
    } catch (error) {
        errorMessage = `Failed to execute query: ${error.message}`;
        console.error('Query execution error:', error);
    } finally {
        isLoading = false;
    }
}

// Generate result summary text
function generateResultSummary(result) {
    if (!result.is_valid) {
        resultSummary = 'Query validation failed';
        return;
    }

    const count = result.result_count || 0;
    const time = result.execution_time_ms || 0;

    // Extract gene/chromosome from query for context
    const gene = extractGene(query);
    const chromosome = extractChromosome(query);

    if (gene) {
        resultSummary = `Found ${count} variants in ${gene} (${time}ms)`;
    } else if (chromosome) {
        resultSummary = `Found ${count} variants on chromosome ${chromosome} (${time}ms)`;
    } else {
        resultSummary = `Found ${count} variants (${time}ms)`;
    }
}

// Extract gene name from query
function extractGene(text) {
    const genes = ['TP53', 'BRCA1', 'BRCA2', 'KRAS', 'EGFR', 'PTEN', 'PIK3CA', 'APC', 'BRAF'];
    const upperText = text.toUpperCase();

    for (const gene of genes) {
        if (upperText.includes(gene)) {
            return gene;
        }
    }

    return null;
}

// Extract chromosome from query
function extractChromosome(text) {
    const match = text.match(/chromosome\s+(\d+|X|Y|MT)/i);
    return match ? match[1] : null;
}

// Get or create user ID
function getUserId() {
    let userId = localStorage.getItem('genomevedic_user_id');
    if (!userId) {
        userId = 'user_' + Math.random().toString(36).substr(2, 9);
        localStorage.setItem('genomevedic_user_id', userId);
    }
    return userId;
}

// Use example query
function useExample(example) {
    query = example.natural_language;
    showAutocomplete = false;
    executeQuery();
}

// Use history query
function useHistoryQuery(entry) {
    query = entry.query;
    showHistory = false;
    executeQuery();
}

// Handle keyboard shortcuts
function handleKeydown(event) {
    if (event.key === 'Enter') {
        executeQuery();
    } else if (event.key === 'Escape') {
        showAutocomplete = false;
        showHistory = false;
    }
}

// Toggle history panel
function toggleHistory() {
    showHistory = !showHistory;
    showAutocomplete = false;
}

// Clear history
function clearHistory() {
    history = [];
    localStorage.removeItem('genomevedic_query_history');
    showHistory = false;
}

// Copy SQL to clipboard
function copySQLToClipboard() {
    if (currentResult && currentResult.generated_sql) {
        navigator.clipboard.writeText(currentResult.generated_sql);
    }
}
</script>

<div class="nl-query-bar">
    <!-- Search Input -->
    <div class="search-container">
        <div class="search-input-wrapper">
            <input
                type="text"
                class="search-input"
                placeholder="Ask a question about genomic variants... (e.g., 'Show me all TP53 mutations')"
                bind:value={query}
                on:input={handleInput}
                on:keydown={handleKeydown}
                disabled={isLoading}
            />
            <div class="search-buttons">
                <button
                    class="history-btn"
                    on:click={toggleHistory}
                    title="Query History"
                >
                    📋
                </button>
                <button
                    class="search-btn"
                    on:click={executeQuery}
                    disabled={isLoading || !query.trim()}
                >
                    {isLoading ? '⏳' : '🔍'} {isLoading ? 'Searching...' : 'Search'}
                </button>
            </div>
        </div>

        <!-- Autocomplete Suggestions -->
        {#if showAutocomplete && filteredExamples.length > 0}
        <div class="autocomplete-panel">
            <div class="autocomplete-header">Suggestions:</div>
            {#each filteredExamples as example}
            <div
                class="autocomplete-item"
                on:click={() => useExample(example)}
            >
                <div class="autocomplete-query">{example.natural_language}</div>
                <div class="autocomplete-desc">{example.description}</div>
            </div>
            {/each}
        </div>
        {/if}

        <!-- History Panel -->
        {#if showHistory && history.length > 0}
        <div class="history-panel">
            <div class="history-header">
                <span>Recent Queries</span>
                <button class="clear-history-btn" on:click={clearHistory}>Clear</button>
            </div>
            {#each history.slice(0, 10) as entry}
            <div
                class="history-item"
                on:click={() => useHistoryQuery(entry)}
            >
                <div class="history-query">{entry.query}</div>
                <div class="history-meta">
                    <span class="history-time">{new Date(entry.timestamp).toLocaleTimeString()}</span>
                    {#if entry.isValid}
                    <span class="history-badge valid">✓</span>
                    {:else}
                    <span class="history-badge invalid">✗</span>
                    {/if}
                </div>
            </div>
            {/each}
        </div>
        {/if}
    </div>

    <!-- Example Queries (Empty State) -->
    {#if showExamples && examples.length > 0 && !showResults}
    <div class="examples-panel">
        <div class="examples-header">Try these example queries:</div>
        <div class="examples-grid">
            {#each examples.slice(0, 6) as example}
            <div
                class="example-card"
                on:click={() => useExample(example)}
            >
                <div class="example-query">{example.natural_language}</div>
                <div class="example-desc">{example.description}</div>
            </div>
            {/each}
        </div>
    </div>
    {/if}

    <!-- Results Panel -->
    {#if showResults && currentResult}
    <div class="results-panel">
        {#if errorMessage}
        <div class="error-message">
            <span class="error-icon">⚠️</span>
            <span class="error-text">{errorMessage}</span>
        </div>
        {/if}

        {#if currentResult.is_valid}
        <div class="result-summary">
            <span class="summary-icon">✓</span>
            <span class="summary-text">{resultSummary}</span>
        </div>

        <div class="sql-display">
            <div class="sql-header">
                <span>Generated SQL:</span>
                <button class="copy-btn" on:click={copySQLToClipboard}>📋 Copy</button>
            </div>
            <pre class="sql-code">{currentResult.generated_sql}</pre>
        </div>

        {#if currentResult.explanation}
        <div class="explanation">
            <strong>Explanation:</strong> {currentResult.explanation}
        </div>
        {/if}
        {/if}
    </div>
    {/if}
</div>

<style>
.nl-query-bar {
    width: 100%;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.search-container {
    position: relative;
    margin-bottom: 20px;
}

.search-input-wrapper {
    display: flex;
    gap: 10px;
    align-items: center;
    background: rgba(255, 255, 255, 0.05);
    border: 2px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 12px;
    transition: all 0.3s ease;
}

.search-input-wrapper:focus-within {
    border-color: rgba(0, 200, 255, 0.5);
    box-shadow: 0 0 20px rgba(0, 200, 255, 0.2);
}

.search-input {
    flex: 1;
    background: transparent;
    border: none;
    color: white;
    font-size: 16px;
    outline: none;
}

.search-input::placeholder {
    color: rgba(255, 255, 255, 0.4);
}

.search-buttons {
    display: flex;
    gap: 8px;
}

.history-btn {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 8px;
    color: white;
    padding: 8px 12px;
    cursor: pointer;
    font-size: 16px;
    transition: all 0.2s ease;
}

.history-btn:hover {
    background: rgba(255, 255, 255, 0.2);
}

.search-btn {
    background: linear-gradient(135deg, #00c8ff, #0080ff);
    border: none;
    border-radius: 8px;
    color: white;
    padding: 10px 20px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
}

.search-btn:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 200, 255, 0.4);
}

.search-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.autocomplete-panel,
.history-panel {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: rgba(20, 20, 40, 0.98);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    margin-top: 8px;
    padding: 12px;
    z-index: 100;
    max-height: 400px;
    overflow-y: auto;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}

.autocomplete-header,
.history-header {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.6);
    text-transform: uppercase;
    margin-bottom: 8px;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.clear-history-btn {
    background: rgba(255, 100, 100, 0.2);
    border: 1px solid rgba(255, 100, 100, 0.4);
    border-radius: 4px;
    color: #ff6464;
    padding: 4px 8px;
    font-size: 11px;
    cursor: pointer;
}

.autocomplete-item,
.history-item {
    padding: 10px;
    margin-bottom: 4px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
}

.autocomplete-item:hover,
.history-item:hover {
    background: rgba(255, 255, 255, 0.1);
}

.autocomplete-query,
.history-query {
    color: white;
    font-size: 14px;
    margin-bottom: 4px;
}

.autocomplete-desc {
    color: rgba(255, 255, 255, 0.5);
    font-size: 12px;
}

.history-meta {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 4px;
}

.history-time {
    color: rgba(255, 255, 255, 0.5);
    font-size: 11px;
}

.history-badge {
    font-size: 12px;
    padding: 2px 6px;
    border-radius: 4px;
}

.history-badge.valid {
    background: rgba(0, 255, 100, 0.2);
    color: #00ff64;
}

.history-badge.invalid {
    background: rgba(255, 100, 100, 0.2);
    color: #ff6464;
}

.examples-panel {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 20px;
}

.examples-header {
    color: rgba(255, 255, 255, 0.8);
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 16px;
}

.examples-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 12px;
}

.example-card {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 12px;
    cursor: pointer;
    transition: all 0.2s ease;
}

.example-card:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(0, 200, 255, 0.4);
    transform: translateY(-2px);
}

.example-query {
    color: white;
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 6px;
}

.example-desc {
    color: rgba(255, 255, 255, 0.5);
    font-size: 12px;
}

.results-panel {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 20px;
}

.error-message {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px;
    background: rgba(255, 100, 100, 0.1);
    border: 1px solid rgba(255, 100, 100, 0.3);
    border-radius: 8px;
    margin-bottom: 16px;
}

.error-icon {
    font-size: 20px;
}

.error-text {
    color: #ff6464;
    font-size: 14px;
}

.result-summary {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px;
    background: rgba(0, 255, 100, 0.1);
    border: 1px solid rgba(0, 255, 100, 0.3);
    border-radius: 8px;
    margin-bottom: 16px;
}

.summary-icon {
    font-size: 20px;
    color: #00ff64;
}

.summary-text {
    color: #00ff64;
    font-size: 14px;
    font-weight: 500;
}

.sql-display {
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 16px;
}

.sql-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    color: rgba(255, 255, 255, 0.8);
    font-size: 13px;
    font-weight: 600;
}

.copy-btn {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 4px;
    color: white;
    padding: 4px 8px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.2s ease;
}

.copy-btn:hover {
    background: rgba(255, 255, 255, 0.2);
}

.sql-code {
    color: #00ff88;
    font-family: 'Courier New', monospace;
    font-size: 13px;
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
}

.explanation {
    color: rgba(255, 255, 255, 0.7);
    font-size: 13px;
    line-height: 1.6;
}
</style>
