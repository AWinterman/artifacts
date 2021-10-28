package artifacts

import (
	"testing"
)

func TestBatchArtifacts(t *testing.T) {
	type args struct {
		batchSize int
		inChan    chan Artifact
	}
	tests := []struct {
		name string
		args args
		want chan []Artifact
	}{
		{
			name: "hello",
			args: args{
				batchSize: 2,
				inChan:    make(chan Artifact),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifacts := BatchArtifacts(tt.args.batchSize, tt.args.inChan)
			go func() { tt.args.inChan <- Artifact{Repository: "1"} }()
			go func() { tt.args.inChan <- Artifact{Repository: "2"} }()
			go func() { tt.args.inChan <- Artifact{Repository: "3"} }()
			go func() { tt.args.inChan <- Artifact{Repository: "4"} }()

			i, ok := <-artifacts

			if !ok {
				t.Fatalf("Oh no %v", i)
			}

			if len(i) > tt.args.batchSize {
				t.Errorf("Found too many elements in batch %v", len(i))
			}

			for _, artifact := range i {
				t.Logf("Batch one: Found %s", artifact.Repository)
			}

			j, ok := <-artifacts

			if !ok {
				t.Fatalf("Oh no %v", j)
			}

			if len(j) > tt.args.batchSize {
				t.Errorf("Found too many elements in batch %v", len(i))
			}

			for _, artifact := range j {
				t.Logf("Batch Two: Found %s", artifact.Repository)
			}

		})
	}
}
