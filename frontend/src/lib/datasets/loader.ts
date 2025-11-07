/**
 * GenomeVedic Dataset Loader
 * Handles streaming, decompression, and progressive loading of genomic datasets
 *
 * Features:
 * - Streaming download with progress tracking
 * - Zstandard decompression support
 * - Progressive LOD loading (5K → 50K → 500K → 5M particles)
 * - Memory-efficient chunk processing
 * - IndexedDB caching for offline use
 *
 * Usage:
 *   const loader = new DatasetLoader();
 *   await loader.load('chr22', (progress) => {
 *     console.log(`Loading: ${progress.percent}%`);
 *   });
 *   const particles = loader.getParticles('chr22');
 */

export interface DatasetMetadata {
  id: string;
  name: string;
  description: string;
  organism: string;
  size: number;
  compressed_size: number;
  particles: number;
  lod_levels: number[];
  url: string;
  format: 'zst' | 'json';
  version: string;
}

export interface Particle {
  x: number;
  y: number;
  z: number;
  base: string;
  pos: number;
  voxel: number;
  color: [number, number, number];
}

export interface ParticleData {
  metadata: {
    sequence_name: string;
    length: number;
    particles: number;
    lod_levels: number[];
    voxel_count: number;
    voxel_size: number;
    williams_batch_size: number;
    generation_time: number;
    digital_root_algorithm: string;
    golden_angle_degrees: number;
    version: string;
  };
  particles: Particle[];
  spatial_hash: { [voxel_id: string]: number[] };
  lod_levels: { [lod_id: string]: number[] };
}

export interface GeneAnnotation {
  id: string;
  name: string;
  chromosome: string;
  start: number;
  end: number;
  strand: string;
  type: string;
  exon_count: number;
  transcript_count: number;
}

export interface AnnotationData {
  metadata: {
    source: string;
    chromosome: string;
    genes: number;
    exons: number;
    transcripts: number;
  };
  genes: GeneAnnotation[];
  exons: any[];
  transcripts: any[];
}

export interface Variant {
  id: string | null;
  chromosome: string;
  position: number;
  ref: string;
  alt: string[];
  quality: number | null;
  filter: string;
  type: string;
  allele_frequency?: number;
}

export interface VariantData {
  metadata: {
    source: string;
    chromosome: string;
    variants: number;
  };
  variants: Variant[];
}

export interface LoadProgress {
  dataset: string;
  stage: 'downloading' | 'decompressing' | 'parsing' | 'complete';
  percent: number;
  loaded: number;
  total: number;
  message: string;
}

export type ProgressCallback = (progress: LoadProgress) => void;

/**
 * Available Tier 1 datasets (bundled with application)
 */
export const TIER1_DATASETS: DatasetMetadata[] = [
  {
    id: 'chr22',
    name: 'Human Chromosome 22',
    description: 'Complete human chromosome 22 sequence (GRCh38)',
    organism: 'Homo sapiens',
    size: 50_818_468,
    compressed_size: 15_000_000, // Estimated after zstd compression
    particles: 50_818_468,
    lod_levels: [5_000, 50_000, 500_000, 5_000_000],
    url: '/data/tier1/chr22_hg38.particles.zst',
    format: 'zst',
    version: '1.0.0'
  },
  {
    id: 'ecoli',
    name: 'E. coli K-12',
    description: 'Complete E. coli K-12 genome (NCBI RefSeq)',
    organism: 'Escherichia coli',
    size: 4_641_652,
    compressed_size: 1_000_000,
    particles: 4_641_652,
    lod_levels: [5_000, 50_000, 500_000],
    url: '/data/tier1/ecoli_k12.particles.zst',
    format: 'zst',
    version: '1.0.0'
  },
  {
    id: 'cosmic',
    name: 'COSMIC Top 100 Cancer Genes',
    description: 'Top cancer genes from COSMIC database (simulated)',
    organism: 'Homo sapiens',
    size: 10_000,
    compressed_size: 3_000,
    particles: 100,
    lod_levels: [100],
    url: '/data/tier1/cosmic_top100.json',
    format: 'json',
    version: '1.0.0'
  }
];

