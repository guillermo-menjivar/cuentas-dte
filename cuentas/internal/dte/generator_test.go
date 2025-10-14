package dte

import "testing"

func TestGenerateNumeroControl(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		name          string
		tipoDte       string
		codEstable    string
		codPuntoVenta string
		sequence      int64
		want          string
	}{
		{
			name:          "casa matriz",
			tipoDte:       "01",
			codEstable:    "M001",
			codPuntoVenta: "P001",
			sequence:      7,
			want:          "DTE-01-M001P001-000000000000007",
		},
		{
			name:          "sucursal",
			tipoDte:       "01",
			codEstable:    "S042",
			codPuntoVenta: "P003",
			sequence:      123,
			want:          "DTE-01-S042P003-000000000000123",
		},
		{
			name:          "bodega",
			tipoDte:       "01",
			codEstable:    "B005",
			codPuntoVenta: "P001",
			sequence:      1,
			want:          "DTE-01-B005P001-000000000000001",
		},
		{
			name:          "patio",
			tipoDte:       "01",
			codEstable:    "P012",
			codPuntoVenta: "P002",
			sequence:      99,
			want:          "DTE-01-P012P002-000000000000099",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.GenerateNumeroControl(tt.tipoDte, tt.codEstable, tt.codPuntoVenta, tt.sequence)
			if got != tt.want {
				t.Errorf("GenerateNumeroControl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatEstablishmentCode(t *testing.T) {
	tests := []struct {
		tipo   string
		number int
		want   string
	}{
		{"02", 1, "M001"},  // Casa Matriz = "02" → M
		{"01", 42, "S042"}, // Sucursal = "01" → S
		{"04", 5, "B005"},  // Bodega = "04" → B
		{"07", 12, "P012"}, // Patio = "07" → P
	}

	for _, tt := range tests {
		got := FormatEstablishmentCode(tt.tipo, tt.number)
		if got != tt.want {
			t.Errorf("FormatEstablishmentCode(%s, %d) = %v, want %v", tt.tipo, tt.number, got, tt.want)
		}
	}
}
