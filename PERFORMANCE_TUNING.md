# Performance Tuning Guide - GenomeVedic.ai
## Optimization Strategies for Billion-Scale Rendering

**Last Updated:** 2025-11-06 (Post Red Team Audit)
**Target:** 60fps sustained, <2GB memory, <5s load time

---

## 🎯 PERFORMANCE BUDGET (Frame Time: 16.67ms @ 60fps)

### **Tier 1: Critical Path (Must Optimize)**

```mathematical
CRITICAL_PATH[CP] = {
  Frustum_culling: 0.5ms (filter 5M voxels to 50K visible),
  GPU_upload: 1.5ms (transfer 50K batch metadata),
  Rendering: 7.0ms (GPU draw calls for visible batches),
  Total: 9.0ms per frame (leaves 7.67ms slack for variance)
}

OPTIMIZATION_PRIORITY[OP] = {
  1_Rendering: 7.0ms (42% of budget, HIGHEST priority),
  2_GPU_upload: 1.5ms (9% of budget, MEDIUM priority),
  3_Frustum_culling: 0.5ms (3% of budget, LOW priority)
}
```

### **Tier 2: Non-Critical (Optimize If Needed)**

```mathematical
NON_CRITICAL[NC] = {
  Camera_update: 0.1ms (quaternion math, already fast),
  UI_overlay: 0.5ms (FPS counter, tooltips),
  User_input: 0.1ms (mouse/keyboard events)
}
```

---

## 🚀 OPTIMIZATION #1: GPU Rendering (7.0ms → 5.0ms target)

### **Current Bottleneck:**
50K visible batches × 0.14ms per batch = 7.0ms

### **Strategy 1: GPU Instancing (Already Planned)**

```javascript
// GOOD: Single draw call per batch (600 particles instanced)
gl.drawElementsInstanced(
    gl.TRIANGLES,
    6,  // 2 triangles per particle
    gl.UNSIGNED_SHORT,
    0,
    600  // Instance count
);

// Total: 50K draw calls (acceptable, but can improve)
```

**Optimization:** Merge batches with same LOD level

```javascript
// BETTER: Batch batches (mega-batches)
// Group adjacent batches into mega-batches of 10K particles each
// Reduces 50K draw calls to 300 mega-batches

const megaBatchSize = 10000;  // 10K particles per mega-batch
const megaBatchCount = 300;   // 50K batches / 10K particles ≈ 300

// Result: 300 draw calls × 0.014ms = 4.2ms (40% FASTER!)
```

**Expected Gain:** 7.0ms → 4.2ms (2.8ms saved)

---

### **Strategy 2: LOD Aggressive Culling**

```javascript
// CURRENT: 3-tier LOD
const LOD_LEVELS = [
    { distance: 100,  particleRatio: 1.0  },  // 100% of particles
    { distance: 500,  particleRatio: 0.5  },  // 50% of particles
    { distance: 2000, particleRatio: 0.1  }   // 10% of particles
];

// OPTIMIZED: 5-tier LOD with exponential falloff
const LOD_LEVELS_OPTIMIZED = [
    { distance: 50,   particleRatio: 1.0   },  // Ultra detail
    { distance: 100,  particleRatio: 0.7   },  // High detail
    { distance: 300,  particleRatio: 0.3   },  // Medium detail
    { distance: 800,  particleRatio: 0.1   },  // Low detail
    { distance: 2000, particleRatio: 0.03  }   // Minimum detail (3% particles)
];

// Effective particles: 50K batches × average 30% = 15K batches
// Result: 15K batches × 0.014ms = 2.1ms (70% FASTER!)
```

**Expected Gain:** 7.0ms → 2.1ms (4.9ms saved)

**Tradeoff:** Visual quality at distance (acceptable for exploration workflow)

---

### **Strategy 3: Geometry Shader Point Sprites (WebGL 2.0)**

