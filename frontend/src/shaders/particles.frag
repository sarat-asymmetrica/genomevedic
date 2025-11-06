#version 300 es
precision highp float;

// Input from vertex shader
in vec4 v_color;
in float v_size;

// Output color
out vec4 fragColor;

void main() {
    // Convert point coordinate to centered [-0.5, 0.5] range
    vec2 coord = gl_PointCoord - vec2(0.5);
    float dist = length(coord);

    // Discard pixels outside circular radius
    // This makes particles appear as circles instead of squares
    if (dist > 0.5) {
        discard;
    }

    // Smooth anti-aliased edge
    // Edge width scales with particle size for consistent appearance
    float edgeWidth = 2.0 / v_size;
    float alpha = 1.0 - smoothstep(0.5 - edgeWidth, 0.5, dist);

    // Brightness falloff toward edges (subtle depth effect)
    float brightness = 1.0 - dist * 0.3;

    // Apply color with anti-aliasing and brightness
    fragColor = vec4(v_color.rgb * brightness, v_color.a * alpha);
}
