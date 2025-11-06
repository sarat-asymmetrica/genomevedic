#version 300 es
precision highp float;

// Per-instance attributes (different for each particle)
in vec3 a_instancePosition;  // Particle position in 3D space (genomic coordinates)
in vec4 a_instanceColor;     // Vedic color (GC content, mutation, digital root)
in float a_instanceSize;     // Size with LOD scaling

// Camera uniforms
uniform mat4 u_viewProjection;  // Combined view-projection matrix
uniform vec3 u_cameraPosition;  // Camera position for distance calculations

// Output to fragment shader
out vec4 v_color;
out float v_size;

void main() {
    // Transform particle position to clip space
    vec4 worldPos = vec4(a_instancePosition, 1.0);
    gl_Position = u_viewProjection * worldPos;

    // Distance-based size attenuation
    // Particles farther from camera appear smaller (realistic depth cue)
    float distance = length(a_instancePosition - u_cameraPosition);
    float distanceFactor = 1.0 / (1.0 + distance * 0.0001);

    // Calculate point size (clamped to reasonable range)
    float baseSize = a_instanceSize * distanceFactor;
    gl_PointSize = clamp(baseSize, 1.0, 64.0);

    // Pass color and size to fragment shader
    v_color = a_instanceColor;
    v_size = gl_PointSize;
}
