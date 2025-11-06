#version 300 es

// ═══════════════════════════════════════════════════════════════════════════
// PARTICLE VERTEX SHADER
// Transforms particle positions and prepares data for fragment shader
// ═══════════════════════════════════════════════════════════════════════════

// ATTRIBUTES (per-particle data from buffers)
in vec2 a_position;      // Particle position (x, y) in pixels
in vec4 a_color;         // Particle color (RGBA, pre-interpolated by Go)
in float a_size;         // Particle size in pixels

// UNIFORMS (shared across all particles)
uniform vec2 u_resolution;      // Screen resolution (width, height)
uniform float u_time;           // Elapsed time in seconds
uniform float u_particle_scale; // Global scale multiplier
uniform float u_particle_alpha; // Global alpha multiplier

// OUTPUT to fragment shader
out vec4 v_color;        // Color for fragment shader

void main() {
    // Convert pixel coordinates to clip space (-1 to 1)
    // Pixel space: (0,0) top-left, (width,height) bottom-right
    // Clip space: (-1,-1) bottom-left, (1,1) top-right
    vec2 clipSpace = (a_position / u_resolution) * 2.0 - 1.0;

    // Flip Y axis (canvas Y is down, WebGL Y is up)
    clipSpace.y *= -1.0;

    // Set position (Z=0 for 2D, W=1 for perspective division)
    gl_Position = vec4(clipSpace, 0.0, 1.0);

    // Set point size (affected by global scale)
    gl_PointSize = a_size * u_particle_scale;

    // Pass color to fragment shader (apply global alpha)
    v_color = vec4(a_color.rgb, a_color.a * u_particle_alpha);
}

// ═══════════════════════════════════════════════════════════════════════════
// DOCUMENTATION
// ═══════════════════════════════════════════════════════════════════════════

/*
VERTEX SHADER PIPELINE:

1. Read per-particle attributes (position, color, size)
2. Transform position from pixel space to clip space
3. Set gl_PointSize for point sprite rendering
4. Pass color to fragment shader

COORDINATE SYSTEMS:

Pixel space (input):
    Origin: (0, 0) top-left
    Range: (0, 0) to (width, height)

Clip space (output):
    Origin: (0, 0) center
    Range: (-1, -1) to (1, 1)
    Y-axis flipped

POINT SPRITES:

gl_PointSize determines the size of the point sprite in pixels.
Fragment shader receives gl_PointCoord (0 to 1) for each fragment.
This allows rendering circular particles (see fragment shader).

PERFORMANCE:

This vertex shader is trivial (~10 instructions).
Bottleneck is in fragment shader (more pixels per particle).
50,000 particles × ~10 instructions = 500K ops (negligible).

VARIABLES:

Attributes (in):
    - Read from vertex buffers
    - Different for each vertex/particle
    - Format: vec2, vec4, float

Uniforms (uniform):
    - Set from CPU once per draw call
    - Same for all vertices
    - Format: vec2, float

Varyings (out):
    - Interpolated across fragments
    - Passed to fragment shader
    - Format: vec4

Built-ins:
    gl_Position: Output position (clip space)
    gl_PointSize: Output point size (pixels)
*/
