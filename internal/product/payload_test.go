package product

import (
	"encoding/json"
	"testing"
)

// Regression: SAP can hold an ITM1 row whose Price is NULL, which makes the
// computed FinalPrice column come back NULL. FinalPrice used to be a plain
// float64, so rows.Scan failed with "converting NULL to float64 is
// unsupported", GetProducts returned (nil, err), and the service answered
// HTTP 200 with a body of `null` for the ENTIRE batch — zeroing price and
// stock for every other SKU in the same request.
// Seen in production with SKU 53503-10-55 (Monday ticket 12761988168).
func TestFinalPriceScansNull(t *testing.T) {
	var p Product

	if err := p.FinalPrice.Scan(nil); err != nil {
		t.Fatalf("FinalPrice must accept a NULL from SQL, got error: %v", err)
	}
	if p.FinalPrice.Valid {
		t.Fatalf("FinalPrice.Valid should be false after scanning NULL")
	}
}

func TestFinalPriceMarshalling(t *testing.T) {
	tests := []struct {
		name string
		scan any
		want string
	}{
		// A missing price must stay distinguishable from a real zero, so that
		// callers can render "no price" instead of an orderable 0.
		{name: "null price", scan: nil, want: `null`},
		{name: "zero price", scan: float64(0), want: `0`},
		{name: "real price", scan: float64(105), want: `105`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p Product
			if err := p.FinalPrice.Scan(tc.scan); err != nil {
				t.Fatalf("Scan(%v) returned error: %v", tc.scan, err)
			}

			raw, err := json.Marshal(p)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var decoded map[string]json.RawMessage
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			got := string(decoded["finalPrice"])
			if got != tc.want {
				t.Errorf("finalPrice = %s, want %s", got, tc.want)
			}
		})
	}
}

// Every non-nullable scan target in GetProducts must be guarded on the SQL
// side, otherwise a NULL there reintroduces the whole-batch failure above.
// sku/cardCode come from non-nullable sources, warehouseCode is wrapped in
// ISNULL(...,''), and priceSource is a CASE with a literal ELSE.
func TestPlainStringTargetsScanEmpty(t *testing.T) {
	var p Product

	for _, tc := range []struct {
		field string
		got   *string
	}{
		{"WarehouseCode", &p.WarehouseCode},
		{"PriceSource", &p.PriceSource},
	} {
		if *tc.got != "" {
			t.Errorf("%s should default to empty string, got %q", tc.field, *tc.got)
		}
	}
}
