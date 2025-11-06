#version 300 es

// ═══════════════════════════════════════════════════════════════════════════
// BACKGROUND VERTEX SHADER
// Generates fullscreen quad without vertex buffer
// ═══════════════════════════════════════════════════════════════════════════

// NO INPUTS (vertex buffer not needed!)
// We generate vertices procedurally using gl_VertexID

void main() {
    // Generate fullscreen quad from vertex ID
    // Vertex 0: (-1, -1) bottom-left
    // Vertex 1: ( 1, -1) bottom-right
    // Vertex 2: (-1,  1) top-left
    // Vertex 3: ( 1,  1) top-right
    //
    // Rendered as TRIANGLE_STRIP:
    //   0---2
    //   |  /|
    //   | / |
    //   |/  |
    //   1---3

    // Compute X coordinate: 0,1 → -1,1 (even/odd)
    float x = float((gl_VertexID & 1) << 1) - 1.0;

    // Compute Y coordinate: 0,0,1,1 → -1,-1,1,1
    float y = float((gl_VertexID & 2)) - 1.0;

    // Set position (covers entire clip space)
    gl_Position = vec4(x, y, 0.0, 1.0);
}

// ═══════════════════════════════════════════════════════════════════════════
// ALTERNATIVE IMPLEMENTATION (more readable, slightly slower)
// ═══════════════════════════════════════════════════════════════════════════

/*
void main() {
    // Array of quad positions (hardcoded)
    vec2 positions[4] = vec2[4](
        vec2(-1.0, -1.0),  // bottom-left
        vec2( 1.0, -1.0),  // bottom-right
        vec2(-1.0,  1.0),  // top-left
        vec2( 1.0,  1.0)   // top-right
    );

    gl_Position = vec4(positions[gl_VertexID], 0.0, 1.0);
}
*/

// ═══════════════════════════════════════════════════════════════════════════
// DOCUMENTATION
// ═══════════════════════════════════════════════════════════════════════════

/*
FULLSCREEN QUAD TECHNIQUE:

Traditional approach:
    1. Create vertex buffer with 4 vertices
    2. Bind buffer
    3. Set attribute pointers
    4. Draw

Procedural approach (this shader):
    1. No vertex buffer needed!
    2. Generate vertices from gl_VertexID
    3. Draw with count=4
    4. Result: Same quad, less overhead

WHY THIS WORKS:

gl_VertexID is built-in vertex index:
    - 0, 1, 2, 3 for 4-vertex draw call
    - We compute position from this index
    - GPU generates vertices on-the-fly

BIT MANIPULATION MAGIC:

For X coordinate:
    gl_VertexID & 1:  extracts lowest bit (0,1,0,1)
    << 1:             shift left (0,2,0,2)
    - 1.0:            offset (-1,1,-1,1)

For Y coordinate:
    gl_VertexID & 2:  extracts second bit (0,0,2,2)
    - 1.0:            offset (-1,-1,1,1)

Result:
    Vertex 0: x=-1, y=-1
    Vertex 1: x= 1, y=-1
    Vertex 2: x=-1, y= 1
    Vertex 3: x= 1, y= 1

TRIANGLE_STRIP WINDING:

With 4 vertices, TRIANGLE_STRIP creates 2 triangles:
    Triangle 1: vertices 0, 1, 2
    Triangle 2: vertices 1, 3, 2 (reuses vertices)

Result: Full quad with 2 triangles, 4 vertices (instead of 6 for TRIANGLES)

PERFORMANCE:

Vertex shader overhead: ~4 GPU cycles per vertex
4 vertices × ~4 cycles = 16 cycles (negligible)
Saving: No buffer allocation, no buffer binding, less state

WHEN TO USE:

✓ Fullscreen effects (post-processing, background)
✓ Simple geometry (quad, triangle)
✓ Procedural shapes (sphere, cube)

✗ Complex meshes (use vertex buffers)
✗ Animated geometry (prefer buffer updates)
✗ Instanced rendering (attributes needed)

FRAGMENT SHADER GETS:

No attributes are passed, but fragment shader can use:
    - gl_FragCoord: pixel position (0,0) to (width,height)
    - Uniforms: u_resolution, u_time, etc.
    - Compute UV: vec2 uv = gl_FragCoord.xy / u_resolution;
*/
