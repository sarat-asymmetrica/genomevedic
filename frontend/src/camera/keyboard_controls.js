/**
 * KeyboardControls - Keyboard input for camera movement
 *
 * - WASD: Move camera
 * - Q/E: Move up/down
 * - Shift: Move faster
 * - R: Reset camera
 */

export class KeyboardControls {
    constructor(camera) {
        this.camera = camera;

        this.keys = new Set();
        this.moveSpeed = 10.0;
        this.fastMoveSpeed = 50.0;

        this.setupEventListeners();
    }

    setupEventListeners() {
        window.addEventListener('keydown', (e) => {
            this.keys.add(e.key.toLowerCase());

            // Reset camera on R
            if (e.key.toLowerCase() === 'r') {
                this.camera.reset();
            }
        });

        window.addEventListener('keyup', (e) => {
            this.keys.delete(e.key.toLowerCase());
        });
    }

    update(deltaTime) {
        if (this.keys.size === 0) return;

        // Determine speed (faster if shift is held)
        const speed = this.keys.has('shift') ? this.fastMoveSpeed : this.moveSpeed;
        const distance = speed * deltaTime;

        // WASD movement
        if (this.keys.has('w')) {
            this.camera.moveForward(distance);
        }
        if (this.keys.has('s')) {
            this.camera.moveForward(-distance);
        }
        if (this.keys.has('a')) {
            this.camera.strafe(-distance);
        }
        if (this.keys.has('d')) {
            this.camera.strafe(distance);
        }

        // QE up/down
        if (this.keys.has('q')) {
            this.camera.elevate(-distance);
        }
        if (this.keys.has('e')) {
            this.camera.elevate(distance);
        }
    }
}
