package device

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeQuerier is a minimal in-memory stand-in for the sqlc device queries.
type fakeQuerier struct {
	stored  string
	hasRow  bool
	addErr  error
	addedID string
}

func (f *fakeQuerier) GetDeviceDetail(context.Context) (string, error) {
	if !f.hasRow {
		return "", errors.New("device not found")
	}
	return f.stored, nil
}

func (f *fakeQuerier) AddDevice(_ context.Context, id string) (string, error) {
	if f.addErr != nil {
		return "", f.addErr
	}
	f.addedID = id
	f.hasRow = true
	f.stored = id
	return id, nil
}

func TestIsUsableSerial(t *testing.T) {
	tests := []struct {
		name   string
		serial string
		want   bool
	}{
		{name: "real serial", serial: "Q9KDVVQ5R9", want: true},
		{name: "serial with surrounding whitespace", serial: "  C02XYZ12345\n", want: true},
		{name: "serial containing digits only", serial: "1234567890", want: true},
		{name: "empty", serial: "", want: false},
		{name: "whitespace only", serial: "   \n", want: false},
		{name: "windows placeholder", serial: "To be filled by O.E.M.", want: false},
		{name: "placeholder is case insensitive", serial: "TO BE FILLED BY O.E.M.", want: false},
		{name: "linux none", serial: "None", want: false},
		{name: "oem default string", serial: "Default string", want: false},
		{name: "generic system serial number", serial: "System Serial Number", want: false},
		{name: "zero", serial: "0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isUsableSerial(tt.serial))
		})
	}
}

func TestGetDeviceDetail_ReturnsPersistedIDWithoutResolvingSerial(t *testing.T) {
	querier := &fakeQuerier{hasRow: true, stored: "PERSISTED-123"}
	svc := &service{
		querier: querier,
		serialFn: func(context.Context) (string, error) {
			t.Fatal("serial must not be resolved when a device id is already stored")
			return "", nil
		},
	}

	require.Equal(t, "PERSISTED-123", svc.GetDeviceDetail(context.Background()))
	require.Empty(t, querier.addedID, "no row should have been inserted")
}

func TestGetDeviceDetail_PersistsResolvedSerial(t *testing.T) {
	querier := &fakeQuerier{}
	svc := &service{
		querier: querier,
		serialFn: func(context.Context) (string, error) {
			return "Q9KDVVQ5R9", nil
		},
	}

	require.Equal(t, "Q9KDVVQ5R9", svc.GetDeviceDetail(context.Background()))
	require.Equal(t, "Q9KDVVQ5R9", querier.addedID)
}

func TestGetDeviceDetail_FallsBackToGeneratedIDWhenSerialUnavailable(t *testing.T) {
	tests := []struct {
		name   string
		serial string
		err    error
	}{
		{name: "command failed", err: errors.New("exit status 1")},
		{name: "empty serial", serial: "", err: nil},
		{name: "firmware placeholder", serial: "To be filled by O.E.M.", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := &fakeQuerier{}
			svc := &service{
				querier: querier,
				serialFn: func(context.Context) (string, error) {
					return tt.serial, tt.err
				},
			}

			got := svc.GetDeviceDetail(context.Background())

			require.NotEmpty(t, got, "a fallback id must always be produced")
			require.NotEqual(t, tt.serial, got, "an unusable serial must not be used as the device id")
			_, err := uuid.Parse(got)
			require.NoError(t, err, "the fallback id should be a uuid")
			require.Equal(t, got, querier.addedID, "the fallback id must be persisted")
		})
	}
}

func TestGetDeviceDetail_FallsBackToGeneratedIDWhenPersistingUnknownOSPlaceholder(t *testing.T) {
	querier := &fakeQuerier{}
	svc := &service{
		querier: querier,
		serialFn: func(context.Context) (string, error) {
			return "None", nil
		},
	}

	got := svc.GetDeviceDetail(context.Background())

	_, err := uuid.Parse(got)
	require.NoError(t, err)
	require.Equal(t, got, querier.addedID)
}

func TestGetDeviceDetail_ReturnsComputedIDWhenPersistingFails(t *testing.T) {
	querier := &fakeQuerier{addErr: errors.New("database is locked")}
	svc := &service{
		querier: querier,
		serialFn: func(context.Context) (string, error) {
			return "Q9KDVVQ5R9", nil
		},
	}

	require.Equal(t, "Q9KDVVQ5R9", svc.GetDeviceDetail(context.Background()))
}
