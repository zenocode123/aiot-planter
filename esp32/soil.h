#ifndef SOIL_H
#define SOIL_H

#include <Arduino.h>

class Soil {
  private:
    int _pin, _dry, _wet;

  public:
    Soil(int pin, int dry, int wet);

    int readPercent();

    int readRaw();
};

#endif