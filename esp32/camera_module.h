#ifndef CAMERA_MODULE_H
#define CAMERA_MODULE_H

#include <Arduino.h>

void init_camera();
void take_and_upload_photo(const String& uploadUrl);

#endif // CAMERA_MODULE_H
