#version 300 es
precision highp float;

// ═══════════════════════════════════════════════════════════════════════════
// BACKGROUND FRAGMENT SHADER
// Quaternion gradient with Perlin noise flow fields
// ═══════════════════════════════════════════════════════════════════════════

// UNIFORMS
uniform vec2 u_resolution;      // Screen resolution
uniform float u_time;           // Elapsed time
uniform vec4 u_color_start;     // Start color (quaternion: w, x, y, z → a, r, g, b)
uniform vec4 u_color_end;       // End color (quaternion)
uniform float u_noise_scale;    // Perlin noise frequency (default: 5.0)
uniform float u_noise_amplitude;// Perlin noise strength (default: 0.2)
uniform int u_regime;           // 0=Exploration, 1=Optimization, 2=Stabilization

// OUTPUT
out vec4 fragColor;

// ═══════════════════════════════════════════════════════════════════════════
// QUATERNION SLERP (Spherical Linear Interpolation)
// ═══════════════════════════════════════════════════════════════════════════

vec4 quatSlerp(vec4 q1, vec4 q2, float t) {
    // Normalize inputs (should already be normalized, but be safe)
    q1 = normalize(q1);
    q2 = normalize(q2);

    // Calculate angle between quaternions
    float dotProd = dot(q1, q2);

    // Ensure shortest path (negate q2 if needed)
    if (dotProd < 0.0) {
        q2 = -q2;
        dotProd = -dotProd;
    }

    // Clamp to valid range (avoid numerical errors in acos)
    dotProd = clamp(dotProd, -1.0, 1.0);

    // If quaternions are very close, use linear interpolation
    const float DOT_THRESHOLD = 0.9995;
    if (dotProd > DOT_THRESHOLD) {
        // Linear interpolation (lerp)
        return normalize(q1 + t * (q2 - q1));
    }

    // Spherical interpolation
    float theta = acos(dotProd);        // Angle between quaternions
    float sinTheta = sin(theta);        // sin(θ)

    float w1 = sin((1.0 - t) * theta) / sinTheta;  // Weight for q1
    float w2 = sin(t * theta) / sinTheta;          // Weight for q2

    return q1 * w1 + q2 * w2;
}

// ═══════════════════════════════════════════════════════════════════════════
// PERLIN NOISE (Simplified 2D)
// ═══════════════════════════════════════════════════════════════════════════

// Hash function for noise (pseudo-random)
float hash(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

// 2D Perlin-like noise
float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);

    // Smooth interpolation (smoothstep)
    f = f * f * (3.0 - 2.0 * f);

    // Get hash values at corners
    float a = hash(i);
    float b = hash(i + vec2(1.0, 0.0));
    float c = hash(i + vec2(0.0, 1.0));
    float d = hash(i + vec2(1.0, 1.0));

    // Bilinear interpolation
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

// Fractal Brownian Motion (multiple octaves of noise)
float fbm(vec2 p) {
    float value = 0.0;
    float amplitude = 0.5;
    float frequency = 1.0;

    // 4 octaves of noise
    for (int i = 0; i < 4; i++) {
        value += amplitude * noise(p * frequency);
        amplitude *= 0.5;  // Each octave half as strong
        frequency *= 2.0;  // Each octave twice as detailed
    }

    return value;
}

// ═══════════════════════════════════════════════════════════════════════════
// MAIN SHADER
// ═══════════════════════════════════════════════════════════════════════════