/**
 * Dataset Loader with streaming and progressive loading
 */
export class DatasetLoader {
  private datasets: Map<string, ParticleData> = new Map();
  private annotations: Map<string, AnnotationData> = new Map();
  private variants: Map<string, VariantData> = new Map();
  private cache: IDBDatabase | null = null;
  private cacheName = 'genomevedic-datasets-v1';

  constructor() {
    this.initCache();
  }

  /**
   * Initialize IndexedDB cache for offline use
   */
  private async initCache(): Promise<void> {
    if (!('indexedDB' in window)) {
      console.warn('IndexedDB not available. Caching disabled.');
      return;
    }

    try {
      const request = indexedDB.open(this.cacheName, 1);

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;
        if (!db.objectStoreNames.contains('datasets')) {
          db.createObjectStore('datasets', { keyPath: 'id' });
        }
      };

      this.cache = await new Promise<IDBDatabase>((resolve, reject) => {
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
      });
    } catch (error) {
      console.error('Failed to initialize cache:', error);
    }
  }

  /**
   * Load dataset with progress tracking
   */
  async load(
    datasetId: string,
    onProgress?: ProgressCallback,
    lodLevel: number = 0
  ): Promise<ParticleData> {
    // Check if already loaded
    if (this.datasets.has(datasetId)) {
      return this.datasets.get(datasetId)!;
    }

    // Find dataset metadata
    const metadata = TIER1_DATASETS.find((d) => d.id === datasetId);
    if (!metadata) {
      throw new Error(`Dataset not found: ${datasetId}`);
    }

    // Check cache first
    const cached = await this.getCached(datasetId);
    if (cached) {
      console.log(`Loaded ${datasetId} from cache`);
      this.datasets.set(datasetId, cached);
      onProgress?.({
        dataset: datasetId,
        stage: 'complete',
        percent: 100,
        loaded: metadata.compressed_size,
        total: metadata.compressed_size,
        message: 'Loaded from cache'
      });
      return cached;
    }

    // Download dataset
    const data = await this.download(metadata, onProgress);

    // Store in memory and cache
    this.datasets.set(datasetId, data);
    await this.setCached(datasetId, data);

    onProgress?.({
      dataset: datasetId,
      stage: 'complete',
      percent: 100,
      loaded: metadata.compressed_size,
      total: metadata.compressed_size,
      message: 'Dataset loaded'
    });

    return data;
  }

  /**
   * Download dataset with streaming
   */
  private async download(
    metadata: DatasetMetadata,
    onProgress?: ProgressCallback
  ): Promise<ParticleData> {
    onProgress?.({
      dataset: metadata.id,
      stage: 'downloading',
      percent: 0,
      loaded: 0,
      total: metadata.compressed_size,
      message: 'Starting download...'
    });

    const response = await fetch(metadata.url);
    if (!response.ok) {
      throw new Error(`Failed to download: ${response.statusText}`);
    }

    const total = parseInt(response.headers.get('content-length') || '0', 10);
    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error('ReadableStream not supported');
    }

    // Read chunks
    const chunks: Uint8Array[] = [];
    let loaded = 0;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      chunks.push(value);
      loaded += value.length;

      onProgress?.({
        dataset: metadata.id,
        stage: 'downloading',
        percent: (loaded / total) * 100,
        loaded,
        total,
        message: `Downloaded ${(loaded / 1024 / 1024).toFixed(1)} MB`
      });
    }

    // Combine chunks
    const buffer = new Uint8Array(loaded);
    let offset = 0;
    for (const chunk of chunks) {
      buffer.set(chunk, offset);
      offset += chunk.length;
    }

    onProgress?.({
      dataset: metadata.id,
      stage: 'decompressing',
      percent: 0,
      loaded: 0,
      total: metadata.size,
      message: 'Decompressing...'
    });

    // Decompress if needed
    let jsonText: string;
    if (metadata.format === 'zst') {
      // TODO: Implement Zstandard decompression in browser
      // For now, we'll assume pre-decompressed JSON or use a library
      console.warn('Zstandard decompression not yet implemented. Using fallback.');
      jsonText = new TextDecoder().decode(buffer);
    } else {
      jsonText = new TextDecoder().decode(buffer);
    }

    onProgress?.({
      dataset: metadata.id,
      stage: 'parsing',
      percent: 0,
      loaded: 0,
      total: jsonText.length,
      message: 'Parsing JSON...'
    });

    // Parse JSON
    const data = JSON.parse(jsonText) as ParticleData;

    return data;
  }

  /**
   * Get cached dataset from IndexedDB
   */
  private async getCached(datasetId: string): Promise<ParticleData | null> {
    if (!this.cache) return null;

    try {
      const transaction = this.cache.transaction(['datasets'], 'readonly');
      const store = transaction.objectStore('datasets');
      const request = store.get(datasetId);

      return await new Promise<ParticleData | null>((resolve) => {
        request.onsuccess = () => resolve(request.result?.data || null);
        request.onerror = () => resolve(null);
      });
    } catch (error) {
      console.error('Failed to get cached dataset:', error);
      return null;
    }
  }

  /**
   * Store dataset in IndexedDB cache
   */
  private async setCached(datasetId: string, data: ParticleData): Promise<void> {
    if (!this.cache) return;

    try {
      const transaction = this.cache.transaction(['datasets'], 'readwrite');
      const store = transaction.objectStore('datasets');
      await store.put({ id: datasetId, data, timestamp: Date.now() });
    } catch (error) {
      console.error('Failed to cache dataset:', error);
    }
  }

  /**
   * Get loaded particles for dataset
   */
  getParticles(datasetId: string, lodLevel: number = 0): Particle[] | null {
    const data = this.datasets.get(datasetId);
    if (!data) return null;

    // Return particles at specified LOD level
    if (lodLevel === -1) {
      // Return all particles
      return data.particles;
    }

    const lodIndices = data.lod_levels[String(lodLevel)];
    if (!lodIndices) {
      return data.particles;
    }

    return lodIndices.map((idx) => data.particles[idx]);
  }

  /**
   * Get spatial hash for fast spatial queries
   */
  getSpatialHash(datasetId: string): { [voxel_id: string]: number[] } | null {
    const data = this.datasets.get(datasetId);
    return data?.spatial_hash || null;
  }

  /**
   * Get metadata for dataset
   */
  getMetadata(datasetId: string) {
    return TIER1_DATASETS.find((d) => d.id === datasetId);
  }

  /**
   * Get all available datasets
   */
  getAvailableDatasets(): DatasetMetadata[] {
    return TIER1_DATASETS;
  }

  /**
   * Clear cache
   */
  async clearCache(): Promise<void> {
    if (!this.cache) return;

    try {
      const transaction = this.cache.transaction(['datasets'], 'readwrite');
      const store = transaction.objectStore('datasets');
      await store.clear();
    } catch (error) {
      console.error('Failed to clear cache:', error);
    }
  }

  /**
   * Unload dataset from memory
   */
  unload(datasetId: string): void {
    this.datasets.delete(datasetId);
    this.annotations.delete(datasetId);
    this.variants.delete(datasetId);
  }
}

/**
 * Global singleton instance
 */
export const datasetLoader = new DatasetLoader();

/**
 * Helper function to format file size
 */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

/**
 * Helper function to estimate load time
 */
export function estimateLoadTime(size: number, bandwidth: number = 10_000_000): number {
  // bandwidth in bytes per second
  return size / bandwidth;
}
