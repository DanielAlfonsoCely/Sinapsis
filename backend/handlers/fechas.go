package handlers

import (
	"fmt"
	"time"
)

// La agenda de citas va de 06:00 a 19:30 (última cita del día), en
// intervalos de media hora.
const (
	horaInicioMin = 6 * 60
	horaFinMin    = 19*60 + 30
)

// horarioValido indica si una fecha/hora cae en la grilla de citas.
func horarioValido(t time.Time) bool {
	minutosDelDia := t.Hour()*60 + t.Minute()
	return t.Minute()%30 == 0 && minutosDelDia >= horaInicioMin && minutosDelDia <= horaFinMin
}

// slotsDelDia devuelve las franjas horarias del día ("HH:MM"), de 06:00 a
// 19:30 en intervalos de media hora.
func slotsDelDia() []string {
	slots := make([]string, 0, (horaFinMin-horaInicioMin)/30+1)
	for m := horaInicioMin; m <= horaFinMin; m += 30 {
		slots = append(slots, fmt.Sprintf("%02d:%02d", m/60, m%60))
	}
	return slots
}
