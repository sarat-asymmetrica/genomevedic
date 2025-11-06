// Wave 1 Demo - Demonstrates FASTQ streaming, voxel grid, and Vedic coloring
package main

import (
	"fmt"
	"log"

	"genomevedic/backend/internal/loader"
	"genomevedic/backend/internal/spatial"
	"genomevedic/backend/internal/vedic"
	"genomevedic/backend/pkg/types"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("GenomeVedic.ai - Wave 1 Demo")
	fmt.Println("==================================================\n")

	// Demo 1: Digital Root Spatial Hashing
	demoDigitalRootHashing()

	// Demo 2: Vedic Color Mapping
	demoVedicColorMapping()

	// Demo 3: Spatial Voxel Grid
	demoSpatialVoxelGrid()

	// Demo 4: Frustum Culling
	demoFrustumCulling()

	// Demo 5: LOD System
	demoLODSystem()

	fmt.Println("\n==================================================")
	fmt.Println("Wave 1 Demo Complete!")
	fmt.Println("==================================================")
}

func demoDigitalRootHashing() {
	fmt.Println("Demo 1: Digital Root Spatial Hashing")
	fmt.Println("-------------------------------------")

	sequence := "ATGCATGCATGC"
	fmt.Printf("Sequence: %s\n", sequence)

	// Test digital root mapping
	for i := 0; i < len(sequence)-2; i++ {
		coords := loader.SequenceTo3D(sequence, i, i)
		fmt.Printf("Position %2d: Base %c → (%.2f, %.2f, %.2f)\n",
			i, sequence[i], coords.X, coords.Y, coords.Z)
	}

	fmt.Println()
}

func demoVedicColorMapping() {
	fmt.Println("Demo 2: Vedic Color Mapping")
	fmt.Println("----------------------------")

	testSequences := []string{
		"AAAA", // AT-rich
		"GCGC", // GC-rich
		"ATGC", // Balanced
	}

	for _, seq := range testSequences {
		// GC Content coloring
		gcColor := vedic.GCContentColor(seq)
		gc := vedic.ComputeGCContent(seq)
		fmt.Printf("Sequence: %s | GC%%: %.1f | Color: RGB(%d, %d, %d)\n",
			seq, gc.Percent, gcColor.R, gcColor.G, gcColor.B)

		// Digital root coloring
		drColor := vedic.DigitalRootColor(seq)
		dr := vedic.ComputeSequenceDigitalRoot(seq)
		fmt.Printf("  Digital Root: %d (%s) | Color: RGB(%d, %d, %d)\n\n",
			dr, vedic.GetDigitalRootPattern(dr), drColor.R, drColor.G, drColor.B)
	}
}

func demoSpatialVoxelGrid() {
	fmt.Println("Demo 3: Spatial Voxel Grid")
	fmt.Println("---------------------------")

	// Create voxel grid
	grid := spatial.NewVoxelGrid(types.VoxelSize)

	// Create sample particles
	particles := []types.Particle{
		{Position: types.Vector3D{X: 1.0, Y: 2.0, Z: 3.0}, Base: 'A'},
		{Position: types.Vector3D{X: 5.0, Y: 6.0, Z: 7.0}, Base: 'T'},
		{Position: types.Vector3D{X: 10.5, Y: 11.5, Z: 12.5}, Base: 'G'},
		{Position: types.Vector3D{X: 15.0, Y: 16.0, Z: 17.0}, Base: 'C'},
	}

	// Insert particles
	for _, p := range particles {
		grid.Insert(p)
	}

	fmt.Printf("Total voxels: %d\n", grid.GetTotalVoxels())
	fmt.Printf("Grid bounds: Min(%.1f, %.1f, %.1f) Max(%.1f, %.1f, %.1f)\n",
		grid.GetBounds().Min.X, grid.GetBounds().Min.Y, grid.GetBounds().Min.Z,
		grid.GetBounds().Max.X, grid.GetBounds().Max.Y, grid.GetBounds().Max.Z)

	// Query a specific voxel
	voxelID := spatial.SpatialHash(particles[0].Position, types.VoxelSize)
	queriedParticles := grid.Query(voxelID)
	fmt.Printf("Voxel (%d, %d, %d) contains %d particle(s)\n\n",
		voxelID.X, voxelID.Y, voxelID.Z, len(queriedParticles))
}

func demoFrustumCulling() {
	fmt.Println("Demo 4: Frustum Culling")
	fmt.Println("-----------------------")

	// Create camera
	camera := types.Camera{
		Position: types.Vector3D{X: 0, Y: 0, Z: 10},
		Target:   types.Vector3D{X: 0, Y: 0, Z: 0},
		Up:       types.Vector3D{X: 0, Y: 1, Z: 0},
		FOV:      60.0,
		Near:     0.1,
		Far:      100.0,
	}

	// Create voxel grid with particles
	grid := spatial.NewVoxelGrid(types.VoxelSize)

	// Add particles in various positions (some in frustum, some out)
	for x := -50.0; x <= 50.0; x += 10.0 {
		for y := -50.0; y <= 50.0; y += 10.0 {
			particle := types.Particle{
				Position: types.Vector3D{X: x, Y: y, Z: 0},
				Base:     'A',
			}
			grid.Insert(particle)
		}
	}

	allVoxels := grid.GetAllVoxels()
	fmt.Printf("Total voxels: %d\n", len(allVoxels))

	// Perform frustum culling
	culler := spatial.NewFrustumCuller(camera)
	visibleVoxels := culler.CullVoxels(allVoxels)

	fmt.Printf("Visible voxels: %d (%.1f%%)\n",
		len(visibleVoxels), float64(len(visibleVoxels))/float64(len(allVoxels))*100.0)
	fmt.Println()
}

func demoLODSystem() {
	fmt.Println("Demo 5: LOD System")
	fmt.Println("------------------")

	// Create camera
	camera := types.Camera{
		Position: types.Vector3D{X: 0, Y: 0, Z: 100},
		Target:   types.Vector3D{X: 0, Y: 0, Z: 0},
		Up:       types.Vector3D{X: 0, Y: 1, Z: 0},
		FOV:      60.0,
		Near:     0.1,
		Far:      1000.0,
	}

	// Create LOD manager
	lodManager := spatial.NewLODManager(camera)

	// Create voxels at various distances
	distances := []float64{10.0, 50.0, 150.0, 300.0, 600.0}

	for _, dist := range distances {
		// Create voxel at distance
		voxel := &types.Voxel{
			ID: types.VoxelID{X: 0, Y: 0, Z: 0},
			Bounds: types.AABB{
				Min: types.Vector3D{X: dist, Y: -5, Z: -5},
				Max: types.Vector3D{X: dist + 10, Y: 5, Z: 5},
			},
			Particles: make([]types.Particle, 1000), // 1000 particles per voxel
		}

		// Apply LOD
		particles := lodManager.ApplyLOD([]*types.Voxel{voxel})

		reduction := spatial.GetParticleReduction(voxel.LODLevel)
		fmt.Printf("Distance: %.0f → LOD Level: %d (%.0f%% particles) → %d particles rendered\n",
			dist, voxel.LODLevel, reduction*100, len(particles))
	}

	fmt.Println()
}

func init() {
	log.SetFlags(0) // Remove timestamp from logs
}
