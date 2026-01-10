package event_network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Tests for TimeUnit constant definitions
func TestTimeUnitConstants(t *testing.T) {
	require.Equal(t, TimeUnit("year"), Year)
	require.Equal(t, TimeUnit("month"), Month)
	require.Equal(t, TimeUnit("day"), Day)
	require.Equal(t, TimeUnit("hour"), Hour)
	require.Equal(t, TimeUnit("minute"), Minute)
	require.Equal(t, TimeUnit("second"), Second)
	require.Equal(t, TimeUnit("millisecond"), Millisecond)
	require.Equal(t, TimeUnit("microsecond"), Microsecond)
}

// Tests for Year conversion
func TestTimeUnit_ToDuration_Year(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One year", 1, time.Hour * 24 * 365},
		{"Two years", 2, time.Hour * 24 * 365 * 2},
		{"Ten years", 10, time.Hour * 24 * 365 * 10},
		{"Zero years", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Year.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Month conversion
func TestTimeUnit_ToDuration_Month(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One month", 1, time.Hour * 24 * 30},
		{"Three months", 3, time.Hour * 24 * 30 * 3},
		{"Twelve months", 12, time.Hour * 24 * 30 * 12},
		{"Zero months", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Month.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Day conversion
func TestTimeUnit_ToDuration_Day(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One day", 1, time.Hour * 24},
		{"Seven days", 7, time.Hour * 24 * 7},
		{"Thirty days", 30, time.Hour * 24 * 30},
		{"Zero days", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Day.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Hour conversion
func TestTimeUnit_ToDuration_Hour(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One hour", 1, time.Hour},
		{"Five hours", 5, time.Hour * 5},
		{"Twenty-four hours", 24, time.Hour * 24},
		{"Zero hours", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Hour.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Minute conversion
func TestTimeUnit_ToDuration_Minute(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One minute", 1, time.Minute},
		{"Thirty minutes", 30, time.Minute * 30},
		{"Sixty minutes", 60, time.Minute * 60},
		{"Zero minutes", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Minute.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Second conversion
func TestTimeUnit_ToDuration_Second(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One second", 1, time.Second},
		{"Ten seconds", 10, time.Second * 10},
		{"Sixty seconds", 60, time.Second * 60},
		{"Zero seconds", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Second.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Millisecond conversion
func TestTimeUnit_ToDuration_Millisecond(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One millisecond", 1, time.Millisecond},
		{"Hundred milliseconds", 100, time.Millisecond * 100},
		{"Thousand milliseconds", 1000, time.Millisecond * 1000},
		{"Zero milliseconds", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Millisecond.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for Microsecond conversion
func TestTimeUnit_ToDuration_Microsecond(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"One microsecond", 1, time.Microsecond},
		{"Thousand microseconds", 1000, time.Microsecond * 1000},
		{"Million microseconds", 1000000, time.Microsecond * 1000000},
		{"Zero microseconds", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Microsecond.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for invalid/unknown TimeUnit
func TestTimeUnit_ToDuration_InvalidUnit(t *testing.T) {
	invalidUnit := TimeUnit("invalid_unit")
	result := invalidUnit.ToDuration(5)
	require.Equal(t, time.Duration(0), result)
}

// Tests for negative values
func TestTimeUnit_ToDuration_NegativeValues(t *testing.T) {
	tests := []struct {
		name     string
		unit     TimeUnit
		input    int
		expected time.Duration
	}{
		{"Negative seconds", Second, -5, time.Second * -5},
		{"Negative minutes", Minute, -10, time.Minute * -10},
		{"Negative hours", Hour, -2, time.Hour * -2},
		{"Negative days", Day, -1, time.Hour * 24 * -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unit.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for large values
func TestTimeUnit_ToDuration_LargeValues(t *testing.T) {
	tests := []struct {
		name     string
		unit     TimeUnit
		input    int
		expected time.Duration
	}{
		{"Large seconds", Second, 1000000, time.Second * 1000000},
		{"Large minutes", Minute, 100000, time.Minute * 100000},
		{"Large hours", Hour, 10000, time.Hour * 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unit.ToDuration(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// Tests for equivalence across time units
func TestTimeUnit_Equivalence(t *testing.T) {
	// Test that 60 seconds equals 1 minute
	sixtySeconds := Second.ToDuration(60)
	oneMinute := Minute.ToDuration(1)
	require.Equal(t, oneMinute, sixtySeconds)

	// Test that 60 minutes equals 1 hour
	sixtyMinutes := Minute.ToDuration(60)
	oneHour := Hour.ToDuration(1)
	require.Equal(t, oneHour, sixtyMinutes)

	// Test that 24 hours equals 1 day
	twentyFourHours := Hour.ToDuration(24)
	oneDay := Day.ToDuration(1)
	require.Equal(t, oneDay, twentyFourHours)

	// Test that 1000 milliseconds equals 1 second
	thousandMilliseconds := Millisecond.ToDuration(1000)
	oneSecond := Second.ToDuration(1)
	require.Equal(t, oneSecond, thousandMilliseconds)

	// Test that 1000000 microseconds equals 1 second
	millionMicroseconds := Microsecond.ToDuration(1000000)
	require.Equal(t, oneSecond, millionMicroseconds)
}

// Tests for time unit ordering
func TestTimeUnit_Ordering(t *testing.T) {
	// Verify that larger time units produce larger durations for same input
	oneYear := Year.ToDuration(1)
	oneMonth := Month.ToDuration(1)
	oneDay := Day.ToDuration(1)
	oneHour := Hour.ToDuration(1)
	oneMinute := Minute.ToDuration(1)
	oneSecond := Second.ToDuration(1)

	require.Greater(t, oneYear, oneMonth)
	require.Greater(t, oneMonth, oneDay)
	require.Greater(t, oneDay, oneHour)
	require.Greater(t, oneHour, oneMinute)
	require.Greater(t, oneMinute, oneSecond)
}

// Tests for TimeUnit receiver method pattern
func TestTimeUnit_ReceiverMethodPattern(t *testing.T) {
	// Test that the method works as a receiver method
	var unit TimeUnit = Hour
	result := unit.ToDuration(3)
	require.Equal(t, time.Hour*3, result)

	// Test with different units
	units := []TimeUnit{Year, Month, Day, Hour, Minute, Second, Millisecond, Microsecond}
	for _, u := range units {
		result := u.ToDuration(1)
		require.NotEqual(t, time.Duration(0), result)
	}
}

// Benchmark tests for conversion performance
func BenchmarkTimeUnit_ToDuration_Year(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Year.ToDuration(10)
	}
}

func BenchmarkTimeUnit_ToDuration_Second(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Second.ToDuration(1000)
	}
}

func BenchmarkTimeUnit_ToDuration_Microsecond(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Microsecond.ToDuration(1000000)
	}
}
