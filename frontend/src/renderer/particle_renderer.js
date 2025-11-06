/**
 * ParticleRenderer - GPU-accelerated particle rendering with instancing
 *
 * Renders millions of particles efficiently using WebGL2 instanced rendering.
 * Single draw call for up to 50K particles per batch.
 */

export class ParticleRenderer {
    constructor(canvas) {
        this.canvas = canvas;
        this.gl = this.initWebGL();

        if (!this.gl) {
            throw new Error('WebGL 2.0 not supported');
        }

        this.program = null;
        this.vao = null;
        this.buffers = {
            position: null,
            color: null,
            size: null
        };

        this.maxParticles = 50000;
        this.particleCount = 0;

        // Pre-allocate typed arrays (reused every frame)
        this.positions = new Float32Array(this.maxParticles * 3);
        this.colors = new Float32Array(this.maxParticles * 4);
        this.sizes = new Float32Array(this.maxParticles);

        this.init();
    }

    initWebGL() {
        const gl = this.canvas.getContext('webgl2', {
            alpha: true,
            depth: true,
            antialias: false,  // We handle AA in shader
            premultipliedAlpha: false,
            preserveDrawingBuffer: false
        });

        if (!gl) {
            console.error('WebGL 2.0 not available');
            return null;
        }

        // Check WebGL capabilities
        const maxInstances = gl.getParameter(gl.MAX_ELEMENTS_INDICES);
        console.log(`WebGL Max Instances: ${maxInstances}`);

        return gl;
    }

    async init() {
        const gl = this.gl;

        // Load and compile shaders
        const vertexShader = await this.loadShader('shaders/particles.vert', gl.VERTEX_SHADER);
        const fragmentShader = await this.loadShader('shaders/particles.frag', gl.FRAGMENT_SHADER);

        // Create shader program
        this.program = this.createProgram(vertexShader, fragmentShader);

        // Setup vertex array object and buffers
        this.setupBuffers();

        // Setup blending for particle transparency
        this.setupBlending();

        console.log('ParticleRenderer initialized');
    }

    async loadShader(url, type) {
        const gl = this.gl;

        // Fetch shader source
        const response = await fetch(url);
        const source = await response.text();

        // Compile shader
        const shader = gl.createShader(type);
        gl.shaderSource(shader, source);
        gl.compileShader(shader);

        // Check compilation status
        if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
            const error = gl.getShaderInfoLog(shader);
            console.error(`Shader compilation error (${url}):`, error);
            gl.deleteShader(shader);
            throw new Error(`Shader compilation failed: ${error}`);
        }

