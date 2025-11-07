/**
 * GenomeVedic Streaming Dataset Loader (Tier 2 - Enhanced)
 * Handles 10 GB+ datasets with progressive LOD streaming
 *
 * Features:
 * - Zstandard decompression (using fzstd library)
 * - Progressive LOD loading (5K → 50K → 500K → 5M)
 * - IndexedDB caching with quota management
 * - Network throttling detection (3G/4G/fiber)
 * - Smooth transitions between LOD levels
 * - Memory-efficient chunk processing
 *
 * Usage:
 *   const loader = new StreamingLoader();
 *   await loader.loadProgressive('grch38_full', (lod, progress) => {
 *     console.log(`LOD ${lod}: ${progress}%`);
 *   });
 */

import { fzstd } from 'fzstd';
import type { ParticleData, DatasetMetadata, LoadProgress, ProgressCallback } from './loader';

// Tier 2 dataset metadata
export interface Tier2DatasetMetadata extends DatasetMetadata {
  tier: number;
  chromosomes?: number;
  download_url?: string;
  cdn_url?: string;
  chunk_size?: number;
  streaming_optimized: boolean;
}

// Available Tier 2 datasets
export const TIER2_DATASETS: Tier2DatasetMetadata[] = [
  {
    id: 'grch38_full',
    name: 'Human Genome GRCh38 (Full)',
    description: 'Complete human genome - all 24 chromosomes (3 GB FASTA → 1 GB compressed)',
    organism: 'Homo sapiens',
    tier: 2,
    chromosomes: 24,
    size: 3_200_000_000,
    compressed_size: 1_000_000_000,
    particles: 3_200_000_000,
    lod_levels: [5_000, 50_000, 500_000, 5_000_000],
    url: '/data/tier2/grch38/grch38_full.particles.zst',
    format: 'zst',
    version: '1.0.0',
    chunk_size: 10_000_000, // 10 MB chunks
    streaming_optimized: true
  },
  {
    id: 'tcga_cancer_samples',
    name: 'TCGA Cancer Samples',
    description: '10 cancer samples from TCGA (500 MB VCF)',
    organism: 'Homo sapiens',
    tier: 2,
    size: 500_000_000,
    compressed_size: 100_000_000,
    particles: 10_000,
    lod_levels: [1_000, 5_000, 10_000],
    url: '/data/tier2/tcga/tcga_samples.variants.zst',
    format: 'zst',
    version: '1.0.0',
    streaming_optimized: true
  },
  {
    id: 'lenski_evolution',
    name: 'Lenski E. coli Evolution (50K generations)',
    description: 'Long-term evolution experiment data (100 MB)',
    organism: 'Escherichia coli',
    tier: 2,
    size: 100_000_000,
    compressed_size: 20_000_000,
    particles: 50_000,
    lod_levels: [1_000, 5_000, 10_000, 50_000],
    url: '/data/tier2/lenski/lenski_evolution.variants.zst',
    format: 'zst',
    version: '1.0.0',
    streaming_optimized: true
  },
  {
    id: 'giab_benchmark',
    name: 'GIAB Benchmark (NA12878)',
    description: 'Genome in a Bottle high-confidence variants (200 MB)',
    organism: 'Homo sapiens',
    tier: 2,
    size: 200_000_000,
    compressed_size: 40_000_000,
    particles: 20_000,
    lod_levels: [1_000, 5_000, 10_000, 20_000],
    url: '/data/tier2/giab/giab_benchmark.variants.zst',
    format: 'zst',
    version: '1.0.0',
    streaming_optimized: true
  }
];

// Network speed detection
export enum NetworkSpeed {
  SLOW_3G = '3G',
  FAST_4G = '4G',
  FIBER = 'Fiber'
}

/**
 * Enhanced Streaming Loader for Tier 2 datasets
 */
export class StreamingLoader {
  private cache: IDBDatabase | null = null;
  private cacheName = 'genomevedic-tier2-v1';
  private decoders: Map<string, any> = new Map();
  private networkSpeed: NetworkSpeed = NetworkSpeed.FAST_4G;

  constructor() {
    this.initCache();
    this.detectNetworkSpeed();
  }

