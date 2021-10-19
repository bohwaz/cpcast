package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/azul3d/engine/binpack"
)

var flagWindowId = flag.String("windowid", "", "window id")
var flagDelay = flag.Int("delay", 250, "delay between frames in milliseconds")

func takeScreenshots(stop chan bool, ssfolder string) {
	for {
		select {
		case <-stop:
			return
		}

		filename := path.Join(ssfolder, fmt.Sprintf("%d.png", time.Now().UnixMilli()))
		cmd := exec.Command("screencapture", "-o", fmt.Sprintf("-l%s", *flagWindowId), "-x", filename)
		cmd.Start()

		time.Sleep(time.Millisecond * time.Duration(*flagDelay))
	}
}

type Pixel struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

type Image [][]Pixel

func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) Pixel {
	return Pixel{uint8(r / 257), uint8(g / 257), uint8(b / 257), uint8(a / 257)}
}

func getPixels(filepath string) (Image, error) {
	file, err := os.Open("./image.png")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]Pixel
	for y := 0; y < height; y++ {
		var row []Pixel
		for x := 0; x < width; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return pixels, nil
}

type Rect struct {
	Y1 int
	X1 int
	Y2 int
	X2 int
}

func getDiffAreas(a, b Image) []Rect {
	if len(a) != len(b) || len(a[0]) != len(b[0]) {
		log.Fatalf(
			"getDiffAreas got images of different sizes, %dx%d vs %dx%d",
			len(a[0]), len(a), len(b[0]), len(b),
		)
	}

	w := len(a[0])
	h := len(a)

	seen := make([][]bool, h)
	for i := range seen {
		seen[i] = make([]bool, w)
	}

	var floodfill func(x, y int, r *Rect)
	floodfill = func(x, y int, r *Rect) {
		if x < r.X1 {
			r.X1 = x
		}
		if y < r.Y1 {
			r.Y1 = y
		}
		if x > r.X2 {
			r.X2 = x
		}
		if y > r.Y2 {
			r.Y2 = y
		}
		seen[y][x] = true

		dy := []int{1, -1, 0, 0}
		dx := []int{0, 0, 1, -1}
		for i := 0; i < len(dy); i++ {
			y2 := y + dy[i]
			x2 := x + dx[i]
			if !seen[y2][x2] {
				floodfill(x2, y2, r)
			}
		}
	}

	rects := []Rect{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !seen[y][x] {
				r := Rect{Y1: y, X1: x, Y2: y, X2: x}
				floodfill(x, y, &r)
				rects = append(rects, r)
			}
		}
	}

	return rects
}

type Sprite struct {
	ID int `json:"id"`
	Y1 int `json:"y1"`
	X1 int `json:"x1"`
	Y2 int `json:"y2"`
	X2 int `json:"x2"`
}

type Change struct {
	ID int `json:"id"`
	X  int `json:"x"`
	Y  int `json:"y"`
}

type FrameInfo struct {
	Timestamp int      `json:"timestamp"`
	Changes   []Change `json:"changes"`
}

type ImagePacker struct {
	Images  []Image
	Sprites []*Sprite
}

func (ip *ImagePacker) Len() int {
	return len(ip.Images)
}

func (ip *ImagePacker) Size(n int) (width, height int) {
	img := ip.Images[n]
	return len(img[0]), len(img)
}

func (ip *ImagePacker) Place(n, x, y int) {
	ip.Sprites = append(ip.Sprites, &Sprite{
		ID: n,
		Y1: y,
		X1: x,
		Y2: y + len(ip.Images[n]),
		X2: x + len(ip.Images[n][0]),
	})
}

func (ip *ImagePacker) CreateImage(filepath string) error {
	w, h := binpack.Pack(ip)
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	for _, sprite := range ip.Sprites {
		img := ip.Images[sprite.ID]
		for y := 0; y < len(img); y++ {
			for x := 0; x < len(img[0]); x++ {
				yf, xf := y+sprite.Y1, x+sprite.X1
				off := (yf*w + xf) * 4
				out.Pix[off+0] = img[y][x].R
				out.Pix[off+1] = img[y][x].G
				out.Pix[off+2] = img[y][x].B
				out.Pix[off+3] = img[y][x].A
			}
		}
	}

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	// bufio.NewWriter(
	return png.Encode(f, out)
}

type File struct {
	Path      string
	Timestamp int
}

func Process(files []File, outputFolder string) error {
	images := []Image{}
	frames := []FrameInfo{}

	var lastFrame [][]Pixel
	for i, file := range files {
		frame, err := getPixels(file.Path)
		if err != nil {
			return err
		}

		if i == 0 {
			// TODO: just add the whole image.
			lastFrame = frame
			continue
		}

		diffAreas := getDiffAreas(frame, lastFrame)
		if len(diffAreas) == 0 {
			continue
		}
		lastFrame = frame

		frameInfo := FrameInfo{}
		for _, area := range diffAreas {
			img := make(Image, area.Y2-area.Y1)
			for y := range img {
				img[y] = make([]Pixel, area.X2-area.X1)
				for x := area.X1; x < area.X2; x++ {
					img[y][x] = frame[area.Y1+y][area.X1+x]
				}
			}

			id := len(images)
			images = append(images, img)
			frameInfo.Changes = append(frameInfo.Changes, Change{
				X:  area.X1,
				Y:  area.Y1,
				ID: id,
			})
		}
		frames = append(frames, frameInfo)
	}

	if err := os.MkdirAll(outputFolder, 0700); err != nil {
		return err
	}

	ip := &ImagePacker{
		Images:  images,
		Sprites: []*Sprite{},
	}

	if err := ip.CreateImage(path.Join(outputFolder, "spritesheet.png")); err != nil {
		return err
	}

	output := map[string]interface{}{
		"frames":  frames,
		"sprites": ip.Sprites,
	}

	data, err := json.Marshal(output)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(outputFolder, "data.json"), data, 0644)
}

func main() {
	flag.Parse()

	if *flagWindowId == "" {
		log.Fatalf("-windowid is required")
	}

	ssfolder, err := ioutil.TempDir("", "cpcast_screenshots")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(ssfolder)

	done := make(chan bool)
	go takeScreenshots(done, ssfolder)

	fmt.Printf("press enter to stop")
	fmt.Scanln()
	done <- true

	files, err := ioutil.ReadDir(ssfolder)
	if err != nil {
		log.Fatal(err)
	}

	allFiles := []File{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".png") {
			continue
		}

		timestampStr := strings.TrimSuffix(filename, path.Ext(filename))
		timestamp, err := strconv.Atoi(timestampStr)
		if err != nil {
			log.Println(err)
			continue
		}

		allFiles = append(allFiles, File{
			Path:      path.Join(ssfolder, filename),
			Timestamp: timestamp,
		})
	}

	sort.Slice(allFiles, func(i, j int) bool {
		a := allFiles[i]
		b := allFiles[j]
		return a.Timestamp < b.Timestamp
	})

	// TODO: customize "output" with a flag
	Process(allFiles, "output")
}
