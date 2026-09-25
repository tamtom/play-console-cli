package onetimeproducts

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func TestOneTimeProductHumanOutputGolden(t *testing.T) {
	product := &androidpublisher.OneTimeProduct{
		PackageName: "com.example.app",
		ProductId:   "coins.v1",
		Listings: []*androidpublisher.OneTimeProductListing{
			{LanguageCode: "en-US", Title: "Coins", Description: "A coin pack"},
		},
		PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{
			{PurchaseOptionId: "default"},
		},
		RegionsVersion: &androidpublisher.RegionsVersion{Version: "2025/02"},
	}
	list := &androidpublisher.ListOneTimeProductsResponse{OneTimeProducts: []*androidpublisher.OneTimeProduct{product}}

	tests := []struct {
		name   string
		value  any
		format string
		golden string
	}{
		{name: "list table", value: list, format: "table", golden: "list.table.golden"},
		{name: "list markdown", value: list, format: "markdown", golden: "list.markdown.golden"},
		{name: "get table", value: product, format: "table", golden: "get.table.golden"},
		{name: "get markdown", value: product, format: "markdown", golden: "get.markdown.golden"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			ctx := shared.ContextWithIO(context.Background(), &stdout, &bytes.Buffer{})
			if err := shared.PrintOutputContext(ctx, test.value, test.format, false); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", test.golden)
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != string(want) {
				t.Fatalf("output differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, stdout.String(), want)
			}
		})
	}
}
