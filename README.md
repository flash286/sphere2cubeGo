# sphere2cube Golang version

Console script written on Go that convert an equirectangular/latlong map into an array of cubemap faces (like you would use to send to OpenGL)

### How to Use:

```bash
go install
go build
./sphere2cubeGo -i panorama.jpg
```

You can specify size of the tile face via param -s

```bash
./sphere2cubeGo -i panorama.jpg -s 2048
```

Also you can specify output dir via param -o


```bash
./sphere2cubeGo -i panorama.jpg -s 2048 -o /path/to/result/dir
```
