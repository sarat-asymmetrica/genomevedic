#version 300 es
precision highp float;

// ═══════════════════════════════════════════════════════════════════════════
// PARTICLE FRAGMENT SHADER
// Renders circular particles with smooth anti-aliasing
// ═══════════════════════════════════════════════════════════════════════════

// INPUT from vertex shader
in vec4 v_color;         // Interpolated color

// OUTPUT color
out vec4 fragColor;

void main() {
    // gl_PointCoord: (0,0) top-left to (1,1) bottom-right of point sprite
    // Center it: (0.5,0.5) → (0,0)
    vec2 coord = gl_PointCoord - 0.5;

    // Calculate distance from center
    float dist = length(coord);

    // Discard fragments outside circle radius
    if (dist > 0.5) {
        discard; // Don't render this fragment (makes square into circle)
    }

    // Smooth edge anti-aliasing (avoid jagged circles)
    // smoothstep creates smooth transition from 1.0 to 0.0
    // Inner edge: 0.4 (fully opaque)
    // Outer edge: 0.5 (fully transparent)
    float alpha = 1.0 - smoothstep(0.4, 0.5, dist);

    // Soft glow effect (optional, brighten center)
    // float glow = 1.0 - (dist * 2.0); // Brighten center
    // vec3 color = v_color.rgb * glow;

    // Output color with anti-aliased alpha
    fragColor = vec4(v_color.rgb, v_color.a * alpha);
}

// ═══════════════════════════════════════════════════════════════════════════
// DOCUMENTATION
// ═══════════════════════════════════════════════════════════════════════════

/*
FRAGMENT SHADER PIPELINE:

1. Receive interpolated color from vertex shader
2. Calculate distance from point sprite center
3. Discard fragments outside circle
4. Apply smooth anti-aliasing at edge
5. Output final color with alpha

ANTI-ALIASING:

Without anti-aliasing:
    if (dist > 0.5) discard;
    Result: Jagged circle edges (1-pixel transitions)

With anti-aliasing:
    alpha = 1.0 - smoothstep(0.4, 0.5, dist);
    Result: Smooth circle edges (gradient transition)

SMOOTHSTEP:

smoothstep(edge0, edge1, x):
    - Returns 0 if x < edge0
    - Returns 1 if x > edge1
    - Smooth Hermite interpolation in between

Example:
    smoothstep(0.4, 0.5, 0.3) = 0.0 (fully opaque)
    smoothstep(0.4, 0.5, 0.45) = 0.5 (half transparent)
    smoothstep(0.4, 0.5, 0.6) = 1.0 (fully transparent)

DISCARD KEYWORD:

Tells GPU to skip this fragment (no write to framebuffer).
Use for:
    - Making non-rectangular shapes (circles, stars)
    - Alpha testing (cutout textures)
    - Particle rendering (square → circle)

PERFORMANCE:

Fragment shader runs for EVERY pixel in EVERY particle.
50,000 particles × 100 pixels/particle = 5 million fragment invocations.
At 60fps: 300 million fragment shader calls per second!

Keep fragment shaders SIMPLE:
    ✓ Simple distance calculations
    ✓ smoothstep
    ✓ Basic color operations
    ✗ Complex noise functions
    ✗ Many texture lookups
    ✗ Branching (if/else)

BLENDING:

With alpha blending enabled:
    finalColor = srcColor * srcAlpha + dstColor * (1 - srcAlpha)

Particles blend smoothly with background and each other.
Order matters (sort back-to-front for correct blending).

GLOW EFFECT:

To add center glow:
    float glow = 1.0 - (dist * 2.0);
    vec3 color = v_color.rgb * (1.0 + glow * 0.5);

Result: Brighter center, fades to edge.

VARIATIONS:

Soft particle:
    alpha = 1.0 - smoothstep(0.0, 0.5, dist);
    Result: Soft edges, no hard circle

Star shape:
    float angle = atan(coord.y, coord.x);
    float starDist = dist * (1.0 + 0.2 * sin(angle * 5.0));
    if (starDist > 0.5) discard;

Square particles:
    // Remove discard and smoothstep
    fragColor = v_color;
*/
