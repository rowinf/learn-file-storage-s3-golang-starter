package main

import "testing"

func Add(a, b int) int {
	return a + b
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 2, 3, 5},
		{"zero", 0, 0, 0},
		{"negative numbers", -2, -3, -5},
		{"mixed", -1, 4, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Add(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestGetVideoAspectRatio(t *testing.T) {
	verticalPath := "samples/boots-video-vertical.mp4"
	horizontalPath := "samples/boots-video-horizontal.mp4"
	tests := []struct {
		path     string
		expected string
	}{
		{verticalPath, "9:16"},
		{horizontalPath, "16:9"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ratio, err := GetVideoAspectRatio(tt.path)
			if err != nil {
				t.Errorf("GetVideoAspectRatio %v", err)
			}
			if ratio != tt.expected {
				t.Errorf("GetVideoAspectRatio %s, %s", ratio, horizontalPath)
			}
		})
	}
}
