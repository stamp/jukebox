#include <PacketSerial.h>

#include <Keypad.h>
#include <SPI.h>
#include <Adafruit_GFX.h>
#include <Max72xxPanel.h>
#include <Adafruit_NeoPixel.h>

int pinCS = 53;
int numberOfHorizontalDisplays = 1;
int numberOfVerticalDisplays = 1;

const byte ROWS = 8;
const byte COLS = 5;
char keys[ROWS][COLS] = {
    {0x01,0x02,0x03,0x04,0x05},
    {0x06,0x07,0x08,0x09,0x0A},
    {0x0B,0x0C,0x0D,0x0E,0x0F},
    {0x10,0x11,0x12,0x13,0x14},
    {0x15,0x16,0x17,0x18,0x19},
    {0x1A,0x1B,0x1C,0x1D,0x1E},
    {0x1F,0x20,0x21,0x22,0x23},
    {0x24,0x25,0x26,0x27,0x28},
};
byte rowPins[ROWS] = {A0, A1, A2, A3, A4, A5, A6, A7}; 
byte colPins[COLS] = {2, 3, 4, 5, 6}; 

Keypad keypad = Keypad( makeKeymap(keys), rowPins, colPins, ROWS, COLS );

Max72xxPanel matrix = Max72xxPanel(pinCS, 1, 1);

#define PIN 9
#define NUM_LEDS 156
Adafruit_NeoPixel strip = Adafruit_NeoPixel(NUM_LEDS, PIN, NEO_GRB + NEO_KHZ800);

PacketSerial serial;

void setup() {  
  serial.setPacketHandler(&onPacket);
  serial.begin(115200);
  
  strip.begin();
  for(int i=0; i<strip.numPixels(); i++) {
    strip.setPixelColor(i, strip.Color(255,0,0));
  }
  strip.show();

  matrix.setIntensity(15);
  matrix.fillScreen(HIGH);
  matrix.write();
  
  delay(1000);
  matrix.fillScreen(LOW);
  matrix.write();
}


void loop() {
    char key = keypad.getKey();

    if (key) {
      uint8_t myPacket[] { key };
      serial.send(myPacket, 1);
    }
    
    serial.update();
}

void onPacket(const uint8_t* tmp, uint16_t size)
{  
  int pointer = 0;
  for ( byte i = 0; i <=5 ; i++ ) {
    byte incomingByte = tmp[pointer];
    pointer++;
    
    matrix.drawPixel(i,0,(incomingByte & 0x01) != 0);
    matrix.drawPixel(i,1,(incomingByte & 0x02) != 0);
    matrix.drawPixel(i,2,(incomingByte & 0x04) != 0);
    matrix.drawPixel(i,3,(incomingByte & 0x08) != 0);
    matrix.drawPixel(i,4,(incomingByte & 0x10) != 0);
    matrix.drawPixel(i,5,(incomingByte & 0x20) != 0);
    matrix.drawPixel(i,6,(incomingByte & 0x40) != 0);
    matrix.drawPixel(i,7,(incomingByte & 0x80) != 0);
  }
  matrix.write();
  
  int i = 0;
  while(pointer+3 < size) {
    byte r = tmp[pointer];
    byte g = tmp[pointer+1];
    byte b = tmp[pointer+2];
    pointer += 3;
    
    strip.setPixelColor(i, strip.Color(b,r,g));
    strip.setPixelColor(NUM_LEDS-i-3, strip.Color(b,r,g));
    i++;
  }
  strip.show();
}
