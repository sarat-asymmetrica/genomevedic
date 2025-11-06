#version 300 es
precision highp float;

// Fullscreen quad vertex positions
in vec2 a_position;

// Output UV coordinates for fragment shader
out vec2 v_uv;

void main() {
    // Pass through position (already in clip space -1 to +1)
    gl_Position = vec4(a_position, 0.0, 1.0);

    // Convert from clip space [-1, 1] to UV space [0, 1]
    v_uv = a_position * 0.5 + 0.5;
}
