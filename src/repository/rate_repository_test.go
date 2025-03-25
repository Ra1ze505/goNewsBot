package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestRateRepository_SaveRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewRateRepository(db)

	rate := &Rate{
		Date: time.Now(),
		Data: json.RawMessage(`{"USD":{"Value":90.0,"Previous":89.0},"EUR":{"Value":100.0,"Previous":99.0}}`),
	}

	mock.ExpectExec("INSERT INTO rates").
		WithArgs(rate.Date, rate.Data).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveRate(rate)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRateRepository_GetRates(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewRateRepository(db)

	expectedRates := &Rates{
		USD: CurrencyRate{
			ID:       "R01235",
			NumCode:  "840",
			CharCode: "USD",
			Nominal:  1,
			Name:     "Доллар США",
			Value:    90.0,
			Previous: 89.0,
		},
		EUR: CurrencyRate{
			ID:       "R01239",
			NumCode:  "978",
			CharCode: "EUR",
			Nominal:  1,
			Name:     "Евро",
			Value:    100.0,
			Previous: 99.0,
		},
	}

	rateData, _ := json.Marshal(expectedRates)
	rate := &Rate{
		Date: time.Now(),
		Data: rateData,
	}

	rows := sqlmock.NewRows([]string{"date", "data"}).
		AddRow(rate.Date, rate.Data)

	mock.ExpectQuery("SELECT date, data FROM rates").
		WillReturnRows(rows)

	actualRates, err := repo.GetRates()
	assert.NoError(t, err)
	assert.Equal(t, expectedRates.USD.Value, actualRates.USD.Value)
	assert.Equal(t, expectedRates.EUR.Value, actualRates.EUR.Value)
	assert.Equal(t, expectedRates.USD.Previous, actualRates.USD.Previous)
	assert.Equal(t, expectedRates.EUR.Previous, actualRates.EUR.Previous)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
