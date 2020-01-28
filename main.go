package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"image/color"
	"image/jpeg"
	"image/png"
)

const (
	timeout = 5 * time.Second
)

var (
	address string

	origin = os.Getenv("ORIGIN_SERVER")
)

type convertedImage struct {
	image  image.Image
	format string
}

type imageConverter func(*http.Response, error) conversionResult

type conversionResult struct {
	converted convertedImage
	err       error
}

type originResponseError struct {
	err string
}

func (e originResponseError) Error() string {
	return e.err
}

func init() {
	host := os.Getenv("APP_HOST")
	port, err := strconv.Atoi(os.Getenv("APP_PORT"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if origin == "" {
		fmt.Println("Environment variable ORIGIN_SERVER is empty - please set in order to proxy requests")
		os.Exit(1)
	}
	address = fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("Starting server at %s\n", address)
	fmt.Printf("Requests will be forwarded to %s\n", origin)
}

func main() {
	server := &http.Server{
		Addr:         address,
		Handler:      http.HandlerFunc(handleGetImage),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	fmt.Println(server.ListenAndServe())
}

func handleGetImage(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r, err := http.NewRequest("GET", fmt.Sprintf("%s%s", origin, req.URL.Path), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}

	result, err := convertWithContext(ctx, r, convert)
	if err != nil {
		w.WriteHeader(http.StatusRequestTimeout)
		fmt.Fprint(w, err)
		return
	}

	if result.err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, result.err)
		return
	}

	if err = writeConvertedImage(w, result.converted.image, result.converted.format); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
	}
}

func convert(res *http.Response, err error) conversionResult {
	if err != nil {
		return conversionResult{err: err}
	}
	if res.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return conversionResult{err: originResponseError{err: err.Error()}}
		}
		return conversionResult{err: originResponseError{err: string(b)}}
	}
	defer res.Body.Close()

	img, format, err := image.Decode(res.Body)
	if err != nil {
		return conversionResult{err: err}
	}

	return conversionResult{
		converted: convertedImage{
			image:  grayscale(img),
			format: format,
		},
		err: nil,
	}
}

func grayscale(old image.Image) image.Image {
	bounds := old.Bounds()
	new := image.NewRGBA(bounds)

	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			new.Set(x, y, color.Gray16Model.Convert(old.At(x, y)))
		}
	}

	return new
}

func convertWithContext(ctx context.Context, req *http.Request, convert imageConverter) (conversionResult, error) {
	done := make(chan conversionResult)
	req = req.WithContext(ctx)
	go sendOriginRequest(req, convert, done)
	select {
	case <-ctx.Done():
		var c conversionResult
		return c, ctx.Err()
	case r := <-done:
		return r, nil
	}
}

func sendOriginRequest(req *http.Request, convert imageConverter, done chan conversionResult) {
	// time.Sleep(5 * time.Second) // simple test for context timeouts
	done <- convert(http.DefaultClient.Do(req))
}

func writeConvertedImage(w http.ResponseWriter, img image.Image, format string) error {
	buf := new(bytes.Buffer)
	switch format {
	case "jpeg":
		if err := jpeg.Encode(buf, img, nil); err != nil {
			return err
		}
	case "png":
		if err := png.Encode(buf, img); err != nil {
			return err
		}
	}
	w.Header().Set("Content-Type", fmt.Sprintf("image/%s", format))
	w.Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}
