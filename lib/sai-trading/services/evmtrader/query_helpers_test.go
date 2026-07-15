package evmtrader

import (
	"testing"
)

func TestParseWrappedIndexHelper(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{
			name:    "valid MarketIndex",
			input:   "MarketIndex(0)",
			want:    0,
			wantErr: false,
		},
		{
			name:    "valid TokenIndex",
			input:   "TokenIndex(5)",
			want:    5,
			wantErr: false,
		},
		{
			name:    "valid UserTradeIndex",
			input:   "UserTradeIndex(123)",
			want:    123,
			wantErr: false,
		},
		{
			name:    "large number",
			input:   "TokenIndex(999999)",
			want:    999999,
			wantErr: false,
		},
		{
			name:    "missing opening parenthesis",
			input:   "MarketIndex 5)",
			want:    0,
			wantErr: true,
		},
		{
			name:    "missing closing parenthesis",
			input:   "MarketIndex(5",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty parentheses",
			input:   "MarketIndex()",
			want:    0,
			wantErr: true,
		},
		{
			name:    "non-numeric value",
			input:   "MarketIndex(abc)",
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative number",
			input:   "MarketIndex(-5)",
			want:    0,
			wantErr: true,
		},
		{
			name:    "plain number without wrapper",
			input:   "5",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWrappedIndex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWrappedIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseWrappedIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMarketIndex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{
			name:    "valid MarketIndex",
			input:   "MarketIndex(0)",
			want:    0,
			wantErr: false,
		},
		{
			name:    "valid MarketIndex with large number",
			input:   "MarketIndex(100)",
			want:    100,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "MarketIndex 5",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMarketIndex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMarketIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMarketIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTokenIndex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{
			name:    "valid TokenIndex",
			input:   "TokenIndex(1)",
			want:    1,
			wantErr: false,
		},
		{
			name:    "zero TokenIndex",
			input:   "TokenIndex(0)",
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "TokenIndex()",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTokenIndex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTokenIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseTokenIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseUserTradeIndex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{
			name:    "valid UserTradeIndex",
			input:   "UserTradeIndex(0)",
			want:    0,
			wantErr: false,
		},
		{
			name:    "large UserTradeIndex",
			input:   "UserTradeIndex(12345)",
			want:    12345,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "UserTradeIndex[5]",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUserTradeIndex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUserTradeIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseUserTradeIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIndexWithFallback(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedPrefix string
		want           uint64
		wantErr        bool
	}{
		{
			name:           "wrapped format",
			input:          "MarketIndex(5)",
			expectedPrefix: "MarketIndex",
			want:           5,
			wantErr:        false,
		},
		{
			name:           "plain number",
			input:          "5",
			expectedPrefix: "MarketIndex",
			want:           5,
			wantErr:        false,
		},
		{
			name:           "TokenIndex wrapped",
			input:          "TokenIndex(10)",
			expectedPrefix: "TokenIndex",
			want:           10,
			wantErr:        false,
		},
		{
			name:           "plain number for TokenIndex",
			input:          "10",
			expectedPrefix: "TokenIndex",
			want:           10,
			wantErr:        false,
		},
		{
			name:           "invalid both formats",
			input:          "abc",
			expectedPrefix: "MarketIndex",
			want:           0,
			wantErr:        true,
		},
		{
			name:           "negative number",
			input:          "-5",
			expectedPrefix: "MarketIndex",
			want:           0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIndexWithFallback(tt.input, tt.expectedPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIndexWithFallback() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseIndexWithFallback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryUnmarshalIndices(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		want     []uint64
		wantOk   bool
	}{
		{
			name:   "direct uint64 array",
			json:   `[0, 1, 2, 5]`,
			want:   []uint64{0, 1, 2, 5},
			wantOk: true,
		},
		{
			name:   "string array with TokenIndex",
			json:   `["TokenIndex(0)", "TokenIndex(1)", "TokenIndex(5)"]`,
			want:   []uint64{0, 1, 5},
			wantOk: true,
		},
		{
			name:   "string array with plain numbers",
			json:   `["0", "1", "5"]`,
			want:   []uint64{0, 1, 5},
			wantOk: true,
		},
		{
			name:   "wrapped data format",
			json:   `{"data": [0, 1, 2]}`,
			want:   []uint64{0, 1, 2},
			wantOk: true,
		},
		{
			name:   "empty array",
			json:   `[]`,
			want:   nil,
			wantOk: false,
		},
		{
			name:   "invalid json",
			json:   `{invalid}`,
			want:   nil,
			wantOk: false,
		},
		{
			name:   "null",
			json:   `null`,
			want:   nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tryUnmarshalIndices([]byte(tt.json))
			if ok != tt.wantOk {
				t.Errorf("tryUnmarshalIndices() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("tryUnmarshalIndices() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tryUnmarshalIndices()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
