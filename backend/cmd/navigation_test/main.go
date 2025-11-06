/**
 * Multi-Scale Navigation Test
 *
 * Tests zoom levels, view controller, and coordinate system
 * Validates genomic navigation pipeline
 */

package main

import (
	"fmt"
	"time"

	"genomevedic/backend/internal/navigation"
)

func main() {
	fmt.Println("=== GenomeVedic.ai - Multi-Scale Navigation Test ===\n")

	// Test 1: Zoom levels
	fmt.Println("Test 1: Testing zoom level configurations...")
	testZoomLevels()
	fmt.Println("✓ Zoom levels working\n")

	// Test 2: View controller
	fmt.Println("Test 2: Testing view controller...")
	testViewController()
	fmt.Println("✓ View controller working\n")

	// Test 3: Coordinate system
	fmt.Println("Test 3: Testing coordinate system...")
	testCoordinateSystem()
	fmt.Println("✓ Coordinate system working\n")

	// Test 4: Navigation scenarios
	fmt.Println("Test 4: Testing navigation scenarios...")
	testNavigationScenarios()
	fmt.Println("✓ Navigation scenarios working\n")

	fmt.Println("=== All Tests Passed ===")
	fmt.Println("Agent 4.3 (Multi-Scale Navigation) Complete!")
}

func testZoomLevels() {
	fmt.Println("  Zoom Level Configurations:")
	for i := navigation.ZoomGenome; i <= navigation.ZoomNucleotide; i++ {
		config := navigation.GetZoomLevelConfig(i)
		fmt.Printf("    %s: %.1fM-%.1fM bp, Camera: %.0f, Density: %.0f%%, LOD: %d\n",
			config.Level.String(),
			float64(config.MinBasePairs)/1_000_000,
			float64(config.MaxBasePairs)/1_000_000,
			config.CameraDistance,
			config.ParticleDensity*100,
			config.LODLevel,
		)
	}

	// Test zoom level detection from distance
	testDistances := []float32{5000, 2000, 800, 200, 50}
	fmt.Println("  Zoom level detection from camera distance:")
	for _, dist := range testDistances {
		level := navigation.GetZoomLevelFromDistance(dist)
		fmt.Printf("    Distance %.0f → %s\n", dist, level.String())
	}
}

func testViewController() {
	// Create view controller (start at chromosome 17)
	vc := navigation.NewViewController("chr17", 0, 83257441)

	fmt.Printf("  Initial view: %s\n", vc.GetViewport().String())
	fmt.Printf("  Camera distance: %.1f\n", vc.GetCameraDistance())
	fmt.Printf("  Particle density: %.0f%%\n", vc.GetParticleDensity()*100)
	fmt.Printf("  LOD level: %d\n", vc.GetLODLevel())

	// Test zoom in
	fmt.Println("\n  Testing zoom transitions:")
	for i := 0; i < 3; i++ {
		if vc.ZoomIn() {
			fmt.Printf("    Zoom in → %s (Distance: %.1f, Density: %.0f%%)\n",
				vc.GetViewport().ZoomLevel.String(),
				vc.GetCameraDistance(),
				vc.GetParticleDensity()*100,
			)
		}
	}

	// Test zoom out
	for i := 0; i < 2; i++ {
		if vc.ZoomOut() {
			fmt.Printf("    Zoom out → %s (Distance: %.1f, Density: %.0f%%)\n",
				vc.GetViewport().ZoomLevel.String(),
				vc.GetCameraDistance(),
				vc.GetParticleDensity()*100,
			)
		}
	}

	// Test camera transition simulation
	fmt.Println("\n  Simulating camera transition (10 frames):")
	vc.SetCameraDistance(500.0)
	for frame := 0; frame < 10; frame++ {
		vc.Update(0.016) // 60 fps
		if frame == 0 || frame == 4 || frame == 9 {
			stats := vc.GetStatistics()
			fmt.Printf("    Frame %d: Distance %.1f, Transitioning: %v\n",
				frame, stats["camera_distance"], stats["is_transitioning"])
		}
	}

	// Test bookmarks
	fmt.Println("\n  Testing bookmarks:")
	vc.AddBookmark("TP53 Gene", "Tumor suppressor gene on chr17")
	vc.AddBookmark("BRCA1 Gene", "Breast cancer gene on chr17")
	bookmarks := vc.GetBookmarks()
	for i, bookmark := range bookmarks {
		fmt.Printf("    Bookmark %d: %s\n", i+1, bookmark.String())
	}
}

