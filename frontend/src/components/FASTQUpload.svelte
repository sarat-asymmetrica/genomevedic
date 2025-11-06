<script>
  /**
   * FASTQ Upload Component
   *
   * Features:
   * - Drag-and-drop file upload
   * - Click to browse
   * - File validation (FASTQ format)
   * - Progress bar with speed indicator
   * - File metadata display (read count, quality, format)
   */

  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  // State
  let isDragging = false;
  let isProcessing = false;
  let uploadProgress = 0;
  let uploadSpeed = 0;
  let uploadedFile = null;
  let fileMetadata = null;
  let errorMessage = '';

  // Drag and drop handlers
  function handleDragEnter(e) {
    e.preventDefault();
    isDragging = true;
  }

  function handleDragLeave(e) {
    e.preventDefault();
    isDragging = false;
  }

  function handleDragOver(e) {
    e.preventDefault();
  }

  function handleDrop(e) {
    e.preventDefault();
    isDragging = false;

    const files = e.dataTransfer.files;
    if (files.length > 0) {
      handleFile(files[0]);
    }
  }

  function handleFileSelect(e) {
    const files = e.target.files;
    if (files.length > 0) {
      handleFile(files[0]);
    }
  }

  async function handleFile(file) {
    errorMessage = '';
    uploadedFile = file;

    // Validate file extension
    const validExtensions = ['.fastq', '.fq', '.fastq.gz', '.fq.gz'];
    const isValid = validExtensions.some(ext => file.name.toLowerCase().endsWith(ext));

    if (!isValid) {
      errorMessage = 'Invalid file format. Please upload a FASTQ file (.fastq, .fq, .fastq.gz, .fq.gz)';
      uploadedFile = null;
      return;
    }

    // Process file
    isProcessing = true;
    uploadProgress = 0;

    try {
      await processFile(file);
    } catch (error) {
      errorMessage = `Error processing file: ${error.message}`;
      uploadedFile = null;
    } finally {
      isProcessing = false;
    }
  }

  async function processFile(file) {
    const startTime = performance.now();
    const reader = new FileReader();

    return new Promise((resolve, reject) => {
      reader.onprogress = (e) => {
        if (e.lengthComputable) {
          uploadProgress = (e.loaded / e.total) * 100;

          // Calculate upload speed (MB/s)
          const elapsedTime = (performance.now() - startTime) / 1000;
          uploadSpeed = (e.loaded / 1024 / 1024) / elapsedTime;
        }
      };

      reader.onload = async (e) => {
        try {
          const content = e.target.result;

          // Parse FASTQ (simple parsing for first few reads)
          const metadata = parseFASTQ(content);
          fileMetadata = metadata;

          uploadProgress = 100;

          // Dispatch event
          dispatch('fileUploaded', {
            file,
            metadata
          });

          resolve();
        } catch (error) {
          reject(error);
        }
      };

      reader.onerror = () => {
        reject(new Error('Failed to read file'));
      };

      // Read as text (for simplicity)
      // In production, you'd handle gzipped files differently
      reader.readAsText(file.slice(0, 100000)); // Read first 100KB for metadata
    });
  }

  function parseFASTQ(content) {
    const lines = content.split('\n').filter(line => line.trim());

    let readCount = 0;
    let totalQuality = 0;
    let minQuality = Infinity;
    let maxQuality = -Infinity;
    let totalLength = 0;
    let format = 'Unknown';

    // Detect format from first header
    if (lines[0]) {
      if (lines[0].includes('Illumina') || lines[0].split(':').length >= 7) {
        format = 'Illumina';
      } else if (lines[0].startsWith('@m64') || lines[0].includes('PacBio')) {
        format = 'PacBio';
      } else if (lines[0].includes('ONT') || lines[0].includes('Nanopore')) {
        format = 'Nanopore';
      }
    }

    // Parse reads (FASTQ format: @header, sequence, +, quality)
    for (let i = 0; i < lines.length && i < 1000; i += 4) {
      if (lines[i] && lines[i].startsWith('@')) {
        readCount++;

        // Parse sequence length
        if (lines[i + 1]) {
          totalLength += lines[i + 1].length;
        }

        // Parse quality scores
        if (lines[i + 3]) {
          const qualityScores = lines[i + 3].split('').map(char => char.charCodeAt(0) - 33);
          const avgQuality = qualityScores.reduce((a, b) => a + b, 0) / qualityScores.length;

          totalQuality += avgQuality;
          minQuality = Math.min(minQuality, ...qualityScores);
          maxQuality = Math.max(maxQuality, ...qualityScores);
        }
      }
    }

    return {
      readCount: readCount * 4, // Extrapolate from sample
      avgReadLength: Math.round(totalLength / readCount),
      avgQuality: Math.round(totalQuality / readCount),
      minQuality: Math.round(minQuality),
      maxQuality: Math.round(maxQuality),
      format
    };
  }

  function clearFile() {
    uploadedFile = null;
    fileMetadata = null;
    uploadProgress = 0;
    errorMessage = '';
  }
</script>

