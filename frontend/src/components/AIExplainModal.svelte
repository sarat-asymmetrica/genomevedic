<script>
	import { onMount } from 'svelte';

	// Props
	export let isOpen = false;
	export let variant = null; // { gene, variant, chromosome, position, refAllele, altAllele }
	export let onClose = () => {};

	// State
	let explanation = '';
	let context = null;
	let loading = false;
	let error = null;
	let cached = false;
	let responseTime = 0;
	let tokensUsed = 0;
	let costUSD = 0;
	let quality = 0;
	let copied = false;

	// API endpoint
	const API_BASE = 'http://localhost:8080';

	// Fetch explanation when modal opens
	$: if (isOpen && variant) {
		fetchExplanation();
	}

	async function fetchExplanation() {
		loading = true;
		error = null;
		explanation = '';
		context = null;
		copied = false;

		const startTime = performance.now();

		try {
			const response = await fetch(`${API_BASE}/api/v1/variants/explain`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					gene: variant.gene,
					variant: variant.variant,
					chromosome: variant.chromosome,
					position: variant.position,
					ref_allele: variant.refAllele || 'A',
					alt_allele: variant.altAllele || 'T',
					include_references: true,
				}),
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || 'Failed to fetch explanation');
			}

			const data = await response.json();

			explanation = data.explanation;
			context = data.context;
			cached = data.cached;
			responseTime = Math.round(data.response_time / 1000000); // Convert nanoseconds to ms
			tokensUsed = data.tokens_used || 0;
			costUSD = data.cost_usd || 0;
			quality = data.quality || 0;

		} catch (err) {
			error = err.message;
			console.error('Error fetching explanation:', err);
		} finally {
			loading = false;
		}
	}

	function copyToClipboard() {
		if (!explanation) return;

		navigator.clipboard.writeText(explanation).then(() => {
			copied = true;
			setTimeout(() => {
				copied = false;
			}, 2000);
		}).catch(err => {
			console.error('Failed to copy:', err);
		});
	}

	function handleClose() {
		onClose();
	}

	function handleBackdropClick(e) {
		if (e.target === e.currentTarget) {
			handleClose();
		}
	}
</script>

