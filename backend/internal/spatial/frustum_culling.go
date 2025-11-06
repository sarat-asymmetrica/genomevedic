// Package spatial - Frustum culling for camera view
package spatial

import (
	"math"
	"genomevedic/backend/pkg/types"
)

// FrustumCuller performs frustum culling to determine visible voxels
type FrustumCuller struct {
	planes types.FrustumPlanes
}

// NewFrustumCuller creates a new frustum culler from a camera
func NewFrustumCuller(camera types.Camera) *FrustumCuller {
	planes := ExtractFrustumPlanes(camera)
	return &FrustumCuller{
		planes: planes,
	}
}

// CullVoxels filters voxels to only those visible in the frustum
// Returns visible voxels (target: ~1% of total voxels)
func (fc *FrustumCuller) CullVoxels(voxels []*types.Voxel) []*types.Voxel {
	visible := make([]*types.Voxel, 0, len(voxels)/100) // Pre-allocate for ~1%

	for _, voxel := range voxels {
		if fc.IsVoxelVisible(voxel.Bounds) {
			voxel.Visible = true
			visible = append(visible, voxel)
		} else {
			voxel.Visible = false
		}
	}

	return visible
}

// IsVoxelVisible tests if a voxel's AABB is visible in the frustum
func (fc *FrustumCuller) IsVoxelVisible(bounds types.AABB) bool {
	// Test AABB against all 6 frustum planes
	for _, plane := range fc.planes {
		if !testAABBPlane(bounds, plane) {
			return false // Outside this plane
		}
	}
	return true // Inside all planes
}

// ExtractFrustumPlanes extracts the 6 frustum planes from a camera
// Planes: left, right, top, bottom, near, far
func ExtractFrustumPlanes(camera types.Camera) types.FrustumPlanes {
	// Build view-projection matrix
	viewMatrix := buildViewMatrix(camera)
	projMatrix := buildProjectionMatrix(camera)
	vpMatrix := multiplyMatrices(projMatrix, viewMatrix)

	// Extract planes from view-projection matrix
	var planes types.FrustumPlanes

	// Left plane: row4 + row1
	planes[0] = types.Plane{
		A: vpMatrix[3] + vpMatrix[0],
		B: vpMatrix[7] + vpMatrix[4],
		C: vpMatrix[11] + vpMatrix[8],
		D: vpMatrix[15] + vpMatrix[12],
	}

	// Right plane: row4 - row1
	planes[1] = types.Plane{
		A: vpMatrix[3] - vpMatrix[0],
		B: vpMatrix[7] - vpMatrix[4],
		C: vpMatrix[11] - vpMatrix[8],
		D: vpMatrix[15] - vpMatrix[12],
	}

	// Top plane: row4 - row2
	planes[2] = types.Plane{
		A: vpMatrix[3] - vpMatrix[1],
		B: vpMatrix[7] - vpMatrix[5],
		C: vpMatrix[11] - vpMatrix[9],
		D: vpMatrix[15] - vpMatrix[13],
	}

	// Bottom plane: row4 + row2
	planes[3] = types.Plane{
		A: vpMatrix[3] + vpMatrix[1],
		B: vpMatrix[7] + vpMatrix[5],
		C: vpMatrix[11] + vpMatrix[9],
		D: vpMatrix[15] + vpMatrix[13],
	}

	// Near plane: row4 + row3
	planes[4] = types.Plane{
		A: vpMatrix[3] + vpMatrix[2],
		B: vpMatrix[7] + vpMatrix[6],
		C: vpMatrix[11] + vpMatrix[10],
		D: vpMatrix[15] + vpMatrix[14],
	}

	// Far plane: row4 - row3
	planes[5] = types.Plane{
		A: vpMatrix[3] - vpMatrix[2],
		B: vpMatrix[7] - vpMatrix[6],
		C: vpMatrix[11] - vpMatrix[10],
		D: vpMatrix[15] - vpMatrix[14],
	}

	// Normalize planes
	for i := range planes {
		planes[i] = normalizePlane(planes[i])
	}

	return planes
}

// testAABBPlane tests if an AABB is on the positive side of a plane
func testAABBPlane(bounds types.AABB, plane types.Plane) bool {
	// Get positive vertex (farthest point in plane normal direction)
	pVertex := bounds.Min

	if plane.A >= 0 {
		pVertex.X = bounds.Max.X
	}
	if plane.B >= 0 {
		pVertex.Y = bounds.Max.Y
	}
	if plane.C >= 0 {
		pVertex.Z = bounds.Max.Z
	}

	// Test if positive vertex is inside plane
	distance := plane.A*pVertex.X + plane.B*pVertex.Y + plane.C*pVertex.Z + plane.D
	return distance >= 0
}

// normalizePlane normalizes a plane equation
func normalizePlane(plane types.Plane) types.Plane {
	length := math.Sqrt(plane.A*plane.A + plane.B*plane.B + plane.C*plane.C)
	if length == 0 {
		return plane
	}

	return types.Plane{
		A: plane.A / length,
		B: plane.B / length,
		C: plane.C / length,
		D: plane.D / length,
	}
}

// buildViewMatrix builds a view matrix from camera
func buildViewMatrix(camera types.Camera) [16]float64 {
	// Compute camera coordinate system
	forward := normalize(subtract(camera.Target, camera.Position))
	right := normalize(cross(forward, camera.Up))
	up := cross(right, forward)

	// Build view matrix (column-major)
	return [16]float64{
		right.X, up.X, -forward.X, 0,
		right.Y, up.Y, -forward.Y, 0,
		right.Z, up.Z, -forward.Z, 0,
		-dot(right, camera.Position), -dot(up, camera.Position), dot(forward, camera.Position), 1,
	}
}

// buildProjectionMatrix builds a perspective projection matrix
func buildProjectionMatrix(camera types.Camera) [16]float64 {
	aspect := 16.0 / 9.0 // Default aspect ratio
	fov := camera.FOV * math.Pi / 180.0
	tanHalfFOV := math.Tan(fov / 2.0)

	f := 1.0 / tanHalfFOV
	rangeInv := 1.0 / (camera.Near - camera.Far)

	// Perspective projection matrix (column-major)
	return [16]float64{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (camera.Near + camera.Far) * rangeInv, -1,
		0, 0, camera.Near * camera.Far * rangeInv * 2, 0,
	}
}

// multiplyMatrices multiplies two 4x4 matrices
func multiplyMatrices(a, b [16]float64) [16]float64 {
	var result [16]float64

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				result[i*4+j] += a[i*4+k] * b[k*4+j]
			}
		}
	}

	return result
}

// Vector math helpers
func subtract(a, b types.Vector3D) types.Vector3D {
	return types.Vector3D{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func normalize(v types.Vector3D) types.Vector3D {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if length == 0 {
		return v
	}
	return types.Vector3D{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

func cross(a, b types.Vector3D) types.Vector3D {
	return types.Vector3D{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func dot(a, b types.Vector3D) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}
