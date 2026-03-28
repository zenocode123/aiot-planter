#include "soil.h"

Soil::Soil(int pin, int dry, int wet)
  : _pin(pin), _dry(dry), _wet(wet) {
  pinMode(_pin, INPUT);
}

int Soil::readPercent() {
  int raw = analogRead(_pin);

  return constrain(map(raw, _dry, _wet, 0, 100), 0, 100);
}

int Soil::readRaw() {
  return analogRead(_pin);
}