        return shader;
    }

    createProgram(vertexShader, fragmentShader) {
        const gl = this.gl;

        const program = gl.createProgram();
        gl.attachShader(program, vertexShader);
        gl.attachShader(program, fragmentShader);
        gl.linkProgram(program);

        // Check linking status
        if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
            const error = gl.getProgramInfoLog(program);
            console.error('Program linking error:', error);
            throw new Error(`Program linking failed: ${error}`);
        }

        return program;
    }

    setupBuffers() {
        const gl = this.gl;

        // Create vertex array object
        this.vao = gl.createVertexArray();
        gl.bindVertexArray(this.vao);

        // Create buffers
        this.buffers.position = gl.createBuffer();
        this.buffers.color = gl.createBuffer();
        this.buffers.size = gl.createBuffer();

        // Position buffer (vec3)
        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
        gl.bufferData(gl.ARRAY_BUFFER, this.positions, gl.DYNAMIC_DRAW);

        const positionLoc = gl.getAttribLocation(this.program, 'a_instancePosition');
        gl.enableVertexAttribArray(positionLoc);
        gl.vertexAttribPointer(positionLoc, 3, gl.FLOAT, false, 0, 0);
        gl.vertexAttribDivisor(positionLoc, 1);  // Instanced

        // Color buffer (vec4)
        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.color);
        gl.bufferData(gl.ARRAY_BUFFER, this.colors, gl.DYNAMIC_DRAW);

        const colorLoc = gl.getAttribLocation(this.program, 'a_instanceColor');
        gl.enableVertexAttribArray(colorLoc);
        gl.vertexAttribPointer(colorLoc, 4, gl.FLOAT, false, 0, 0);
        gl.vertexAttribDivisor(colorLoc, 1);  // Instanced

        // Size buffer (float)
        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.size);
        gl.bufferData(gl.ARRAY_BUFFER, this.sizes, gl.DYNAMIC_DRAW);

        const sizeLoc = gl.getAttribLocation(this.program, 'a_instanceSize');
        gl.enableVertexAttribArray(sizeLoc);
        gl.vertexAttribPointer(sizeLoc, 1, gl.FLOAT, false, 0, 0);
        gl.vertexAttribDivisor(sizeLoc, 1);  // Instanced

        gl.bindVertexArray(null);
    }

    setupBlending() {
        const gl = this.gl;

        // Enable additive blending for overlapping particles
        gl.enable(gl.BLEND);
        gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
        gl.blendEquation(gl.FUNC_ADD);

        // Enable depth testing
        gl.enable(gl.DEPTH_TEST);
        gl.depthFunc(gl.LESS);

        // Enable point sprite rendering
        gl.enable(gl.VERTEX_PROGRAM_POINT_SIZE);
    }

    updateParticles(particles) {
        // Fill typed arrays with particle data
        this.particleCount = Math.min(particles.length, this.maxParticles);

        for (let i = 0; i < this.particleCount; i++) {
            const p = particles[i];

            // Position
            this.positions[i * 3 + 0] = p.x;
            this.positions[i * 3 + 1] = p.y;
            this.positions[i * 3 + 2] = p.z;

            // Color (RGBA)
            this.colors[i * 4 + 0] = p.r;
            this.colors[i * 4 + 1] = p.g;
            this.colors[i * 4 + 2] = p.b;
            this.colors[i * 4 + 3] = p.a !== undefined ? p.a : 1.0;

            // Size
            this.sizes[i] = p.size !== undefined ? p.size : 5.0;
        }

        // Upload to GPU (only updated portion)
        const gl = this.gl;

        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
        gl.bufferSubData(gl.ARRAY_BUFFER, 0, this.positions.subarray(0, this.particleCount * 3));

        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.color);
        gl.bufferSubData(gl.ARRAY_BUFFER, 0, this.colors.subarray(0, this.particleCount * 4));

        gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.size);
        gl.bufferSubData(gl.ARRAY_BUFFER, 0, this.sizes.subarray(0, this.particleCount));
    }

    render(camera) {
        const gl = this.gl;

        // Clear framebuffer
        gl.clearColor(0.0, 0.0, 0.0, 1.0);
        gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);

        if (this.particleCount === 0) {
            return;
        }

        // Use shader program
        gl.useProgram(this.program);

        // Upload camera uniforms
        const vpLoc = gl.getUniformLocation(this.program, 'u_viewProjection');
        gl.uniformMatrix4fv(vpLoc, false, camera.viewProjectionMatrix);

        const camPosLoc = gl.getUniformLocation(this.program, 'u_cameraPosition');
        gl.uniform3fv(camPosLoc, camera.position);

        // Bind VAO and draw
        gl.bindVertexArray(this.vao);
        gl.drawArraysInstanced(gl.POINTS, 0, 1, this.particleCount);
        gl.bindVertexArray(null);
    }

    resize(width, height) {
        this.canvas.width = width;
        this.canvas.height = height;
        this.gl.viewport(0, 0, width, height);
    }

    destroy() {
        const gl = this.gl;

        // Delete buffers
        gl.deleteBuffer(this.buffers.position);
        gl.deleteBuffer(this.buffers.color);
        gl.deleteBuffer(this.buffers.size);

        // Delete VAO
        gl.deleteVertexArray(this.vao);

        // Delete program
        gl.deleteProgram(this.program);
    }
}
