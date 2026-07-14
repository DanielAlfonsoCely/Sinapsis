package handlers

import (
	"testing"
	"time"
)

// parseFechaHora interpreta el valor de un <input type="datetime-local">:
// "YYYY-MM-DDTHH:MM", con segundos opcionales. Rechaza fechas sin hora.
//
// No la usa ningún handler (Create parsea a mano con time.ParseInLocation
// para fijar la zona horaria de Bogotá); vive aquí, junto a su test, como
// utilidad de parseo probada por si se vuelve a necesitar.
func parseFechaHora(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05", s)
}

func TestParseFechaHora(t *testing.T) {
	t.Run("acepta datetime-local", func(t *testing.T) {
		got, err := parseFechaHora("2026-07-15T10:30")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if got.Hour() != 10 || got.Minute() != 30 || got.Day() != 15 {
			t.Errorf("fecha mal parseada: %v", got)
		}
	})
	t.Run("acepta segundos opcionales", func(t *testing.T) {
		if _, err := parseFechaHora("2026-07-15T10:30:45"); err != nil {
			t.Errorf("debería aceptar segundos: %v", err)
		}
	})
	t.Run("rechaza fecha sin hora", func(t *testing.T) {
		if _, err := parseFechaHora("2026-07-15"); err == nil {
			t.Error("una fecha sin hora debería ser rechazada")
		}
	})
	t.Run("rechaza basura", func(t *testing.T) {
		if _, err := parseFechaHora("mañana a las 3"); err == nil {
			t.Error("un texto libre debería ser rechazado")
		}
	})
}