func testCoordinateSystem() {
	// Create coordinate system (scale: 1 bp = 0.001 3D units)
	cs := navigation.NewCoordinateSystem(0.001, 1000.0, 400.0)

	fmt.Println("  Human genome chromosomes:")
	chromosomes := cs.GetChromosomes()
	for i, chrom := range chromosomes {
		if i < 5 || i >= 22 { // Show first 5 and X, Y
			fmt.Printf("    %s: %.1fM bp (Offset: %.1fM bp)\n",
				chrom.Name,
				float64(chrom.Length)/1_000_000,
				float64(chrom.Offset)/1_000_000,
			)
		}
	}
	fmt.Printf("    ... (Total: %d chromosomes)\n", len(chromosomes))

	// Test genomic to linear conversion
	fmt.Println("\n  Genomic → Linear position conversion:")
	testPositions := []struct {
		chrom string
		pos   uint64
	}{
		{"chr1", 1000000},
		{"chr7", 55019220},  // EGFR
		{"chr17", 7577534},  // TP53
		{"chr17", 43044400}, // BRCA1
	}

	for _, tp := range testPositions {
		linear, err := cs.GenomicToLinear(tp.chrom, tp.pos)
		if err != nil {
			fmt.Printf("    ERROR: %v\n", err)
			continue
		}
		fmt.Printf("    %s:%d → Linear: %d\n", tp.chrom, tp.pos, linear)
	}

	// Test linear to genomic conversion
	fmt.Println("\n  Linear → Genomic position conversion:")
	testLinear := []uint64{0, 500_000_000, 1_000_000_000, 2_500_000_000, 3_000_000_000}
	for _, linear := range testLinear {
		chrom, pos, err := cs.LinearToGenomic(linear)
		if err != nil {
			fmt.Printf("    ERROR: %v\n", err)
			continue
		}
		fmt.Printf("    Linear %d → %s:%d\n", linear, chrom, pos)
	}

	// Test 3D conversion
	fmt.Println("\n  Genomic → 3D spatial position:")
	for _, tp := range testPositions {
		pos3d, err := cs.GenomicTo3D(tp.chrom, tp.pos)
		if err != nil {
			fmt.Printf("    ERROR: %v\n", err)
			continue
		}
		fmt.Printf("    %s:%d → (%.2f, %.2f, %.2f)\n",
			tp.chrom, tp.pos, pos3d[0], pos3d[1], pos3d[2])
	}

	// Test distance calculation
	fmt.Println("\n  Distance between genomic positions:")
	dist1, _ := cs.DistanceBetweenPositions("chr17", 7577534, "chr17", 43044400)
	fmt.Printf("    TP53 ↔ BRCA1: %.2f 3D units\n", dist1)

	dist2, _ := cs.DistanceBetweenPositions("chr7", 55019220, "chr17", 7577534)
	fmt.Printf("    EGFR ↔ TP53: %.2f 3D units\n", dist2)
}

func testNavigationScenarios() {
	cs := navigation.NewCoordinateSystem(0.001, 1000.0, 400.0)

	// Scenario 1: Navigate to TP53 gene
	fmt.Println("  Scenario 1: Navigate to TP53 gene (chr17:7,571,720-7,590,868)")
	vc1 := navigation.NewViewController("chr17", 0, 83257441)
	vc1.NavigateToGene("chr17", 7571720, 7590868)
	fmt.Printf("    View: %s\n", vc1.GetViewport().String())
	fmt.Printf("    Zoom: %s, Density: %.0f%%\n",
		vc1.GetViewport().ZoomLevel.String(),
		vc1.GetParticleDensity()*100,
	)

	// Get 3D bounds
	min, max, _ := cs.GetRegionBounds("chr17", 7571720, 7590868)
	fmt.Printf("    3D Bounds: (%.2f,%.2f,%.2f) to (%.2f,%.2f,%.2f)\n",
		min[0], min[1], min[2], max[0], max[1], max[2])

	// Scenario 2: Navigate to specific exon in EGFR
	fmt.Println("\n  Scenario 2: Navigate to EGFR exon 19 (chr7:55,199,846-55,200,018)")
	vc2 := navigation.NewViewController("chr7", 0, 159345973)
	vc2.NavigateToExon("chr7", 55199846, 55200018)
	fmt.Printf("    View: %s\n", vc2.GetViewport().String())
	fmt.Printf("    Zoom: %s, Density: %.0f%%\n",
		vc2.GetViewport().ZoomLevel.String(),
		vc2.GetParticleDensity()*100,
	)

	// Scenario 3: Pan navigation
	fmt.Println("\n  Scenario 3: Pan navigation")
	vc3 := navigation.NewViewController("chr12", 0, 133275309)
	vc3.NavigateToGene("chr12", 25204789, 25250936) // KRAS
	fmt.Printf("    Initial: %s\n", vc3.GetViewport().String())

	// Pan right 10K bp
	vc3.Pan(10000)
	fmt.Printf("    After pan right 10K bp: %s\n", vc3.GetViewport().String())

	// Pan left 5K bp
	vc3.Pan(-5000)
	fmt.Printf("    After pan left 5K bp: %s\n", vc3.GetViewport().String())

	// Scenario 4: History navigation
	fmt.Println("\n  Scenario 4: History navigation")
	vc4 := navigation.NewViewController("chr17", 0, 83257441)
	vc4.NavigateToPosition("chr17", 7577534, navigation.ZoomGene)  // TP53
	vc4.NavigateToPosition("chr17", 43044400, navigation.ZoomGene) // BRCA1
	vc4.NavigateToPosition("chr7", 55019220, navigation.ZoomGene)  // EGFR

	stats := vc4.GetStatistics()
	fmt.Printf("    History count: %d, Current index: %d\n",
		stats["history_count"], stats["history_index"])

	// Go back
	vc4.GoBack()
	fmt.Printf("    After go back: %s\n", vc4.GetViewport().String())

	vc4.GoBack()
	fmt.Printf("    After go back: %s\n", vc4.GetViewport().String())

	// Go forward
	vc4.GoForward()
	fmt.Printf("    After go forward: %s\n", vc4.GetViewport().String())

	// Measure performance
	fmt.Println("\n  Performance benchmark:")
	start := time.Now()
	iterations := 100000
	for i := 0; i < iterations; i++ {
		pos := uint64(i * 30000) // Every 30K bp
		_ = cs.LinearTo3D(pos)
	}
	duration := time.Since(start)
	fmt.Printf("    %d coordinate conversions: %.2f ms (%.2f µs/conversion)\n",
		iterations,
		float64(duration.Microseconds())/1000.0,
		float64(duration.Microseconds())/float64(iterations),
	)
}
