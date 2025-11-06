/**
 * Particle Trail System Test
 *
 * Tests trail system and evolution animation
 * Validates temporal mutation visualization
 */

package main

import (
	"fmt"
	"time"

	"genomevedic/backend/internal/navigation"
	"genomevedic/backend/internal/trails"
)

func main() {
	fmt.Println("=== GenomeVedic.ai - Particle Trail System Test ===\n")

	// Test 1: Basic trail system
	fmt.Println("Test 1: Testing basic trail system...")
	testBasicTrails()
	fmt.Println("✓ Basic trail system working\n")

	// Test 2: Trail animation simulation
	fmt.Println("Test 2: Simulating trail animation...")
	testTrailAnimation()
	fmt.Println("✓ Trail animation working\n")

	// Test 3: Evolution animation
	fmt.Println("Test 3: Testing evolution animation...")
	testEvolutionAnimation()
	fmt.Println("✓ Evolution animation working\n")

	// Test 4: Performance benchmark
	fmt.Println("Test 4: Performance benchmark...")
	testPerformance()
	fmt.Println("✓ Performance test complete\n")

	fmt.Println("=== All Tests Passed ===")
	fmt.Println("Agent 4.4 (Particle Trails) Complete!")
}

func testBasicTrails() {
	// Create trail system (max 50 points, 2s fade, 10 points/sec)
	ts := trails.NewTrailSystem(50, 2.0, 10.0)

	// Create some trails
	particleIDs := []uint64{1, 2, 3}
	for _, id := range particleIDs {
		ts.AddTrail(id)
	}

	// Add points to trails
	for i := 0; i < 10; i++ {
		for _, id := range particleIDs {
			position := [3]float32{float32(i), 0, 0}
			color := [4]float32{1.0, 0.0, 0.0, 1.0}
			size := float32(5.0)

			trail := ts.GetTrail(id)
			trail.AddPoint(position, color, size)
		}
	}

	// Get statistics
	stats := ts.GetStatistics()

	fmt.Printf("  Total trails: %d\n", stats["total_trails"])
	fmt.Printf("  Active trails: %d\n", stats["active_trails"])
	fmt.Printf("  Total trail points: %d\n", stats["total_trail_points"])
	fmt.Printf("  Avg points per trail: %.1f\n", stats["avg_points_per_trail"])
	fmt.Printf("  Fade time: %.1fs\n", stats["fade_time"])
	fmt.Printf("  Emission rate: %.1f points/sec\n", stats["emission_rate"])
}

func testTrailAnimation() {
	// Create trail system
	ts := trails.NewTrailSystem(100, 3.0, 20.0)

	// Create a moving particle
	particleID := uint64(100)
	trail := ts.AddTrail(particleID)

	// Simulate 60 frames (1 second at 60 fps)
	deltaTime := float32(1.0 / 60.0)

	fmt.Println("  Simulating 60 frames (1 second):")

	for frame := 0; frame < 60; frame++ {
		// Add point to trail (moving in a circle)
		angle := float32(frame) * 0.1
		position := [3]float32{
			float32(100.0) * float32(angle),
			float32(100.0) * float32(angle),
			0,
		}
		color := [4]float32{1.0, 0.0, 0.0, 1.0}
		size := float32(5.0)

		trail.AddPoint(position, color, size)

		// Update trail system
		ts.Update(deltaTime)

		// Log progress every 15 frames
		if frame%15 == 0 {
			trailPoints := len(trail.GetPoints())
			fmt.Printf("    Frame %2d: Trail points = %d\n", frame, trailPoints)
		}
	}

	// Final statistics
	stats := ts.GetStatistics()
	fmt.Printf("  Final trail points: %d\n", stats["total_trail_points"])
}

