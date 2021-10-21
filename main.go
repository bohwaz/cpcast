package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"math"
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
var flagOutputFolder = flag.String("output", "", "output folder")

func takeScreenshots(stop chan bool, ssfolder string) {
	for {
		select {
		case <-stop:
			return
		default:
		}

		filename := path.Join(ssfolder, fmt.Sprintf("%d.png", time.Now().UnixMilli()))
		args := []string{"-o", fmt.Sprintf("-l%s", *flagWindowId), "-x", filename}
		cmd := exec.Command("screencapture", args...)
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
	file, err := os.Open(filepath)
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

func expandRect(r *Rect, w, h, delta int) {
	r.X1 -= delta
	if r.X1 < 0 {
		r.X1 = 0
	}

	r.Y1 -= delta
	if r.Y1 < 0 {
		r.Y1 = 0
	}

	r.X2 += delta
	if r.X2 >= w {
		r.X2 = w - 1
	}

	r.Y2 += delta
	if r.Y2 >= h {
		r.Y2 = h - 1
	}
}

func diff(a, b Image) []Rect {
	if len(a) != len(b) || len(a[0]) != len(b[0]) {
		log.Fatalf(
			"diff got images of different sizes, %dx%d vs %dx%d",
			len(a[0]), len(a), len(b[0]), len(b),
		)
	}

	w := len(a[0])
	h := len(a)

	seen := make([][]bool, h)
	for y := range seen {
		seen[y] = make([]bool, w)
	}

	areDifferent := func(x, y int) bool {
		pa := a[y][x]
		pb := b[y][x]

		distance := math.Abs(float64(pa.R-pb.R)) +
			math.Abs(float64(pa.G-pb.G)) +
			math.Abs(float64(pa.B-pb.B)) +
			math.Abs(float64(pa.A-pb.A))

		return distance > 6 // random number lol
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

		for x2 := x - 2; x2 <= x+2; x2++ {
			for y2 := y - 2; y2 <= y+2; y2++ {
				if 0 <= x2 && x2 < w && 0 <= y2 && y2 < h {
					if areDifferent(x2, y2) && !seen[y2][x2] {
						floodfill(x2, y2, r)
					}
				}
			}
		}
	}

	rects := []Rect{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if areDifferent(x, y) && !seen[y][x] {
				r := Rect{Y1: y, X1: x, Y2: y, X2: x}
				floodfill(x, y, &r)
				expandRect(&r, w, h, 4)
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

func parseFlags() {
	flag.Parse()

	if *flagWindowId == "" {
		log.Fatalf("-windowid is required")
	}
	if *flagDelay == 0 {
		log.Fatalf("-delay is required and can't be 0")
	}
	if *flagOutputFolder == "" {
		log.Fatalf("-output is required and can't be 0")
	}
}

func main() {
	parseFlags()

	ssfolder, err := ioutil.TempDir("", "cpcast_screenshots")
	fmt.Printf("%s\n", ssfolder)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(ssfolder)

	done := make(chan bool)
	go takeScreenshots(done, ssfolder)

	log.Printf("press enter to stop")
	fmt.Scanln()
	done <- true

	files, err := ioutil.ReadDir(ssfolder)
	if err != nil {
		log.Fatal(err)
	}

	type File struct {
		Path      string
		Timestamp int
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
		return allFiles[i].Timestamp < allFiles[j].Timestamp
	})

	images := []Image{}

	type Change struct {
		ID int `json:"-"`
		X  int `json:"x"`
		Y  int `json:"y"`
		X1 int `json:"x1"`
		Y1 int `json:"y1"`
		X2 int `json:"x2"`
		Y2 int `json:"y2"`
	}

	type FrameInfo struct {
		Timestamp int       `json:"timestamp"`
		Changes   []*Change `json:"changes"`
	}

	frames := []*FrameInfo{}

	var lastFrame [][]Pixel
	for i, file := range allFiles {
		// log.Printf("%d) %s", i, file.Path)

		frame, err := getPixels(file.Path)
		if err != nil {
			log.Fatal(err)
		}

		var diffRegions []Rect

		if i == 0 {
			region := Rect{X1: 0, Y1: 0, X2: len(frame[0]) - 1, Y2: len(frame) - 1}
			diffRegions = []Rect{region}
		} else {
			diffRegions = diff(frame, lastFrame)
			if len(diffRegions) > 50 {
				// if we have a fuckload of tiny regions, just combine into one region
				superRegion := diffRegions[0]
				for _, region := range diffRegions {
					if region.X1 < superRegion.X1 {
						superRegion.X1 = region.X1
					}
					if region.X2 > superRegion.X2 {
						superRegion.X2 = region.X2
					}
					if region.Y1 < superRegion.Y1 {
						superRegion.Y1 = region.Y1
					}
					if region.Y2 > superRegion.Y2 {
						superRegion.Y2 = region.Y2
					}
				}
				expandRect(&superRegion, len(frame[0]), len(frame), 4)
				diffRegions = []Rect{superRegion}
			}
		}

		lastFrame = frame

		if len(diffRegions) == 0 {
			continue
		}

		log.Printf("frame %d, found %d regions", i, len(diffRegions))

		frameInfo := &FrameInfo{
			Timestamp: file.Timestamp,
		}

		for _, region := range diffRegions {
			// print out first 20 regions to get a sense
			/*
				if i < 20 {
					log.Printf(
						"region found: top = %d, left = %d, right = %d, bottom = %d",
						region.Y1, region.X1, region.X2, region.Y2,
					)
				}
			*/

			frameInfo.Changes = append(frameInfo.Changes, &Change{
				X:  region.X1,
				Y:  region.Y1,
				ID: len(images),
			})

			img := make(Image, region.Y2-region.Y1+1)
			for y := range img {
				w := region.X2 - region.X1 + 1
				img[y] = make([]Pixel, w)
				for x := 0; x < w; x++ {
					img[y][x] = frame[region.Y1+y][region.X1+x]
				}
			}
			images = append(images, img)
		}
		frames = append(frames, frameInfo)
	}

	if err := os.MkdirAll(*flagOutputFolder, 0700); err != nil {
		log.Fatal(err)
	}

	log.Printf("gathered %d images", len(images))

	ip := &ImagePacker{
		Images:  images,
		Sprites: []*Sprite{},
	}

	if err := ip.CreateImage(path.Join(*flagOutputFolder, "spritesheet.png")); err != nil {
		log.Fatal(err)
	}

	spritesByID := map[int]*Sprite{}
	for _, sprite := range ip.Sprites {
		spritesByID[sprite.ID] = sprite
	}

	for _, frame := range frames {
		for _, change := range frame.Changes {
			sprite := spritesByID[change.ID]
			change.X1 = sprite.X1
			change.Y1 = sprite.Y1
			change.X2 = sprite.X2
			change.Y2 = sprite.Y2
		}
	}

	// write frames into compressed format

	jsonframes := []interface{}{}
	for _, frame := range frames {
		jsonchanges := [][]int{}
		for _, change := range frame.Changes {
			jsonchange := []int{
				change.X,
				change.Y,
				change.X1,
				change.Y1,
				change.X2,
				change.Y2,
			}
			jsonchanges = append(jsonchanges, jsonchange)
		}

		jsonframe := []interface{}{
			frame.Timestamp,
			jsonchanges,
		}
		jsonframes = append(jsonframes, jsonframe)
	}

	data, err := json.Marshal(jsonframes)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile(path.Join(*flagOutputFolder, "data.json"), data, 0644); err != nil {
		log.Fatal(err)
	}

	log.Printf("done!")
}