{#if isOpen}
	<div class="modal-backdrop" on:click={handleBackdropClick}>
		<div class="modal-container">
			<!-- Header -->
			<div class="modal-header">
				<h2>AI Variant Explanation</h2>
				<button class="close-button" on:click={handleClose}>×</button>
			</div>

			<!-- Variant info -->
			{#if variant}
				<div class="variant-info">
					<span class="variant-gene">{variant.gene}</span>
					<span class="variant-separator">·</span>
					<span class="variant-name">{variant.variant}</span>
					<span class="variant-separator">·</span>
					<span class="variant-position">{variant.chromosome}:{variant.position}</span>
				</div>
			{/if}

			<!-- Content -->
			<div class="modal-content">
				{#if loading}
					<div class="loading-state">
						<div class="spinner"></div>
						<p>Analyzing variant with GPT-4...</p>
						<p class="loading-subtitle">Fetching data from ClinVar, COSMIC, gnomAD, and PubMed</p>
					</div>
				{:else if error}
					<div class="error-state">
						<div class="error-icon">⚠️</div>
						<h3>Error</h3>
						<p>{error}</p>
						<button class="retry-button" on:click={fetchExplanation}>Retry</button>
					</div>
				{:else if explanation}
					<div class="explanation-container">
						<!-- Explanation text -->
						<div class="explanation-text">
							{explanation}
						</div>

						<!-- Context data -->
						{#if context}
							<div class="context-section">
								<h3>Data Sources</h3>
								<div class="context-grid">
									{#if context.ClinVar}
										<div class="context-item">
											<strong>ClinVar:</strong>
											<span class:found={context.ClinVar.found}>
												{context.ClinVar.found
													? `${context.ClinVar.pathogenicity} (${context.ClinVar.review_status})`
													: 'No data'}
											</span>
										</div>
									{/if}
									{#if context.COSMIC}
										<div class="context-item">
											<strong>COSMIC:</strong>
											<span class:found={context.COSMIC.found}>
												{context.COSMIC.found
													? `${context.COSMIC.frequency} samples${context.COSMIC.is_hotspot ? ' (Hotspot)' : ''}`
													: 'No data'}
											</span>
										</div>
									{/if}
									{#if context.GnomAD}
										<div class="context-item">
											<strong>gnomAD:</strong>
											<span class:found={context.GnomAD.found}>
												{context.GnomAD.found
													? `AF = ${context.GnomAD.allele_frequency.toExponential(2)}`
													: 'No data'}
											</span>
										</div>
									{/if}
									{#if context.PubMed}
										<div class="context-item">
											<strong>PubMed:</strong>
											<span class:found={context.PubMed.found}>
												{context.PubMed.found
													? `${context.PubMed.total_count} publications`
													: 'No data'}
											</span>
										</div>
									{/if}
								</div>
							</div>
						{/if}

						<!-- Metadata -->
						<div class="metadata-section">
							<div class="metadata-item">
								<span class="metadata-label">Response Time:</span>
								<span class="metadata-value">{responseTime}ms</span>
							</div>
							<div class="metadata-item">
								<span class="metadata-label">Cache:</span>
								<span class="metadata-value cache-badge" class:cached={cached}>
									{cached ? 'HIT' : 'MISS'}
								</span>
							</div>
							{#if !cached}
								<div class="metadata-item">
									<span class="metadata-label">Tokens:</span>
									<span class="metadata-value">{tokensUsed}</span>
								</div>
								<div class="metadata-item">
									<span class="metadata-label">Cost:</span>
									<span class="metadata-value">${costUSD.toFixed(4)}</span>
								</div>
							{/if}
							<div class="metadata-item">
								<span class="metadata-label">Quality:</span>
								<span class="metadata-value">{(quality * 100).toFixed(0)}%</span>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="modal-footer">
				<button class="copy-button" on:click={copyToClipboard} disabled={!explanation}>
					{copied ? 'Copied!' : 'Copy Explanation'}
				</button>
				<button class="close-footer-button" on:click={handleClose}>Close</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		animation: fadeIn 0.2s ease-out;
	}

	@keyframes fadeIn {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	.modal-container {
		background: #1a1a2e;
		border-radius: 12px;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
		max-width: 800px;
		width: 90%;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		animation: slideUp 0.3s ease-out;
		border: 1px solid rgba(100, 255, 218, 0.2);
	}

	@keyframes slideUp {
		from {
			transform: translateY(20px);
			opacity: 0;
		}
		to {
			transform: translateY(0);
			opacity: 1;
		}
	}

	.modal-header {
		padding: 24px;
		border-bottom: 1px solid rgba(100, 255, 218, 0.2);
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.modal-header h2 {
		margin: 0;
		color: #64ffda;
		font-size: 24px;
		font-weight: 600;
	}

	.close-button {
		background: none;
		border: none;
		color: #ccd6f6;
		font-size: 36px;
		cursor: pointer;
		padding: 0;
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		transition: all 0.2s;
	}

	.close-button:hover {
		background: rgba(100, 255, 218, 0.1);
		color: #64ffda;
	}

	.variant-info {
		padding: 16px 24px;
		background: rgba(100, 255, 218, 0.05);
		border-bottom: 1px solid rgba(100, 255, 218, 0.2);
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.variant-gene {
		color: #64ffda;
		font-weight: 700;
		font-size: 18px;
	}

	.variant-separator {
		color: #8892b0;
	}

	.variant-name {
		color: #ccd6f6;
		font-weight: 600;
		font-size: 16px;
	}

	.variant-position {
		color: #8892b0;
		font-size: 14px;
	}

	.modal-content {
		padding: 24px;
		overflow-y: auto;
		flex: 1;
		min-height: 300px;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 300px;
		gap: 16px;
	}

	.spinner {
		width: 48px;
		height: 48px;
		border: 4px solid rgba(100, 255, 218, 0.2);
		border-top-color: #64ffda;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.loading-state p {
		color: #ccd6f6;
		margin: 0;
	}

	.loading-subtitle {
		color: #8892b0 !important;
		font-size: 14px;
	}

	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 300px;
		gap: 16px;
	}

	.error-icon {
		font-size: 48px;
	}

	.error-state h3 {
		color: #ff6b6b;
		margin: 0;
	}

	.error-state p {
		color: #ccd6f6;
		margin: 0;
		text-align: center;
	}

	.retry-button {
		background: #64ffda;
		color: #0a192f;
		border: none;
		padding: 10px 20px;
		border-radius: 6px;
		cursor: pointer;
		font-weight: 600;
		transition: all 0.2s;
	}

	.retry-button:hover {
		background: #52ccb0;
		transform: translateY(-2px);
	}

	.explanation-container {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.explanation-text {
		color: #ccd6f6;
		line-height: 1.8;
		font-size: 16px;
		white-space: pre-wrap;
		background: rgba(100, 255, 218, 0.03);
		padding: 20px;
		border-radius: 8px;
		border-left: 4px solid #64ffda;
	}

	.context-section {
		background: rgba(100, 255, 218, 0.05);
		padding: 20px;
		border-radius: 8px;
	}

	.context-section h3 {
		color: #64ffda;
		margin: 0 0 16px 0;
		font-size: 18px;
	}

	.context-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 12px;
	}

	.context-item {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.context-item strong {
		color: #8892b0;
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.context-item span {
		color: #666;
		font-size: 14px;
	}

	.context-item span.found {
		color: #ccd6f6;
	}

	.metadata-section {
		display: flex;
		flex-wrap: wrap;
		gap: 16px;
		padding: 16px;
		background: rgba(0, 0, 0, 0.2);
		border-radius: 8px;
	}

	.metadata-item {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.metadata-label {
		color: #8892b0;
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.metadata-value {
		color: #ccd6f6;
		font-weight: 600;
		font-size: 14px;
	}

	.cache-badge {
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 12px;
	}

	.cache-badge.cached {
		background: rgba(100, 255, 218, 0.2);
		color: #64ffda;
	}

	.cache-badge:not(.cached) {
		background: rgba(255, 107, 107, 0.2);
		color: #ff6b6b;
	}

	.modal-footer {
		padding: 20px 24px;
		border-top: 1px solid rgba(100, 255, 218, 0.2);
		display: flex;
		justify-content: flex-end;
		gap: 12px;
	}

	.copy-button {
		background: #64ffda;
		color: #0a192f;
		border: none;
		padding: 10px 20px;
		border-radius: 6px;
		cursor: pointer;
		font-weight: 600;
		transition: all 0.2s;
	}

	.copy-button:hover:not(:disabled) {
		background: #52ccb0;
		transform: translateY(-2px);
	}

	.copy-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.close-footer-button {
		background: transparent;
		color: #8892b0;
		border: 1px solid #8892b0;
		padding: 10px 20px;
		border-radius: 6px;
		cursor: pointer;
		font-weight: 600;
		transition: all 0.2s;
	}

	.close-footer-button:hover {
		border-color: #64ffda;
		color: #64ffda;
	}
</style>
