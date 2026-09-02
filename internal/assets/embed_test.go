package assets

import "testing"

func TestHelmetsFS_ContainsSeededPennStateAsset(t *testing.T) {
	data, err := HelmetsFS.ReadFile("helmets/213.png")
	if err != nil {
		t.Fatalf("HelmetsFS.ReadFile(helmets/213.png) failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("HelmetsFS.ReadFile(helmets/213.png) returned empty data")
	}
}
