/**
 * MouseControls - Mouse input handling for camera
 *
 * - Left drag: Rotate camera
 * - Scroll wheel: Zoom in/out
 * - Right drag: Pan camera
 */

export class MouseControls {
    constructor(camera, canvas) {
        this.camera = camera;
        this.canvas = canvas;

        this.isDragging = false;
        this.isRightDragging = false;
        this.lastMouseX = 0;
        this.lastMouseY = 0;

        this.rotationSensitivity = 0.005;  // Radians per pixel
        this.panSensitivity = 2.0;         // Units per pixel
        this.zoomSensitivity = 50.0;       // Units per scroll notch

        this.setupEventListeners();
    }

    setupEventListeners() {
        // Mouse down
        this.canvas.addEventListener('mousedown', (e) => {
            if (e.button === 0) {
                // Left button - rotate
                this.isDragging = true;
                this.lastMouseX = e.clientX;
                this.lastMouseY = e.clientY;
                e.preventDefault();
            } else if (e.button === 2) {
                // Right button - pan
                this.isRightDragging = true;
                this.lastMouseX = e.clientX;
                this.lastMouseY = e.clientY;
                e.preventDefault();
            }
        });

        // Mouse up
        this.canvas.addEventListener('mouseup', (e) => {
            this.isDragging = false;
            this.isRightDragging = false;
        });

        // Mouse leave
        this.canvas.addEventListener('mouseleave', () => {
            this.isDragging = false;
            this.isRightDragging = false;
        });

        // Mouse move
        this.canvas.addEventListener('mousemove', (e) => {
            if (this.isDragging) {
                const deltaX = e.clientX - this.lastMouseX;
                const deltaY = e.clientY - this.lastMouseY;

                // Rotate camera (yaw = X movement, pitch = Y movement)
                this.camera.rotate(
                    -deltaX * this.rotationSensitivity,  // Yaw
                    -deltaY * this.rotationSensitivity   // Pitch
                );

                this.lastMouseX = e.clientX;
                this.lastMouseY = e.clientY;
            } else if (this.isRightDragging) {
                const deltaX = e.clientX - this.lastMouseX;
                const deltaY = e.clientY - this.lastMouseY;

                // Pan camera
                this.camera.strafe(deltaX * this.panSensitivity);
                this.camera.elevate(-deltaY * this.panSensitivity);

                this.lastMouseX = e.clientX;
                this.lastMouseY = e.clientY;
            }
        });

        // Mouse wheel
        this.canvas.addEventListener('wheel', (e) => {
            e.preventDefault();

            // Zoom in/out (move camera forward/back)
            const zoomDelta = -Math.sign(e.deltaY) * this.zoomSensitivity;
            this.camera.moveForward(zoomDelta);
        });

        // Disable context menu on canvas
        this.canvas.addEventListener('contextmenu', (e) => {
            e.preventDefault();
        });
    }
}