```glsl
// CURRENT: 2 triangles per particle (6 vertices)
// OPTIMIZED: Point sprites (1 vertex per particle)

// Vertex shader
#version 300 es
in vec3 a_position;
in vec4 a_color;
out vec4 v_color;

void main() {
    gl_Position = u_viewProjectionMatrix * vec4(a_position, 1.0);
    gl_PointSize = 5.0;  // Particle size in pixels
    v_color = a_color;
}

// Fragment shader
#version 300 es
precision mediump float;
in vec4 v_color;
out vec4 fragColor;

void main() {
    vec2 coord = gl_PointCoord - vec2(0.5);
    if (length(coord) > 0.5) discard;  // Circular particles
    fragColor = v_color;
}
```

**Expected Gain:** 6 vertices → 1 vertex = 6× reduction in vertex processing
- GPU time: 7.0ms → 1.2ms (83% FASTER!)

**Tradeoff:** Point sprites have size limits (typically 64 pixels max)

---

## 🧮 OPTIMIZATION #2: Frustum Culling (0.5ms → 0.2ms target)

### **Current Bottleneck:**
Test 5M voxels against 6 frustum planes

### **Strategy 1: Spatial Grid Acceleration**

```go
// CURRENT: Linear search through 5M voxels
for _, voxel := range allVoxels {
    if frustum.TestVoxel(voxel.Bounds) {
        visibleVoxels = append(visibleVoxels, voxel)
    }
}

// OPTIMIZED: Octree with early rejection
type Octree struct {
    Root *OctreeNode
}

func (ot *Octree) QueryFrustum(frustum FrustumPlanes) []Voxel {
    return ot.Root.queryRecursive(frustum)
}

func (node *OctreeNode) queryRecursive(frustum FrustumPlanes) []Voxel {
    // Test node bounds against frustum
    if !frustum.TestAABB(node.Bounds) {
        return nil  // Entire subtree culled!
    }

    // If leaf, return voxels
    if node.IsLeaf {
        return node.Voxels
    }

    // Recurse into children
    var visible []Voxel
    for _, child := range node.Children {
        visible = append(visible, child.queryRecursive(frustum)...)
    }
    return visible
}
```

**Expected Gain:** 5M tests → ~1000 octree node tests (5000× reduction!)
- Culling time: 0.5ms → 0.0001ms (5000× FASTER!)

**Tradeoff:** Octree construction time (one-time cost at startup)

---

### **Strategy 2: SIMD Vectorization (Go Assembly)**

```go
// CURRENT: Scalar frustum test (1 voxel at a time)
func (f FrustumPlanes) TestVoxel(bounds VoxelBounds) bool {
    for _, plane := range f {
        if plane.Distance(bounds.Center) < -bounds.Radius {
            return false  // Outside frustum
        }
    }
    return true
}

// OPTIMIZED: SIMD test (4 voxels at a time using AVX)
// (Requires cgo or Go assembly)
// Expected: 4× throughput improvement
```

**Expected Gain:** 0.5ms → 0.125ms (4× FASTER with SIMD)

**Tradeoff:** Platform-specific code, added complexity

---

## 💾 OPTIMIZATION #3: Memory Bandwidth (1.5ms → 0.8ms target)

### **Current Bottleneck:**
Upload 50K batch metadata to GPU (1.2 MB per frame)

### **Strategy 1: Persistent GPU Buffers**

```javascript
// CURRENT: Upload batch data every frame
gl.bufferData(gl.ARRAY_BUFFER, batchData, gl.DYNAMIC_DRAW);

// OPTIMIZED: Upload once, update only changed batches
gl.bufferData(gl.ARRAY_BUFFER, batchData, gl.STATIC_DRAW);  // Initial upload

// Per frame: Only update visible flag (1 byte per batch)
gl.bufferSubData(gl.ARRAY_BUFFER, offset, visibilityFlags);  // 50K bytes instead of 1.2MB
```

**Expected Gain:** 1.2 MB → 50 KB upload per frame (24× reduction!)
- Upload time: 1.5ms → 0.06ms (25× FASTER!)

---

### **Strategy 2: Compression (Color Palette)**

