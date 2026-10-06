package documents

import (
	"strings"
	"testing"
	"time"
)

// "בקשות החזרה" are SAP Return Requests (ORRR), a different table from
// Returns (ORDN). Reading them through the Returns entry was the bug.
func TestReturnRequestReadsORRR(t *testing.T) {
	for _, raw := range []string{"ReturnRequest", "returnrequest", "ORRR"} {
		docType, ok := normalizeSapDocType(raw)
		if !ok || docType != "ReturnRequest" {
			t.Fatalf("normalizeSapDocType(%q) = %q, %v; want ReturnRequest, true", raw, docType, ok)
		}
	}

	notRepair := "101"
	countQuery, keysQuery, err := buildSapDocumentsQueries("mssql", SapDocumentsQuery{
		DocType:               "ReturnRequest",
		DateFrom:              time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
		DateTo:                time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC),
		WarehouseCodeNotEqual: &notRepair,
		SortBy:                "DocDate",
		SortDir:               "desc",
		Page:                  1,
		PageSize:              10,
	})
	if err != nil {
		t.Fatalf("buildSapDocumentsQueries: %v", err)
	}

	for name, query := range map[string]string{"count": countQuery.Query, "keys": keysQuery.Query} {
		if !strings.Contains(query, "FROM ORRR H") || !strings.Contains(query, "FROM RRR1 L") {
			t.Errorf("%s query does not read ORRR/RRR1:\n%s", name, query)
		}
		if strings.Contains(query, "ORDN") || strings.Contains(query, "RDN1") {
			t.Errorf("%s query still reads Returns tables:\n%s", name, query)
		}
	}
}

func TestReturnsStillReadORDN(t *testing.T) {
	docType, ok := normalizeSapDocType("Returns")
	if !ok || docType != "Returns" {
		t.Fatalf("normalizeSapDocType(Returns) = %q, %v", docType, ok)
	}
	if got := sapDocTableByType["Returns"]; got.Header != "ORDN" || got.Lines != "RDN1" {
		t.Fatalf("Returns maps to %s/%s, want ORDN/RDN1", got.Header, got.Lines)
	}
}