void main() {
    // Normalized coordinates (0 to 1)
    vec2 uv = gl_FragCoord.xy / u_resolution;

    // Time-based flow (animate noise)
    float flowTime = u_time * 0.1;

    // Add Perlin noise for organic variation
    float n = fbm(uv * u_noise_scale + flowTime);

    // Interpolation factor (left-to-right gradient + noise)
    float t = uv.x + n * u_noise_amplitude;

    // Regime-based behavior
    if (u_regime == 0) {
        // EXPLORATION: More chaotic, vibrant
        t += sin(u_time * 2.0 + uv.y * 10.0) * 0.1;
    } else if (u_regime == 1) {
        // OPTIMIZATION: Moderate variation
        t += sin(u_time * 1.0 + uv.y * 5.0) * 0.05;
    }
    // STABILIZATION (u_regime == 2): Stable, no additional variation

    // Clamp t to [0, 1]
    t = clamp(t, 0.0, 1.0);

    // Quaternion SLERP between start and end colors
    vec4 q = quatSlerp(u_color_start, u_color_end, t);

    // Convert quaternion to RGB
    // Quaternion format: (w, x, y, z) → (a, r, g, b)
    // For color, we use (x, y, z) as RGB, w as alpha
    vec3 color = vec3(q.y, q.z, q.w); // Extract RGB from quaternion

    // Normalize color to [0, 1] range
    color = clamp(color, 0.0, 1.0);

    // Optional: Add subtle vignette (darken edges)
    float vignette = 1.0 - 0.3 * length(uv - 0.5);
    color *= vignette;

    // Output final color (opaque)
    fragColor = vec4(color, 1.0);
}

// ═══════════════════════════════════════════════════════════════════════════
// DOCUMENTATION
// ═══════════════════════════════════════════════════════════════════════════

/*
QUATERNION COLOR INTERPOLATION:

Why quaternions for colors?
    - Smooth, perceptually uniform transitions
    - No muddy intermediate colors (RGB lerp problem)
    - Mathematically elegant

RGB lerp problem:
    Blue (0,0,1) → Yellow (1,1,0)
    Midpoint: (0.5, 0.5, 0.5) = Gray (muddy!)

Quaternion slerp:
    Blue → Yellow passes through Cyan (vibrant!)

PERLIN NOISE:

Adds organic variation to gradient:
    - fbm(): Fractal Brownian Motion (multiple octaves)
    - Creates natural-looking patterns
    - Animates over time (flowTime)

REGIME BEHAVIOR:

Exploration (30%):
    - Chaotic, vibrant
    - More noise, faster animation
    - Discovers edge cases

Optimization (20%):
    - Moderate variation
    - Refining, tuning
    - Balanced chaos and stability

Stabilization (50%):
    - Stable, predictable
    - Minimal variation
    - Locked for testing

COORDINATE SYSTEMS:

gl_FragCoord:
    - Pixel coordinates (0,0) to (width,height)
    - Bottom-left origin in WebGL

uv:
    - Normalized coordinates (0,0) to (1,1)
    - Used for effects independent of resolution

PERFORMANCE:

Fragment shader runs for EVERY pixel:
    1920×1080 = 2,073,600 pixels per frame
    At 60fps = 124 million invocations per second!

Optimizations:
    ✓ Simple noise function (not true Perlin, faster)
    ✓ Limited octaves (4 instead of 8)
    ✓ No texture lookups
    ✓ Minimal branching

GPU is parallel beast:
    - Each pixel processed simultaneously
    - 2 million pixels = easy for modern GPU
    - Bottleneck is memory bandwidth, not compute

CUSTOMIZATION:

Change gradient direction:
    float t = uv.y; // Top-to-bottom
    float t = length(uv - 0.5); // Radial

Add more noise:
    float n = fbm(uv * 10.0 + flowTime); // Higher frequency

Animate differently:
    float flowTime = u_time * 0.5; // Slower
    float flowTime = sin(u_time * 0.2); // Oscillate

VIGNETTE EFFECT:

Darkens edges, focuses center:
    float vignette = 1.0 - 0.3 * length(uv - 0.5);
    color *= vignette;

Adjust strength:
    0.0 = no vignette
    0.5 = moderate darkening
    1.0 = black edges

DEBUGGING:

View noise only:
    fragColor = vec4(vec3(n), 1.0);

View interpolation factor:
    fragColor = vec4(vec3(t), 1.0);

View quaternion components:
    fragColor = vec4(q.xyz, 1.0);
*/