```javascript
// CURRENT: RGBA color per particle (4 bytes)
// A=red, T=blue, G=green, C=yellow (only 4 colors needed!)

// OPTIMIZED: 2-bit palette index
const colorPalette = [
    [1.0, 0.0, 0.0, 1.0],  // Red (A)
    [0.0, 0.0, 1.0, 1.0],  // Blue (T)
    [0.0, 1.0, 0.0, 1.0],  // Green (G)
    [1.0, 1.0, 0.0, 1.0]   // Yellow (C)
];

// Store palette index (2 bits) instead of full RGBA (32 bits)
// Compression: 16× smaller color data!
```

**Expected Gain:** Color data: 4 bytes → 0.25 bytes (16× reduction!)

---

## ⚡ OPTIMIZATION #4: FASTQ Parsing (<5s target)

### **Current Estimate:**
Single-threaded: 15 seconds (200 MB/s × 3GB)

### **Strategy 1: Parallel Parsing**

```go
// OPTIMIZED: Split FASTQ into chunks, parse in parallel
func StreamFASTQParallel(filepath string, numWorkers int) <-chan Sequence {
    ch := make(chan Sequence, 10000)

    // Worker pool
    var wg sync.WaitGroup
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            // Parse chunk assigned to this worker
            parseChunk(filepath, workerID, numWorkers, ch)
        }(i)
    }

    // Close channel when all workers done
    go func() {
        wg.Wait()
        close(ch)
    }()

    return ch
}

// Expected: 4 workers on 4-core CPU → 4× speedup
// 15s / 4 = 3.75s (MEETS <5s target!)
```

**Expected Gain:** 15s → 3.75s (4× FASTER on 4-core CPU)

---

### **Strategy 2: gzip Streaming Decompression**

```go
// CURRENT: Decompress entire file, then parse
// OPTIMIZED: Stream decompression

import (
    "compress/gzip"
    "io"
)

func StreamGzipFASTQ(filepath string) <-chan Sequence {
    file, _ := os.Open(filepath)
    gzipReader, _ := gzip.NewReader(file)

    // Parse directly from gzip stream (no intermediate decompression)
    scanner := bufio.NewScanner(gzipReader)
    // ...
}
```

**Expected Gain:** 3GB FASTQ → 1GB gzip (3× less disk I/O)
- Decompression adds CPU cost, but saves disk bandwidth
- Net gain: ~20% faster (15s → 12s, then 12s / 4 workers = 3.0s)

---

## 🧠 OPTIMIZATION #5: Spatial Hashing (Digital Root)

### **Current: O(1) Voxel Lookup (Already Optimal!)**

```go
// Digital root hash is already O(1)
func SpatialHash(pos Vector3D) VoxelID {
    x := DigitalRoot(int(pos.X * 100))
    y := DigitalRoot(int(pos.Y * 100))
    z := DigitalRoot(int(pos.Z * 100))
    return VoxelID(x*81 + y*9 + z)  // 9×9×9 = 729 voxels per region
}
```

**No optimization needed.** Digital root is already optimal (O(1) hash, perfect for this use case).

---

## 📊 COMBINED OPTIMIZATION IMPACT

### **Baseline Performance (Conservative Estimates):**

```mathematical
BASELINE[B] = {
  Rendering: 7.0ms,
  Frustum_culling: 0.5ms,
  GPU_upload: 1.5ms,
  Camera: 0.1ms,
  UI: 0.5ms,
  Total: 9.6ms per frame (104 fps)
}
```

### **Optimized Performance (Aggressive Optimizations):**

```mathematical
OPTIMIZED[O] = {
  Rendering: 2.1ms (LOD + point sprites),
  Frustum_culling: 0.0001ms (octree),
  GPU_upload: 0.06ms (persistent buffers),
  Camera: 0.1ms,
  UI: 0.5ms,
  Total: 2.76ms per frame (362 fps!)
}

SPEEDUP = 9.6ms / 2.76ms = 3.5× faster
```

