package update_status

import "testing"

func TestUpdateStatusString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status UpdateStatus
		value  byte
		want   string
	}{
		{name: "accepted", status: UpdateStatusAccepted, value: 0, want: "accepted"},
		{name: "running", status: UpdateStatusRunning, value: 1, want: "running"},
		{name: "succeeded", status: UpdateStatusSucceeded, value: 2, want: "succeeded"},
		{name: "failed", status: UpdateStatusFailed, value: 3, want: "failed"},
		{name: "unknown", status: UpdateStatus(255), value: 255, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if byte(tt.status) != tt.value {
				t.Fatalf("numeric value = %d, want %d", tt.status, tt.value)
			}
			if got := tt.status.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