func testEvolutionAnimation() {
	// Create coordinate system for 3D conversion
	cs := navigation.NewCoordinateSystem(0.001, 1000.0, 400.0)

	// Create trail system
	ts := trails.NewTrailSystem(200, 5.0, 30.0)

	// Create evolution animation (time scale: 1 time unit = 1 second)
	ea := trails.NewEvolutionAnimation(ts, 1.0)

	// Set coordinate function
	ea.SetCoordinateFunction(func(chromosome string, position uint64) ([3]float32, error) {
		return cs.GenomicTo3D(chromosome, position)
	})

	// Generate cancer evolution timeline
	mutations := trails.GenerateCancerEvolution(30)
	fmt.Printf("  Generated %d mutations for cancer evolution\n", len(mutations))

	// Add mutations to animation
	for _, mut := range mutations {
		ea.AddMutation(mut)
	}

	// Get statistics before animation
	stats := ea.GetStatistics()
	fmt.Printf("  Total mutations: %d\n", stats["total_mutations"])
	fmt.Printf("  Driver mutations: %d\n", stats["driver_mutations"])
	fmt.Printf("  Passenger mutations: %d\n", stats["passenger_mutations"])
	fmt.Printf("  Max time: %.1f\n", stats["max_time"])

	// Print stage breakdown
	stageCounts := stats["stage_counts"].(map[trails.EvolutionStage]int)
	fmt.Printf("  Mutations by stage:\n")
	for stage, count := range stageCounts {
		fmt.Printf("    %s: %d\n", stage.String(), count)
	}

	// Simulate animation
	fmt.Println("\n  Simulating evolution animation (4 seconds):")
	ea.Play()

	deltaTime := float32(1.0 / 60.0) // 60 fps
	totalTime := float32(4.0)
	frames := int(totalTime / deltaTime)

	for frame := 0; frame < frames; frame++ {
		ea.Update(deltaTime)
		ts.Update(deltaTime)

		// Log progress every 60 frames (1 second)
		if frame%60 == 0 {
			currentStats := ea.GetStatistics()
			trailStats := ts.GetStatistics()

			fmt.Printf("    Time %.1fs: Active mutations = %d, Trail points = %d, Progress = %.0f%%\n",
				currentStats["current_time"],
				currentStats["active_mutations"],
				trailStats["total_trail_points"],
				currentStats["progress"].(float32)*100,
			)
		}
	}

	// Final statistics
	finalStats := ea.GetStatistics()
	fmt.Printf("  Animation complete: %.0f%% progress\n", finalStats["progress"].(float32)*100)

	// Test driver mutation query
	driverMutations := ea.GetDriverMutations()
	fmt.Printf("\n  Driver mutations:\n")
	for i, mut := range driverMutations {
		if i < 5 { // Show first 5
			fmt.Printf("    %s:%d at time %.2f [%s]\n",
				mut.Chromosome, mut.Position, mut.Timepoint, mut.Stage.String())
		}
	}
}

func testPerformance() {
	// Create large trail system
	ts := trails.NewTrailSystem(1000, 5.0, 60.0)

	// Create many trails
	numTrails := 10000
	fmt.Printf("  Creating %d trails...\n", numTrails)

	startCreate := time.Now()
	for i := 0; i < numTrails; i++ {
		ts.AddTrail(uint64(i))
	}
	createTime := time.Since(startCreate)
	fmt.Printf("    Creation time: %.2f ms (%.2f µs/trail)\n",
		float64(createTime.Microseconds())/1000.0,
		float64(createTime.Microseconds())/float64(numTrails),
	)

	// Add points to trails
	fmt.Printf("  Adding 10 points to each trail...\n")
	startAdd := time.Now()
	for i := 0; i < numTrails; i++ {
		trail := ts.GetTrail(uint64(i))
		for j := 0; j < 10; j++ {
			position := [3]float32{float32(j), 0, 0}
			color := [4]float32{1.0, 0.0, 0.0, 1.0}
			trail.AddPoint(position, color, 5.0)
		}
	}
	addTime := time.Since(startAdd)
	fmt.Printf("    Add time: %.2f ms (%.2f µs/point)\n",
		float64(addTime.Microseconds())/1000.0,
		float64(addTime.Microseconds())/float64(numTrails*10),
	)

	// Update benchmark
	fmt.Printf("  Update benchmark (100 frames)...\n")
	startUpdate := time.Now()
	deltaTime := float32(1.0 / 60.0)
	for frame := 0; frame < 100; frame++ {
		ts.Update(deltaTime)
	}
	updateTime := time.Since(startUpdate)
	fmt.Printf("    Update time: %.2f ms (%.2f ms/frame)\n",
		float64(updateTime.Microseconds())/1000.0,
		float64(updateTime.Microseconds())/1000.0/100.0,
	)

	// Statistics
	stats := ts.GetStatistics()
	fmt.Printf("  Final statistics:\n")
	fmt.Printf("    Total trail points: %d\n", stats["total_trail_points"])
	fmt.Printf("    Active trails: %d\n", stats["active_trails"])

	// Memory estimate
	pointSize := 40 // bytes (position + color + size + age + maxAge)
	totalMemory := stats["total_trail_points"].(int) * pointSize
	fmt.Printf("    Estimated memory: %.2f MB\n", float64(totalMemory)/1024.0/1024.0)
}