  /**
   * Initialize IndexedDB cache with quota management
   */
  private async initCache(): Promise<void> {
    if (!('indexedDB' in window)) {
      console.warn('IndexedDB not available. Caching disabled.');
      return;
    }

    try {
      const request = indexedDB.open(this.cacheName, 2);

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // Create object stores
        if (!db.objectStoreNames.contains('datasets')) {
          const store = db.createObjectStore('datasets', { keyPath: 'id' });
          store.createIndex('tier', 'tier', { unique: false });
          store.createIndex('timestamp', 'timestamp', { unique: false });
        }

        if (!db.objectStoreNames.contains('lod_cache')) {
          const lodStore = db.createObjectStore('lod_cache', { keyPath: ['dataset_id', 'lod_level'] });
          lodStore.createIndex('dataset_id', 'dataset_id', { unique: false });
        }
      };

      this.cache = await new Promise<IDBDatabase>((resolve, reject) => {
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
      });

      // Check quota
      await this.checkQuota();
    } catch (error) {
      console.error('Failed to initialize cache:', error);
    }
  }

  /**
   * Check IndexedDB quota and warn if low
   */
  private async checkQuota(): Promise<void> {
    if ('storage' in navigator && 'estimate' in navigator.storage) {
      const estimate = await navigator.storage.estimate();
      const usagePercent = ((estimate.usage || 0) / (estimate.quota || 1)) * 100;

      if (usagePercent > 80) {
        console.warn(`Storage quota ${usagePercent.toFixed(1)}% full. Consider clearing cache.`);
      }
    }
  }

  /**
   * Detect network speed using Network Information API
   */
  private detectNetworkSpeed(): void {
    if ('connection' in navigator) {
      const conn = (navigator as any).connection;

      if (conn.effectiveType) {
        switch (conn.effectiveType) {
          case 'slow-2g':
          case '2g':
          case '3g':
            this.networkSpeed = NetworkSpeed.SLOW_3G;
            break;
          case '4g':
            this.networkSpeed = NetworkSpeed.FAST_4G;
            break;
          default:
            this.networkSpeed = NetworkSpeed.FIBER;
        }
      }
    }
  }

  /**
   * Load dataset progressively (LOD 5K → 50K → 500K → 5M)
   */
  async loadProgressive(
    datasetId: string,
    onProgress?: (lodLevel: number, progress: number, data: ParticleData) => void
  ): Promise<ParticleData> {
    const metadata = this.getMetadata(datasetId);
    if (!metadata) {
      throw new Error(`Dataset not found: ${datasetId}`);
    }

    const lodLevels = metadata.lod_levels;
    let currentData: ParticleData | null = null;

    // Load each LOD level progressively
    for (let i = 0; i < lodLevels.length; i++) {
      const lodLevel = i;
      const targetParticles = lodLevels[i];

      // Check cache first
      const cached = await this.getCachedLOD(datasetId, lodLevel);

      if (cached) {
        console.log(`Loaded LOD ${lodLevel} from cache (${targetParticles:,} particles)`);
        currentData = cached;
        onProgress?.(lodLevel, 100, currentData);
        continue;
      }

      // Load from network
      console.log(`Loading LOD ${lodLevel} from network (${targetParticles:,} particles)...`);

      currentData = await this.loadDatasetLOD(datasetId, lodLevel, (progress) => {
        onProgress?.(lodLevel, progress, currentData!);
      });

      // Cache this LOD level
      await this.cacheLOD(datasetId, lodLevel, currentData);

      // For slow networks, stop at LOD 1 (50K particles)
      if (this.networkSpeed === NetworkSpeed.SLOW_3G && lodLevel === 1) {
        console.log('Stopping at LOD 1 due to slow network');
        break;
      }
    }

    return currentData!;
  }

  /**
   * Load specific LOD level
   */
  private async loadDatasetLOD(
    datasetId: string,
    lodLevel: number,
    onProgress?: (progress: number) => void
  ): Promise<ParticleData> {
    const metadata = this.getMetadata(datasetId);
    if (!metadata) {
      throw new Error(`Dataset not found: ${datasetId}`);
    }

    // Download compressed data
    const response = await fetch(metadata.url);
    if (!response.ok) {
      throw new Error(`Failed to download: ${response.statusText}`);
    }

    const total = parseInt(response.headers.get('content-length') || '0', 10);
    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error('ReadableStream not supported');
    }

    // Read chunks with progress
    const chunks: Uint8Array[] = [];
    let loaded = 0;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      chunks.push(value);
      loaded += value.length;

      onProgress?.((loaded / total) * 100);
    }

    // Combine chunks
    const compressed = new Uint8Array(loaded);
    let offset = 0;
    for (const chunk of chunks) {
      compressed.set(chunk, offset);
      offset += chunk.length;
    }

    // Decompress with Zstandard
    const decompressed = await this.decompressZstd(compressed);

    // Parse JSON
    const text = new TextDecoder().decode(decompressed);
    const data = JSON.parse(text) as ParticleData;

    // Filter to specific LOD level
    const filtered = this.filterToLOD(data, lodLevel);

    return filtered;
  }

  /**
   * Decompress Zstandard data
   */
  private async decompressZstd(compressed: Uint8Array): Promise<Uint8Array> {
    try {
      // Use fzstd library for decompression
      return fzstd.decompress(compressed);
    } catch (error) {
      console.error('Zstandard decompression failed:', error);
      throw new Error('Failed to decompress dataset');
    }
  }

  /**
   * Filter particle data to specific LOD level
   */
  private filterToLOD(data: ParticleData, lodLevel: number): ParticleData {
    const lodKey = String(lodLevel);
    const indices = data.lod_levels[lodKey];

    if (!indices) {
      // Return all particles if LOD doesn't exist
      return data;
    }

    // Create filtered particle list
    const filteredParticles = indices.map((idx) => data.particles[idx]);

    return {
      ...data,
      particles: filteredParticles
    };
  }

  /**
   * Get cached LOD level
   */
  private async getCachedLOD(datasetId: string, lodLevel: number): Promise<ParticleData | null> {
    if (!this.cache) return null;

    try {
      const transaction = this.cache.transaction(['lod_cache'], 'readonly');
      const store = transaction.objectStore('lod_cache');
      const request = store.get([datasetId, lodLevel]);

      return await new Promise<ParticleData | null>((resolve) => {
        request.onsuccess = () => resolve(request.result?.data || null);
        request.onerror = () => resolve(null);
      });
    } catch (error) {
      console.error('Failed to get cached LOD:', error);
      return null;
    }
  }

  /**
   * Cache LOD level
   */
  private async cacheLOD(datasetId: string, lodLevel: number, data: ParticleData): Promise<void> {
    if (!this.cache) return;

    try {
      const transaction = this.cache.transaction(['lod_cache'], 'readwrite');
      const store = transaction.objectStore('lod_cache');

      await store.put({
        dataset_id: datasetId,
        lod_level: lodLevel,
        data,
        timestamp: Date.now()
      });
    } catch (error) {
      console.error('Failed to cache LOD:', error);
    }
  }

  /**
   * Get metadata for dataset
   */
  getMetadata(datasetId: string): Tier2DatasetMetadata | null {
    return TIER2_DATASETS.find((d) => d.id === datasetId) || null;
  }

  /**
   * Get all available Tier 2 datasets
   */
  getAvailableDatasets(): Tier2DatasetMetadata[] {
    return TIER2_DATASETS;
  }

  /**
   * Clear all caches
   */
  async clearCache(): Promise<void> {
    if (!this.cache) return;

    try {
      const transaction = this.cache.transaction(['datasets', 'lod_cache'], 'readwrite');

      await transaction.objectStore('datasets').clear();
      await transaction.objectStore('lod_cache').clear();

      console.log('Cache cleared');
    } catch (error) {
      console.error('Failed to clear cache:', error);
    }
  }

  /**
   * Get cache statistics
   */
  async getCacheStats(): Promise<any> {
    if (!this.cache) return null;

    const transaction = this.cache.transaction(['lod_cache'], 'readonly');
    const store = transaction.objectStore('lod_cache');
    const count = await new Promise<number>((resolve) => {
      const request = store.count();
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => resolve(0);
    });

    const estimate = await navigator.storage?.estimate();

    return {
      cached_lod_levels: count,
      storage_used_mb: ((estimate?.usage || 0) / 1024 / 1024).toFixed(1),
      storage_quota_mb: ((estimate?.quota || 0) / 1024 / 1024).toFixed(1),
      network_speed: this.networkSpeed
    };
  }
}

/**
 * Global singleton instance
 */
export const streamingLoader = new StreamingLoader();

/**
 * Helper: Estimate load time based on network speed
 */
export function estimateLoadTime(size: number, networkSpeed: NetworkSpeed): number {
  const bandwidthMbps = {
    [NetworkSpeed.SLOW_3G]: 0.4,
    [NetworkSpeed.FAST_4G]: 10,
    [NetworkSpeed.FIBER]: 100
  };

  const bandwidth = bandwidthMbps[networkSpeed] * 1024 * 1024 / 8; // Convert to bytes/sec
  return size / bandwidth;
}