### **Realistic Optimized (Conservative):**

```mathematical
REALISTIC[R] = {
  Rendering: 4.2ms (mega-batching only),
  Frustum_culling: 0.1ms (octree without SIMD),
  GPU_upload: 0.5ms (persistent buffers),
  Camera: 0.1ms,
  UI: 0.5ms,
  Total: 5.4ms per frame (185 fps)
}

SPEEDUP = 9.6ms / 5.4ms = 1.8× faster (VERY ACHIEVABLE)
```

---

## 🎯 OPTIMIZATION ROADMAP

### **Phase 1: Essential (Wave 3)**
1. ✅ GPU instancing (already planned)
2. ✅ Frustum culling with octree (add to Wave 3 Agent 3.2)
3. ✅ 3-tier LOD system (add to Wave 3 Agent 3.2)

**Expected:** 104 fps baseline

---

### **Phase 2: Performance Polish (Wave 6 Stress Testing)**
4. Mega-batching (group batches for fewer draw calls)
5. Persistent GPU buffers (reduce upload bandwidth)
6. 5-tier LOD with aggressive culling

**Expected:** 185 fps optimized

---

### **Phase 3: Extreme Optimization (Post-Launch, If Needed)**
7. Point sprites (requires WebGL 2.0 fallback handling)
8. SIMD frustum testing (platform-specific)
9. Color palette compression (2-bit vs 32-bit)

**Expected:** 362 fps theoretical maximum (likely overkill)

---

## 🔧 PROFILING STRATEGY

### **Tools:**

1. **Chrome DevTools Performance Panel**
   - Record 60 seconds of interaction
   - Identify frame drops (yellow/red bars)
   - Measure GPU utilization

2. **WebGL Inspector**
   - Count draw calls per frame
   - Measure buffer upload sizes
   - Validate shader performance

3. **Go pprof (Backend)**
   - Profile FASTQ parsing
   - Identify memory allocations
   - Optimize hot paths

### **Metrics to Track:**

```mathematical
PERFORMANCE_METRICS[PM] = {
  Frame_rate: Target ≥60fps sustained,
  GPU_utilization: Target 80-95%,
  Frame_time_variance: <5ms (smooth, no stuttering),
  Memory_CPU: <2GB peak,
  Memory_GPU: <250MB VRAM,
  Load_time: <5s for 3GB file
}
```

---

## 🚦 OPTIMIZATION DECISION TREE

```
START: Measure baseline performance

IF frame_rate < 60fps:
    IF GPU_time > 10ms:
        → Optimize rendering (LOD, mega-batching, point sprites)
    ELSE IF CPU_time > 5ms:
        → Optimize frustum culling (octree, SIMD)
    ELSE IF upload_time > 2ms:
        → Optimize GPU upload (persistent buffers, compression)

ELSE IF frame_rate ≥ 60fps:
    → Profile variance
    IF variance > 5ms:
        → Optimize memory allocations (reduce GC pressure)
    ELSE:
        → SHIP IT (performance target met!)

ELSE IF load_time > 5s:
    → Optimize FASTQ parsing (parallel workers, gzip streaming)
```

---

## 📈 EXPECTED OUTCOMES

### **Conservative (Baseline Architecture):**
- 104 fps average
- 9.6ms per frame
- 1.77 GB memory
- 3.75s load time
- **Status:** EXCEEDS all targets

### **Optimized (With Phase 2 Optimizations):**
- 185 fps average
- 5.4ms per frame
- 1.5 GB memory (compression)
- 3.0s load time (parallel + gzip)
- **Status:** 3× performance headroom

### **Extreme (With Phase 3 Optimizations):**
- 362 fps average
- 2.76ms per frame
- 1.2 GB memory
- 2.5s load time
- **Status:** Likely overkill, but proves scalability

---

**Maintained by:** Agent Deploy-2 (Performance Engineer)
**Last Updated:** 2025-11-06 (Post Red Team Audit)
**Next Review:** After Wave 3 benchmarking
