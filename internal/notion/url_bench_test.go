package notion

import "testing"

func BenchmarkExtractIDFromNotionURL(b *testing.B) {
	input := "https://www.notion.so/Some-Page-12345678123412341234123456789012"
	for i := 0; i < b.N; i++ {
		_, _ = ExtractIDFromNotionURL(input)
	}
}
