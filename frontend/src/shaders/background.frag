#version 300 es
precision highp float;

in vec2 v_uv;
out vec4 fragColor;

uniform float u_time;        // Seconds elapsed for animation
uniform vec4 u_colorTop;     // Top gradient color (RGBA)
uniform vec4 u_colorBottom;  // Bottom gradient color (RGBA)

// Perlin noise hash function
float hash(vec3 p) {
    p = fract(p * vec3(443.897, 441.423, 437.195));
    p += dot(p, p.yxz + 19.19);
    return fract((p.x + p.y) * p.z);
}

// 3D Perlin noise
float perlin(vec3 p) {
    vec3 i = floor(p);
    vec3 f = fract(p);

    // Cubic Hermite interpolation
    f = f * f * (3.0 - 2.0 * f);

    // Sample 8 corners of cube
    float c000 = hash(i + vec3(0.0, 0.0, 0.0));
    float c001 = hash(i + vec3(0.0, 0.0, 1.0));
    float c010 = hash(i + vec3(0.0, 1.0, 0.0));
    float c011 = hash(i + vec3(0.0, 1.0, 1.0));
    float c100 = hash(i + vec3(1.0, 0.0, 0.0));
    float c101 = hash(i + vec3(1.0, 0.0, 1.0));
    float c110 = hash(i + vec3(1.0, 1.0, 0.0));
    float c111 = hash(i + vec3(1.0, 1.0, 1.0));

    // Trilinear interpolation
    float x00 = mix(c000, c100, f.x);
    float x01 = mix(c001, c101, f.x);
    float x10 = mix(c010, c110, f.x);
    float x11 = mix(c011, c111, f.x);

    float y0 = mix(x00, x10, f.y);
    float y1 = mix(x01, x11, f.y);

    return mix(y0, y1, f.z);
}

// Fractional Brownian Motion (layered noise)
float fbm(vec3 p) {
    float value = 0.0;
    float amplitude = 0.5;
    float frequency = 1.0;

    for (int i = 0; i < 5; i++) {
        value += amplitude * perlin(p * frequency);
        frequency *= 2.0;
        amplitude *= 0.5;
    }

    return value;
}

// Quaternion multiplication
vec4 quatMul(vec4 q1, vec4 q2) {
    return vec4(
        q1.w * q2.x + q1.x * q2.w + q1.y * q2.z - q1.z * q2.y,
        q1.w * q2.y - q1.x * q2.z + q1.y * q2.w + q1.z * q2.x,
        q1.w * q2.z + q1.x * q2.y - q1.y * q2.x + q1.z * q2.w,
        q1.w * q2.w - q1.x * q2.x - q1.y * q2.y - q1.z * q2.z
    );
}

// Quaternion slerp (spherical linear interpolation)
vec4 quatSlerp(vec4 q1, vec4 q2, float t) {
    float dotProduct = dot(q1, q2);

    // Ensure shortest path
    if (dotProduct < 0.0) {
        q2 = -q2;
        dotProduct = -dotProduct;
    }

    // Linear interpolation for small angles
    if (dotProduct > 0.9995) {
        return normalize(mix(q1, q2, t));
    }

    // Slerp formula
    float theta = acos(clamp(dotProduct, -1.0, 1.0));
    float sinTheta = sin(theta);
    float w1 = sin((1.0 - t) * theta) / sinTheta;
    float w2 = sin(t * theta) / sinTheta;

    return q1 * w1 + q2 * w2;
}

// Convert quaternion to RGB color
vec3 quatToRGB(vec4 q) {
    // Normalize quaternion
    q = normalize(q);

    // Extract rotation axis and angle
    float angle = 2.0 * acos(clamp(q.w, -1.0, 1.0));
    vec3 axis = length(q.xyz) > 0.0001 ? normalize(q.xyz) : vec3(0, 1, 0);

    // Map to RGB using axis components and angle
    float hue = angle / (2.0 * 3.14159);
    vec3 color = vec3(
        0.5 + 0.5 * axis.x * cos(angle),
        0.5 + 0.5 * axis.y * cos(angle + 2.094),  // 120° phase shift
        0.5 + 0.5 * axis.z * cos(angle + 4.189)   // 240° phase shift
    );

    return clamp(color, 0.0, 1.0);
}

void main() {
    // Convert colors to quaternions
    vec4 q1 = vec4(normalize(u_colorTop.rgb), 0.5);
    vec4 q2 = vec4(normalize(u_colorBottom.rgb), 0.5);

    // Normalize quaternions
    q1 = normalize(q1);
    q2 = normalize(q2);

    // Vertical gradient parameter
    float t = v_uv.y;

    // Add animated Perlin noise
    vec3 noiseCoord = vec3(v_uv * 3.0, u_time * 0.1);
    float noise = fbm(noiseCoord);

    // Perturb gradient with noise
    t += noise * 0.15;
    t = clamp(t, 0.0, 1.0);

    // Quaternion slerp between colors
    vec4 q = quatSlerp(q1, q2, t);

    // Convert quaternion to RGB
    vec3 color = quatToRGB(q);

    // Add subtle noise texture
    color += vec3(noise * 0.05);

    fragColor = vec4(color, 1.0);
}