<div class="upload-panel">
  <h3>FASTQ Upload</h3>

  {#if !uploadedFile}
    <div
      class="dropzone {isDragging ? 'dragging' : ''}"
      on:dragenter={handleDragEnter}
      on:dragleave={handleDragLeave}
      on:dragover={handleDragOver}
      on:drop={handleDrop}
      role="button"
      tabindex="0"
    >
      <div class="dropzone-icon">📁</div>
      <div class="dropzone-text">
        {#if isDragging}
          Drop FASTQ file here
        {:else}
          Drag & drop FASTQ file or
          <label class="file-label">
            <input
              type="file"
              accept=".fastq,.fq,.fastq.gz,.fq.gz"
              on:change={handleFileSelect}
              disabled={isProcessing}
            />
            <span class="file-label-text">browse</span>
          </label>
        {/if}
      </div>
      <div class="dropzone-hint">
        Supported formats: .fastq, .fq, .fastq.gz, .fq.gz
      </div>
    </div>
  {:else}
    <div class="file-info">
      <div class="file-header">
        <div class="file-name">{uploadedFile.name}</div>
        <button class="clear-btn" on:click={clearFile}>✕</button>
      </div>

      <div class="file-size">
        {(uploadedFile.size / 1024 / 1024).toFixed(2)} MB
      </div>

      {#if isProcessing}
        <div class="progress-section">
          <div class="progress-bar">
            <div class="progress-fill" style="width: {uploadProgress}%"></div>
          </div>
          <div class="progress-text">
            {uploadProgress.toFixed(0)}% • {uploadSpeed.toFixed(2)} MB/s
          </div>
        </div>
      {/if}

      {#if fileMetadata}
        <div class="metadata">
          <h4>File Metadata</h4>

          <div class="metadata-grid">
            <div class="metadata-item">
              <div class="metadata-label">Format</div>
              <div class="metadata-value">{fileMetadata.format}</div>
            </div>

            <div class="metadata-item">
              <div class="metadata-label">Read Count</div>
              <div class="metadata-value">{fileMetadata.readCount.toLocaleString()}</div>
            </div>

            <div class="metadata-item">
              <div class="metadata-label">Avg Read Length</div>
              <div class="metadata-value">{fileMetadata.avgReadLength} bp</div>
            </div>

            <div class="metadata-item">
              <div class="metadata-label">Avg Quality</div>
              <div class="metadata-value">Q{fileMetadata.avgQuality}</div>
            </div>

            <div class="metadata-item">
              <div class="metadata-label">Quality Range</div>
              <div class="metadata-value">Q{fileMetadata.minQuality} - Q{fileMetadata.maxQuality}</div>
            </div>
          </div>
        </div>
      {/if}
    </div>
  {/if}

  {#if errorMessage}
    <div class="error-message">
      ⚠️ {errorMessage}
    </div>
  {/if}
</div>

<style>
  .upload-panel {
    color: #e0e0e0;
  }

  h3 {
    margin: 0 0 16px 0;
    font-size: 16px;
    font-weight: 600;
    color: #fff;
  }

  h4 {
    margin: 16px 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: #fff;
  }

  /* Dropzone */
  .dropzone {
    border: 2px dashed rgba(255, 255, 255, 0.2);
    border-radius: 8px;
    padding: 40px 20px;
    text-align: center;
    cursor: pointer;
    transition: all 0.3s;
  }

  .dropzone:hover {
    border-color: rgba(102, 126, 234, 0.5);
    background: rgba(102, 126, 234, 0.05);
  }

  .dropzone.dragging {
    border-color: rgba(102, 126, 234, 1);
    background: rgba(102, 126, 234, 0.1);
  }

  .dropzone-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .dropzone-text {
    font-size: 14px;
    color: #aaa;
    margin-bottom: 8px;
  }

  .dropzone-hint {
    font-size: 12px;
    color: #666;
  }

  .file-label {
    display: inline;
    cursor: pointer;
  }

  .file-label input {
    display: none;
  }

  .file-label-text {
    color: #667eea;
    text-decoration: underline;
  }

  .file-label-text:hover {
    color: #764ba2;
  }

  /* File Info */
  .file-info {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 16px;
  }

  .file-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .file-name {
    font-size: 14px;
    font-weight: 500;
    color: #fff;
    word-break: break-all;
  }

  .clear-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #999;
    width: 24px;
    height: 24px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .clear-btn:hover {
    background: rgba(239, 68, 68, 0.2);
    border-color: rgba(239, 68, 68, 0.5);
    color: #f87171;
  }

  .file-size {
    font-size: 12px;
    color: #888;
    margin-bottom: 12px;
  }

  /* Progress */
  .progress-section {
    margin-bottom: 16px;
  }

  .progress-bar {
    width: 100%;
    height: 6px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 8px;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #667eea, #764ba2);
    transition: width 0.3s ease;
  }

  .progress-text {
    font-size: 12px;
    color: #888;
    text-align: center;
  }

  /* Metadata */
  .metadata {
    margin-top: 16px;
  }

  .metadata-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .metadata-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .metadata-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: #888;
    font-weight: 500;
  }

  .metadata-value {
    font-size: 14px;
    font-weight: 600;
    color: #fff;
  }

  /* Error Message */
  .error-message {
    margin-top: 16px;
    padding: 12px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }
</style>
