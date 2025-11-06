/**
 * QuaternionCamera - Smooth camera with quaternion rotations (no gimbal lock)
 *
 * Uses quaternion slerp for smooth interpolated rotations.
 * Avoids gimbal lock issues inherent in Euler angles.
 */

import { quat, vec3, mat4 } from '../utils/gl_matrix.js';

export class QuaternionCamera {
    constructor(fov = 60, aspect = 16/9, near = 0.1, far = 100000) {
        // Camera transform
        this.position = vec3.fromValues(0, 0, 1000);
        this.rotation = quat.create();  // Identity quaternion
        this.target = quat.create();    // Target rotation for slerp

        // Projection parameters
        this.fov = fov;
        this.aspect = aspect;
        this.near = near;
        this.far = far;

        // Smooth interpolation
        this.slerpSpeed = 0.15;  // 15% interpolation per frame

        // Matrices (cached)
        this.viewMatrix = mat4.create();
        this.projectionMatrix = mat4.create();
        this.viewProjectionMatrix = mat4.create();

        this.updateProjectionMatrix();
        this.updateViewMatrix();
    }

    /**
     * Rotate camera by delta angles (in radians)
     */
    rotate(deltaYaw, deltaPitch) {
        // Create quaternions for yaw (Y axis) and pitch (X axis)
        const yawQuat = quat.create();
        const pitchQuat = quat.create();

        quat.setAxisAngle(yawQuat, [0, 1, 0], deltaYaw);
        quat.setAxisAngle(pitchQuat, [1, 0, 0], deltaPitch);

        // Combine rotations: target = yaw * pitch * target
        quat.multiply(this.target, yawQuat, this.target);
        quat.multiply(this.target, this.target, pitchQuat);

        // Normalize to prevent drift
        quat.normalize(this.target, this.target);
    }

    /**
     * Move camera in local space
     */
    move(direction, speed) {
        // Transform direction by camera rotation
        const rotatedDir = vec3.create();
        vec3.transformQuat(rotatedDir, direction, this.rotation);

        // Scale by speed
        vec3.scale(rotatedDir, rotatedDir, speed);

        // Add to position
        vec3.add(this.position, this.position, rotatedDir);
    }

    /**
     * Move camera forward/backward
     */
    moveForward(distance) {
        this.move([0, 0, -distance], 1);
    }

    /**
     * Move camera left/right
     */
    strafe(distance) {
        this.move([distance, 0, 0], 1);
    }

    /**
     * Move camera up/down
     */
    elevate(distance) {
        this.move([0, distance, 0], 1);
    }

    /**
     * Look at a specific point
     */
    lookAt(target) {
        const direction = vec3.create();
        vec3.subtract(direction, target, this.position);
        vec3.normalize(direction, direction);

        // Calculate rotation quaternion to look at target
        const up = vec3.fromValues(0, 1, 0);
        const right = vec3.create();
        vec3.cross(right, up, direction);
        vec3.normalize(right, right);

        const actualUp = vec3.create();
        vec3.cross(actualUp, direction, right);

        // Build rotation matrix
        const rotMat = mat4.create();
        rotMat[0] = right[0];
        rotMat[1] = right[1];
        rotMat[2] = right[2];
        rotMat[4] = actualUp[0];
        rotMat[5] = actualUp[1];
        rotMat[6] = actualUp[2];
        rotMat[8] = -direction[0];
        rotMat[9] = -direction[1];
        rotMat[10] = -direction[2];

        // Convert to quaternion
        quat.fromMat3(this.target, rotMat);
        quat.normalize(this.target, this.target);
    }

    /**
     * Update camera (call every frame)
     */
    update(deltaTime) {
        // Smooth quaternion slerp
        quat.slerp(this.rotation, this.rotation, this.target, this.slerpSpeed);

        // Update matrices
        this.updateViewMatrix();
        this.updateViewProjectionMatrix();
    }

    updateViewMatrix() {
        // Build view matrix from quaternion and position
        mat4.fromRotationTranslation(this.viewMatrix, this.rotation, this.position);
        mat4.invert(this.viewMatrix, this.viewMatrix);
    }

    updateProjectionMatrix() {
        mat4.perspective(
            this.projectionMatrix,
            this.fov * Math.PI / 180,
            this.aspect,
            this.near,
            this.far
        );
    }

    updateViewProjectionMatrix() {
        mat4.multiply(
            this.viewProjectionMatrix,
            this.projectionMatrix,
            this.viewMatrix
        );
    }

    /**
     * Set aspect ratio (call on window resize)
     */
    setAspect(aspect) {
        this.aspect = aspect;
        this.updateProjectionMatrix();
        this.updateViewProjectionMatrix();
    }

    /**
     * Get forward direction vector
     */
    getForward() {
        const forward = vec3.fromValues(0, 0, -1);
        vec3.transformQuat(forward, forward, this.rotation);
        return forward;
    }

    /**
     * Get right direction vector
     */
    getRight() {
        const right = vec3.fromValues(1, 0, 0);
        vec3.transformQuat(right, right, this.rotation);
        return right;
    }

    /**
     * Get up direction vector
     */
    getUp() {
        const up = vec3.fromValues(0, 1, 0);
        vec3.transformQuat(up, up, this.rotation);
        return up;
    }

    /**
     * Reset camera to default position/rotation
     */
    reset() {
        vec3.set(this.position, 0, 0, 1000);
        quat.identity(this.rotation);
        quat.identity(this.target);
        this.updateViewMatrix();
        this.updateViewProjectionMatrix();
    }
}
