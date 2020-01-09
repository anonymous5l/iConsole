# iConsole

## About

pure golang communication

this project just for learn `iOS` `iTunes` communication

reference:

[libimobiledevice](https://github.com/libimobiledevice)

## Tools

### devices

list all iOS devices

```bash
./iconsole devices
    
iPhone AnonymousPhone 13.3
    ConnectionType: Network
    UDID: XXXXXXXX-XXXXXXXXXXXXXXXX
iPad AnonymousResearch 13.2.3
    ConnectionType: USB
    UDID: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```
    
### syslog

show all device system log like `deviceconsole`

```base
./iconsole syslog -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

Jan  9 18:18:00 AnonymousResearch backboardd(CoreBrightness)[67] <Notice>: Lcurrent=101.0476 Lr=0.3557 DR=200.0000 factor=0.0000
Jan  9 18:18:00 AnonymousResearch backboardd(CoreBrightness)[67] <Notice>: Lcurrent=101.0476 Lr=0.3557 DR=200.0000 factor=0.0000
Jan  9 18:18:00 AnonymousResearch trustd[118] <Notice>: cert[0]: MissingIntermediate =(leaf)[force]> 0
Jan  9 18:18:00 AnonymousResearch trustd[118] <Notice>: cert[0]: NonEmptySubject =(path)[]> 0
Jan  9 18:18:00 AnonymousResearch trustd[118] <Notice>: cert[0]: MissingIntermediate =(leaf)[force]> 0
Jan  9 18:18:00 AnonymousResearch trustd[118] <Notice>: cert[0]: NonEmptySubject =(path)[]> 0
...
```
    
### simlocation

Simulate device location include convert coordinate u can go anywhere

stander coordinate wgs84

default coordinate gcj02

> Remember: that you have to mount the Developer disk image on your device, if you want to use the `DTSimulateLocation` service.

#### start
```bash
./iconsole simlocation start -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX -lat xx.xxx -lon xx.xxx
```

#### stop
```bash
./iconsole simlocation stop -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

### screenshot

capture device screen file format `tiff` auto save work path. 

format `ScreenShot 2006-01-02 15.04.05.tiff` 

```bash
./iconsole screenshot -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

### sync

enable disable get Wi-Fi communication

```bash
./iconsole sync -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

Device enable WiFi connections
```

```bash
./iconsole sync enable -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

Successd
```

```bash
./iconsole sync disable -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

Successd
```

### relay

relay device port to local normal usage for `debugserver`

transport communication

examples:

```bash
./iconsole relay -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX tcp :1234
```

```bash
./iconsole relay -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX unix /opt/xx
```

```bash
./iconsole relay -u XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX tcp 127.0.0.1:1234
```
